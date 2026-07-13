# Next.js Standalone Review Fixes

## 目标

修复本次 standalone 实现 review 中确认的全部问题：

1. `Layer.Spread` 不具备目录展开语义，导致启动命令找不到 `server.js`。
2. `SPA_OUTPUT_DIR` 强制 SPA 时仍进入 standalone 分支。
3. 直接用正则改写 Next 配置兼容性不足，包装式和函数式配置会失败。
4. 自定义 `BUILD_CMD` 存在时仍被 `package.json scripts.build` 校验拒绝。
5. 当前测试未覆盖最终部署目录和计划行为。

## 修复方案

### 1. 使用独立部署根目录

在 build step 执行完 `next build` 后创建固定目录 `/railpack/next-standalone`：

- 将 `<app>/<distDir>/standalone/.` 的内容复制到 `/railpack/next-standalone/`。
- 将 `<app>/<distDir>/static` 复制到 `/railpack/next-standalone/<app>/<distDir>/static`。
- 将可选 `<app>/public` 复制到 `/railpack/next-standalone/<app>/public`。
- DeployInputs 仅 include `/railpack/next-standalone`，不设置 `Spread`。
- 启动命令使用绝对路径：
  - 单应用：`node /railpack/next-standalone/server.js`
  - workspace：`node /railpack/next-standalone/<app>/server.js`

这样不依赖 BuildKit 对 include 目录的重定位，启动路径与实际复制路径完全一致。

### 2. 统一产物类型判定

使用最终 `isSPA := p.isSPA(ctx)` 决定部署模式：

```go
isNextStandalone := nextAppErr == nil && !isSPA
```

因此 `SPA_OUTPUT_DIR`、Next export 等静态场景不会注入 standalone。自定义 start command 导致 `isSPA=false` 时，非 export Next 项目仍进入 standalone；Next export 项目需继续通过 `isNextSPA` 保持静态产物语义，避免把 export 覆盖成 standalone。实现中会显式组合“强制 SPA”和“Next export”结果，而不是重复推导互相冲突的布尔值。

### 3. 用包装配置兼容用户配置

不再对配置对象做正则注入。构建时：

- 将原配置重命名为同目录下的 Railpack 临时文件。
- 创建与原配置格式匹配的 wrapper 配置。
- wrapper 加载原配置，等待 Promise（如有），调用函数配置（如有），最后返回 `{ ...resolvedConfig, output: "standalone" }`。
- 对 `.js` 根据项目 package type 选择 CJS/ESM wrapper；`.mjs` 使用 ESM；`.ts` 使用 TypeScript ESM import。
- 没有配置时生成最小 `next.config.mjs`。
- 构建层是临时文件系统，不修改宿主源码。
- `distDir` 不再依赖静态解析：构建后通过 shell 在应用目录查找 standalone 产物，并要求唯一匹配，从而支持动态 `distDir`。

如果 Next 对某种配置格式无法加载 wrapper，应在 build 阶段产生明确错误，而不是在 Plan 阶段拒绝常见配置。

### 4. 使用最终构建步骤判断

移除 `p.packageJson.HasScript("build")` 前置校验。调用 `p.Build` 并应用用户 config/env override 后，根据 build step 最终 commands 判断是否存在实际构建命令；若没有才返回 standalone 缺少 build command 的错误。

需要遵循 GenerateContext config 合并顺序，确保 `BUILD_CMD`/`--build-cmd` 可以替换 provider 默认命令。

### 5. 测试

新增/调整测试覆盖：

- 单应用计划部署 `/railpack/next-standalone`，启动绝对 `server.js`。
- workspace 启动 `/railpack/next-standalone/apps/web/server.js`。
- Deploy layer 不使用 `Spread`。
- `SPA_OUTPUT_DIR` 保持 Caddy/static 部署且不写 standalone wrapper。
- package.json 无 build script、但存在自定义 `BUILD_CMD` 时允许规划。
- 动态函数配置、Sentry/插件包装配置可通过 wrapper 兼容。
- `.mjs`、CJS `.js`、ESM `.js`、`.ts` wrapper 内容正确。
- build commands 顺序：准备 wrapper → build → 整理 standalone 目录。
- 运行 `go test ./core/providers/node`、相关 BuildKit 测试、`go vet`、`git diff --check`。
- Docker 可用时执行实际镜像验证；不可用则记录环境限制。

## 影响文件

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go`
  - 修正模式判定、build command 校验、部署目录整理、DeployInputs 和启动命令。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next.go`
  - 删除脆弱的对象正则改写和静态 distDir 限制。
  - 增加配置 wrapper、临时配置路径和部署根路径 helper。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next_test.go`
  - 更新 helper 测试并增加 wrapper 格式测试。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node_test.go` 或必要的新测试文件
  - 增加最终 Plan 行为测试。

## 预期结果

- standalone server 的启动路径与镜像实际路径一致。
- 强制 SPA 配置不再被 standalone 覆盖。
- 常见动态、函数和插件包装配置能够保留原逻辑并追加 standalone output。
- 自定义 build command 正常工作。
- 测试能够捕获目录布局、模式选择和命令优先级回归。
