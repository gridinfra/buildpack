# Provider Runtime 与 Expose 元数据增强总结

## 完成内容

### 统一 Provider 元数据契约

- 在 `core/generate/metadata.go` 新增 `ProviderMetadata`：

```go
type ProviderMetadata struct {
    Runtime string
    Expose  string
}
```

- 在 `core/providers/provider.go` 的 `Provider` 接口新增：

```go
Metadata(ctx *generate.GenerateContext) generate.ProviderMetadata
```

数据类型放在 `generate` 包而不是 `providers` 包，以避免 `providers -> node -> providers` 等 Go 循环依赖。

### 核心 BuildResult 写入

`GenerateBuildPlan` 在实际 Provider 的 `Plan` 成功后读取统一元数据并写入：

```go
providerMetadata := providerToUse.Metadata(ctx)
ctx.Metadata.Set("runtime", providerMetadata.Runtime)
ctx.Metadata.Set("expose", providerMetadata.Expose)
```

因此最终 `BuildResult.Metadata` 会输出：

- `runtime`：规范化框架或运行时名称。
- `expose`：有可靠默认端口时输出字符串端口；无可靠端口时省略。

未改变：

- `BuildResult.Metadata map[string]string` 类型。
- Provider 检测顺序。
- `providers` 和 `DetectedProviders` 现有语义。
- 构建计划、启动命令和 OCI 镜像配置。
- `nodeRuntime`、`pythonRuntime`、`javaFramework` 等既有元数据字段。

### Provider 映射

已覆盖 `GetLanguageProviders` 中全部 14 个 Provider：

| Provider/框架 | runtime | expose |
|---|---|---|
| Next.js server | `nextjs` | `3000` |
| Next.js static | `nextjs` | `80` |
| Nuxt/Remix/TanStack Start | 对应框架名 | `3000` |
| React Router SSR | `react-router` | `3000` |
| Node SPA | 对应框架名 | `80` |
| Astro server | `astro` | `4321` |
| 普通 Node | `nodejs` | 省略 |
| Bun | `bun` | 省略 |
| Django/Flask/FastAPI/FastHTML | 对应框架名 | `8000` |
| 普通 Python | `python` | 省略 |
| Laravel | `laravel` | `80` |
| 普通 PHP | `php` | `80` |
| Rails | `rails` | `3000` |
| 普通 Ruby | `ruby` | 省略 |
| Phoenix | `phoenix` | `4000` |
| 普通 Elixir | `elixir` | 省略 |
| .NET | `dotnet` | `3000` |
| Staticfile | `staticfile` | `80` |
| Gin | `gin` | 省略 |
| 普通 Go | `go` | 省略 |
| Spring Boot | `spring-boot` | 省略 |
| 普通 Java | `java` | 省略 |
| Rust/Deno/Gleam/C++/Shell | 对应稳定运行时名 | 省略 |

端口仅在仓库模板、自动生成启动命令或已有集成样例提供明确依据时设置，没有为任意 Go、Java、Rust、Node 等应用猜测端口。

### 测试覆盖

新增或扩展测试覆盖：

- Node：Next server/static、Vite SPA、Astro server、普通 Node，以及旧 `nodeRuntime` 保留。
- Python：Django、Flask、FastAPI、FastHTML、普通 Python，以及旧 `pythonRuntime` 保留。
- PHP/Laravel、Ruby/Rails、Elixir/Phoenix、.NET、Staticfile。
- Go runtime fallback 与空 expose。
- 最终 `BuildResult.Metadata` 中的 `runtime`、`expose` 和旧字段兼容性。
- 显式 Provider 使用实际执行 Provider 的统一元数据。

## 验证结果

### 通过

- `mise run check`
  - `go vet ./...`
  - `go fmt ./...`
  - `golangci-lint run`：0 issues
  - `go mod verify`：通过
- `mise run test`
  - 全量短测试通过。
  - `core`、全部 Provider、BuildKit、CLI、内部工具及短模式 integration_tests 包均通过。
- core 元数据定向测试通过。
- Node、Python、PHP、Ruby、Elixir、.NET、Staticfile、Go 与 Provider 包定向测试通过。
- `git diff --check` 通过。
- BuildPlan snapshot 无变化。

### 环境限制

尝试运行以下容器集成样例：

- `node-next`
- `python-django`
- `php-laravel-12-react`

三个样例均正常生成构建计划，但在镜像构建阶段无法连接 BuildKit。尝试执行仓库标准任务 `mise run run-buildkit-container` 时，Docker 报错：

```text
Cannot connect to the Docker daemon at unix:///Users/yaozaiyong/.docker/run/docker.sock.
Is the docker daemon running?
```

用户选择不启动 Docker，因此未继续容器级集成验证。这是本地环境限制，不是代码或构建计划生成失败。

## 工作区说明

以下未跟踪内容不属于本次任务，未修改或删除：

- `.idea/`
- `cli/go_build_github_com_railwayapp_railpack`
- `railpack-info.json`
- `railpack-plan.json`
