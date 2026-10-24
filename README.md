# guild-go

Guild of Pioneers 的 Go 重写版：一个仓库、一个 Go 二进制、一个镜像。

- 取消 project / task / action 三层结构，统一为任意深度的节点树。
- 节点包含标题、描述、预期完成时间、实际完成时间、标签、手动排序、负责人/成员。
- 线性评论 + 固定 emoji 反应 + 评论附图；查看父节点时展示每个直接子节点的最新评论。
- 3D 关系图谱：读取完整节点列表后在浏览器渲染，支持俯视镜头、节点聚焦和节点上方只读评论流（复用 `GET /api/nodes`，无额外接口）。
- 删除节点会级联删除整棵子树、评论和附件。
- Keycloak OIDC 登录，权限解析使用 [`bitnp-authz`](https://git.bitnp.net/BITNP/bitnp-authz)。
- 前端 Vue 3 + Vite + Tailwind，中文高密度三栏；由 Go `embed` 直接提供静态文件。
- 外部 PostgreSQL + 外部 S3 兼容存储；不迁移旧数据。

## 快速开始

### 1. 准备环境

- Go 1.27+
- Node.js 24+ / pnpm 11+
- 外部 PostgreSQL（16+）
- Keycloak confidential client
- S3 兼容桶（用于评论图片）
- Go 1.27+（无需访问 `git.bitnp.net`：`bitnp-authz` 已内嵌在 `third_party/`）

复制环境变量并填写：

```bash
cp .env.example .env
```

关键项：

| 变量 | 说明 |
| --- | --- |
| `APP_PUBLIC_URL` | 对外 HTTPS 源，例如 `https://guild.example.com` |
| `APP_SECURE_COOKIE` | 生产为 `true` |
| `DATABASE_URL` | PostgreSQL DSN |
| `OIDC_ISSUER` | Keycloak realm issuer |
| `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET` | confidential client |
| `KEYCLOAK_ADMIN_*` | 只读 Admin API，用于显示其他用户姓名/头像/职务；可留空 |
| `S3_ENDPOINT` / `S3_BUCKET` / `S3_ACCESS_KEY` / `S3_SECRET_KEY` | 只保留这 4 个 S3 变量 |

S3 的 region 固定为 `us-east-1`，path-style 固定开启；附件大小 5MB、每条评论最多 4 张，这些是代码常量，不通过环境变量配置。

### 2. 构建前端并启动

```bash
cd web
pnpm install --frozen-lockfile
pnpm build
cd ..
go run ./cmd/guild
```

打开 `http://localhost:8080`。首次启动会自动执行嵌入式数据库迁移。

### 3. 开发模式

两个终端：

```bash
# 终端 1
go run ./cmd/guild

# 终端 2
cd web
pnpm dev
```

Vite 开发服务器在 `http://localhost:5173`，把 `/api` 代理到 `127.0.0.1:8080`。

需要在没有 OIDC 的情况下调试时，可仅对本次后端进程启用开发登录旁路：

```bash
set -a
source .env
set +a
APP_DEV_AUTH_BYPASS=true go run ./cmd/guild
```

旁路身份默认是 `dev-user`，拥有完整管理权限，并会写入本地 `users` 表。可以通过 `APP_DEV_AUTH_SUBJECT` 和 `APP_DEV_AUTH_NAME` 修改。该选项默认关闭，绝不能在共享或生产环境启用。

开启后的行为与限制：

- 该进程内的所有请求都直接以管理员身份通过，不再校验 OIDC，也不会过期；`OIDC_*` 因此不再是必填项。
- 不要把这个开关写进 `compose.yaml` 使用的 `.env`：`env_file` 会把配置原样带进容器。
- 旁路下 `POST /api/auth/logout` 只会清除 Cookie，管理员身份仍会保留；要恢复真实登录，重启进程并去掉该开关。

### 4. 单镜像构建

```bash
docker build -t guild-go .
```

或：

```bash
docker compose up -d --build
```

`compose.yaml` 只启动应用；PostgreSQL、S3、Keycloak 都是外部服务。镜像使用多阶段构建：Node 构建前端 → Go 从 `third_party/bitnp-authz` 编译并 embed 前端 → Alpine 非 root 运行。仓库不提交 `vendor/`，只把 `bitnp-authz` 一个模块的源码内嵌在 `third_party/`，其余依赖仍走 Go module proxy。

## 仓库结构

```
cmd/guild/              Go 入口
internal/auth/          OIDC、内存会话、CSRF、登录/回调
internal/users/         已知用户索引、Keycloak Admin API 显示缓存
internal/nodes/         节点树、成员、继承权限、排序
internal/comments/      评论、反应、附件元数据
internal/storage/       S3 上传/读取/删除
internal/database/      pgxpool + 嵌入式迁移
internal/httpapi/       路由、错误、静态 SPA、中间件
internal/web/           go:embed 前端 dist
web/                    Vue 3 + Vite 前端
third_party/bitnp-authz/  内嵌的 bitnp-authz Go 源码快照
doc/API.md              API 契约
```

没有 pnpm workspace，也没有把前端作为独立发布物；前端源码在 `web/`，Vite 输出到 `internal/web/dist`，Go 二进制嵌入后单进程提供。

## 权限模型

只存直接成员关系，继承关系实时计算：

```text
D_leader(n) = n 的直接负责人
D_member(n) = n 的全部直接成员（包含直接负责人）
A(n)        = n 及所有祖先
S(n)        = n 及所有后代

E_leader(n) = union D_leader(a) for a in A(n)
E_member(n) = union D_member(d) for d in S(n) ∪ E_leader(n)
```

含义：

- **负责人管理权向下继承**：根负责人可编辑/删除/管理整棵子树。
- **成员身份向上继承**：子节点成员自动成为所有祖先的有效成员。
- **祖先负责人可在后代评论**；父节点普通成员不能评论未加入的子节点。
- 只保存直接关系，不写冗余继承表。

| 操作 | 条件 |
| --- | --- |
| 查看节点/评论/附件 | 任意准入成员 |
| 加/取消表情 | 任意准入成员 |
| 创建根节点、根排序 | manager（任一部门 admin）或系统管理员 |
| 创建子节点 | 父节点有效负责人或系统管理员 |
| 编辑/完成/标签/预期完成时间 | 节点有效负责人或系统管理员 |
| 删除节点、管理成员、子节点排序 | 节点有效负责人或系统管理员 |
| 发表评论 | 节点有效成员或系统管理员 |
| 删除评论 | 评论作者或系统管理员 |

创建节点时可以指定负责人/成员；未指定负责人时创建者自动成为直接负责人。每节点至少保留一个直接负责人，负责人和成员不能重叠。系统管理员可直接评论，但不会自动计入有效成员。

当前版本不支持移动节点；建错父节点需要删除后重建。删除是级联的，前端会展示子树规模并二次确认。

## Keycloak

创建 confidential client，启用 authorization code flow + PKCE，回调：

```text
{APP_PUBLIC_URL}/api/auth/callback
```

默认 scopes 需要包含 `openid profile email roles`，并确保 `roles` scope 的 mapper 把 realm roles 写入 ID token（`realm_access.roles`）。应用只认 `permission-*` 角色做授权，`position-*` 只用于展示。

邮箱 mapper 需将 `email` 写入 ID token。邮箱仅向本人展示，不要求 `email_verified` 为 `true`，也不用于识别用户或判断权限。

其他用户的姓名/头像/职务由只读 service account 查询 Admin API：

```text
GET /admin/realms/{realm}/users/{subject}
GET /admin/realms/{realm}/users/{subject}/role-mappings/realm/composite
```

service account 需要 `realm-management`（本部署中为 `master-realm`）的 `view-users`。结果缓存 10 分钟，失败缓存 30 秒；Admin API 不可用时回退为 subject、无头像、无职务。该数据不参与授权。

## 评论图片

- 每条评论最多 4 张，单张 ≤ 5MB。
- 只接受 JPEG / PNG / WebP / GIF；服务端用文件头检测，拒绝 SVG。
- 图片逐张上传，失败时只重试当前文件；发表评论时一次性关联已上传图片。
- 未发表的图片仅供上传者关联到原节点，24 小时后过期；服务启动时及每小时清理。
- 对象 key 前缀 `guild-go/comments/...`，bucket 保持私有。
- 图片通过 `GET /api/attachments/{id}` 经应用鉴权后流式返回。
- 移除图片、清理过期上传、删除评论/节点时尽力删除 S3 对象；失败只记日志，可能产生孤儿对象。

## 会话与单实例

会话保存在进程内内存中，默认 30 分钟滑动过期。重启服务会让所有用户重新登录；当前设计面向单实例部署，不引入 Redis 或会话表。CSRF token 存在会话中，所有写请求需要 `X-CSRF-Token`。

## 测试

```bash
go test ./...
cd web && pnpm build
```

数据库集成测试需要 `TEST_DATABASE_URL`：

```bash
TEST_DATABASE_URL='postgres://user:pass@host:5432/guild_go_test?sslmode=disable' \
  go test ./internal/integration -v
```

集成测试会清空测试库中的表，请只指向专用测试库。

## 文档

- [API 契约](doc/API.md)
- 旧 Spring 项目与旧 Vue 前端仓库保持不动；本项目是新的 `guild-go` 仓库。
