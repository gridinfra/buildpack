# Provider Runtime/Expose Review 修复计划

- [x] Task 1: 修复 Node workspace Next.js 元数据识别
    - 1.1: 增加复用现有 workspace 遍历的 Next.js 存在性判断
    - 1.2: 让统一 runtime 检测使用 workspace-aware 判断
    - 1.3: 保持旧 isNext 调用点和多个 Next 应用构建语义不变
    - 1.4: 为 node-turborepo 增加 runtime、expose 和旧 nodeRuntime 测试

- [x] Task 2: 补齐 Ruby Rack 默认端口元数据
    - 2.1: 在非 Rails 且存在 config.ru 时设置 expose 3000
    - 2.2: 保持 Rack runtime 为 ruby，Rails 分支优先
    - 2.3: 使用 ruby-sinatra fixture 增加 Metadata 测试

- [x] Task 3: 运行检查和回归测试
    - 3.1: 运行 mise run check
    - 3.2: 运行 Node 与 Ruby Provider 定向测试
    - 3.3: 运行 core Metadata 定向测试和全量短测试
    - 3.4: 检查 git diff、snapshot 和非预期文件变化
