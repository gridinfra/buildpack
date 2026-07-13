# Next.js Standalone Review Fixes 总结

## 修复问题

1. 修复 `Layer.Spread` 被误当作目录展开的问题。
   - 构建后将 standalone 产物整理到固定目录 `/railpack/next-standalone`。
   - DeployInputs 直接 include 该绝对目录，不设置 `Spread`。
   - 启动命令固定为 `node /railpack/next-standalone/server.js`。

2. 修复强制 SPA 与 standalone 同时生效的问题。
   - standalone 判定使用最终 `isSPA` 结果。
   - `RAILPACK_SPA_OUTPUT_DIR` 和 Next export 保持 Caddy/static 部署，不注入 wrapper。

3. 修复直接正则改写 Next 配置的兼容性问题。
   - 原配置在 build step 中改名保留。
   - 新配置作为 wrapper 加载原配置。
   - 支持 ESM、CommonJS、TypeScript、函数配置、Promise 和插件包装结果。
   - wrapper 在解析原配置后覆盖 `output: "standalone"`。

4. 修复自定义 build command 被拒绝或丢失 standalone 命令的问题。
   - 接受 package script 或 `ctx.Config.Steps["build"]` 中的 executable command。
   - 自定义 COPY 之后、首个 executable command 之前注入 wrapper。
   - 用户 build 命令之后固定执行 standalone 整理。
   - 自定义 build 的 deploy output 固定为 `/railpack/next-standalone`，不再额外复制全量应用。

5. 修复 workspace 与 tracing root 下 server 路径不稳定的问题。
   - 精确选择 `standalone/server.js` 或 `standalone/<workspace>/server.js`。
   - 必要时在部署根创建相对符号链接 `server.js`。
   - static/public 放到实际 server 所在目录。
   - 不再递归要求整个 traced 文件树只能有一个名为 `server.js` 的文件。

6. 修复 Next export 空白格式和动态 output 的误判。
   - 支持 `output : "export"`、跨行空白等写法。
   - 通过轻量词法扫描忽略 JS/TS 注释并保留字符串。
   - 动态 `output` 配置不强制覆盖为 standalone，保持原 Node 部署路径。

## 修改文件

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/package_json.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node_test.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next_test.go`
- `/Users/yaozaiyong/Downloads/buildpack/core/__snapshots__/TestGenerateBuildPlanForExamples_node-next_1.snap.json`
- `/Users/yaozaiyong/Downloads/buildpack/core/__snapshots__/TestGenerateBuildPlanForExamples_node-turborepo_1.snap.json`

## 验证结果

通过：

- `go test ./core/providers/node`
- `go test ./buildkit/...`
- `go test ./core/generate ./core/plan`
- `go test ./core -run TestGenerateBuildPlanForExamples`
- 119 个示例 snapshot 全部通过
- `go vet ./core/... ./buildkit/...`
- `gofmt`
- `git diff --check`
- 普通 Next、Next SPA、workspace Next、自定义 build command 的 CLI plan 检查
- 二次只读代码 review
- `aiscan-cli save-repair` 修复记录保存成功

实际镜像验证：

- 本地 Docker 和 BuildKit 容器可用。
- 构建进入 BuildKit 后，拉取 `ghcr.io/railwayapp/railpack-builder` 与 `railpack-runtime` 镜像时网络超时。
- 因外部 GHCR 连接超时，未能完成镜像启动与 HTTP 请求验证；代码、BuildPlan、BuildKit 单测与 snapshot 均已通过。

## 修复统计

- 发现问题：6 个
- 已修复问题：6 个
- 修复代码文件：5 个
- 更新 snapshot：2 个
