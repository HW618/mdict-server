# 开发指南

## 概述

本文档为 Mdict Server 项目的开发者提供详细的开发环境搭建、代码规范和贡献指南。

---

## 开发环境要求

### 必需工具

| 工具 | 版本要求 | 说明 |
|------|----------|------|
| Go | 1.22+ | 编程语言 |
| Git | 2.30+ | 版本控制 |
| SQLite | 3.40+ | 数据库（开发时使用文件） |
| Make | 3.81+ | 构建工具（可选） |

### 推荐工具

| 工具 | 用途 |
|------|------|
| VS Code / GoLand | IDE |
| Postman / Insomnia | API 测试 |
| Docker Desktop | 容器开发 |
| Air | 热重载工具 |

### Go 环境配置

```bash
# 检查 Go 版本
go version

# 设置 Go 代理（国内用户）
go env -w GOPROXY=https://goproxy.cn,direct

# 启用 Go Modules
go env -w GO111MODULE=on
```

---

## 项目结构

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
├── .env.example
├── PRD.md
└── README.md
```

### 目录职责

| 目录 | 职责 |
|------|------|
| `cmd/server` | 程序入口，负责初始化配置、启动服务 |
| `internal/config` | 环境变量读取、配置验证、默认值设置 |
| `internal/auth` | JWT 生成/验证、认证/授权中间件 |
| `internal/dict` | 词典文件解析、查询算法、缓存管理 |
| `internal/handlers` | HTTP 请求处理、业务逻辑 |
| `internal/models` | 数据模型定义、验证规则 |
| `internal/store` | 数据库操作、数据持久化 |
| `internal/middleware` | HTTP 中间件（CORS、限流、日志） |
| `templates` | HTML 模板文件 |
| `migrations` | 数据库迁移脚本 |

---

## 快速开始

### 1. 克隆项目

```bash
git clone https://github.com/HW618/mdict-server.git
cd mdict-server
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置环境变量

```bash
# 复制示例配置
cp .env.example .env

# 编辑配置文件
vim .env
```

**最小配置：**

```bash
ADMIN_USER=admin
ADMIN_PASS=your-password
JWT_SECRET=your-secret-key
```

### 4. 准备词典文件

```bash
# 创建词典目录
mkdir -p dicts

# 复制词典文件
cp /path/to/your/dictionary.mdx dicts/
```

### 5. 启动开发服务器

```bash
# 直接运行
go run cmd/server/main.go

# 或使用 Air 热重载
air
```

### 6. 访问服务

- Web 界面: http://localhost:8080
- 管理后台: http://localhost:8080/admin
- API 文档: http://localhost:8080/api/v1/health

---

## 开发规范

### Go 代码规范

#### 1. 命名规范

```go
// 包名：小写单词，无下划线
package config

// 常量：驼峰或全大写下划线
const MaxRetries = 3
const MAX_BUFFER_SIZE = 1024

// 变量：驼峰命名
var userID string
var userName string

// 函数/方法：驼峰命名，首字母大小写决定可见性
func GetUserByID(id string) (*User, error) {}
func (s *UserService) validate() error {}

// 接口：通常以 -er 结尾
type Reader interface {}
type Writer interface {}
```

#### 2. 注释规范

```go
// Package config handles application configuration loading and validation.
package config

// Config represents the application configuration loaded from environment variables.
type Config struct {
    // ServerAddr is the address the server listens on.
    ServerAddr string `env:"SERVER_ADDR" default:"0.0.0.0"`
    
    // ServerPort is the port the server listens on.
    ServerPort int `env:"SERVER_PORT" default:"8080"`
}

// Load reads configuration from environment variables and validates them.
// It returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
    // ...
}
```

#### 3. 错误处理

```go
// 使用自定义错误类型
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed: %s - %s", e.Field, e.Message)
}

// 错误包装
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}

// 错误检查
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

#### 4. 并发安全

```go
type DictEngine struct {
    mu      sync.RWMutex
    dicts   map[string]*Dict
}

func (e *DictEngine) GetDict(id string) (*Dict, bool) {
    e.mu.RLock()
    defer e.mu.RUnlock()
    dict, ok := e.dicts[id]
    return dict, ok
}

func (e *DictEngine) LoadDict(id string, dict *Dict) {
    e.mu.Lock()
    defer e.mu.Unlock()
    e.dicts[id] = dict
}
```

### Git 提交规范

#### Commit Message 格式

```
<type>(<scope>): <subject>

[optional body]

[optional footer]
```

#### Type 类型

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | 修复 bug |
| `docs` | 文档更新 |
| `style` | 代码格式调整（不影响逻辑） |
| `refactor` | 代码重构 |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `chore` | 构建/工具链更新 |

#### 示例

```bash
# 新功能
git commit -m "feat(search): add fuzzy search support"

# 修复 bug
git commit -m "fix(auth): fix token refresh race condition"

# 文档更新
git commit -m "docs(api): update search endpoint documentation"

# 重构
git commit -m "refactor(dict): extract search logic to separate package"
```

---

## 测试

### 测试类型

| 类型 | 目录 | 说明 |
|------|------|------|
| 单元测试 | `*_test.go` | 测试单个函数/方法 |
| 集成测试 | `tests/` | 测试模块间交互 |
| E2E 测试 | `tests/e2e/` | 测试完整流程 |

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/config/...

# 运行带覆盖率的测试
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 运行特定测试
go test -run TestLoadConfig ./internal/config/

# 运行基准测试
go test -bench=. ./internal/dict/
```

### 测试规范

```go
package config

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestLoadConfig_Success(t *testing.T) {
    // Arrange
    t.Setenv("SERVER_PORT", "9090")
    t.Setenv("ADMIN_USER", "testuser")
    
    // Act
    cfg, err := Load()
    
    // Assert
    require.NoError(t, err)
    assert.Equal(t, 9090, cfg.ServerPort)
    assert.Equal(t, "testuser", cfg.AdminUser)
}

func TestLoadConfig_MissingRequired(t *testing.T) {
    // Arrange
    t.Setenv("JWT_SECRET", "")
    
    // Act
    _, err := Load()
    
    // Assert
    require.Error(t, err)
    assert.Contains(t, err.Error(), "JWT_SECRET")
}
```

### Mock 示例

```go
//go:generate mockgen -source=store.go -destination=mock_store.go -package=store

type UserStore interface {
    GetUser(id string) (*User, error)
    CreateUser(user *User) error
}

// Mock 实现
type MockUserStore struct {
    mock.Mock
}

func (m *MockUserStore) GetUser(id string) (*User, error) {
    args := m.Called(id)
    return args.Get(0).(*User), args.Error(1)
}
```

---

## 依赖管理

### 添加依赖

```bash
# 添加依赖
go get github.com/new/package@latest

# 更新依赖
go get -u github.com/new/package

# 整理依赖
go mod tidy
```

### 主要依赖

| 依赖 | 用途 |
|------|------|
| `github.com/gin-gonic/gin` | Web 框架 |
| `github.com/lib-x/mdx` | Mdx/Mdd 解析 |
| `modernc.org/sqlite` | SQLite 驱动 |
| `github.com/golang-jwt/jwt/v5` | JWT 处理 |
| `github.com/rs/zerolog` | 结构化日志 |
| `github.com/microcosm-cc/bluemonday` | HTML 消毒 |
| `golang.org/x/crypto` | bcrypt 加密 |

---

## 调试技巧

### 1. 日志级别

```bash
# 开发环境使用 debug 级别
LOG_LEVEL=debug go run cmd/server/main.go
```

### 2. 环境变量调试

```bash
# 打印所有环境变量
go run cmd/server/main.go -debug-env
```

### 3. 数据库调试

```bash
# 查看 SQLite 数据库
sqlite3 data/mdict.db

# 查看表结构
.tables
.schema users
```

### 4. API 调试

```bash
# 使用 curl 调试
curl -v http://localhost:8080/api/v1/health

# 使用 jq 格式化 JSON
curl -s http://localhost:8080/api/v1/dicts | jq .
```

### 5. 性能分析

```go
import _ "net/http/pprof"

// 在 main.go 中添加
go func() {
    http.ListenAndServe("localhost:6060", nil)
}()
```

```bash
# 访问 pprof
go tool pprof http://localhost:6060/debug/pprof/profile
```

---

## 常见问题

### Q: 编译错误 "undefined: sqlite"

A: 需要安装 CGO 依赖或使用纯 Go 实现：

```bash
# 方案1：安装 CGO 依赖
sudo apt-get install gcc

# 方案2：使用纯 Go 实现
# go.mod 中已使用 modernc.org/sqlite，无需 CGO
```

### Q: 如何重置管理员密码？

A: 删除数据库文件或直接修改：

```bash
# 方案1：删除数据库（会丢失所有数据）
rm data/mdict.db

# 方案2：使用 SQL 修改
sqlite3 data/mdict.db
UPDATE users SET password='$2a$12$...' WHERE username='admin';
```

### Q: 如何添加新的 API 端点？

A: 步骤如下：

1. 在 `internal/handlers/` 添加处理器
2. 在 `cmd/server/main.go` 注册路由
3. 在 `internal/auth/middleware.go` 配置权限
4. 编写测试
5. 更新 API 文档

### Q: 如何调试 JWT Token？

A: 使用 jwt.io 或命令行工具：

```bash
# 解码 Token
echo "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." | cut -d. -f2 | base64 -d | jq .
```

---

## 贡献流程

### 1. Fork 项目

```bash
# 在 GitHub 上 Fork 项目
# 克隆你的 Fork
git clone https://github.com/your-username/mdict-server.git
cd mdict-server

# 添加上游仓库
git remote add upstream https://github.com/HW618/mdict-server.git
```

### 2. 创建功能分支

```bash
# 同步上游代码
git fetch upstream
git checkout -b feature/your-feature upstream/main
```

### 3. 开发和测试

```bash
# 编写代码
# 运行测试
go test ./...

# 检查代码风格
golangci-lint run
```

### 4. 提交代码

```bash
git add .
git commit -m "feat(scope): your feature description"
```

### 5. 创建 Pull Request

```bash
git push origin feature/your-feature
# 在 GitHub 上创建 PR
```

### PR 检查清单

- [ ] 代码通过所有测试
- [ ] 代码符合项目规范
- [ ] 已添加必要的测试
- [ ] 已更新相关文档
- [ ] Commit message 符合规范
- [ ] 没有引入新的 lint 警告

---

## IDE 配置

### VS Code

安装扩展：
- Go (官方)
- Go Test Explorer
- Error Lens
- GitLens

配置 `.vscode/settings.json`：

```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.lintOnSave": "package",
    "go.testOnSave": true,
    "editor.formatOnSave": true
}
```

### GoLand

1. 打开项目
2. 配置 Go SDK：File → Project Structure → SDKs
3. 启用 Go Modules：Preferences → Go → Go Modules
4. 配置运行配置：Run → Edit Configurations

---

## 更新日志

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2026-06-12 | 初始版本 |
