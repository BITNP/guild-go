# API Reference

本文是 `guild-go` 的 API 契约。所有业务接口前缀为 `/api`，前端与 API 同源。

## 约定

### 认证

浏览器通过 Keycloak OIDC 登录，服务端保持 HttpOnly 会话 Cookie `guild_session`。只有 ID token 的 `realm_access.roles` 中存在已识别 `permission-*` 角色的用户才会被准入。

- `GET /api/auth/status`、`GET /api/auth/login`、`GET /api/auth/callback`、`GET /api/healthz` 公开。
- 其余接口要求有效会话。
- 所有 POST / PUT / PATCH / DELETE 要求 CSRF：先 `GET /api/auth/csrf`，再带 `X-CSRF-Token`。
- 会话默认 30 分钟滑动过期，保存在进程内存中；服务重启后失效。

### 状态码

| 状态码 | 含义 |
| --- | --- |
| 200 | 读取/更新成功 |
| 201 | 创建成功 |
| 204 | 删除/排序/登出成功 |
| 400 | 参数或请求体不合法 |
| 401 | 未登录或会话过期 |
| 403 | 权限不足或 CSRF 无效 |
| 404 | 资源不存在 |
| 413 | 图片/请求体超过大小限制 |
| 500 | 服务器内部错误 |

错误体：

```json
{
  "status": 403,
  "error": "FORBIDDEN",
  "message": "只有节点负责人可以执行此操作"
}
```

CSRF 失败时 `error` 为 `CSRF_INVALID`。

### 时间与 ID

- 所有时间字段为 RFC 3339 UTC 字符串，例如 `2026-10-21T15:30:00Z`。
- `dueAt` 是预期完成时间，可为 `null`。
- `completedAt` 是实际完成时间，`null` 表示未完成；完成时由服务端写入当前时间，重新打开时清空。
- 节点、评论、附件 ID 为正整数。

### 用户形状

`UserSummary`：

```json
{
  "subject": "b1f0c9d2-...",
  "userName": "张三",
  "avatar": "https://login.example.com/.../avatar.png",
  "positions": [
    {
      "code": "position-techdept-head",
      "label": "技术部部长",
      "scope": "department",
      "department": "techdept"
    }
  ]
}
```

`department` 取值：`techdept`、`clinic`、`digimedia`、`techsvc`。`positions` 只用于展示，永不参与授权。

`Profile` 在 `UserSummary` 基础上增加：

```json
{
  "email": "alice@example.com",
  "permissions": {
    "systemAdmin": false,
    "manager": true,
    "accessDepartments": ["techdept"],
    "managedDepartments": ["techdept"]
  }
}
```

`email` 和 `permissions` 仅对本人返回；查看他人时为 `null`。

## Auth

### `GET /api/auth/login?returnTo=/`

开始 OIDC 登录。`returnTo` 必须是本站相对路径；非法值回退为 `/`。

成功时 302 到 Keycloak。发现失败时 302 到 `/login?error=provider_unavailable`。

### `GET /api/auth/callback`

OIDC 回调。校验 state、nonce、issuer、audience、签名和 PKCE。成功后设置会话 Cookie 并跳回 `returnTo`；失败跳 `/login`：

- `not_member`：没有已识别的 `permission-*` 角色
- `no_roles`：ID token 没有 realm roles
- `provider_unavailable`：无法加载 OIDC discovery
- `oidc_failed`：state/nonce/token 交换等失败

### `GET /api/auth/status`

公开，返回当前会话状态：

```json
{"state":"active","user":{ "...Profile..." }}
```

或：

```json
{"state":"anonymous","user":null}
```

`Cache-Control: no-store`。

### `GET /api/auth/csrf`

要求登录。返回：

```json
{"token":"opaque","headerName":"X-CSRF-Token"}
```

### `GET /api/auth/me`

要求登录。返回当前用户 `Profile`。

### `POST /api/auth/logout`

要求登录 + CSRF。删除服务端会话并清 Cookie，返回 204。不登出 Keycloak。

## Users

### `GET /api/users`

要求登录。返回已知用户目录，按 subject 排序。当前用户资料来自会话；其他用户通过 Keycloak Admin API 查询并缓存。Admin API 不可用时回退为 subject、无头像、空职务。

返回 `UserSummary[]`。

### `GET /api/users/{subject}`

要求登录。返回一个用户的 `Profile`。未知 subject 返回 404。查看他人时 `email` 和 `permissions` 为 `null`。

## Overview

### `GET /api/overview`

要求登录。返回总览数据：

```json
{
  "roots": [
    {
      "id": 1,
      "parentId": null,
      "title": "技术部",
      "dueAt": null,
      "completedAt": null,
      "tags": ["部门"],
      "sortOrder": 0,
      "childCount": 5,
      "commentCount": 12,
      "latestCommentAt": "2026-10-21T15:30:00Z",
      "myRelation": "leader",
      "updatedAt": "2026-10-21T15:30:00Z",
      "leaders": [{ "...UserSummary..." }],
      "latestComment": {
        "id": 99,
        "author": { "...UserSummary..." },
        "bodyPreview": "周五晚上可以吗？",
        "createdAt": "2026-10-21T15:30:00Z",
        "reactionSummary": [
          {"emoji": "thumbs_up", "count": 2, "reactedByMe": false}
        ],
        "attachmentCount": 0
      }
    }
  ],
  "recentComments": [
    {
      "id": 100,
      "nodeId": 13,
      "nodeTitle": "面试安排",
      "rootTitle": "技术部",
      "author": { "...UserSummary..." },
      "bodyPreview": "……",
      "createdAt": "2026-10-21T16:00:00Z",
      "reactionSummary": []
    }
  ]
}
```

`myRelation` 为 `leader`、`member` 或 `null`；leader 优先。`latestComment` 为 `null` 表示暂无评论。

## Nodes

节点权限由有效负责人/有效成员决定：

```text
E_leader(n) = 本节点及所有祖先的直接负责人
E_member(n) = 本节点及所有后代的直接成员（含负责人）∪ E_leader(n)
```

### `GET /api/nodes`

要求登录。返回全部节点的扁平列表，供侧栏建树、搜索和筛选：

```json
[
  {
    "id": 12,
    "parentId": 3,
    "title": "秋季招新",
    "dueAt": "2026-11-01T12:00:00Z",
    "completedAt": null,
    "tags": ["招新", "技术部"],
    "sortOrder": 2,
    "childCount": 4,
    "commentCount": 7,
    "latestCommentAt": "2026-10-21T15:30:00Z",
    "myRelation": "member",
    "updatedAt": "2026-10-21T15:30:00Z"
  }
]
```

`myRelation` 是当前用户对该节点的有效关系。

关系图谱直接复用该接口渲染：一次取回全部节点后在浏览器侧布局，因此没有单独的分页或邻居接口。

### `GET /api/nodes/{id}`

要求登录。返回节点详情：

```json
{
  "id": 12,
  "parentId": 3,
  "title": "秋季招新",
  "description": "……",
  "dueAt": "2026-11-01T12:00:00Z",
  "completedAt": null,
  "tags": ["招新"],
  "sortOrder": 2,
  "createdBySubject": "b1f0...",
  "createdBy": { "...UserSummary..." },
  "createdAt": "2026-10-01T00:00:00Z",
  "updatedAt": "2026-10-21T15:30:00Z",
  "breadcrumb": [
    {"id": 1, "title": "公会"},
    {"id": 3, "title": "技术部"}
  ],
  "directLeaders": [{ "...UserSummary..." }],
  "directMembers": [],
  "inheritedLeaderCount": 1,
  "inheritedMemberCount": 3,
  "effectiveMemberCount": 5,
  "subtreeNodeCount": 14,
  "subtreeCommentCount": 23,
  "subtreeAttachmentCount": 4,
  "permissions": {
    "canEdit": true,
    "canDelete": true,
    "canComment": true,
    "canManageMembers": true,
    "canAddChild": true,
    "canReorderChildren": true
  },
  "children": [
    {
      "id": 13,
      "parentId": 12,
      "title": "面试安排",
      "dueAt": null,
      "completedAt": null,
      "tags": ["面试"],
      "sortOrder": 0,
      "childCount": 0,
      "commentCount": 3,
      "latestCommentAt": "2026-10-21T15:00:00Z",
      "myRelation": "member",
      "updatedAt": "2026-10-21T15:00:00Z",
      "latestComment": {
        "id": 99,
        "author": { "...UserSummary..." },
        "bodyPreview": "周五晚上可以吗？",
        "createdAt": "2026-10-21T15:00:00Z",
        "reactionSummary": [],
        "attachmentCount": 0
      }
    }
  ]
}
```

- `breadcrumb` 只包含祖先，不含当前节点。
- `children` 按 `sortOrder`、`id` 排序。
- `latestComment` 为直接子节点最新 1 条评论；`bodyPreview` 截断约 200 字。
- `subtree*` 统计用于删除确认。
- `permissions.canComment` 对系统管理员为 `true`，即使不是有效成员。

### `POST /api/nodes`

要求登录 + CSRF。

创建根节点需要 manager（任一部门 admin）或系统管理员；创建子节点需要父节点有效负责人或系统管理员。

请求：

```json
{
  "parentId": 3,
  "title": "面试安排",
  "description": "……",
  "dueAt": "2026-11-01T12:00:00Z",
  "tags": ["面试"],
  "leaderSubjects": ["subject-a"],
  "memberSubjects": ["subject-b"]
}
```

- `parentId` 为 `null` 或省略表示根节点。
- `title` 1–200 字符；`description` ≤ 20,000；标签 ≤ 10 个、每个 ≤ 32 字符。
- 所有 subject 必须是已知用户。
- `leaderSubjects` 与 `memberSubjects` 不能重叠。
- 未指定负责人时，创建者自动成为直接负责人。
- `memberSubjects` 省略或为空时，复制父节点当前的直接成员，并排除已选负责人；根节点保持为空。复制后的名单不会随父节点后续变更自动更新。
- 新节点追加到同级末尾。

成功返回 201 + `NodeDetail`。

### `PUT /api/nodes/{id}`

要求节点有效负责人或系统管理员 + CSRF。替换可编辑字段：

```json
{
  "title": "面试安排",
  "description": "……",
  "dueAt": "2026-11-01T12:00:00Z",
  "tags": ["面试"],
  "completed": true
}
```

- `dueAt` 传 `null` 清除预期完成时间。
- `completed=true` 且当前未完成时写入 `completedAt=now()`；已完成再传 `true` 不改变原时间；`false` 清空。
- 返回更新后的 `NodeDetail`。

### `DELETE /api/nodes/{id}`

要求节点有效负责人或系统管理员 + CSRF。级联删除子树、成员、评论、反应和附件元数据，并尽力删除 S3 对象。返回 204。

### `PUT /api/nodes/order`

要求登录 + CSRF。

请求：

```json
{"parentId": 3, "ids": [13, 12, 15]}
```

- `parentId=null` 表示根节点；根排序要求 manager 或系统管理员。
- 子节点排序要求父节点有效负责人或系统管理员。
- `ids` 必须完整覆盖该父节点的全部直接子节点/全部根节点，不能重复，否则 400。
- 成功 204。

### `GET /api/nodes/{id}/members`

要求登录。返回有效负责人/成员及其来源：

```json
{
  "leaders": [
    {
      "user": { "...UserSummary..." },
      "direct": true,
      "sources": []
    },
    {
      "user": { "...UserSummary..." },
      "direct": false,
      "sources": [{"id": 1, "title": "公会"}]
    }
  ],
  "members": [
    {
      "user": { "...UserSummary..." },
      "direct": false,
      "sources": [{"id": 13, "title": "面试安排"}]
    }
  ]
}
```

`leaders` 为有效负责人；`members` 为有效成员中不是负责人的部分。`direct=false` 表示继承，`sources` 为来源节点。

### `PUT /api/nodes/{id}/members`

要求节点有效负责人或系统管理员 + CSRF。只替换直接关系，继承关系自动重算。

```json
{
  "leaders": ["subject-a"],
  "members": ["subject-b", "subject-c"]
}
```

- `leaders` 不能为空。
- `members` 省略或为空时，复制父节点当前的直接成员，并排除已选负责人；根节点保持为空。非空时按指定名单保存。复制后的名单不会随父节点后续变更自动更新。
- 负责人和成员不能重叠。
- 所有 subject 必须已知。
- 返回更新后的 `NodeDetail`。

### 不存在的接口

当前版本不支持移动节点。建错父节点只能删除后重建。

## Comments

评论是线性结构，没有楼中楼。时间倒序分页，默认每页 30 条。

### `GET /api/nodes/{id}/comments?before={commentId}&limit=30`

要求登录。返回：

```json
{
  "items": [
    {
      "id": 99,
      "nodeId": 12,
      "author": { "...UserSummary..." },
      "body": "周五晚上可以吗？",
      "createdAt": "2026-10-21T15:30:00Z",
      "attachments": [
        {
          "id": 7,
          "url": "/api/attachments/7",
          "contentType": "image/png",
          "sizeBytes": 123456
        }
      ],
      "reactions": [
        {"emoji": "thumbs_up", "count": 2, "reactedByMe": true}
      ],
      "canDelete": true
    }
  ],
  "hasMore": true,
  "nextBefore": 71
}
```

- `before` 为上一页最后一条评论 ID，可省略。
- `limit` 默认 30，最大 100。
- 评论删除权限：作者或系统管理员。

### `POST /api/nodes/{id}/comments`

要求节点有效成员或系统管理员 + CSRF。JSON 请求体：

| 字段 | 说明 |
| --- | --- |
| `body` | 纯文本，≤ 5000 字，可为空但必须至少有一张图片 |
| `attachmentIds` | 0–4 个已上传图片 ID，按展示顺序排列 |

图片必须由当前用户上传到当前节点，尚未用于其他评论，且上传未超过 24 小时。
评论与图片关联在同一事务中完成；失败时保留已上传图片，重试发表无需再次上传。成功 201 + `Comment`。

### `POST /api/nodes/{id}/comment-images`

要求节点有效成员或系统管理员 + CSRF。`multipart/form-data` 的 `file` 字段只接收一个文件，≤ 5MB。
允许 JPEG、PNG、WebP、GIF；服务端根据文件头检测，拒绝 SVG。
成功 201 + `{"id": 123}`，供发表评论时使用。每张图片独立上传和重试，上传本身不创建评论。
未发表的图片不开放读取，24 小时后过期；服务启动时及每小时清理过期图片。

### `DELETE /api/comment-images/{id}`

要求上传者 + CSRF。删除自己尚未发表的图片及 S3 对象，返回 204。
图片不存在、不属于自己或已发表时返回 404。

### `DELETE /api/comments/{id}`

要求评论作者或系统管理员 + CSRF。删除评论、反应、附件元数据和 S3 对象。返回 204。

### `GET /api/comments/{id}/reactions/{emoji}`

要求登录。任意准入成员都可查看指定表情的人员，按回应时间从早到晚排列。
返回 `UserSummary[]`（`subject`、`userName`、`avatar`、`positions`），不包含邮箱和权限。
无人使用该表情时返回 `[]`；评论不存在返回 404，表情不受支持返回 400。

### `PUT /api/comments/{id}/reactions/{emoji}`

要求登录 + CSRF。任意准入成员都可反应。`emoji` 取值：

```text
thumbs_up, thumbs_down, laugh, hooray, confused, heart, rocket, eyes
```

切换当前用户的反应，返回：

```json
{
  "reactions": [
    {"emoji": "thumbs_up", "count": 2, "reactedByMe": true}
  ]
}
```

### `GET /api/attachments/{id}`

要求登录。从 S3 流式返回图片：

- `Content-Type`：存储时的媒体类型
- `Cache-Control: private, max-age=3600`
- `X-Content-Type-Options: nosniff`
- `Content-Disposition: inline`

不存在返回 404。

## Health

### `GET /api/healthz`

公开。数据库可达时：

```json
{"status":"ok"}
```

不可达时 503。

## 权限速查

| 操作 | 条件 |
| --- | --- |
| 查看节点/评论/附件 | 任意准入成员 |
| 加/取消表情 | 任意准入成员 |
| 创建根节点、根排序 | manager 或系统管理员 |
| 创建子节点 | 父节点 `E_leader` 或系统管理员 |
| 编辑/完成/标签/预期完成时间 | 节点 `E_leader` 或系统管理员 |
| 删除节点 | 节点 `E_leader` 或系统管理员 |
| 管理直接负责人/成员 | 节点 `E_leader` 或系统管理员 |
| 调整直接子节点顺序 | 节点 `E_leader` 或系统管理员 |
| 发表评论 | 节点 `E_member` 或系统管理员 |
| 删除评论 | 评论作者或系统管理员 |

有效关系只存直接成员，继承关系实时计算。`position-*` 只用于展示，永远不授权。
