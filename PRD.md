# Mdict Server - 产品需求文档 (PRD)

## 版本信息

| 版本 | 日期 | 作者 | 说明 |
|------|------|------|------|
| 1.0 | 2026-06-12 | - | 初始版本 |

---

## 1. 项目概述

### 1.1 项目背景

开发一个基于 Go 语言的在线 Mdx/Mdd 词典查询与管理服务，支持 Docker 容器化部署。该服务为个人和团队提供便捷的词典查询能力，同时支持 AI 智能体集成。

### 1.2 核心价值

- **轻量化部署**：单二进制文件 + SQLite，无需外部依赖
- **高性能查询**：内存映射词典文件，毫秒级响应
- **安全可靠**：JWT 鉴权 + RBAC 权限控制
- **AI 友好**：自动生成 Agent Skill 配置文件

### 1.3 目标用户

| 用户类型 | 使用场景 |
|----------|----------|
| 个人用户 | 本地词典查询、学习辅助 |
| 团队部署 | 共享词典资源、统一管理 |
| AI 智能体 | 通过 API 自动查询单词释义 |

---

## 2. 技术栈与架构

### 2.1 技术选型

| 组件 | 技术选择 | 版本要求 | 选择理由 |
|------|----------|----------|----------|
| 编程语言 | Go | 1.22+ | 高性能、静态编译、部署简单 |
| Web 框架 | Gin | v1.10+ | 高性能、生态成熟、中间件丰富 |
| 词典解析 | lib-x/mdx | latest | 专为 Mdx/Mdd 格式设计 |
| 数据库 | SQLite | 3.40+ | 轻量、嵌入式、无需额外部署 |
| 驱动 | modernc.org/sqlite | latest | 纯 Go 实现，无 CGO 依赖 |
| 前端框架 | Alpine.js | v3 | 轻量、无构建步骤、声明式 |
| CSS 框架 | Tailwind CSS | v3 (CDN) | 实用优先、响应式、无构建 |
| HTML 消毒 | bluemonday | latest | Go 生态标准、配置灵活 |
| 日志库 | zerolog | latest | 结构化 JSON、高性能 |
| JWT 库 | golang-jwt/jwt/v5 | v5 | 标准库、维护活跃 |

### 2.2 系统架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      Nginx / Reverse Proxy                   │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│                    Mdict Server (Go)                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │  Gin Router  │  │  Middleware  │  │  Static Assets   │  │
│  │  (REST API)  │──│  (JWT/RBAC)  │──│  (Embedded HTML) │  │
│  └──────┬───────┘  └──────────────┘  └──────────────────┘  │
│         │                                                    │
│  ┌──────▼───────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │   Handlers   │  │  Dict Engine │  │  User Store      │  │
│  │  (Business)  │──│  (lib-x/mdx) │  │  (SQLite)        │  │
│  └──────────────┘  └──────┬───────┘  └──────────────────┘  │
│                           │                                  │
└───────────────────────────┼──────────────────────────────────┘
                            │
┌───────────────────────────▼──────────────────────────────────┐
│                     File System                              │
│  /dicts/*.mdx  /dicts/*.mdd  /data/mdict.db                 │
└──────────────────────────────────────────────────────────────┘
```

### 2.3 目录结构

```
mdict-server/
├── cmd/
│   └── server/
│       └── main.go                 # 程序入口
├── internal/
│   ├── config/
│   │   └── config.go               # 配置加载与验证
│   ├── auth/
│   │   ├── jwt.go                  # JWT 生成与验证
│   │   └── middleware.go           # 认证/授权中间件
│   ├── dict/
│   │   ├── engine.go               # 词典引擎核心
│   │   ├── loader.go               # 词典文件加载器
│   │   └── search.go               # 搜索算法实现
│   ├── handlers/
│   │   ├── auth.go                 # 认证处理器
│   │   ├── dict.go                 # 词典管理处理器
│   │   ├── search.go               # 查询处理器
│   │   ├── user.go                 # 用户管理处理器
│   │   ├── skill.go                # Agent Skill 处理器
│   │   └── health.go               # 健康检查处理器
│   ├── models/
│   │   ├── user.go                 # 用户模型
│   │   └── dict.go                 # 词典元数据模型
│   ├── store/
│   │   ├── sqlite.go               # SQLite 连接管理
│   │   ├── user_store.go           # 用户数据访问
│   │   └── dict_store.go           # 词典状态存储
│   └── middleware/
│       ├── cors.go                 # CORS 中间件
│       ├── ratelimit.go            # 限流中间件
│       └── logger.go               # 请求日志中间件
├── templates/
│   ├── index.html                  # 查询主页
│   └── admin.html                  # 管理后台
├── migrations/
│   └── 001_init.sql                # 数据库初始化脚本
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
├── .github/
│   └── workflows/
│       └── docker-build.yml
├── .env.example                    # 环境变量示例
├── PRD.md                          # 本文档
└── README.md
```

---

## 3. 环境变量配置

### 3.1 配置变量表

| 变量名 | 类型 | 默认值 | 必填 | 说明 |
|--------|------|--------|------|------|
| `SERVER_ADDR` | string | `0.0.0.0` | 否 | 服务监听地址 |
| `SERVER_PORT` | int | `8080` | 否 | 服务监听端口 |
| `DICT_DIR` | string | `./dicts` | 否 | 词典文件目录路径 |
| `DATA_DIR` | string | `./data` | 否 | 数据持久化目录（SQLite） |
| `ADMIN_USER` | string | *随机生成* | 否 | 管理员用户名 |
| `ADMIN_PASS` | string | *随机生成* | 否 | 管理员密码 |
| `JWT_SECRET` | string | *随机生成* | 否 | JWT 签名密钥 |
| `JWT_ACCESS_TTL` | duration | `2h` | 否 | Access Token 有效期 |
| `JWT_REFRESH_TTL` | duration | `168h` | 否 | Refresh Token 有效期（7天） |
| `SKILL_SERVER_URL` | string | `http://localhost:8080` | 否 | Agent Skill 服务器地址 |
| `MAX_UPLOAD_SIZE` | string | `500MB` | 否 | 单文件上传大小限制 |
| `RATE_LIMIT` | int | `100` | 否 | 每分钟请求限制（0=不限制） |
| `LOG_LEVEL` | string | `info` | 否 | 日志级别 (debug/info/warn/error) |
| `LOG_FORMAT` | string | `json` | 否 | 日志格式 (json/text) |
| `CORS_ORIGINS` | string | `*` | 否 | 允许的跨域来源（逗号分隔） |

### 3.2 初始化逻辑

```go
// 启动时执行以下步骤：
// 1. 加载 .env 文件（如果存在）
// 2. 读取环境变量，未设置则使用默认值
// 3. 验证配置合法性（端口范围、目录权限等）
// 4. 如果 ADMIN_USER/ADMIN_PASS 未设置，随机生成并打印到日志
// 5. 如果 JWT_SECRET 未设置，随机生成 32 字节密钥
// 6. 确保 DICT_DIR 和 DATA_DIR 目录存在
```

### 3.3 随机凭证生成规则

| 字段 | 生成规则 | 示例 |
|------|----------|------|
| ADMIN_USER | 8位随机字符串（小写字母+数字） | `admin_a3x8k2` |
| ADMIN_PASS | 16位随机字符串（大小写+数字+特殊字符） | `Kj9#mP2$xN5@wR8` |
| JWT_SECRET | 32字节 Base64 编码随机串 | `a2V5LW1kc2VydmVyLXNlY3JldA==` |

**重要**：随机生成的凭证必须在首次启动日志中醒目标记，并建议用户设置环境变量持久化。

---

## 4. 数据库设计

### 4.1 SQLite 数据库文件

- **路径**：`{DATA_DIR}/mdict.db`
- **WAL 模式**：启用，提升并发读取性能

### 4.2 表结构

#### users 表

```sql
CREATE TABLE IF NOT EXISTS users (
    id          TEXT PRIMARY KEY,           -- UUID
    username    TEXT NOT NULL UNIQUE,       -- 用户名
    password    TEXT NOT NULL,              -- bcrypt 哈希后的密码
    api_token   TEXT UNIQUE,               -- API 访问 Token（长期有效）
    can_use_api     BOOLEAN DEFAULT FALSE, -- 是否允许调用 API
    is_dict_admin   BOOLEAN DEFAULT FALSE, -- 是否为词典管理员
    is_user_admin   BOOLEAN DEFAULT FALSE, -- 是否为用户管理员
    is_active       BOOLEAN DEFAULT TRUE,  -- 账户是否激活
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_api_token ON users(api_token);
```

#### dicts 表

```sql
CREATE TABLE IF NOT EXISTS dicts (
    id          TEXT PRIMARY KEY,           -- MD5 哈希前8位
    filename    TEXT NOT NULL UNIQUE,       -- 原始文件名
    title       TEXT,                       -- 词典标题（从 mdx 元数据读取）
    description TEXT,                       -- 词典描述
    file_size   INTEGER NOT NULL,           -- 文件大小（字典）
    entry_count INTEGER DEFAULT 0,          -- 词条数量
    is_enabled  BOOLEAN DEFAULT TRUE,       -- 是否启用
    has_mdd     BOOLEAN DEFAULT FALSE,      -- 是否有配套 mdd 文件
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

#### refresh_tokens 表

```sql
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id          TEXT PRIMARY KEY,           -- UUID
    user_id     TEXT NOT NULL,              -- 关联用户
    token       TEXT NOT NULL UNIQUE,       -- Refresh Token
    expires_at  DATETIME NOT NULL,          -- 过期时间
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
```

### 4.3 初始数据

系统首次启动时自动创建默认管理员账户：

```sql
-- 仅在 users 表为空时插入
INSERT INTO users (id, username, password, can_use_api, is_dict_admin, is_user_admin)
VALUES ('admin-001', '{ADMIN_USER}', '{bcrypt_hash}', TRUE, TRUE, TRUE);
```

---

## 5. 核心功能模块

### 5.1 词典引擎模块

#### 5.1.1 词典加载

**启动时加载流程：**

1. 扫描 `DICT_DIR` 目录下所有 `.mdx` 文件
2. 对每个 `.mdx` 文件：
   - 计算文件 MD5 哈希作为 ID（前8位）
   - 检查是否存在同名 `.mdd` 文件
   - 使用 `lib-x/mdx` 解析词典元数据（标题、描述）
   - 统计词条数量
   - 将元数据存入 SQLite
3. 对已在数据库中但文件不存在的词典，标记为 `is_enabled = FALSE`

**词典文件命名规范：**
- 词典文件：`{name}.mdx`
- 媒体文件：`{name}.mdd`（可选，与 .mdx 同名）
- 示例：`oxford.mdx` + `oxford.mdd`

#### 5.1.2 查询接口

**精确查询流程：**

```
请求: GET /api/v1/search?word=hello&dict_id=oxford
处理:
  1. 参数校验（word 必填，dict_id 可选）
  2. 如果指定 dict_id，查询特定词典
  3. 如果未指定，查询所有已启用词典
  4. 对每个词典执行精确匹配
  5. 合并结果，返回
响应: JSON 包含单词释义 HTML
```

**模糊查询流程：**

```
请求: GET /api/v1/search/fuzzy?keyword=hel&page=1&page_size=20
处理:
  1. 参数校验（keyword 必填，最小2字符）
  2. 遍历已启用词典
  3. 执行前缀匹配或包含匹配
  4. 去重并排序
  5. 分页返回
响应: JSON 包含匹配的单词列表
```

#### 5.1.3 并发控制

- 使用 `sync.RWMutex` 保护词典实例的并发访问
- 读操作（查询）使用读锁，支持并发
- 写操作（加载/卸载词典）使用写锁，互斥
- 每个词典实例独立，避免全局锁竞争

#### 5.1.4 静态资源服务

```
请求: GET /api/v1/assets/{dict_id}/{path}
处理:
  1. 验证 dict_id 存在且已启用
  2. 从对应的 .mdd 文件读取资源
  3. 根据文件扩展名设置 Content-Type
  4. 返回二进制数据
响应: 带正确 MIME 类型的二进制内容
```

**MIME 类型映射：**

| 扩展名 | MIME 类型 |
|--------|-----------|
| .png | image/png |
| .jpg/.jpeg | image/jpeg |
| .gif | image/gif |
| .mp3 | audio/mpeg |
| .wav | audio/wav |
| .ogg | audio/ogg |
| .css | text/css |
| .js | application/javascript |

### 5.2 认证与授权模块

#### 5.2.1 JWT Token 机制

**双 Token 架构：**

| Token 类型 | 有效期 | 用途 | 存储位置 |
|------------|--------|------|----------|
| Access Token | 2小时 | API 认证 | 客户端内存/Header |
| Refresh Token | 7天 | 刷新 Access Token | 数据库 + HttpOnly Cookie |

**Token Payload 结构：**

```json
{
  "sub": "user-uuid",
  "username": "john",
  "permissions": {
    "can_use_api": true,
    "is_dict_admin": false,
    "is_user_admin": false
  },
  "iat": 1718188800,
  "exp": 1718196000
}
```

#### 5.2.2 认证流程

**登录流程：**

```
POST /api/v1/auth/login
Request:
{
  "username": "john",
  "password": "secret"
}

Response (成功):
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_in": 7200,
    "user": {
      "id": "uuid",
      "username": "john",
      "permissions": {...}
    }
  }
}
```

**Token 刷新流程：**

```
POST /api/v1/auth/refresh
Request:
{
  "refresh_token": "eyJ..."
}

Response (成功):
{
  "code": 0,
  "message": "success",
  "data": {
    "access_token": "eyJ...",
    "expires_in": 7200
  }
}
```

#### 5.2.3 API Token 机制

每个用户拥有一个长期有效的 API Token，用于：
- Agent Skill 配置
- 外部系统集成
- 脚本调用

**API Token 特点：**
- 长期有效（不过期，除非手动重置）
- 只能由管理员生成或重置
- 与 JWT Access Token 权限相同
- 通过 `Authorization: Bearer <API_TOKEN>` 传递

#### 5.2.4 权限中间件

```go
// 中间件执行顺序：
// 1. CORS 中间件
// 2. 请求日志中间件
// 3. 限流中间件
// 4. JWT 认证中间件（提取用户信息）
// 5. 权限检查中间件（检查具体权限）
// 6. 业务处理器
```

**权限检查规则：**

| 接口 | 所需权限 |
|------|----------|
| POST /api/v1/auth/login | 无（公开） |
| POST /api/v1/auth/refresh | 无（使用 Refresh Token） |
| GET /api/v1/search* | `can_use_api` 或 Web 登录 |
| GET /api/v1/dicts | `can_use_api` 或 Web 登录 |
| PATCH /api/v1/dicts/:id/status | `is_dict_admin` |
| POST /api/v1/dicts/upload | `is_dict_admin` |
| GET /api/v1/users | `is_user_admin` |
| POST /api/v1/users | `is_user_admin` |
| DELETE /api/v1/users/:id | `is_user_admin` |
| PUT /api/v1/users/:id/permissions | `is_user_admin` |

### 5.3 用户管理模块

#### 5.3.1 用户 CRUD

**创建用户：**

```
POST /api/v1/users
Request:
{
  "username": "newuser",
  "password": "optional-auto-generate",
  "permissions": {
    "can_use_api": true,
    "is_dict_admin": false,
    "is_user_admin": false
  }
}

Response:
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "uuid",
    "username": "newuser",
    "api_token": "mdtk_xxxx...",
    "permissions": {...}
  }
}
```

**管理员保护规则：**
- 管理员账户（`is_user_admin = TRUE`）不能被删除
- 管理员的 `is_user_admin` 权限不能被移除
- 至少保留一个管理员账户

#### 5.3.2 API Token 管理

- 创建用户时自动生成 API Token
- API Token 格式：`mdtk_` + 32位随机字符串
- 管理员可重置用户的 API Token
- 用户可查看自己的 API Token（通过 Web 界面）

### 5.4 Web 界面模块

#### 5.4.1 前端技术栈

| 技术 | 版本 | 引入方式 | 用途 |
|------|------|----------|------|
| Alpine.js | 3.x | CDN | 响应式数据绑定 |
| Tailwind CSS | 3.x | CDN | 样式框架 |
| marked.js | latest | CDN | Markdown 渲染（可选） |
| DOMPurify | latest | CDN | HTML 消毒 |

#### 5.4.2 查询主页 (/)

**布局设计：**

```
┌─────────────────────────────────────────────────────┐
│  Logo   [词典选择下拉]   [搜索框]        [登录按钮]  │
├─────────────────────────────────────────────────────┤
│                                                     │
│     ┌───────────────────────────────────────┐       │
│     │                                       │       │
│     │         查询结果展示区域              │       │
│     │         (HTML 渲染)                  │       │
│     │                                       │       │
│     └───────────────────────────────────────┘       │
│                                                     │
│  [音频播放器]  [图片查看器]                          │
│                                                     │
└─────────────────────────────────────────────────────┘
```

**交互流程：**

1. 页面加载时获取已启用词典列表
2. 用户输入单词，支持：
   - 回车查询
   - 点击搜索按钮
   - 输入防抖（300ms）
3. 显示查询结果（HTML 渲染）
4. 自动处理结果中的图片和音频标签

**XSS 防护：**

```javascript
// 使用 DOMPurify 消毒 HTML
const cleanHtml = DOMPurify.sanitize(rawHtml, {
  ALLOWED_TAGS: ['div', 'span', 'p', 'b', 'i', 'u', 'a', 'img', 'audio', 'source', 'br', 'hr', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'table', 'tr', 'td', 'th', 'thead', 'tbody', 'font', 'sup', 'sub', 'blockquote'],
  ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'class', 'style', 'width', 'height', 'controls', 'autoplay', 'loop', 'type', 'color', 'size', 'face']
});
```

#### 5.4.3 管理后台 (/admin)

**布局设计：**

```
┌─────────────────────────────────────────────────────┐
│  Logo   [导航: 词典管理 | 用户管理]     [登出按钮]   │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌─────────────────────────────────────────────┐    │
│  │  词典管理                                   │    │
│  │  ┌─────────────────────────────────────┐    │    │
│  │  │ [拖拽上传区域]                       │    │    │
│  │  └─────────────────────────────────────┘    │    │
│  │  ┌─────────────────────────────────────┐    │    │
│  │  │ 词典列表 (表格)                     │    │    │
│  │  │ 文件名 | 大小 | 条目数 | 状态 | 操作│    │    │
│  │  └─────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────┘    │
│                                                     │
│  ┌─────────────────────────────────────────────┐    │
│  │  用户管理                                   │    │
│  │  ┌─────────────────────────────────────┐    │    │
│  │  │ [新建用户按钮]                       │    │    │
│  │  └─────────────────────────────────────┘    │    │
│  │  ┌─────────────────────────────────────┐    │    │
│  │  │ 用户列表 (表格)                     │    │    │
│  │  │ 用户名 | 权限 | API Token | 操作   │    │    │
│  │  └─────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────┘    │
│                                                     │
└─────────────────────────────────────────────────────┘
```

**管理功能：**

1. **词典管理**
   - 拖拽或点击上传 .mdx/.mdd 文件
   - 显示上传进度条
   - Toggle 开关启用/禁用词典
   - 删除词典（二次确认）
   - 显示词典详细信息

2. **用户管理**
   - 新建用户表单（模态框）
   - 权限勾选框
   - 复制 API Token 按钮
   - 重置 API Token（二次确认）
   - 删除用户（二次确认）

---

## 6. API 接口规范

### 6.1 通用响应格式

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

### 6.2 错误码定义

| 错误码 | HTTP 状态码 | 说明 |
|--------|-------------|------|
| 0 | 200 | 成功 |
| 40001 | 400 | 请求参数错误 |
| 40002 | 400 | 请求体解析失败 |
| 40101 | 401 | 未提供认证信息 |
| 40102 | 401 | Token 已过期 |
| 40103 | 401 | Token 无效 |
| 40104 | 401 | 用户名或密码错误 |
| 40301 | 403 | 权限不足 |
| 40401 | 404 | 资源不存在 |
| 40901 | 409 | 资源已存在（用户名重复） |
| 41301 | 413 | 文件过大 |
| 42901 | 429 | 请求过于频繁 |
| 50001 | 500 | 服务器内部错误 |
| 50002 | 500 | 数据库错误 |
| 50003 | 500 | 词典文件读取错误 |

### 6.3 API 端点详细定义

#### 6.3.1 认证接口

**POST /api/v1/auth/login**

```
描述: 用户登录
认证: 无需
Content-Type: application/json

请求体:
{
  "username": string,    // 必填，用户名
  "password": string     // 必填，密码
}

成功响应 (200):
{
  "code": 0,
  "data": {
    "access_token": string,   // JWT Access Token
    "refresh_token": string,  // JWT Refresh Token
    "expires_in": number,     // Access Token 过期秒数
    "user": {
      "id": string,
      "username": string,
      "permissions": {
        "can_use_api": boolean,
        "is_dict_admin": boolean,
        "is_user_admin": boolean
      }
    }
  }
}

错误响应:
- 40104: 用户名或密码错误
- 42901: 登录尝试过于频繁
```

**POST /api/v1/auth/refresh**

```
描述: 刷新 Access Token
认证: 无需（使用 Refresh Token）

请求体:
{
  "refresh_token": string   // 必填，Refresh Token
}

成功响应 (200):
{
  "code": 0,
  "data": {
    "access_token": string,
    "expires_in": number
  }
}

错误响应:
- 40102: Refresh Token 已过期
- 40103: Refresh Token 无效
```

**POST /api/v1/auth/logout**

```
描述: 用户登出，吊销 Refresh Token
认证: 需要

请求体:
{
  "refresh_token": string   // 可选，不传则吊销所有
}

成功响应 (200):
{
  "code": 0,
  "message": "Logged out successfully"
}
```

#### 6.3.2 查询接口

**GET /api/v1/search**

```
描述: 精确查询单词
认证: 需要 (can_use_api 或 Web 登录)

查询参数:
- word: string, 必填，查询的单词
- dict_id: string, 可选，指定词典 ID

成功响应 (200):
{
  "code": 0,
  "data": {
    "word": "hello",
    "results": [
      {
        "dict_id": "oxford",
        "dict_name": "Oxford Dictionary",
        "html": "<div>...</div>",
        "has_audio": true,
        "audio_url": "/api/v1/assets/oxford/hello.mp3"
      }
    ]
  }
}

错误响应:
- 40001: word 参数缺失
- 40401: 未找到匹配结果
```

**GET /api/v1/search/fuzzy**

```
描述: 模糊查询单词
认证: 需要 (can_use_api 或 Web 登录)

查询参数:
- keyword: string, 必填，查询关键词（最少2字符）
- dict_id: string, 可选，指定词典 ID
- page: int, 可选，默认 1
- page_size: int, 可选，默认 20，最大 100

成功响应 (200):
{
  "code": 0,
  "data": {
    "items": [
      {
        "word": "hello",
        "dict_id": "oxford",
        "dict_name": "Oxford Dictionary"
      }
    ],
    "total": 150,
    "page": 1,
    "page_size": 20,
    "total_pages": 8
  }
}
```

#### 6.3.3 词典管理接口

**GET /api/v1/dicts**

```
描述: 获取词典列表
认证: 需要

成功响应 (200):
{
  "code": 0,
  "data": [
    {
      "id": "oxford",
      "filename": "oxford.mdx",
      "title": "Oxford Advanced Dictionary",
      "description": "...",
      "file_size": 52428800,
      "entry_count": 85000,
      "is_enabled": true,
      "has_mdd": true,
      "created_at": "2026-06-12T10:00:00Z"
    }
  ]
}
```

**PATCH /api/v1/dicts/:id/status**

```
描述: 切换词典启用状态
认证: 需要 (is_dict_admin)

路径参数:
- id: string, 词典 ID

请求体:
{
  "is_enabled": boolean
}

成功响应 (200):
{
  "code": 0,
  "message": "Dictionary status updated"
}
```

**POST /api/v1/dicts/upload**

```
描述: 上传词典文件
认证: 需要 (is_dict_admin)
Content-Type: multipart/form-data

表单字段:
- file: File, 必填，.mdx 或 .mdd 文件

成功响应 (200):
{
  "code": 0,
  "data": {
    "id": "new_dict_id",
    "filename": "new_dict.mdx",
    "file_size": 10485760
  }
}

错误响应:
- 40001: 文件格式错误
- 41301: 文件过大
- 40901: 词典已存在
```

**GET /api/v1/dicts/:id/download**

```
描述: 下载词典文件
认证: 需要 (is_dict_admin)

路径参数:
- id: string, 词典 ID

响应: 文件流
Content-Disposition: attachment; filename="oxford.mdx"
```

**DELETE /api/v1/dicts/:id**

```
描述: 删除词典
认证: 需要 (is_dict_admin)

路径参数:
- id: string, 词典 ID

成功响应 (200):
{
  "code": 0,
  "message": "Dictionary deleted"
}
```

#### 6.3.4 用户管理接口

**GET /api/v1/users**

```
描述: 获取用户列表
认证: 需要 (is_user_admin)

成功响应 (200):
{
  "code": 0,
  "data": [
    {
      "id": "uuid",
      "username": "john",
      "api_token": "mdtk_xxxx...",  // 仅管理员可见
      "permissions": {
        "can_use_api": true,
        "is_dict_admin": false,
        "is_user_admin": false
      },
      "is_active": true,
      "created_at": "2026-06-12T10:00:00Z"
    }
  ]
}
```

**POST /api/v1/users**

```
描述: 创建新用户
认证: 需要 (is_user_admin)

请求体:
{
  "username": string,         // 必填
  "password": string,         // 可选，不填则自动生成
  "permissions": {
    "can_use_api": boolean,
    "is_dict_admin": boolean,
    "is_user_admin": boolean
  }
}

成功响应 (201):
{
  "code": 0,
  "data": {
    "id": "uuid",
    "username": "newuser",
    "api_token": "mdtk_xxxx...",
    "password": "auto-generated-if-empty",  // 仅创建时返回
    "permissions": {...}
  }
}

错误响应:
- 40901: 用户名已存在
```

**DELETE /api/v1/users/:id**

```
描述: 删除用户
认证: 需要 (is_user_admin)

路径参数:
- id: string, 用户 ID

成功响应 (200):
{
  "code": 0,
  "message": "User deleted"
}

错误响应:
- 40301: 不能删除管理员账户
- 40401: 用户不存在
```

**PUT /api/v1/users/:id/permissions**

```
描述: 修改用户权限
认证: 需要 (is_user_admin)

路径参数:
- id: string, 用户 ID

请求体:
{
  "can_use_api": boolean,
  "is_dict_admin": boolean,
  "is_user_admin": boolean
}

成功响应 (200):
{
  "code": 0,
  "message": "Permissions updated"
}

错误响应:
- 40301: 不能移除最后一个管理员的权限
```

**POST /api/v1/users/:id/reset-token**

```
描述: 重置用户 API Token
认证: 需要 (is_user_admin)

路径参数:
- id: string, 用户 ID

成功响应 (200):
{
  "code": 0,
  "data": {
    "api_token": "mdtk_new_token..."
  }
}
```

#### 6.3.5 Agent Skill 接口

**GET /api/v1/skill.json**

```
描述: 获取 Agent Skill 配置文件
认证: 无需

成功响应 (200):
{
  "openapi": "3.0.0",
  "info": {
    "title": "Mdict Dictionary Service",
    "description": "Query word definitions from Mdx dictionaries. Server URL: {SKILL_SERVER_URL}. Use Authorization: Bearer <YOUR_TOKEN> header.",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "{SKILL_SERVER_URL}"
    }
  ],
  "paths": {
    "/api/v1/search": {
      "get": {
        "operationId": "searchWord",
        "summary": "Search word definition",
        "description": "Query the exact definition of a word from enabled dictionaries",
        "parameters": [
          {
            "name": "word",
            "in": "query",
            "required": true,
            "schema": {
              "type": "string"
            },
            "description": "The word to search"
          },
          {
            "name": "dict_id",
            "in": "query",
            "required": false,
            "schema": {
              "type": "string"
            },
            "description": "Optional dictionary ID to search in"
          }
        ],
        "security": [
          {
            "bearerAuth": []
          }
        ],
        "responses": {
          "200": {
            "description": "Word definition found",
            "content": {
              "application/json": {
                "schema": {
                  "$ref": "#/components/schemas/SearchResult"
                }
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "securitySchemes": {
      "bearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      }
    },
    "schemas": {
      "SearchResult": {
        "type": "object",
        "properties": {
          "code": { "type": "integer" },
          "data": {
            "type": "object",
            "properties": {
              "word": { "type": "string" },
              "results": {
                "type": "array",
                "items": {
                  "type": "object",
                  "properties": {
                    "dict_name": { "type": "string" },
                    "html": { "type": "string" }
                  }
                }
              }
            }
          }
        }
      }
    }
  }
}
```

#### 6.3.6 系统接口

**GET /api/v1/health**

```
描述: 健康检查
认证: 无需

成功响应 (200):
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime": 3600,
  "dicts_loaded": 3
}
```

**GET /api/v1/stats**

```
描述: 系统统计（管理员）
认证: 需要 (is_user_admin)

成功响应 (200):
{
  "code": 0,
  "data": {
    "total_dicts": 3,
    "enabled_dicts": 2,
    "total_users": 5,
    "total_entries": 250000,
    "uptime_seconds": 86400
  }
}
```

---

## 7. 安全设计

### 7.1 认证安全

| 措施 | 说明 |
|------|------|
| 密码哈希 | 使用 bcrypt，cost=12 |
| Token 安全 | JWT 使用 HS256 签名，密钥长度 >= 32 字节 |
| 登录限流 | 同一 IP 每分钟最多 5 次登录尝试 |
| 会话管理 | Refresh Token 支持吊销，登出时清除 |

### 7.2 API 安全

| 措施 | 说明 |
|------|------|
| CORS | 可配置允许的来源，默认仅允许同源 |
| 限流 | 全局每分钟 100 次请求（可配置） |
| 输入校验 | 所有输入参数严格校验 |
| SQL 注入 | 使用参数化查询，禁止字符串拼接 |
| 路径遍历 | 文件路径严格校验，禁止 `..` |

### 7.3 前端安全

| 措施 | 说明 |
|------|------|
| XSS 防护 | 使用 DOMPurify 消毒所有 HTML |
| CSP | 设置 Content-Security-Policy 头 |
| Cookie | HttpOnly、Secure、SameSite 属性 |

### 7.4 文件上传安全

| 措施 | 说明 |
|------|------|
| 文件类型 | 仅允许 .mdx 和 .mdd 扩展名 |
| 文件大小 | 默认限制 500MB（可配置） |
| 文件名校验 | 过滤特殊字符，防止路径遍历 |
| 病毒扫描 | 建议生产环境集成扫描（不在本项目范围） |

---

## 8. 日志与监控

### 8.1 日志格式

```json
{
  "level": "info",
  "time": "2026-06-12T10:00:00Z",
  "caller": "handlers/search.go:42",
  "message": "Search completed",
  "request_id": "req_abc123",
  "user_id": "user_xyz",
  "word": "hello",
  "dict_count": 3,
  "duration_ms": 15
}
```

### 8.2 日志级别

| 级别 | 使用场景 |
|------|----------|
| DEBUG | 详细的调试信息，开发环境使用 |
| INFO | 一般操作日志，如登录、查询 |
| WARN | 警告信息，如配置缺失使用默认值 |
| ERROR | 错误信息，如数据库操作失败 |

### 8.3 审计日志

以下操作记录审计日志：

- 用户登录/登出
- 用户创建/删除/权限修改
- 词典上传/删除/状态切换
- API Token 重置

---

## 9. 部署方案

### 9.1 Docker 部署

**Dockerfile：**

```dockerfile
# 构建阶段
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mdict-server ./cmd/server

# 运行阶段
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/mdict-server .
RUN mkdir -p /dicts /data

EXPOSE 8080
VOLUME ["/dicts", "/data"]

ENTRYPOINT ["./mdict-server"]
```

**Docker Compose：**

```yaml
version: '3.8'

services:
  mdict-server:
    image: ghcr.io/HW618/mdict-server:latest
    container_name: mdict-server
    ports:
      - "8080:8080"
    volumes:
      - ./dicts:/dicts
      - ./data:/data
    environment:
      - DICT_DIR=/dicts
      - DATA_DIR=/data
      - JWT_SECRET=your-secret-key-here
      - ADMIN_USER=admin
      - ADMIN_PASS=your-password
    restart: unless-stopped
```

### 9.2 环境变量配置示例

```bash
# .env.example
SERVER_ADDR=0.0.0.0
SERVER_PORT=8080
DICT_DIR=./dicts
DATA_DIR=./data
ADMIN_USER=admin
ADMIN_PASS=change-me-in-production
JWT_SECRET=generate-a-random-32-byte-string
JWT_ACCESS_TTL=2h
JWT_REFRESH_TTL=168h
SKILL_SERVER_URL=http://localhost:8080
MAX_UPLOAD_SIZE=500MB
RATE_LIMIT=100
LOG_LEVEL=info
LOG_FORMAT=json
CORS_ORIGINS=*
```

### 9.3 CI/CD 配置

**GitHub Actions：**

```yaml
name: Build and Push Docker Image

on:
  push:
    branches: [main]
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Run tests
        run: go test ./...

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ghcr.io/${{ github.repository }}
          tags: |
            type=ref,event=branch
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
```

---

## 10. 性能要求

| 指标 | 要求 |
|------|------|
| 单次查询响应时间 | < 100ms (精确查询) |
| 模糊查询响应时间 | < 500ms (前100条) |
| 并发用户支持 | >= 100 |
| 内存占用 | < 500MB (基础运行) |
| 启动时间 | < 10s (加载3个词典) |

---

## 11. 测试策略

### 11.1 单元测试

- 覆盖率目标：>= 70%
- 重点模块：config、auth、dict、store
- 使用 `testing` 包和 `testify` 断言库

### 11.2 集成测试

- API 端点测试
- 数据库操作测试
- 词典文件解析测试

### 11.3 E2E 测试

- 完整用户流程测试
- 使用 Docker Compose 搭建测试环境

---

## 12. 后续扩展（不在 v1.0 范围）

| 功能 | 优先级 | 说明 |
|------|--------|------|
| 多词典联合查询优化 | P2 | 并行查询多个词典 |
| 查询历史记录 | P3 | 保存用户查询历史 |
| 收藏功能 | P3 | 收藏单词和释义 |
| 词典分组管理 | P3 | 按语言/类型分组 |
| 批量导入导出 | P3 | 批量操作支持 |
| WebSocket 实时通知 | P4 | 词典更新通知 |
| 多语言界面 | P4 | 国际化支持 |

---

## 13. 术语表

| 术语 | 说明 |
|------|------|
| Mdx | Mdict 词典文件格式，包含词条和释义 |
| Mdd | Mdict 媒体资源文件格式，包含图片、音频等 |
| JWT | JSON Web Token，用于 API 认证 |
| RBAC | Role-Based Access Control，基于角色的访问控制 |
| Agent Skill | AI 智能体工具配置文件，描述 API 能力 |

---

## 14. 变更记录

| 日期 | 版本 | 变更内容 |
|------|------|----------|
| 2026-06-12 | 1.0 | 初始版本，完善所有需求细节 |
