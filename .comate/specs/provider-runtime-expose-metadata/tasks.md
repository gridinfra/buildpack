# Provider Runtime 与 Expose 元数据实施计划

- [x] Task 1: 扩展 Provider 元数据契约与核心写入链路
    - 1.1: 在 Provider 包定义统一的 ProviderMetadata 值对象
    - 1.2: 为 Provider 接口增加 Metadata 方法
    - 1.3: 在 GenerateBuildPlan 的 Provider Plan 成功后写入 runtime 与 expose
    - 1.4: 保持已有 providers、DetectedProviders 和 provider-specific metadata 行为不变

- [x] Task 2: 为 Node 与 Python Provider 实现框架和端口元数据
    - 2.1: 复用 Node 现有 runtime 检测并规范化 next/node 为 nextjs/nodejs
    - 2.2: 区分 Node SPA 与 SSR 默认端口并处理 Astro server
    - 2.3: 复用 Python runtime 检测并为已识别 Web 框架设置 8000
    - 2.4: 保留 nodeRuntime、pythonRuntime 等既有字段
    - 2.5: 补充 Node 和 Python 的表驱动单元测试

- [x] Task 3: 为有框架检测或固定默认端口的 Provider 实现元数据
    - 3.1: 为 PHP/Laravel 输出 php 或 laravel，并设置 expose 80
    - 3.2: 为 Ruby/Rails 输出 ruby 或 rails，并仅为 Rails 设置 expose 3000
    - 3.3: 为 Elixir/Phoenix 输出 elixir 或 phoenix，并仅为 Phoenix 设置 expose 4000
    - 3.4: 为 .NET 输出 dotnet，并设置 expose 3000
    - 3.5: 为 Staticfile 输出 staticfile，并设置 expose 80
    - 3.6: 扩展对应 Provider 单元测试覆盖框架分支和默认端口

- [x] Task 4: 为其余语言 Provider 实现 runtime fallback
    - 4.1: 为 Go/Gin 输出 go 或 gin，不设置 expose
    - 4.2: 为 Java/Spring Boot 输出 java 或 spring-boot，不设置 expose
    - 4.3: 为 Rust、Deno、Gleam、C++、Shell 输出稳定 runtime，不设置 expose
    - 4.4: 补充或扩展单元测试验证 runtime fallback 与空 expose

- [x] Task 5: 验证 BuildResult 统一元数据和兼容性
    - 5.1: 在 core 测试中覆盖 Next.js、普通 Node、SPA、有端口框架和无端口运行时
    - 5.2: 验证显式 Provider 使用实际 Provider 的统一元数据
    - 5.3: 验证未检测 Provider 时不写 runtime 和 expose
    - 5.4: 验证旧 metadata 字段仍然存在且值不变

- [x] Task 6: 运行项目检查和相关测试
    - 6.1: 运行 mise run check 并修复本次变更引入的问题
    - 6.2: 运行 core 与 Provider 相关单元测试
    - 6.3: 运行代表性的 Node、Python 和 PHP 集成测试或仓库等价测试任务
    - 6.4: 检查 git diff，确认未包含无关改动或非预期 snapshot 变化
