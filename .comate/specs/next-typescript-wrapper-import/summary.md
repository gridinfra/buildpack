# Next.js 构建前脚本设置 Standalone 实施总结

## 完成内容

- 删除 Next 配置 wrapper、原配置重命名和跨配置 import 方案。
- 新增临时 Node.js 脚本 asset `/railpack/scripts/configure-next-standalone.mjs`，在 `next build` 前直接修改构建文件系统中的 Next 配置。
- 支持 `next.config.js`、`next.config.mjs`、`next.config.ts`：
  - 替换已有静态 `output`；
  - 向直接导出的 `nextConfig`、ESM 对象或 CommonJS 对象注入 `output: "standalone"`；
  - 无配置时生成最小 `next.config.mjs`；
  - 动态 output、插件包装和多候选配置明确失败且不改原文件。
- 保持宿主工作区不变，不再生成 `.ts` import，消除 TS5097。
- 在 build 后将 standalone、`.next/static` 和 `public` 整理到 `/railpack/next-standalone`，使用固定入口 `node /railpack/next-standalone/server.js`。
- 支持 root Next 应用和 workspace Next 应用，prepare 命令使用 shell 执行并处理两种 server.js 布局。
- 保持 Next SPA/static export 行为；显式 start command 禁用 standalone 并保留完整 Node 部署输入。
- 自定义 build command 使用 `plan.Spread`，保证配置脚本在首个可执行 build 前运行，部署整理在全部 build 命令后运行。
- 修复 `distDir` 任意空白格式解析、错误文本、静态检查和命令引号问题。

## 测试与验证

- `mise run check`：通过，golangci-lint 0 issues，模块校验通过。
- `go test -short ./... -count=1`：通过。
- `go test ./core -run '^TestGenerateBuildPlanForExamples$' -count=1`：通过。
- `go test -short ./buildkit -count=1`：通过。
- Node provider 全量测试：通过。
- 配置脚本实际由 Node 执行，覆盖 TS、MJS、CJS、已有 output、无配置、动态配置和插件包装失败场景。
- CLI 计划验证：root Next、Next SPA、workspace Next 均成功生成并符合预期。
- 更新且仅更新两个相关快照：`node-next`、`node-turborepo`。
- BuildKit 实际构建 `examples/node-next`：通过。
  - TypeScript 配置修改成功；
  - Next.js production build 成功；
  - standalone prepare 成功；
  - 镜像成功加载并启动；
  - HTTP `/` 返回 200；
  - 镜像大小约 144.8 MB。
