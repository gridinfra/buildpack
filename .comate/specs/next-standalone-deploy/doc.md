# Next.js Standalone Deployment

## 背景与目标

`NodeProvider.Plan` 当前将 Node 构建步骤作为部署输入，并对普通 Next.js 使用包管理器脚本或 `next start` 启动。由于构建步骤包含完整应用、开发依赖和 `node_modules`，非静态 Next.js 的运行镜像明显偏大。

本变更仅针对满足以下条件的项目：

- `package.json` 依赖中包含 `next`；
- 项目不是 Next.js SPA/static export，即 `isNextSPA(ctx) == false`；
- 没有用户通过环境变量显式覆盖启动命令。

目标是在构建时启用 Next.js `output: "standalone"`，运行镜像仅部署 standalone 服务端产物、`.next/static` 和可选 `public`，并通过 standalone 生成的 `server.js` 启动。Next.js SPA、CRA、Vite、普通 Node 项目和用户显式启动命令保持原行为。

## 当前处理链路

`NodeProvider.Plan` 位于 `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go`：

1. 校验并读取 `package.json`，设置 Node 元数据。
2. 计算 `isSPA := p.isSPA(ctx)`。
3. 创建 mise、install、build 步骤。
4. build 步骤输入完整本地应用与 install 层，并执行 build script。
5. SPA 时部署静态输出目录并使用 Caddy。
6. 非 SPA 时把完整 build 层加入 `DeployInputs`，再由 `getStartCommand` 选择 Procfile、环境变量、package script 或框架默认命令。

Next.js 辅助逻辑位于 `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next.go`：

- `isNextSPA` 根据 Next 配置中的 `output: "export"`、`next export` 脚本等判断静态导出。
- `getNextOutputDirectory` 仅服务 SPA，默认返回 `out`。
- `getNextStartCommand` 当前默认返回 `next start`。

## 设计方案

### 1. 精确区分 standalone Next.js

在 `Plan` 中复用一次框架判断：

```go
isNextStandalone := p.isNext() && !isSPA
```

该判断发生在 build/deploy/start command 规划之前，确保三个环节使用相同条件，避免“生成 standalone 但仍部署全量目录”或“部署精简目录但仍执行 next start”的不一致。

### 2. 构建前设置 Next output

在 `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next.go` 增加 Next standalone 配置命令生成逻辑，并在 `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go` 的 build commands 中，将该命令放在用户 build command 之前。

处理原则：

- 不修改宿主机上的源文件；修改只发生在 BuildKit build step 的临时文件系统。
- 已存在 `next.config.js`、`next.config.mjs` 或 `next.config.ts` 时，向配置对象注入/覆盖 `output: "standalone"`。
- 没有 Next 配置文件时，生成最小 `next.config.mjs`：

```js
const nextConfig = { output: "standalone" };
export default nextConfig;
```

- 配置文件位于 `package.json` 对应的应用目录；根应用为 `/app`，workspace 应用为 `/app/<package-path>`。
- 保留用户其他 Next 配置。对于常见的对象导出形式（`export default nextConfig`、`module.exports = nextConfig`、直接对象导出），将 output 写入导出的对象。
- 若现有配置为函数、条件分支或其他无法可靠静态改写的动态形式，不静默替换整个配置；规划应返回明确错误，指出配置文件及不支持的导出形式，防止构建出行为错误的镜像。
- 若用户已配置 `output: "export"`，应继续由 `isNextSPA` 路径处理，不覆盖为 standalone。
- 若用户已配置其他 `output`，非 SPA 路径统一覆盖为 `standalone`，因为部署输入和启动方式依赖该产物契约。

实现应优先采用小范围、可测试的 Go 字符串转换函数生成最终配置内容，再通过 shell heredoc 在 build step 内写入；不引入 JavaScript AST 依赖。转换函数必须对注释、空对象、多行对象和单双引号 output 属性有覆盖测试。

### 3. standalone 产物与目录语义

Next build 默认生成：

```text
<app>/.next/standalone/
  server.js                         # 单应用
  <workspace-path>/server.js        # monorepo/workspace 应用
  node_modules/                     # traced runtime dependencies
  package.json
```

`.next/static` 和 `public` 不会自动复制进 standalone 目录，因此部署时必须额外加入。

为让运行镜像目录与 Next 生成的相对路径一致，`DeployInputs` 调整为：

- standalone 根目录：从 build 层复制 `<app>/.next/standalone`，使用 `Spread: true` 展开到 `/app`；
- 静态资源：复制 `<app>/.next/static` 到 standalone server 对应应用目录下的 `.next/static`；
- public：存在时复制 `<app>/public` 到 standalone server 对应应用目录下的 `public`。

单应用的运行目录：

```text
/app/server.js
/app/.next/static
/app/public
```

workspace 应用（例 `apps/web`）的运行目录：

```text
/app/apps/web/server.js
/app/apps/web/.next/static
/app/apps/web/public
```

workspace 模式保留 standalone 根部可能包含的共享 traced 文件和根 `node_modules`，不能只复制 `<workspace-path>` 子目录。

部署输入使用 `plan.NewStepLayer(buildStep.Name, ...)` 的 include/rename/spread 能力表达，不额外运行复制命令；可选 `public` 仅在源应用中存在时加入，避免不存在路径导致构建失败。

### 4. 启动命令

在 `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go` 调整启动命令选择：

- `RAILPACK_START_CMD` 等显式环境变量仍拥有最高优先级，不强制改写用户命令。
- 没有显式覆盖时，非 SPA Next.js 使用 standalone server：
  - 单应用：`node server.js`
  - workspace 应用：`node <package-path>/server.js`
- 不再使用 `next start` 或 package manager 的 `start` script，因为精简镜像中不保证存在 package manager CLI 或完整依赖树。
- 启动命令使用 POSIX 路径拼接，且路径需经过命令参数安全引用，避免 workspace 路径中的特殊字符被 shell 解释。
- Next standalone server 原生读取 `PORT` 和 `HOSTNAME`；保持现有运行时环境传递机制。

为保持优先级清晰，新增专用 helper（例如 `getNextStandaloneStartCommand`），并在通用 `getStartCommand` 的用户覆盖逻辑之后、package script/framework fallback 之前调用。

### 5. `DeployInputs` 调整

非 SPA 分支拆分：

```go
switch {
case isSPA:
    // 现有 Caddy 静态部署
case isNextStandalone:
    // 仅 standalone + static + optional public
case buildStep != nil:
    // 其他 Node 项目继续部署完整 build 层
}
```

若没有 build script/build step，则不能产生 `.next/standalone`，规划阶段返回明确错误。普通 Node 项目仍允许无 build step 并部署 install/local 输入，行为不变。

### 6. 数据流

```text
package.json + Next config
  -> Initialize: 识别 Next、workspace 与应用路径
  -> Plan: isSPA / isNextStandalone
  -> build step 临时改写 Next config
  -> next build
  -> <app>/.next/standalone + <app>/.next/static + optional public
  -> DeployInputs 精确复制并保持应用相对目录
  -> node [<workspace>/]server.js
```

## 影响文件

### 修改

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go`
  - `NodeProvider.Plan`：计算 standalone 条件；在 build 前注入配置；调整 `DeployInputs`；校验 build step。
  - `NodeProvider.getStartCommand`：在保留显式环境变量优先级的前提下选择 standalone 启动命令。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next.go`
  - 增加 standalone 常量、应用目录/产物路径 helper、配置转换及构建命令生成、standalone 启动命令。
  - 保持现有 SPA 检测与静态输出逻辑。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node_test.go`
  - 增加 Next standalone 规划测试：构建命令顺序、部署输入、启动命令。
  - 增加单应用、SPA、workspace 路径和显式启动命令覆盖测试。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next_test.go`（若不存在则新增，属于必要测试文件）
  - 表驱动测试配置转换、配置文件选择、路径与启动命令。

### 测试夹具

优先复用：

- `/Users/yaozaiyong/Downloads/buildpack/examples/node-next`
- `/Users/yaozaiyong/Downloads/buildpack/examples/node-next-spa`
- `/Users/yaozaiyong/Downloads/buildpack/examples/node-turborepo`

仅当现有夹具无法表达动态配置或缺失配置场景时，才新增最小测试夹具。

## 边界条件与异常处理

- Next SPA：保持 `output: export`、Caddy 和 `out` 部署，不注入 standalone。
- 无 Next 配置：在构建层生成配置，不污染源码工作区。
- 多个 Next 配置文件：沿用 `nextConfigFiles` 顺序选择首个，并保持与 SPA 检测一致；测试固定该行为。
- 动态/不可解析配置：返回可操作错误，不覆盖用户配置。
- 无 build script：返回错误，避免部署阶段引用不存在的 standalone 产物。
- 无 `public`：不加入该输入；`.next/static` 是 Next build 必需产物并始终加入。
- workspace：启动路径指向 workspace server.js，static/public 放在同一 workspace 相对目录；standalone 根目录整体展开以保留共享依赖。
- 自定义 Next `distDir`：standalone 与 static 实际根目录会变化。现有代码不解析该选项；本次应从配置转换结果识别静态字符串 `distDir` 并据此计算路径。动态 `distDir` 与动态配置一样返回明确错误，避免目录错误。
- 自定义 `outputFileTracingRoot`：Next 会把根结构编码进 standalone 产物；整体展开 standalone 根目录可保留该结构，但启动 server 的相对路径必须由应用 workspace 路径决定并通过测试验证。
- 用户显式 start command：继续执行用户命令；即使构建产物为 standalone，也不改变既有覆盖约定。

## 验证方案

1. Go 单元测试：
   - `go test ./core/providers/node/...`
   - 配置注入/覆盖与拒绝动态配置。
   - SPA 不受影响。
   - 单应用与 workspace 的部署 Layer include/spread/rename。
   - standalone start command 与显式覆盖优先级。

2. 构建计划快照/断言：
   - `examples/node-next` 只部署 standalone、static、public，不部署完整 build layer。
   - `examples/node-next-spa` 仍使用 Caddy 和 `out`。
   - `examples/node-turborepo` 的 server/static/public 路径保持 `apps/web` 相对结构。

3. 集成构建（仓库现有工具可用时）：
   - 实际构建 `examples/node-next` 镜像。
   - 检查镜像包含 `/app/server.js`、`/app/.next/static`、`/app/public`，且不包含完整开发依赖目录。
   - 启动容器并请求页面/静态资源，确认 server 与资源路径正确。
   - 实际构建 workspace Next 示例并检查 `/app/apps/web/server.js` 及资源请求。

## 预期结果

- 非 SPA Next.js 自动生成 standalone 产物并以 `node server.js` 运行。
- 运行镜像不再携带完整源代码和开发依赖，镜像体积显著降低。
- 单应用与 workspace 的 server、static、public 路径正确。
- SPA 与其他 Node 框架行为无回归。
- 无法可靠生成 standalone 的配置在规划/构建前明确失败，而不是生成不可启动镜像。
