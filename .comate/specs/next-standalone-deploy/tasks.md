# 实现 Next.js Standalone 精简部署

- [x] Task 1: 实现 Next.js standalone 配置与路径辅助逻辑
    - 1.1: 在 next.go 中增加 standalone 输出、server、static 和 public 路径常量及 helper
    - 1.2: 实现 Next 配置文件选择与应用目录定位
    - 1.3: 实现对象配置中 output standalone 的注入或覆盖
    - 1.4: 支持无配置文件时生成最小 next.config.mjs
    - 1.5: 识别静态 distDir 并拒绝无法安全处理的动态配置
    - 1.6: 实现单应用与 workspace 的 standalone 启动命令
    - 1.7: 增加配置转换、异常和路径 helper 的表驱动单元测试

- [x] Task 2: 将 standalone 配置接入 NodeProvider 构建规划
    - 2.1: 在 Plan 中统一计算 Next SPA 与 standalone 条件
    - 2.2: 在 Next build command 前加入临时配置写入命令
    - 2.3: 确保配置修改仅发生在构建层且不修改宿主源码
    - 2.4: 对缺少 build script 或无法生成 standalone 产物的项目返回明确错误
    - 2.5: 增加构建命令顺序及 SPA 不注入配置的测试

- [x] Task 3: 调整 standalone 的 DeployInputs 与启动命令
    - 3.1: 将 standalone 根目录作为 spread 部署输入展开到 /app
    - 3.2: 将 .next/static 放入 standalone server 对应应用目录
    - 3.3: 仅在 public 存在时加入 public 部署输入
    - 3.4: 保持 workspace standalone 根目录中的共享 traced 文件
    - 3.5: 在显式环境变量覆盖之后选择 node server.js 启动命令
    - 3.6: 保持 SPA、其他 Node 项目和用户显式启动命令的原有行为
    - 3.7: 增加单应用、SPA、workspace 与显式覆盖的构建计划测试

- [x] Task 4: 执行单元与构建计划验证
    - 4.1: 运行 node provider 单元测试并修复失败
    - 4.2: 运行相关 core 测试以检查共享计划结构回归
    - 4.3: 检查 node-next、node-next-spa 和 node-turborepo 的生成计划
    - 4.4: 执行仓库可用的格式化和静态检查

- [x] Task 5: 验证实际 Next.js 镜像产物与运行目录
    - 5.1: 使用仓库现有构建入口构建 node-next 示例镜像
    - 5.2: 检查 /app/server.js、/app/.next/static 和 /app/public
    - 5.3: 启动镜像并验证页面及静态资源响应
    - 5.4: 在环境允许时构建 workspace Next 示例并检查 apps/web/server.js 路径
    - 5.5: 记录因 Docker、网络或依赖不可用而无法执行的集成验证
