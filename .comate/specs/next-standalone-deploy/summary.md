# Next.js Standalone Deployment 实施总结

## 完成内容

- 非 SPA Next.js 构建前自动将 Next 配置的 `output` 注入或覆盖为 `"standalone"`。
- 配置修改通过 build step asset 写入临时构建文件系统，不修改用户宿主源码。
- 支持 `next.config.js`、`next.config.mjs`、`next.config.ts` 的常见对象导出形式。
- 无配置文件时生成最小 `next.config.mjs`。
- 支持静态字符串 `distDir`；动态配置、动态 `distDir`、越界目录和多个 Next 应用会返回明确错误。
- Next SPA 保持 `output: export`、`out` 和 Caddy 部署流程。
- 普通 Next 与唯一 Next workspace 应用均可定位正确应用目录。
- `next build` 后把 `.next/static` 和可选 `public` 复制进 standalone 树。
- 部署输入只包含 mise 运行时和展开后的 standalone 根目录，不再包含完整源码、开发依赖和完整 build 层。
- 单应用以 `node server.js` 启动；workspace 以 `node <workspace>/server.js` 启动。
- 显式 Railpack deploy start command 保持覆盖优先级；package.json 的普通 start script 不再覆盖 standalone server 命令。

## 修改文件

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go`
  - 在 `NodeProvider.Plan` 中统一规划 Next standalone 配置、构建命令、资源复制、部署输入和启动命令。
  - 支持 workspace Next 应用进入 standalone 流程。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next.go`
  - 新增 Next 应用定位、配置选择与转换、`distDir` 校验、standalone 路径和启动命令 helper。
  - SPA 检测改为使用实际 Next workspace package 的 scripts/config。

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/next_test.go`
  - 新增配置转换、动态配置错误、`distDir` 和单应用/workspace 路径测试。

## 生成计划验证

### `examples/node-next`

- start command：`node server.js`
- deploy input：`.next/standalone`，`spread: true`
- build commands 顺序：写入 `/app/next.config.ts` → `npm run build` → 复制 `.next/static` 与 `public`

### `examples/node-next-spa`

- start command：Caddy
- deploy input：`out`
- 未注入 standalone 配置

### `examples/node-turborepo`

- start command：`node apps/web/server.js`
- deploy input：`apps/web/.next/standalone`，`spread: true`
- 配置写入：`/app/apps/web/next.config.js`
- static/public 复制到 `apps/web/.next/standalone/apps/web/` 下，展开后得到 `/app/apps/web/.next/static` 和 `/app/apps/web/public`

## 验证结果

已通过：

- `go test ./core/providers/node`
- `go vet ./core/providers/node`
- `git diff --check`
- 三个 Next 示例的 CLI build plan 生成与结构化字段检查

未执行：

- 实际 Docker 镜像构建、容器目录检查和 HTTP 请求验证。
- 原因：本机 Docker daemon 未运行，`docker info` 无法连接 `/Users/yaozaiyong/.docker/run/docker.sock`。

## 注意事项

- 多个 Next workspace 应用无法唯一决定部署入口时会明确失败，需要用户先选择单个应用。
- 动态 Next 配置不会被覆盖，以避免破坏用户配置逻辑。
- 工作区中原有未跟踪 `.idea/` 目录未修改。
