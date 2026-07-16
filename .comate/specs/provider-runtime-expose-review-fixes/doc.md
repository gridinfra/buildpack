# Provider Runtime/Expose Review 修复

## 背景与目标

针对 `provider-runtime-expose-metadata` 实现进行深度 Code Review 后，确认两个有明确代码证据的问题：

1. Node workspace 的 Next.js 应用可能位于子包，现有统一 runtime 只检查根 `package.json`，导致输出 `runtime=nodejs` 且缺少 `expose=3000`。
2. Ruby Rack 应用存在 `config.ru` 时，Provider 生成的启动命令明确使用 `${PORT:-3000}`，但统一元数据没有输出 `expose=3000`。

本次只修复这两个问题，不改变统一 Metadata 契约、其他 Provider 映射和 expose 的产品语义。

## 修复方案

### Node workspace Next.js 检测

当前 `getRuntime` 调用 `isNext()`，后者通过 `hasDependency("next")` 只读取根 `p.packageJson`：

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go:542-544`
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go:717-748`

而现有 `getPackagesWithFramework` 已遍历 workspace root 与全部子包：

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go:648-665`

新增一个 workspace-aware 的布尔判断，复用该遍历：

```go
func (p *NodeProvider) hasNextPackage(ctx *generate.GenerateContext) bool {
    packages, _ := p.getPackagesWithFramework(ctx, func(pkg *WorkspacePackage, _ *generate.GenerateContext) bool {
        return pkg.PackageJson.hasDependency("next")
    })
    return len(packages) > 0
}
```

`getRuntime` 的服务端 Next 分支改用该判断。现有 `isNext()` 保留给只应检查根包的旧调用点，避免扩大其他行为变化。

不直接复用 `getNextPackage`，因为它在多个 Next 子包时返回错误，而统一 runtime 只需要判断“是否存在 Next.js 应用”，多个候选也应输出框架类型。

测试扩展 `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node_test.go` 的 Metadata 表驱动用例，加入 `examples/node-turborepo`，预期：

```text
runtime=nextjs
expose=3000
```

同时断言旧 `nodeRuntime=next` 保持兼容。

### Ruby Rack 默认端口

当前 Ruby `Metadata` 只对 Rails 设置 3000：

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/ruby/ruby.go:26-30`

但 `GetStartCommand` 对非 Rails 的 `config.ru` 生成：

```text
bundle exec rackup config.ru -o 0.0.0.0 -p ${PORT:-3000}
```

修复为：

```go
func (p *RubyProvider) Metadata(ctx *generate.GenerateContext) generate.ProviderMetadata {
    if p.usesRails(ctx) {
        return generate.ProviderMetadata{Runtime: "rails", Expose: "3000"}
    }
    if ctx.App.HasFile("config.ru") {
        return generate.ProviderMetadata{Runtime: "ruby", Expose: "3000"}
    }
    return generate.ProviderMetadata{Runtime: "ruby"}
}
```

使用 `App.HasFile`，符合项目文件访问约定。扩展 `/Users/yaozaiyong/Downloads/buildpack/core/providers/ruby/ruby_test.go`，使用已有 `examples/ruby-sinatra` fixture 验证 `runtime=ruby`、`expose=3000`。

## 数据流与兼容性

```text
NodeProvider.Initialize -> workspace 已构建
NodeProvider.Plan -> 旧 nodeRuntime 写入
NodeProvider.Metadata -> workspace-aware Next 判断 -> nextjs/3000

RubyProvider.Plan -> GetStartCommand 根据 config.ru 生成 rackup/3000
RubyProvider.Metadata -> 同一 config.ru 特征 -> ruby/3000
```

兼容性保证：

- 不修改 `ProviderMetadata`、`BuildResult.Metadata` 类型。
- 不改变 Node 构建命令、workspace 选择、多个 Next 应用的构建错误行为。
- 不改变旧 `nodeRuntime` 的值。
- Ruby Rack runtime 仍为 `ruby`，只补充 expose。
- 不改 Procfile 或用户自定义 start command 语义。

## 影响文件

- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node.go`
  - 增加 workspace-aware Next 存在性判断。
  - `getRuntime` 使用新判断。
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/node/node_test.go`
  - 增加 turborepo Metadata 用例。
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/ruby/ruby.go`
  - Rack `config.ru` 设置 expose 3000。
- `/Users/yaozaiyong/Downloads/buildpack/core/providers/ruby/ruby_test.go`
  - 增加 Sinatra/Rack Metadata 用例。

## 边界条件

- workspace 未初始化或为空：Next 判断返回 false，不 panic。
- workspace 包含多个 Next 应用：runtime 仍为 `nextjs`；构建阶段原有单应用选择规则不变。
- 根包与子包均有 Next：runtime 为 `nextjs`。
- Rails 同时存在 `config.ru`：Rails 分支优先，runtime 保持 `rails`。
- 普通 Ruby 无 `config.ru`：继续省略 expose。

## 验证方案

遵循仓库命令约定：

1. `mise run check`
2. Node 与 Ruby Provider 定向测试。
3. core Metadata 定向测试及 `mise run test` 全量短测试。
4. `git diff --check` 与 snapshot 变化检查。

容器集成测试依赖本地 Docker/BuildKit；若 Docker daemon 仍不可用，记录环境限制，不以代码方式绕过。

## 预期结果

- `examples/node-turborepo` 输出 `runtime=nextjs`、`expose=3000`，旧 `nodeRuntime=next`。
- `examples/ruby-sinatra` 输出 `runtime=ruby`、`expose=3000`。
- 普通 Node、普通 Ruby 及其他 Provider 行为不变。
