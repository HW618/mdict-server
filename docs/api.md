# API 接口文档

## 概述

Mdict Server 提供 RESTful API 接口，用于词典查询、用户管理和系统管理。

### 基础信息

| 项目 | 说明 |
|------|------|
| 基础 URL | `http://localhost:8080/api/v1` |
| 认证方式 | JWT Bearer Token |
| 内容类型 | `application/json` |
| 字符编码 | UTF-8 |

### 认证方式

除登录接口外，所有 API 都需要在请求头中携带 JWT Token：

```http
Authorization: Bearer <your-token>
```

### 通用响应格式

**成功响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**错误响应：**

```json
{
  "code": 40001,
  "message": "Invalid username or password",
  "data": null
}
```

**分页响应：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [ ... ],
    "total": 100,
    "page": 1,
    "page_size": 20,
    "total_pages": 5
  }
}
```

### 错误码

| 错误码 | HTTP 状态码 | 说明 |
|--------|-------------|------|
| `0` | 200 | 成功 |
| `40001` | 400 | 请求参数错误 |
| `40002` | 400 | 请求体解析失败 |
| `40101` | 401 | 未提供认证信息 |
| `40102` | 401 | Token 已过期 |
| `40103` | 401 | Token 无效 |
| `40104` | 401 | 用户名或密码错误 |
| `40301` | 403 | 权限不足 |
| `40401` | 404 | 资源不存在 |
| `40901` | 409 | 资源已存在（用户名重复） |
| `41301` | 413 | 文件过大 |
| `42901` | 429 | 请求过于频繁 |
| `50001` | 500 | 服务器内部错误 |
| `50002` | 500 | 数据库错误 |
| `50003` | 500 | 词典文件读取错误 |

---

## 认证接口

### POST /api/v1/auth/login

用户登录，获取 JWT Token。

**认证：** 无需

**请求体：**

```json
{
  "username": "admin",
  "password": "your-password"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 用户名 |
| `password` | string | ✅ | 密码 |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 7200,
    "user": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "admin",
      "permissions": {
        "can_use_api": true,
        "is_dict_admin": true,
        "is_user_admin": true
      }
    }
  }
}
```

**错误响应：**

- `40104` - 用户名或密码错误
- `42901` - 登录尝试过于频繁（5次/分钟）

**curl 示例：**

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "your-password"}'
```

---

### POST /api/v1/auth/refresh

刷新 Access Token。

**认证：** 无需（使用 Refresh Token）

**请求体：**

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `refresh_token` | string | ✅ | Refresh Token |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 7200
  }
}
```

**错误响应：**

- `40102` - Refresh Token 已过期
- `40103` - Refresh Token 无效

**curl 示例：**

```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "your-refresh-token"}'
```

---

### POST /api/v1/auth/logout

用户登出，吊销 Refresh Token。

**认证：** 需要

**请求体：**

```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `refresh_token` | string | ❌ | Refresh Token，不传则吊销所有 |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "Logged out successfully"
}
```

**curl 示例：**

```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer your-access-token" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "your-refresh-token"}'
```

---

## 查询接口

### GET /api/v1/search

精确查询单词释义。

**认证：** 需要（`can_use_api` 权限或 Web 登录）

**查询参数：**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `word` | string | ✅ | 查询的单词 |
| `dict_id` | string | ❌ | 指定词典 ID，不传则查询所有启用的词典 |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "word": "hello",
    "results": [
      {
        "dict_id": "oxford",
        "dict_name": "Oxford Advanced Dictionary",
        "html": "<div class=\"entry\">...</div>",
        "has_audio": true,
        "audio_url": "/api/v1/assets/oxford/hello.mp3"
      }
    ]
  }
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `word` | string | 查询的单词 |
| `results` | array | 查询结果列表 |
| `results[].dict_id` | string | 词典 ID |
| `results[].dict_name` | string | 词典名称 |
| `results[].html` | string | 单词释义 HTML（已消毒） |
| `results[].has_audio` | boolean | 是否有音频 |
| `results[].audio_url` | string | 音频文件 URL（如果有） |

**错误响应：**

- `40001` - word 参数缺失
- `40401` - 未找到匹配结果

**curl 示例：**

```bash
# 查询所有词典
curl "http://localhost:8080/api/v1/search?word=hello" \
  -H "Authorization: Bearer your-token"

# 查询指定词典
curl "http://localhost:8080/api/v1/search?word=hello&dict_id=oxford" \
  -H "Authorization: Bearer your-token"
```

---

### GET /api/v1/search/fuzzy

模糊查询单词。

**认证：** 需要（`can_use_api` 权限或 Web 登录）

**查询参数：**

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `keyword` | string | ✅ | - | 查询关键词（最少2字符） |
| `dict_id` | string | ❌ | - | 指定词典 ID |
| `page` | int | ❌ | 1 | 页码 |
| `page_size` | int | ❌ | 20 | 每页数量（最大100） |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "items": [
      {
        "word": "hello",
        "dict_id": "oxford",
        "dict_name": "Oxford Advanced Dictionary"
      },
      {
        "word": "help",
        "dict_id": "oxford",
        "dict_name": "Oxford Advanced Dictionary"
      }
    ],
    "total": 150,
    "page": 1,
    "page_size": 20,
    "total_pages": 8
  }
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `items` | array | 匹配的单词列表 |
| `items[].word` | string | 单词 |
| `items[].dict_id` | string | 词典 ID |
| `items[].dict_name` | string | 词典名称 |
| `total` | int | 总匹配数 |
| `page` | int | 当前页码 |
| `page_size` | int | 每页数量 |
| `total_pages` | int | 总页数 |

**错误响应：**

- `40001` - keyword 参数缺失或长度不足

**curl 示例：**

```bash
# 基本模糊查询
curl "http://localhost:8080/api/v1/search/fuzzy?keyword=hel" \
  -H "Authorization: Bearer your-token"

# 带分页的查询
curl "http://localhost:8080/api/v1/search/fuzzy?keyword=hel&page=1&page_size=10" \
  -H "Authorization: Bearer your-token"
```

---

## 词典管理接口

### GET /api/v1/dicts

获取词典列表。

**认证：** 需要

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "oxford",
      "filename": "oxford.mdx",
      "title": "Oxford Advanced Dictionary",
      "description": "Oxford Advanced Learner's Dictionary",
      "file_size": 52428800,
      "entry_count": 85000,
      "is_enabled": true,
      "has_mdd": true,
      "created_at": "2026-06-12T10:00:00Z",
      "updated_at": "2026-06-12T10:00:00Z"
    }
  ]
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 词典 ID（MD5 哈希前8位） |
| `filename` | string | 文件名 |
| `title` | string | 词典标题（从元数据读取） |
| `description` | string | 词典描述 |
| `file_size` | int64 | 文件大小（字节） |
| `entry_count` | int64 | 词条数量 |
| `is_enabled` | boolean | 是否启用 |
| `has_mdd` | boolean | 是否有配套 mdd 文件 |
| `created_at` | string | 创建时间（ISO 8601） |
| `updated_at` | string | 更新时间（ISO 8601） |

**curl 示例：**

```bash
curl http://localhost:8080/api/v1/dicts \
  -H "Authorization: Bearer your-token"
```

---

### PATCH /api/v1/dicts/:id/status

切换词典启用状态。

**认证：** 需要（`is_dict_admin` 权限）

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | string | 词典 ID |

**请求体：**

```json
{
  "is_enabled": false
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `is_enabled` | boolean | ✅ | 目标状态 |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "Dictionary status updated"
}
```

**错误响应：**

- `40401` - 词典不存在

**curl 示例：**

```bash
curl -X PATCH http://localhost:8080/api/v1/dicts/oxford/status \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{"is_enabled": false}'
```

---

### POST /api/v1/dicts/upload

上传词典文件。

**认证：** 需要（`is_dict_admin` 权限）

**请求类型：** `multipart/form-data`

**表单字段：**

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `file` | File | ✅ | .mdx 或 .mdd 文件 |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "Dictionary uploaded successfully",
  "data": {
    "id": "new_dict_id",
    "filename": "new_dict.mdx",
    "file_size": 10485760
  }
}
```

**错误响应：**

- `40001` - 文件格式错误（仅支持 .mdx/.mdd）
- `40901` - 词典已存在
- `41301` - 文件过大（默认限制 500MB）

**curl 示例：**

```bash
curl -X POST http://localhost:8080/api/v1/dicts/upload \
  -H "Authorization: Bearer your-token" \
  -F "file=@/path/to/dictionary.mdx"
```

---

### GET /api/v1/dicts/:id/download

下载词典文件。

**认证：** 需要（`is_dict_admin` 权限）

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | string | 词典 ID |

**成功响应 (200)：**

返回文件流，Content-Type 为 `application/octet-stream`。

**错误响应：**

- `40401` - 词典不存在

**curl 示例：**

```bash
curl -O -J http://localhost:8080/api/v1/dicts/oxford/download \
  -H "Authorization: Bearer your-token"
```

---

### DELETE /api/v1/dicts/:id

删除词典。

**认证：** 需要（`is_dict_admin` 权限）

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | string | 词典 ID |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "Dictionary deleted"
}
```

**错误响应：**

- `40401` - 词典不存在

**curl 示例：**

```bash
curl -X DELETE http://localhost:8080/api/v1/dicts/oxford \
  -H "Authorization: Bearer your-token"
```

---

## 用户管理接口

### GET /api/v1/users

获取用户列表。

**认证：** 需要（`is_user_admin` 权限）

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "username": "john",
      "api_token": "mdtk_a1b2c3d4e5f6...",
      "permissions": {
        "can_use_api": true,
        "is_dict_admin": false,
        "is_user_admin": false
      },
      "is_active": true,
      "created_at": "2026-06-12T10:00:00Z",
      "updated_at": "2026-06-12T10:00:00Z"
    }
  ]
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 用户 ID（UUID） |
| `username` | string | 用户名 |
| `api_token` | string | API Token（仅管理员可见） |
| `permissions` | object | 权限配置 |
| `permissions.can_use_api` | boolean | 是否允许调用 API |
| `permissions.is_dict_admin` | boolean | 是否为词典管理员 |
| `permissions.is_user_admin` | boolean | 是否为用户管理员 |
| `is_active` | boolean | 账户是否激活 |
| `created_at` | string | 创建时间 |
| `updated_at` | string | 更新时间 |

**curl 示例：**

```bash
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer your-token"
```

---

### POST /api/v1/users

创建新用户。

**认证：** 需要（`is_user_admin` 权限）

**请求体：**

```json
{
  "username": "newuser",
  "password": "optional-password",
  "permissions": {
    "can_use_api": true,
    "is_dict_admin": false,
    "is_user_admin": false
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | ✅ | 用户名（唯一） |
| `password` | string | ❌ | 密码，不填则自动生成 |
| `permissions` | object | ❌ | 权限配置，默认全部为 false |
| `permissions.can_use_api` | boolean | ❌ | 是否允许调用 API |
| `permissions.is_dict_admin` | boolean | ❌ | 是否为词典管理员 |
| `permissions.is_user_admin` | boolean | ❌ | 是否为用户管理员 |

**成功响应 (201)：**

```json
{
  "code": 0,
  "message": "User created successfully",
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "username": "newuser",
    "api_token": "mdtk_x1y2z3...",
    "password": "auto-generated-password",
    "permissions": {
      "can_use_api": true,
      "is_dict_admin": false,
      "is_user_admin": false
    }
  }
}
```

**注意：** `password` 字段仅在创建时返回一次，后续无法查看。

**错误响应：**

- `40901` - 用户名已存在

**curl 示例：**

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newuser",
    "permissions": {
      "can_use_api": true,
      "is_dict_admin": false,
      "is_user_admin": false
    }
  }'
```

---

### DELETE /api/v1/users/:id

删除用户。

**认证：** 需要（`is_user_admin` 权限）

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | string | 用户 ID |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "User deleted"
}
```

**错误响应：**

- `40301` - 不能删除管理员账户
- `40401` - 用户不存在

**curl 示例：**

```bash
curl -X DELETE http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440001 \
  -H "Authorization: Bearer your-token"
```

---

### PUT /api/v1/users/:id/permissions

修改用户权限。

**认证：** 需要（`is_user_admin` 权限）

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | string | 用户 ID |

**请求体：**

```json
{
  "can_use_api": true,
  "is_dict_admin": true,
  "is_user_admin": false
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `can_use_api` | boolean | ✅ | 是否允许调用 API |
| `is_dict_admin` | boolean | ✅ | 是否为词典管理员 |
| `is_user_admin` | boolean | ✅ | 是否为用户管理员 |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "Permissions updated"
}
```

**错误响应：**

- `40301` - 不能移除最后一个管理员的权限
- `40401` - 用户不存在

**curl 示例：**

```bash
curl -X PUT http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440001/permissions \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "can_use_api": true,
    "is_dict_admin": true,
    "is_user_admin": false
  }'
```

---

### POST /api/v1/users/:id/reset-token

重置用户 API Token。

**认证：** 需要（`is_user_admin` 权限）

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `id` | string | 用户 ID |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "API token reset successfully",
  "data": {
    "api_token": "mdtk_new_token..."
  }
}
```

**错误响应：**

- `40401` - 用户不存在

**curl 示例：**

```bash
curl -X POST http://localhost:8080/api/v1/users/550e8400-e29b-41d4-a716-446655440001/reset-token \
  -H "Authorization: Bearer your-token"
```

---

## Agent Skill 接口

### GET /api/v1/skill.json

获取 Agent Skill 配置文件（OpenAPI 3.0 格式）。

**认证：** 无需

**成功响应 (200)：**

返回 OpenAPI 3.0 规范的 JSON 文件，包含：
- 服务基本信息
- 服务器地址
- 搜索接口定义
- 认证方式说明

**curl 示例：**

```bash
curl http://localhost:8080/api/v1/skill.json
```

**在 AI 智能体中使用：**

1. 获取 skill.json 文件
2. 配置到你的 AI 智能体平台（如 Dify、Coze、GPTs）
3. 在配置中填入你的 API Token

---

## 系统接口

### GET /api/v1/health

健康检查。

**认证：** 无需

**成功响应 (200)：**

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 3600,
  "dicts_loaded": 3
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `status` | string | 服务状态（healthy/unhealthy） |
| `version` | string | 服务版本号 |
| `uptime` | int64 | 运行时间（秒） |
| `dicts_loaded` | int | 已加载词典数量 |

**curl 示例：**

```bash
curl http://localhost:8080/api/v1/health
```

---

### GET /api/v1/stats

系统统计信息。

**认证：** 需要（`is_user_admin` 权限）

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_dicts": 3,
    "enabled_dicts": 2,
    "total_users": 5,
    "total_entries": 250000,
    "uptime_seconds": 86400
  }
}
```

**响应字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `total_dicts` | int | 词典总数 |
| `enabled_dicts` | int | 已启用词典数 |
| `total_users` | int | 用户总数 |
| `total_entries` | int64 | 总词条数 |
| `uptime_seconds` | int64 | 运行时间（秒） |

**curl 示例：**

```bash
curl http://localhost:8080/api/v1/stats \
  -H "Authorization: Bearer your-token"
```

---

## 静态资源接口

### GET /api/v1/assets/:dict_id/*path

获取词典的静态资源（图片、音频等）。

**认证：** 需要

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `dict_id` | string | 词典 ID |
| `path` | string | 资源路径 |

**成功响应 (200)：**

返回资源文件，Content-Type 根据文件扩展名自动设置。

**支持的 MIME 类型：**

| 扩展名 | MIME 类型 |
|--------|-----------|
| `.png` | `image/png` |
| `.jpg` / `.jpeg` | `image/jpeg` |
| `.gif` | `image/gif` |
| `.mp3` | `audio/mpeg` |
| `.wav` | `audio/wav` |
| `.ogg` | `audio/ogg` |
| `.css` | `text/css` |
| `.js` | `application/javascript` |

**错误响应：**

- `40401` - 词典不存在或资源不存在

**curl 示例：**

```bash
# 获取图片
curl http://localhost:8080/api/v1/assets/oxford/images/hello.png \
  -H "Authorization: Bearer your-token" \
  -o hello.png

# 获取音频
curl http://localhost:8080/api/v1/assets/oxford/audio/hello.mp3 \
  -H "Authorization: Bearer your-token" \
  -o hello.mp3
```

---

## 权限说明

| 接口 | 所需权限 |
|------|----------|
| POST /api/v1/auth/login | 无（公开） |
| POST /api/v1/auth/refresh | 无（使用 Refresh Token） |
| POST /api/v1/auth/logout | 需要登录 |
| GET /api/v1/search* | `can_use_api` 或 Web 登录 |
| GET /api/v1/dicts | 需要登录 |
| PATCH /api/v1/dicts/:id/status | `is_dict_admin` |
| POST /api/v1/dicts/upload | `is_dict_admin` |
| GET /api/v1/dicts/:id/download | `is_dict_admin` |
| DELETE /api/v1/dicts/:id | `is_dict_admin` |
| GET /api/v1/users | `is_user_admin` |
| POST /api/v1/users | `is_user_admin` |
| DELETE /api/v1/users/:id | `is_user_admin` |
| PUT /api/v1/users/:id/permissions | `is_user_admin` |
| POST /api/v1/users/:id/reset-token | `is_user_admin` |
| GET /api/v1/skill.json | 无（公开） |
| GET /api/v1/health | 无（公开） |
| GET /api/v1/stats | `is_user_admin` |
| GET /api/v1/assets/:id/* | 需要登录 |

---

## 限流说明

| 接口类型 | 限流规则 |
|----------|----------|
| 登录接口 | 5 次/分钟/IP |
| 查询接口 | 100 次/分钟/用户 |
| 管理接口 | 60 次/分钟/用户 |
| 上传接口 | 10 次/分钟/用户 |

超过限流将返回 `42901` 错误码。

---

## 更新日志

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2026-06-12 | 初始版本 |
