# 用构建前脚本设置 Next.js Standalone

- [x] Task 1: 实现 Next 配置修改脚本
    - 1.1: 定义临时 Node 脚本 asset 和固定脚本路径
    - 1.2: 支持替换已有静态 output
    - 1.3: 支持 nextConfig 对象、直接 ESM 对象和 CommonJS 对象注入
    - 1.4: 支持无配置时生成最小 next.config.mjs
    - 1.5: 对动态、插件和多候选配置明确失败
    - 1.6: 使用 shell quote 生成脚本执行命令

- [x] Task 2: 移除 wrapper 并接入 NodeProvider Plan
    - 2.1: 删除 getNextStandaloneWrapper 和原配置重命名逻辑
    - 2.2: 删除仅供 wrapper 使用的 PackageJson.Type
    - 2.3: 在 build command 前写入并执行配置脚本
    - 2.4: 保持自定义 build 的 setup/build/prepare 顺序
    - 2.5: 保持 SPA、动态 output 与固定部署根行为

- [x] Task 3: 迁移单元与计划测试
    - 3.1: 删除 wrapper import 测试
    - 3.2: 实际执行脚本验证 TS、MJS、CJS 和已有 output
    - 3.3: 验证无配置生成及不支持配置失败不改文件
    - 3.4: 更新根应用、workspace、SPA 和自定义 build 计划断言
    - 3.5: 更新两个 Next snapshots

- [x] Task 4: 修复本轮检查和 review 遗留项
    - 4.1: 修复 ST1005、QF1001 和未使用方法
    - 4.2: 修复 distDir 空白格式回归
    - 4.3: 显式 startCommand 时保持完整 Node 部署输入
    - 4.4: 复用 plan.Spread 并补充关键 why 注释
    - 4.5: 拆分超长 Go 命令构造行

- [x] Task 5: 执行完整验证
    - 5.1: 运行 mise run check
    - 5.2: 运行 Node provider、core 和 BuildKit 测试
    - 5.3: 验证全部示例 snapshots
    - 5.4: 生成普通 Next、SPA、workspace 和自定义 build 计划
    - 5.5: 网络允许时执行实际镜像构建，否则记录限制
