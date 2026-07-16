# Runtime 与 Expose 支持清单

`runtime` 和 `expose` 会写入构建结果的 `metadata`：

```json
{
  "metadata": {
    "runtime": "nextjs",
    "expose": "3000"
  }
}
```

> `expose` 为空时，该字段不会写入 Metadata。

## 有明确默认端口的 Runtime

| Runtime | Expose | 适用场景 |
|---|---:|---|
| `nextjs` | `80` | Next.js 静态导出或 SPA |
| `nextjs` | `3000` | Next.js SSR、Server 或 Workspace 应用 |
| `nuxt` | `3000` | Nuxt 服务端应用 |
| `remix` | `3000` | Remix 服务端应用 |
| `tanstack-start` | `3000` | TanStack Start |
| `react-router` | `80` | React Router SPA |
| `react-router` | `3000` | React Router SSR |
| `astro` | `80` | Astro 静态站点 |
| `astro` | `4321` | Astro Server 或 SSR |
| `vite` | `80` | Vite SPA |
| `cra` | `80` | Create React App |
| `angular` | `80` | Angular SPA |
| `expo` | `80` | Expo Web SPA |
| `static` | `80` | 强制 Node SPA 或 Staticfile 静态文件服务 |
| `django` | `8000` | Django |
| `flask` | `8000` | Flask |
| `fastapi` | `8000` | FastAPI |
| `fasthtml` | `8000` | FastHTML |
| `laravel` | `80` | Laravel |
| `php` | `80` | 普通 PHP |
| `rails` | `3000` | Ruby on Rails |
| `ruby` | `3000` | Rack 应用，存在 `config.ru` |
| `phoenix` | `4000` | Phoenix |
| `dotnet` | `3000` | .NET |

## 不设置默认端口的 Runtime

| Runtime | Expose | 说明 |
|---|---:|---|
| `nodejs` | — | 普通 Node.js，端口由应用决定 |
| `bun` | — | 普通 Bun 应用，端口由应用决定 |
| `vite` | — | 非 SPA Vite 场景 |
| `python` | — | 普通 Python |
| `ruby` | — | 非 Rails、非 Rack 的普通 Ruby |
| `elixir` | — | 普通 Elixir |
| `go` | — | 普通 Go |
| `gin` | — | Gin 端口由应用代码决定 |
| `java` | — | 普通 Java |
| `spring-boot` | — | 使用动态 `$PORT`，未定义固定 fallback |
| `rust` | — | 普通 Rust |
| `deno` | — | Deno 端口由应用代码决定 |
| `gleam` | — | Gleam |
| `cpp` | — | C/C++ |
| `shell` | — | Shell 脚本行为未知 |

## 按 Provider 查看

| Provider | 支持的 Runtime |
|---|---|
| Node | `nodejs`、`bun`、`nextjs`、`nuxt`、`remix`、`tanstack-start`、`react-router`、`astro`、`vite`、`cra`、`angular`、`expo`、`static` |
| Python | `python`、`django`、`flask`、`fastapi`、`fasthtml` |
| PHP | `php`、`laravel` |
| Ruby | `ruby`、`rails` |
| Elixir | `elixir`、`phoenix` |
| Go | `go`、`gin` |
| Java | `java`、`spring-boot` |
| .NET | `dotnet` |
| Rust | `rust` |
| Deno | `deno` |
| Gleam | `gleam` |
| C++ | `cpp` |
| Staticfile | `static` |
| Shell | `shell` |

## Node.js 特殊端口规则

同一个 Node runtime 可能因 SPA、静态导出或服务端模式使用不同端口：

```text
Next.js SPA/Static       -> nextjs / 80
Next.js SSR/Server       -> nextjs / 3000
Next.js Workspace        -> nextjs / 3000
React Router SPA         -> react-router / 80
React Router SSR         -> react-router / 3000
Astro Static             -> astro / 80
Astro Server             -> astro / 4321
Vite SPA                 -> vite / 80
Create React App         -> cra / 80
Angular SPA              -> angular / 80
Expo Web SPA             -> expo / 80
普通 Node.js             -> nodejs / 无 expose
普通 Bun                 -> bun / 无 expose
```

## 实现位置

- 统一 Metadata 类型：`core/generate/metadata.go`
- 核心 Metadata 写入：`core/core.go`
- Provider 接口：`core/providers/provider.go`
- Node 映射：`core/providers/node/node.go`
- Python 映射：`core/providers/python/python.go`
- PHP 映射：`core/providers/php/php.go`
- Ruby 映射：`core/providers/ruby/ruby.go`
- Elixir 映射：`core/providers/elixir/elixir.go`
- Go 映射：`core/providers/golang/golang.go`
- Java 映射：`core/providers/java/java.go`
- .NET 映射：`core/providers/dotnet/dotnet.go`
- Staticfile 映射：`core/providers/staticfile/staticfile.go`
