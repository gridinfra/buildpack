# 修复 Next.js Standalone Review 问题

- [x] Task 1: 重构 standalone 配置包装与路径 helper
    - 1.1: 删除正则对象改写和静态 distDir 依赖
    - 1.2: 实现无配置、ESM、CommonJS 和 TypeScript wrapper
    - 1.3: 保留函数、Promise 和插件包装配置的返回结果
    - 1.4: 增加固定部署根目录和绝对启动路径 helper
    - 1.5: 更新配置与路径单元测试

- [x] Task 2: 修复 Plan 模式与构建命令判定
    - 2.1: 统一 SPA、Next export 和 standalone 条件
    - 2.2: 保证 SPA_OUTPUT_DIR 不进入 standalone
    - 2.3: 通过 build step 最终命令判断是否存在构建命令
    - 2.4: 保留 BUILD_CMD 与显式配置覆盖能力
    - 2.5: 保持其他 Node provider 行为不变

- [x] Task 3: 修复 standalone 部署目录和启动命令
    - 3.1: build 后定位唯一 standalone 产物目录
    - 3.2: 将 standalone 内容整理到 /railpack/next-standalone
    - 3.3: 将 static 和可选 public 放入对应应用相对目录
    - 3.4: DeployInputs 仅复制固定部署根且不使用 Spread
    - 3.5: 单应用和 workspace 使用绝对 server.js 启动路径

- [x] Task 4: 增加计划级与回归测试
    - 4.1: 覆盖单应用 standalone 的 commands、inputs 和 start command
    - 4.2: 覆盖 workspace standalone 路径
    - 4.3: 覆盖 SPA_OUTPUT_DIR 与 Next export
    - 4.4: 覆盖自定义 BUILD_CMD 且无 package build script
    - 4.5: 覆盖动态和包装式 Next 配置

- [x] Task 5: 执行完整验证并整理结果
    - 5.1: 运行 Node provider 完整单元测试
    - 5.2: 运行相关 BuildKit 与 core 测试
    - 5.3: 运行 go vet、格式化和 diff 检查
    - 5.4: 生成普通 Next、SPA 和 workspace 构建计划并核对字段
    - 5.5: 在 Docker 可用时执行镜像运行验证，否则记录限制
