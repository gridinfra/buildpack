# Provider Runtime 与 Expose 元数据增强

## 背景与目标

当前构建结果通过 `BuildResult.Metadata map[string]string` 输出元数据（`/Users/yaozaiyong/Downloads/buildpack/core/core.go:36-44`），语言 Provider 按固定顺序检测并选择第一个命中的实现（`/Users/yaozaiyong/Downloads/buildpack/core/providers/provider.go:33-50`）。部分 Provider 已写入私有字段，例如 Node 的 `nodeRuntime`、Python 的 `pythonRuntime`，但调用方缺少统一字段来识别实际框架及其默认监听端口。

本次增强统一新增：

- `metadata.runtime`：检测到框架时写规范化框架名；未检测到框架时写语言/运行时名。
- `metadata.expose`：仅当 Provider 或框架存在可靠、稳定且有仓库实现依据的默认监听端口时写入十进制端口字符串。

示例：

```json
{
  "metadata": {
    "providers": "node",
    "runtime": "nextjs",
    "expose": "3000",
    "nodeRuntime": "next",
    "nodePackageManager": "npm"
  }
}
```

普通 Node 项目应至少输出：

```json
{
  "metadata": {
    "providers": "node",
    "runtime": "nodejs"
  }
}
```

普通 Node 不设置 `expose`，因为任意 Node 服务没有统一默认监听端口。

## 需求场景与处理逻辑

### 场景一：检测到明确框架

Provider 完成 `Initialize` 和 `Plan` 后，向核心返回统一的运行时元数据。核心将非空值写入 `ctx.Metadata`：

```go
metadata := providerToUse.Metadata(ctx)
ctx.Metadata.Set("runtime", metadata.Runtime)
ctx.Metadata.Set("expose", metadata.Expose)
```

例如：

- Next.js：`runtime=nextjs`、`expose=3000`。
- Vite SPA：`runtime=vite`、`expose=80`。
- Django：`runtime=django`、`expose=8000`。
- Laravel：`runtime=laravel`、`expose=80`。

### 场景二：检测到语言但未检测到框架

写入稳定的语言/运行时名称：

- Node：`nodejs`
- Python：`python`
- Go：`go`
- Java：`java`
- Ruby：`ruby`
- 其他 Provider 使用其规范运行时名。

若没有可靠的通用 Web 端口，不写 `expose`。`Metadata.Set` 已忽略空字符串（`/Users/yaozaiyong/Downloads/buildpack/core/generate/metadata.go:13-19`），因此无需写占位值或 `0`。

### 场景三：没有检测到 Provider

保持现状：不写 `runtime` 和 `expose`。本次不改变 `DetectedProviders` 的现有行为，也不调整 Provider 选择优先级。

### 场景四：显式指定 Provider

统一元数据描述实际执行的 `providerToUse`，而不是仅描述自动检测的 Provider。现有 `providers` 和 `DetectedProviders` 语义保持不变，避免扩大兼容性变更范围。

### 场景五：用户覆盖启动命令或 `PORT`

`expose` 表达框架/Provider 生成逻辑的默认端口，不解析任意 Procfile、用户自定义 start command 或运行时环境中的 `PORT`。本次不修改构建计划、启动命令、OCI `ExposedPorts` 或端口映射行为。

## 架构与技术方案

### 1. 扩展 Provider 契约

在 `/Users/yaozaiyong/Downloads/buildpack/core/providers/provider.go` 增加统一值对象并扩展接口：

```go
type ProviderMetadata struct {
    Runtime string
    Expose  string
}

type Provider interface {
    Name() string
    Detect(ctx *generate.GenerateContext) (bool, error)
    Initialize(ctx *generate.GenerateContext) error
    Plan(ctx *generate.GenerateContext) error
    Metadata(ctx *generate.GenerateContext) ProviderMetadata
    CleansePlan(buildPlan *plan.BuildPlan)
    StartCommandHelp() string
}
```

设计理由：

- 强制所有语言 Provider 明确声明统一 runtime，避免核心层按具体 Provider 类型做 `switch`。
- 框架检测继续封装在各 Provider 内，复用已经存在的 `isNext`、`getRuntime`、`usesRails`、`getRuntime` 等逻辑。
- 返回值不直接修改 map，便于单元测试和后续扩展。
- `Expose` 保持字符串，与现有 `map[string]string` API 兼容。

### 2. 核心层统一落盘

在 `/Users/yaozaiyong/Downloads/buildpack/core/core.go` 中，于 `providerToUse.Plan(ctx)` 成功后调用 `Metadata(ctx)`，并通过现有 `Metadata.Set` 写入：

```go
if providerToUse != nil {
    if err := providerToUse.Plan(ctx); err != nil {
        // 保持现有错误处理
    }

    providerMetadata := providerToUse.Metadata(ctx)
    ctx.Metadata.Set("runtime", providerMetadata.Runtime)
    ctx.Metadata.Set("expose", providerMetadata.Expose)
}
```

放在 `Plan` 之后可确保 Provider 已完成初始化和框架判定所依赖的状态准备。已有 `nodeRuntime`、`pythonRuntime`、`javaFramework`、布尔框架字段继续由原逻辑写入，不删除、不改名。

### 3. Runtime 与 Expose 映射

以下映射优先复用仓库已有检测函数。端口只采用代码模板、自动生成启动命令或现有集成测试中明确的默认端口。

| Provider | 检测结果 | `runtime` | `expose` | 依据/说明 |
|---|---|---:|---:|---|
| PHP | Laravel | `laravel` | `80` | `usesLaravel`; Caddy `{$PORT:80}` |
| PHP | 普通 PHP | `php` | `80` | Provider 使用默认 Caddy 配置 |
| Go | Gin | `gin` | 不写 | 复用 `isGin`; 应用监听端口由代码决定 |
| Go | 普通 Go | `go` | 不写 | 无统一监听端口 |
| Java | Spring Boot | `spring-boot` | 不写 | 复用现有框架检测；启动参数使用 `$PORT`，仓库未固定 fallback |
| Java | 普通 Java | `java` | 不写 | 无统一监听端口 |
| Rust | Rust | `rust` | 不写 | 当前没有可靠框架检测和统一端口 |
| Ruby | Rails | `rails` | `3000` | 复用 `usesRails`; 自动 Rails server 命令 |
| Ruby | 普通 Ruby | `ruby` | 不写 | 任意 Ruby 应用无统一端口 |
| Elixir | Phoenix | `phoenix` | `4000` | 仅在现有依赖/项目结构可明确识别 Phoenix 时设置 |
| Elixir | 普通 Elixir | `elixir` | 不写 | 无统一端口 |
| Python | Django | `django` | `8000` | 复用 `getRuntime`; 自动 gunicorn 默认绑定 |
| Python | Flask | `flask` | `8000` | 复用 `getRuntime` |
| Python | FastAPI | `fastapi` | `8000` | 复用 `getRuntime` |
| Python | FastHTML | `fasthtml` | `8000` | 复用 `getRuntime` |
| Python | 普通 Python | `python` | 不写 | 无统一监听端口 |
| Deno | Deno | `deno` | 不写 | 应用代码决定端口 |
| .NET | .NET | `dotnet` | `3000` | Provider 设置 `ASPNETCORE_URLS=http://[::]:${PORT:-3000}` |
| Node | Next.js | `nextjs` | `3000` | 现有 `nodeRuntime=next`; Next 集成样例端口 |
| Node | Nuxt | `nuxt` | `3000` | 现有框架检测及集成样例 |
| Node | Remix | `remix` | `3000` | 现有框架检测及集成样例 |
| Node | TanStack Start | `tanstack-start` | `3000` | 现有框架检测及集成样例 |
| Node | Astro server | `astro` | `4321` | 现有框架检测及集成样例 |
| Node | React Router SSR | `react-router` | `3000` | 现有框架检测及集成样例 |
| Node | SPA（Vite、CRA、Angular、Expo、静态导出 Next、React Router SPA、Astro static） | 对应框架名；Next 使用 `nextjs` | `80` | SPA Caddy 模板使用 `{$PORT:80}` |
| Node | Bun（非其他框架） | `bun` | 不写 | Bun 应用本身无统一端口 |
| Node | 普通 Node | `nodejs` | 不写 | 用户要求的默认 fallback |
| Gleam | Gleam | `gleam` | 不写 | 无统一端口 |
| C++ | C++ | `cpp` | 不写 | 无统一端口 |
| Staticfile | 静态文件 | `staticfile` | `80` | Caddy 默认端口 |
| Shell | Shell | `shell` | 不写 | 脚本行为未知 |

Node 的统一值与现有 `nodeRuntime` 有意区分：

```go
func normalizeNodeRuntime(runtime string) string {
    switch runtime {
    case "next":
        return "nextjs"
    case "node":
        return "nodejs"
    default:
        return runtime
    }
}
```

不修改现有 `getRuntime()` 返回值，避免破坏既有 `nodeRuntime=next/node` 的兼容性。

### 4. 避免重复和昂贵检测

- Node、Python 的 `Metadata` 复用现有 `getRuntime(ctx)`，然后基于结果计算端口。
- PHP、Go、Ruby、Java 复用当前写私有 metadata 时已经使用的检测函数。
- 不新增全仓库文件内容扫描；如果某 Provider 的框架检测目前只存在于 `Plan` 局部变量，应将结果缓存到 Provider 字段或提取现有轻量判断，而不是重复遍历项目。
- `Metadata` 不返回错误，因为其依据均来自已完成的初始化状态和容错型检测方法；Provider 的读取/解析错误仍由 `Initialize`/`Plan` 负责。

## 数据流

```text
GenerateBuildPlan
  -> getProviders
      -> Detect
      -> Initialize
  -> providerToUse.Plan(ctx)
      -> 保留现有 provider-specific metadata
  -> providerToUse.Metadata(ctx)
      -> ProviderMetadata{Runtime, Expose}
  -> ctx.Metadata.Set("runtime", ...)
  -> ctx.Metadata.Set("expose", ...)
  -> ctx.Generate()
  -> BuildResult.Metadata = ctx.Metadata.Properties
  -> JSON 输出 metadata.runtime / metadata.expose
```

## 影响文件

### 核心契约与调用链

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/provider.go`
  - 新增 `ProviderMetadata`。
  - `Provider` 接口新增 `Metadata` 方法。
- `/Users/yaozaiyong/Downloads/buildpack/core/core.go`
  - `GenerateBuildPlan` 在 Provider 计划成功后写统一 `runtime`/`expose`。
- `/Users/yaozaiyong/Downloads/buildpack/core/core_test.go`
  - 增加最终 `BuildResult.Metadata` 的框架、fallback、端口及缺省行为断言。

### Provider 实现

以下现有 Provider 文件增加 `Metadata` 方法；只在必要时复用或提取已有框架检测：

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/php/php.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/golang/golang.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/java/java.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/rust/rust.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/ruby/ruby.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/elixir/elixir.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/python/python.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/deno/deno.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/dotnet/dotnet.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/gleam/gleam.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/cpp/cpp.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/staticfile/staticfile.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/shell/shell.go`

### Provider 测试

优先扩展现有表驱动测试，不新建无必要 fixture：

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node_test.go`
  - Next.js：`nextjs/3000`。
  - Next static 或 Vite SPA：对应 runtime/`80`。
  - 普通 Node：`nodejs`/空 expose。
  - Astro server：`astro/4321`。
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/python/python_test.go`
  - Django/Flask/FastAPI/FastHTML 为 `8000`，普通 Python为空。
- 其他已有 Provider 测试文件
  - 至少覆盖有框架分支或固定端口的 PHP、Ruby、Elixir、.NET、Staticfile。
  - 基础 Provider 覆盖 runtime fallback 与空 expose。

如现有测试文件不存在，优先在最接近的已有测试中覆盖；仅当无法合理覆盖时新增对应 `_test.go`（测试代码属于实现所必需）。

## 边界条件与异常处理

1. **空 runtime**：所有注册在 `GetLanguageProviders` 的实现必须返回非空 runtime；测试逐个验证接口实现。
2. **无默认端口**：返回空字符串，由 `Metadata.Set` 跳过；禁止写 `0`、`unknown` 或猜测端口。
3. **SPA 与 SSR 同框架**：先判断实际 SPA 模式。SPA 端口为 80；SSR 使用框架服务端默认值。例如 Next static 为 80，Next server 为 3000。
4. **框架重叠**：沿用现有 Provider 内部优先级，不引入第二套检测顺序。例如 React Router + Vite 保持识别为 React Router。
5. **显式 start command/Procfile**：不尝试解析命令推导端口，也不覆盖 runtime 框架判断。
6. **Provider Plan 失败**：保持现有失败返回，不输出成功 BuildResult，也不额外调用统一 Metadata。
7. **兼容性**：不改变 `BuildResult.Metadata` 类型，不删除旧 metadata，不改 `DetectedProviders`，不改变 JSON 字段结构。
8. **安全与性能**：不执行项目脚本、不运行依赖代码；只使用现有静态文件/依赖检测。避免新增递归全项目扫描。

## 验证方案

遵循仓库约定，不直接调用 `go`：

1. 运行格式、静态检查与单元测试入口：

```bash
mise run check
```

2. 运行 core 与 Provider 相关测试（使用仓库 `mise` 任务；若没有细粒度任务，则使用项目约定的测试任务）。
3. 验证代表性已有 examples：
   - `node-next`
   - 一个 Node SPA（如 `node-vite-react` 或 `node-next-spa`）
   - `python-django`
   - `php-laravel-*`
4. 不更新 BuildPlan snapshot，除非测试证明统一 Metadata 已被纳入 snapshot；当前 snapshot 只比较 `Plan`，预计无需变化。

## 预期结果

- 所有成功选择的语言 Provider 均输出统一 `metadata.runtime`。
- 检测到 Next.js 时输出 `runtime=nextjs`；普通 Node 输出 `runtime=nodejs`。
- 有可靠默认端口的框架/Provider 输出字符串形式 `metadata.expose`。
- 无可靠默认端口时不出现 `expose`。
- 现有 provider-specific metadata、构建计划、启动命令、Provider 检测顺序及 JSON 结构保持兼容。
