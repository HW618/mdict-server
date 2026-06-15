# 部署指南

## 概述

本文档提供 Mdict Server 的多种部署方案，包括 Docker 单机部署、Docker Compose 部署和生产环境配置。

---

## 部署方式

| 方式 | 适用场景 | 复杂度 |
|------|----------|--------|
| Docker 单机 | 个人使用、快速体验 | ⭐ |
| Docker Compose | 小团队、开发环境 | ⭐⭐ |
| Kubernetes | 大规模生产环境 | ⭐⭐⭐ |

---

## Docker 单机部署

### 前提条件

- Docker 20.10+
- 至少 512MB 可用内存
- 至少 1GB 可用磁盘空间

### 快速启动

```bash
# 创建数据目录
mkdir -p dicts data

# 复制词典文件
cp /path/to/your/*.mdx dicts/

# 运行容器
docker run -d \
  --name mdict-server \
  -p 8080:8080 \
  -v $(pwd)/dicts:/dicts \
  -v $(pwd)/data:/data \
  -e ADMIN_USER=admin \
  -e ADMIN_PASS=your-secure-password \
  -e JWT_SECRET=your-secret-key-at-least-32-chars \
  ghcr.io/HW618/mdict-server:latest
```

### 查看日志

```bash
# 实时日志
docker logs -f mdict-server

# 最近 100 行
docker logs --tail 100 mdict-server
```

### 停止/重启

```bash
# 停止
docker stop mdict-server

# 重启
docker restart mdict-server

# 删除
docker rm -f mdict-server
```

---

## Docker Compose 部署

### 前提条件

- Docker 20.10+
- Docker Compose v2.0+

### 部署步骤

#### 1. 克隆项目

```bash
git clone https://github.com/HW618/mdict-server.git
cd mdict-server
```

#### 2. 创建环境配置

```bash
# 创建 .env 文件
cat > .env << 'EOF'
# 管理员配置
ADMIN_USER=admin
ADMIN_PASS=your-secure-password

# JWT 密钥（至少 32 字符）
JWT_SECRET=your-secret-key-at-least-32-chars

# 服务器配置
SERVER_PORT=8080
LOG_LEVEL=info

# 词典目录
DICT_DIR=/dicts
DATA_DIR=/data
EOF
```

#### 3. 准备词典文件

```bash
# 创建词典目录
mkdir -p dicts

# 复制词典文件
cp /path/to/your/*.mdx dicts/
cp /path/to/your/*.mdd dicts/  # 如果有
```

#### 4. 启动服务

```bash
# 前台启动（查看日志）
docker-compose up

# 后台启动
docker-compose up -d

# 查看状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

#### 5. 访问服务

- Web 界面: http://localhost:8080
- 管理后台: http://localhost:8080/admin
- 健康检查: http://localhost:8080/api/v1/health

### Docker Compose 配置文件

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
      - ADMIN_USER=${ADMIN_USER:-admin}
      - ADMIN_PASS=${ADMIN_PASS}
      - JWT_SECRET=${JWT_SECRET}
      - DICT_DIR=/dicts
      - DATA_DIR=/data
      - SERVER_PORT=8080
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - LOG_FORMAT=${LOG_FORMAT:-json}
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

---

## 生产环境配置

### 环境变量详解

#### 基础配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_ADDR` | `0.0.0.0` | 监听地址 |
| `SERVER_PORT` | `8080` | 监听端口 |
| `DICT_DIR` | `./dicts` | 词典文件目录 |
| `DATA_DIR` | `./data` | 数据存储目录 |

#### 安全配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `ADMIN_USER` | *随机* | 管理员用户名 |
| `ADMIN_PASS` | *随机* | 管理员密码 |
| `JWT_SECRET` | *随机* | JWT 签名密钥（至少 32 字符） |
| `JWT_ACCESS_TTL` | `2h` | Access Token 有效期 |
| `JWT_REFRESH_TTL` | `168h` | Refresh Token 有效期 |

#### 限制配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `MAX_UPLOAD_SIZE` | `500MB` | 单文件上传大小限制 |
| `RATE_LIMIT` | `100` | 每分钟请求限制（0=不限制） |

#### 日志配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `LOG_LEVEL` | `info` | 日志级别 (debug/info/warn/error) |
| `LOG_FORMAT` | `json` | 日志格式 (json/text) |

#### 跨域配置

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `CORS_ORIGINS` | `*` | 允许的跨域来源（逗号分隔） |

### 安全最佳实践

#### 1. 密钥管理

```bash
# 生成随机 JWT 密钥
JWT_SECRET=$(openssl rand -base64 32)

# 生成随机管理员密码
ADMIN_PASS=$(openssl rand -base64 16)

# 使用 Docker secrets（Swarm 模式）
echo "$JWT_SECRET" | docker secret create jwt_secret -
```

#### 2. 网络隔离

```yaml
# docker-compose.yml
services:
  mdict-server:
    networks:
      - internal
    # 不暴露端口到外网，通过反向代理访问

networks:
  internal:
    driver: bridge
```

#### 3. 资源限制

```yaml
services:
  mdict-server:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

#### 4. 只读文件系统

```yaml
services:
  mdict-server:
    read_only: true
    tmpfs:
      - /tmp
    volumes:
      - ./dicts:/dicts:ro
      - ./data:/data
```

---

## 反向代理配置

### Nginx 配置

```nginx
server {
    listen 80;
    server_name mdict.example.com;
    
    # 重定向到 HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name mdict.example.com;
    
    # SSL 证书
    ssl_certificate /etc/letsencrypt/live/mdict.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mdict.example.com/privkey.pem;
    
    # SSL 配置
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    
    # 安全头
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # 代理配置
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket 支持（如果需要）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # 超时配置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # 缓冲配置
        proxy_buffering on;
        proxy_buffer_size 4k;
        proxy_buffers 8 4k;
    }
    
    # 上传大小限制
    client_max_body_size 500M;
    
    # 静态文件缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        proxy_pass http://localhost:8080;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
```

### Caddy 配置

```
mdict.example.com {
    reverse_proxy localhost:8080
    
    header {
        X-Frame-Options "SAMEORIGIN"
        X-Content-Type-Options "nosniff"
        X-XSS-Protection "1; mode=block"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
    
    encode gzip
    
    @static {
        path *.js *.css *.png *.jpg *.jpeg *.gif *.ico *.svg
    }
    header @static Cache-Control "public, max-age=31536000, immutable"
}
```

---

## 数据备份

### 备份策略

```bash
#!/bin/bash
# backup.sh - 每日备份脚本

BACKUP_DIR="/backup/mdict"
DATE=$(date +%Y%m%d_%H%M%S)

# 创建备份目录
mkdir -p $BACKUP_DIR

# 备份数据库
cp data/mdict.db "$BACKUP_DIR/mdict_$DATE.db"

# 备份词典文件（可选，因为词典文件通常较大）
# tar -czf "$BACKUP_DIR/dicts_$DATE.tar.gz" dicts/

# 清理 30 天前的备份
find $BACKUP_DIR -name "*.db" -mtime +30 -delete

echo "Backup completed: $BACKUP_DIR/mdict_$DATE.db"
```

### 定时备份

```bash
# 添加到 crontab
crontab -e

# 每天凌晨 2 点备份
0 2 * * * /path/to/backup.sh >> /var/log/mdict-backup.log 2>&1
```

### 恢复备份

```bash
# 停止服务
docker-compose down

# 恢复数据库
cp /backup/mdict/mdict_20260612_020000.db data/mdict.db

# 启动服务
docker-compose up -d
```

---

## 监控配置

### 健康检查

```bash
# 检查服务状态
curl -f http://localhost:8080/api/v1/health || echo "Service unhealthy"

# 检查返回值
STATUS=$(curl -s http://localhost:8080/api/v1/health | jq -r '.status')
if [ "$STATUS" != "healthy" ]; then
    echo "Service is $STATUS"
    # 发送告警
fi
```

### 日志监控

```bash
# 实时监控错误日志
docker logs -f mdict-server 2>&1 | grep -i error

# 统计每小时请求数
docker logs mdict-server 2>&1 | \
  grep "request" | \
  awk '{print $1}' | \
  cut -d: -f1,2 | \
  sort | uniq -c
```

### Prometheus 指标（可选扩展）

如果需要集成 Prometheus，可以在应用中添加指标端点：

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'mdict-server'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

---

## 故障排查

### 常见问题

#### 1. 服务无法启动

```bash
# 检查日志
docker logs mdict-server

# 常见原因：
# - 端口被占用
# - 环境变量未设置
# - 数据目录权限问题
```

#### 2. 词典加载失败

```bash
# 检查词典文件权限
ls -la dicts/

# 检查文件格式
file dicts/*.mdx

# 检查日志中的错误信息
docker logs mdict-server | grep -i "dict"
```

#### 3. 数据库错误

```bash
# 检查数据库文件
ls -la data/

# 检查数据库完整性
sqlite3 data/mdict.db "PRAGMA integrity_check;"
```

#### 4. 性能问题

```bash
# 检查资源使用
docker stats mdict-server

# 检查内存限制
docker inspect mdict-server | grep -i memory
```

### 日志级别

| 级别 | 使用场景 |
|------|----------|
| `debug` | 开发调试，显示详细信息 |
| `info` | 生产环境，显示正常操作 |
| `warn` | 警告信息，不影响运行 |
| `error` | 错误信息，需要关注 |

---

## 升级指南

### Docker 升级

```bash
# 拉取最新镜像
docker pull ghcr.io/HW618/mdict-server:latest

# 停止旧容器
docker-compose down

# 启动新容器
docker-compose up -d

# 验证版本
curl http://localhost:8080/api/v1/health | jq '.version'
```

### 数据迁移

```bash
# 备份旧数据
cp data/mdict.db data/mdict.db.backup

# 启动新版本（自动迁移）
docker-compose up -d

# 检查迁移日志
docker logs mdict-server | grep -i "migration"
```

---

## 环境变量速查表

```bash
# 最小配置
ADMIN_USER=admin
ADMIN_PASS=your-password
JWT_SECRET=your-secret-key-at-least-32-chars

# 完整配置
SERVER_ADDR=0.0.0.0
SERVER_PORT=8080
DICT_DIR=/dicts
DATA_DIR=/data
ADMIN_USER=admin
ADMIN_PASS=your-secure-password
JWT_SECRET=your-secret-key-at-least-32-chars
JWT_ACCESS_TTL=2h
JWT_REFRESH_TTL=168h
SKILL_SERVER_URL=https://mdict.example.com
MAX_UPLOAD_SIZE=500MB
RATE_LIMIT=100
LOG_LEVEL=info
LOG_FORMAT=json
CORS_ORIGINS=https://mdict.example.com
```

---

## 更新日志

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2026-06-12 | 初始版本 |
