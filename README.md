# Mdict Server

[English](#english) | [中文](#中文)

---

## 中文

一个基于 Go 语言的在线 Mdx/Mdd 词典查询与管理服务，支持 Docker 容器化部署。

### ✨ 功能特性

- 📚 **多词典支持** - 同时加载和查询多个 .mdx/.mdd 词典文件
- 🔍 **智能查询** - 支持精确查询和模糊查询（前缀匹配）
- 🎵 **多媒体支持** - 完美渲染图片和音频资源
- 👥 **用户管理** - 基于角色的访问控制（RBAC）
- 🔐 **安全认证** - JWT Token 认证机制
- 🤖 **AI 集成** - 自动生成 Agent Skill 配置文件
- 🎨 **美观界面** - 现代化 Web 管理面板
- 🐳 **容器部署** - 支持 Docker 一键部署

### 🚀 快速开始

#### 使用 Docker（推荐）

```bash
# 创建数据目录
mkdir -p dicts data

# 复制词典文件到 dicts 目录
cp your-dictionary.mdx dicts/

# 运行容器
docker run -d \
  --name mdict-server \
  -p 8080:8080 \
  -v $(pwd)/dicts:/dicts \
  -v $(pwd)/data:/data \
  -e ADMIN_USER=admin \
  -e ADMIN_PASS=your-password \
  -e JWT_SECRET=your-secret-key \
  ghcr.io/HW618/mdict-server:latest
```

#### 使用 Docker Compose

```bash
# 克隆项目
git clone https://github.com/HW618/mdict-server.git
cd mdict-server

# 创建 .env 文件
cat > .env << EOF
ADMIN_USER=admin
ADMIN_PASS=your-password
JWT_SECRET=your-secret-key
EOF

# 启动服务
docker-compose up -d
```

访问 http://localhost:8080 开始使用。

### 📖 文档

- [API 接口文档](docs/api.md) - 完整的 REST API 参考
- [开发指南](docs/development.md) - 开发环境搭建和代码规范
- [部署指南](docs/deployment.md) - Docker 部署和生产环境配置
- [使用指南](docs/usage.md) - 用户手册和管理员手册
- [产品需求文档](PRD.md) - 详细的功能需求和技术规格

### ⚙️ 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SERVER_ADDR` | `0.0.0.0` | 监听地址 |
| `SERVER_PORT` | `8080` | 监听端口 |
| `DICT_DIR` | `./dicts` | 词典文件目录 |
| `DATA_DIR` | `./data` | 数据存储目录 |
| `ADMIN_USER` | *随机生成* | 管理员用户名 |
| `ADMIN_PASS` | *随机生成* | 管理员密码 |
| `JWT_SECRET` | *随机生成* | JWT 签名密钥 |
| `LOG_LEVEL` | `info` | 日志级别 |

更多配置项请参考 [部署指南](docs/deployment.md)。

### 🛠️ 开发

```bash
# 克隆项目
git clone https://github.com/HW618/mdict-server.git
cd mdict-server

# 安装依赖
go mod download

# 运行测试
go test ./...

# 启动开发服务器
go run cmd/server/main.go
```

详细开发指南请参考 [开发指南](docs/development.md)。

### 📦 技术栈

- **后端**: Go 1.22+ / Gin / SQLite
- **前端**: Alpine.js / Tailwind CSS
- **词典引擎**: lib-x/mdx
- **部署**: Docker / GitHub Actions

### 🤝 贡献

欢迎贡献！请阅读 [贡献指南](CONTRIBUTING.md) 了解详情。

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

### 🙏 致谢

- [lib-x/mdx](https://github.com/lib-x/mdx) - Mdx/Mdd 文件解析库
- [Gin](https://github.com/gin-gonic/gin) - Web 框架
- [Alpine.js](https://alpinejs.dev/) - 前端框架
- [Tailwind CSS](https://tailwindcss.com/) - CSS 框架

---

## English

An online Mdx/Mdd dictionary query and management service built with Go, supporting Docker containerized deployment.

### ✨ Features

- 📚 **Multi-dictionary Support** - Load and query multiple .mdx/.mdd dictionary files simultaneously
- 🔍 **Smart Search** - Exact match and fuzzy search (prefix matching)
- 🎵 **Multimedia Support** - Perfect rendering of images and audio resources
- 👥 **User Management** - Role-Based Access Control (RBAC)
- 🔐 **Secure Authentication** - JWT Token authentication
- 🤖 **AI Integration** - Auto-generate Agent Skill configuration files
- 🎨 **Beautiful UI** - Modern web admin panel
- 🐳 **Container Deployment** - Docker one-click deployment

### 🚀 Quick Start

#### Using Docker (Recommended)

```bash
# Create data directories
mkdir -p dicts data

# Copy dictionary files to dicts directory
cp your-dictionary.mdx dicts/

# Run container
docker run -d \
  --name mdict-server \
  -p 8080:8080 \
  -v $(pwd)/dicts:/dicts \
  -v $(pwd)/data:/data \
  -e ADMIN_USER=admin \
  -e ADMIN_PASS=your-password \
  -e JWT_SECRET=your-secret-key \
  ghcr.io/HW618/mdict-server:latest
```

#### Using Docker Compose

```bash
# Clone project
git clone https://github.com/HW618/mdict-server.git
cd mdict-server

# Create .env file
cat > .env << EOF
ADMIN_USER=admin
ADMIN_PASS=your-password
JWT_SECRET=your-secret-key
EOF

# Start service
docker-compose up -d
```

Visit http://localhost:8080 to get started.

### 📖 Documentation

- [API Documentation](docs/api.md) - Complete REST API reference
- [Development Guide](docs/development.md) - Development environment setup and coding standards
- [Deployment Guide](docs/deployment.md) - Docker deployment and production configuration
- [Usage Guide](docs/usage.md) - User and administrator manual
- [Product Requirements](PRD.md) - Detailed functional requirements and technical specifications

### 🛠️ Development

```bash
# Clone project
git clone https://github.com/HW618/mdict-server.git
cd mdict-server

# Install dependencies
go mod download

# Run tests
go test ./...

# Start development server
go run cmd/server/main.go
```

For detailed development guide, see [Development Guide](docs/development.md).

### 📦 Tech Stack

- **Backend**: Go 1.22+ / Gin / SQLite
- **Frontend**: Alpine.js / Tailwind CSS
- **Dictionary Engine**: lib-x/mdx
- **Deployment**: Docker / GitHub Actions

### 🤝 Contributing

Contributions are welcome! Please read [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the project
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

### 📄 License

This project is licensed under the MIT License - see [LICENSE](LICENSE) file for details

### 🙏 Acknowledgments

- [lib-x/mdx](https://github.com/lib-x/mdx) - Mdx/Mdd file parsing library
- [Gin](https://github.com/gin-gonic/gin) - Web framework
- [Alpine.js](https://alpinejs.dev/) - Frontend framework
- [Tailwind CSS](https://tailwindcss.com/) - CSS framework
