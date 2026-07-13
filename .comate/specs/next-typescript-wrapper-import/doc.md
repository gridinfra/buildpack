# Next.js Standalone Config Script

## 目标

废弃 `getNextStandaloneWrapper` 及原配置重命名方案。改为在 Next build 前执行一个临时 Node.js 脚本，直接修改构建层中的 `next.config.js`、`next.config.mjs` 或 `next.config.ts`，设置：

```js
output: "standalone"
```

修改只发生在 BuildKit build step 的临时文件系统，不修改用户宿主工作区。

## 当前问题

wrapper 方案会让 `next.config.ts` 生成以下 import：

```ts
import originalConfig from "./next.config.railpack-original.ts";
```

未启用 `allowImportingTsExtensions` 时会触发 TS5097。即使移除扩展名，wrapper 仍会改变原配置的模块加载方式，并增加 ESM、CommonJS、TypeScript 和 Next 版本兼容成本。

## 新方案

### 1. 生成临时配置修改脚本

在 `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next.go` 中提供脚本内容常量或生成函数。脚本使用 Node 标准库 `fs`、`path`，接收 Next 配置文件路径：

```text
node /railpack/scripts/configure-next-standalone.mjs <config-path>
```

脚本执行逻辑：

1. 读取目标配置文件。
2. 忽略注释后识别配置形态。
3. 若存在静态 `output: "..."` 或 `output: '...'`，原地替换为 `output: "standalone"`。
4. 若存在常见配置对象声明，在对象起始位置注入 `output: "standalone",`：
   - `const nextConfig = { ... }`
   - `let nextConfig = { ... }`
   - `var nextConfig = { ... }`
   - 带 TypeScript 类型的 `const nextConfig: NextConfig = { ... }`
   - `export default { ... }`
   - `module.exports = { ... }`
5. 若没有 Next 配置文件，生成最小 `next.config.mjs`。
6. 写回配置文件并输出明确日志。
7. 若配置是函数、插件调用、变量引用、对象 spread 组合且无法安全定位导出的对象，脚本以非零状态退出并说明不支持的配置形式；不得覆盖整个配置或静默继续。

### 2. Build step 顺序

`NodeProvider.Plan` 中 standalone build 命令顺序调整为：

```text
1. 写入 configure-next-standalone.mjs asset
2. node /railpack/scripts/configure-next-standalone.mjs <next-config-path>
3. 执行默认或用户自定义 build command
4. 整理 standalone、static、public 到固定部署根
```

删除：

- 原配置 `mv` 命令。
- `NextStandaloneConfigAsset` wrapper asset。
- `getNextOriginalConfigPath`。
- `getNextStandaloneWrapper`。
- `PackageJson.Type` 字段（若不再有其他调用方）。

新增独立脚本 asset，例如：

```text
NextStandaloneConfigScriptAsset = "configure-next-standalone.mjs"
```

脚本写到 `/railpack/scripts/configure-next-standalone.mjs`，避免与用户应用文件冲突。

### 3. SPA 与动态 output

- 明确静态 `output: "export"` 的项目继续走 SPA，不执行 standalone 修改脚本。
- `RAILPACK_SPA_OUTPUT_DIR` 强制 SPA 时不执行脚本。
- 动态 `output` 无法在 Plan 阶段可靠判断时，不应自动进入 standalone；继续保持保守策略，避免反转用户配置。
- 普通非 SPA Next 项目执行脚本。

### 4. 安全与边界

- 配置路径由 Go 端作为独立命令参数进行 shell quote。
- 脚本只允许修改 `next.config.js`、`next.config.mjs`、`next.config.ts`。
- 脚本读取和写入失败必须返回非零退出码。
- 脚本在写入前确认只发生一次目标替换或注入，匹配多个候选对象时失败。
- 不引入 JavaScript AST 依赖，不修改 package.json、tsconfig.json 或 lockfile。
- 不在最终镜像中部署临时脚本或修改后的源码，只部署 `/railpack/next-standalone`。

## 影响文件

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next.go`
  - 删除 wrapper helper。
  - 增加配置修改脚本内容与命令 helper。
  - 保留 Next 应用定位、SPA 检测和 standalone 产物整理。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go`
  - build 前写入并执行配置修改脚本。
  - 删除原配置重命名和 wrapper 写入命令。
  - 保持自定义 build 命令的前后顺序约束。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/package_json.go`
  - 删除仅供 wrapper 判断模块类型使用的 `Type` 字段。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next_test.go`
  - 删除 wrapper 测试。
  - 增加脚本内容、命令路径和配置转换测试。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node_test.go`
  - 更新计划级 asset 和命令顺序断言。

- 相关 Next build plan snapshots。

## 验证

1. 使用临时目录实际执行生成的 Node 脚本，验证：
   - `next.config.ts` 对象配置。
   - `.mjs` ESM 对象配置。
   - `.js` CommonJS 对象配置。
   - 已有 `output` 被替换。
   - 无配置时生成最小配置。
   - 动态/插件配置明确失败且原文件不变。
2. `mise run check`。
3. Node provider 单元测试与全部 Next snapshot。
4. `mise run cli -- plan examples/node-next`、Next SPA、workspace 计划检查。
5. BuildKit/GHCR 网络允许时执行实际 Next image build。

## 预期结果

- `next.config.ts` 不再产生额外 import，因此彻底消除 TS5097。
- build 前配置文件被直接设置为 standalone。
- 常见对象配置保持原有其他选项。
- 不支持的动态配置明确失败，不产生不可启动镜像。
- standalone 部署根、启动命令、SPA 和自定义 build 行为保持不变。
