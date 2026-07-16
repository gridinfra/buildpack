# Provider Runtime/Expose Review 修复总结

## Review 结论

深度 Code Review 最终确认两个问题：

1. Node workspace 子包中的 Next.js 无法被统一 runtime 检测识别。
2. Ruby Rack 已有明确 3000 默认端口，但统一 Metadata 未输出 expose。

初审中关于“自定义 start command 应清空 expose”和“Python Web 框架不应默认写 8000”的 finding 经 Meta Review 排除，因为本功能的既定语义是框架/Provider 默认端口，而不是最终命令解析结果。

## 完成修复

### Node workspace Next.js

- 在 `core/providers/node/node.go` 增加 `hasNextPackage(ctx)`。
- 复用现有 `getPackagesWithFramework` 遍历 workspace root 与子包。
- `getRuntime` 使用 workspace-aware 判断返回 `next`。
- 保留原 `isNext()` 供根包构建变量逻辑使用，避免扩大行为变化。
- 多个 Next 子包只影响原有构建选择规则，不影响元数据对框架类型的判断。

结果：

```text
examples/node-turborepo
runtime=nextjs
expose=3000
nodeRuntime=next
```

### Ruby Rack

- 在 `core/providers/ruby/ruby.go` 的 Rails 分支之后检查 `config.ru`。
- Rack 应用保持 `runtime=ruby`，新增 `expose=3000`。
- Rails 优先级不变。

结果：

```text
examples/ruby-sinatra
runtime=ruby
expose=3000
```

## 测试增强

- Node Metadata 表驱动测试新增 `node-turborepo`。
- Node 测试精确断言统一 runtime、expose 和旧 `nodeRuntime` 值。
- Ruby Metadata 表驱动测试新增 `ruby-sinatra` Rack 用例。

## 验证结果

全部通过：

- `mise run check`
  - go vet
  - go fmt
  - golangci-lint：0 issues
  - go mod verify
- Node Provider 测试
- Ruby Provider 测试
- core Metadata 定向测试
- `mise run test` 全量短测试
- `git diff --check`
- BuildPlan snapshot 无变化

未修改或删除工作区中与本任务无关的 `.idea/`、构建二进制和 `railpack-*.json` 文件。
