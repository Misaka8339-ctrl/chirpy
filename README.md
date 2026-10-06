# Chirpy

基于 Go 和 PostgreSQL 构建的简易社交消息 REST API。

本项目在 Boot.dev「Learn HTTP Servers in Go」课程中逐步完成，使用 Go 标准库实现 HTTP 路由、用户认证、消息管理和支付 webhook，重点练习后端服务的完整开发流程。

## 功能

- 用户注册、登录及邮箱和密码更新
- 使用 Argon2id 存储和验证密码哈希
- JWT access token 认证，有效期为 1 小时
- Refresh token 刷新与撤销，有效期为 60 天
- 创建、查询和删除 Chirp
- 仅允许作者删除自己的 Chirp
- 正文最多 140 个 Unicode 码点，并替换指定违禁词
- 按作者筛选 Chirp，按创建时间升序或降序排列
- 通过 Polka webhook 升级 Chirpy Red 会员，使用 API key 验证请求
- 静态文件服务、访问计数及健康检查
- 仅开发环境可用的数据重置接口

## 技术栈

| 技术 | 用途 |
|---|---|
| Go / net/http | HTTP 服务、路由和中间件 |
| PostgreSQL | 持久化存储 |
| Goose | 数据库结构迁移 |
| SQLC | 根据 SQL 生成类型安全的 Go 查询代码 |
| lib/pq | PostgreSQL 驱动 |
| argon2id | 密码哈希 |
| golang-jwt/jwt/v5 | JWT 签发与验证 |
| godotenv | 加载本地环境变量 |

## 项目结构

```text
chirpy/
├── main.go              # 服务配置、依赖初始化和路由注册
├── handler_*.go         # HTTP 处理器
├── json_helpers.go      # JSON 响应辅助函数
├── internal/
│   ├── auth/            # 密码、JWT、令牌和 API key 工具及测试
│   └── database/        # SQLC 生成的数据库访问代码
├── sql/
│   ├── schema/          # Goose 数据库迁移
│   └── queries/         # SQLC 查询
├── assets/              # 静态资源
├── index.html
├── sqlc.yaml
├── .env.example         # 本地配置模板
├── go.mod
└── go.sum
```

## 本地运行

以下命令适用于 Linux / WSL Bash。

### 1. 准备环境

需要安装：

- Go 1.22 或更高版本；实际使用版本还需满足 go.mod 和依赖要求
- PostgreSQL 15 或更高版本
- Goose
- SQLC

安装命令行工具：

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
```

确认工具可用：

```bash
go version
psql --version
goose -version
sqlc version
```

### 2. 获取项目

```bash
git clone https://github.com/Misaka8339-ctrl/chirpy.git
cd chirpy
go mod download
```

### 3. 准备数据库

启动 PostgreSQL：

```bash
sudo service postgresql start
```

进入数据库客户端：

```bash
sudo -u postgres psql
```

创建数据库；如果已经存在，跳过创建：

```sql
CREATE DATABASE chirpy;
```

确保数据库用户已配置可用于本地连接的密码，然后退出：

```text
\q
```

### 4. 配置环境变量

```bash
cp .env.example .env
```

修改 `.env`：

```dotenv
DB_URL="postgres://postgres:YOUR_PASSWORD@localhost:5432/chirpy?sslmode=disable"
PLATFORM="dev"
JWT_SECRET="YOUR_RANDOM_SECRET"
POLKA_KEY="YOUR_POLKA_API_KEY"
```

生成 JWT 密钥：

```bash
openssl rand -base64 64
```

将输出拼成一行后填入 `JWT_SECRET`。`POLKA_KEY` 使用课程或支付方提供的值。

`.env` 包含敏感信息，不应提交到 Git。

### 5. 执行迁移并生成代码

将下方连接字符串替换为 `.env` 中的实际 `DB_URL`：

```bash
goose -dir sql/schema postgres \
  "postgres://postgres:YOUR_PASSWORD@localhost:5432/chirpy?sslmode=disable" up

sqlc generate
```

Goose 修改实际数据库结构；SQLC 根据 SQL 生成 Go 代码，不能替代数据库迁移。

### 6. 启动服务

在项目根目录执行：

```bash
go build -o out && ./out
```

默认监听端口为 `8080`。

- 网站：http://localhost:8080/app/
- 健康检查：http://localhost:8080/api/healthz
- 访问统计：http://localhost:8080/admin/metrics

## API

需要用户认证的接口使用：

```http
Authorization: Bearer <access_token>
```

| 方法 | 路径 | 功能 | 认证或限制 |
|---|---|---|---|
| GET | `/api/healthz` | 健康检查 | 无 |
| POST | `/api/users` | 注册用户 | 无 |
| PUT | `/api/users` | 更新自己的邮箱和密码 | Access token |
| POST | `/api/login` | 登录并获取两种令牌 | 邮箱和密码 |
| POST | `/api/refresh` | 获取新的 access token | Refresh token |
| POST | `/api/revoke` | 撤销 refresh token | Refresh token |
| POST | `/api/chirps` | 创建 Chirp | Access token |
| GET | `/api/chirps` | 查询 Chirp 列表 | 无 |
| GET | `/api/chirps/{chirpID}` | 查询单条 Chirp | 无 |
| DELETE | `/api/chirps/{chirpID}` | 删除 Chirp | Access token，且必须是作者 |
| POST | `/api/polka/webhooks` | 接收会员升级通知 | Polka API key |
| GET | `/admin/metrics` | 查看静态文件请求计数 | 无 |
| POST | `/admin/reset` | 删除全部用户并清零计数 | 仅 `PLATFORM=dev` |

重置会通过外键级联删除用户关联的 Chirp 和 refresh token，但保留表结构。

### 注册与登录

注册：

```bash
curl -i -X POST http://localhost:8080/api/users \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"example-password"}'
```

登录：

```bash
curl -i -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"example-password"}'
```

登录成功返回用户信息及：

- `token`：1 小时有效的 access token
- `refresh_token`：60 天有效的 refresh token

用户响应包含 `id`、`created_at`、`updated_at`、`email` 和 `is_chirpy_red`，不包含密码或密码哈希。

### 创建 Chirp

```bash
ACCESS_TOKEN='替换为登录返回的token'

curl -i -X POST http://localhost:8080/api/chirps \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"body":"Hello, Chirpy!"}'
```

作者身份由 JWT 确定，请求正文无需提供 `user_id`。

正文超过 140 个 Unicode 码点时返回 `400`。独立出现的 `kerfuffle`、`sharbert`、`fornax` 会被替换为 `****`，匹配时忽略大小写；带标点的词不会被视为完全匹配。

### 筛选与排序

```bash
curl "http://localhost:8080/api/chirps"
curl "http://localhost:8080/api/chirps?sort=desc"
curl "http://localhost:8080/api/chirps?author_id=USER_UUID&sort=asc"
```

| 参数 | 含义 | 默认行为 |
|---|---|---|
| `author_id` | 按作者 UUID 筛选，在数据库执行 | 不筛选作者 |
| `sort` | `asc` 或 `desc`，按 `created_at` 排序 | `asc` |

没有匹配记录时返回 `[]`。当前尚未实现分页。

### 刷新与撤销令牌

```bash
REFRESH_TOKEN='替换为登录返回的refresh_token'

curl -i -X POST http://localhost:8080/api/refresh \
  -H "Authorization: Bearer $REFRESH_TOKEN"

curl -i -X POST http://localhost:8080/api/revoke \
  -H "Authorization: Bearer $REFRESH_TOKEN"
```

刷新成功返回新的 `token`。撤销成功返回 `204`，没有正文。

撤销 refresh token 不会立即使已经签发的 access token 失效；已有 access token 仍可使用到过期。

### Polka webhook

请求头：

```http
Authorization: ApiKey <POLKA_KEY>
```

请求正文：

```json
{
  "event": "user.upgraded",
  "data": {
    "user_id": "替换为用户UUID"
  }
}
```

- 缺少或错误的 API key：`401`
- 认证通过，但事件不是 `user.upgraded`：`204`
- 用户升级成功：`204`
- 用户不存在：`404`

### 错误响应

业务处理器使用以下 JSON 结构返回错误：

```json
{
  "error": "错误说明"
}
```

常用状态码：

| 状态码 | 含义 |
|---|---|
| 200 | 查询或更新成功 |
| 201 | 创建成功 |
| 204 | 操作成功，无响应正文 |
| 400 | 请求参数错误 |
| 401 | 缺少或无效的认证凭据 |
| 403 | 无权执行该操作 |
| 404 | 资源不存在 |
| 409 | 资源冲突，例如更新邮箱时重复 |
| 500 | 服务端处理失败 |

## 开发与测试

格式化、运行测试、构建：

```bash
go fmt ./...
go test ./...
go build -o out
```

强制重新执行认证工具包的测试：

```bash
go test -v -count=1 ./internal/auth
```

修改 SQL 查询后执行：

```bash
sqlc generate
```

修改数据库结构时，应新增迁移文件并执行 `goose up`。

SQLC 生成的 `internal/database` 文件不应手动修改。

Boot.dev CLI 测试需要保持服务器运行，并在另一个终端执行课程页面提供的测试命令。

## 当前范围

这是一个本地学习项目，尚未作为生产服务部署：

- 当前配置使用 HTTP，生产环境需要 HTTPS。
- 列表接口尚未实现分页。
- 修改密码不会自动撤销已有令牌。
- 当前静态文件服务以项目根目录为根，仅适合本地练习；部署前需要限制为专用的公开资源目录。
- 管理接口尚未加入管理员身份认证。

## 致谢

项目基于 Boot.dev「Learn HTTP Servers in Go」课程完成。