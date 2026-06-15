# 使用指南

## 概述

本文档为 Mdict Server 的用户提供详细的使用说明，包括 Web 界面操作、API 调用和 Agent 集成。

---

## 快速开始

### 1. 访问服务

启动服务后，在浏览器中访问：

```
http://localhost:8080
```

### 2. 登录管理后台

访问管理后台：

```
http://localhost:8080/admin
```

使用管理员账户登录：
- 用户名：`admin`（或你在环境变量中设置的 `ADMIN_USER`）
- 密码：启动时生成的随机密码（查看日志）或你设置的 `ADMIN_PASS`

### 3. 上传词典

1. 登录管理后台
2. 在"词典管理"区域点击"上传"或拖拽文件
3. 选择 `.mdx` 文件（可选附带同名 `.mdd` 文件）
4. 等待上传完成

### 4. 开始查询

1. 返回首页 http://localhost:8080
2. 在搜索框中输入单词
3. 按回车或点击搜索按钮
4. 查看查询结果

---

## Web 界面使用

### 查询页面 (/)

#### 界面布局

```
┌─────────────────────────────────────────────────────────────┐
│  [Logo]  [词典选择▼]  [搜索框________________]  [搜索]  [登录] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   ┌─────────────────────────────────────────────────────┐   │
│   │                                                     │   │
│   │                 查询结果展示区                       │   │
│   │                                                     │   │
│   │   hello                                             │   │
│   │   /həˈloʊ/                                         │   │
│   │                                                     │   │
│   │   interj. 你好；喂                                  │   │
│   │   n. 问候；招呼                                      │   │
│   │   vi. 打招呼                                        │   │
│   │                                                     │   │
│   │   [🔊 发音]                                         │   │
│   │                                                     │   │
│   └─────────────────────────────────────────────────────┘   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 功能说明

1. **词典选择**
   - 点击下拉菜单选择要查询的词典
   - 默认查询所有已启用的词典
   - 可以选择特定词典进行查询

2. **搜索功能**
   - 在搜索框中输入单词
   - 按回车键或点击搜索按钮进行查询
   - 支持实时输入建议（模糊查询）

3. **查询结果**
   - 显示单词的完整释义
   - 支持图片显示
   - 支持音频播放（点击发音按钮）
   - HTML 内容已安全消毒，防止 XSS 攻击

4. **音频播放**
   - 如果单词有音频，会显示播放按钮
   - 点击按钮播放发音
   - 支持 MP3、WAV、OGG 格式

#### 快捷键

| 快捷键 | 功能 |
|--------|------|
| `Enter` | 执行查询 |
| `Esc` | 清空搜索框 |

---

### 管理后台 (/admin)

#### 界面布局

```
┌─────────────────────────────────────────────────────────────┐
│  [Logo]  [词典管理]  [用户管理]              [admin] [登出]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  词典管理                                            │    │
│  │  ┌───────────────────────────────────────────────┐  │    │
│  │  │  拖拽 .mdx/.mdd 文件到此处或点击上传          │  │    │
│  │  └───────────────────────────────────────────────┘  │    │
│  │                                                     │    │
│  │  ┌──────┬────────┬────────┬───────┬──────┬───────┐  │    │
│  │  │ 状态 │ 文件名 │ 大小   │ 条目数│ 日期 │ 操作  │  │    │
│  │  ├──────┼────────┼────────┼───────┼──────┼───────┤  │    │
│  │  │  ●   │ oxford │ 50MB   │ 85000 │ ...  │ 🗑️   │  │    │
│  │  │  ○   │ collins│ 30MB   │ 62000 │ ...  │ 🗑️   │  │    │
│  │  └──────┴────────┴────────┴───────┴──────┴───────┘  │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  用户管理                                            │    │
│  │  ┌───────────────────────────────────────────────┐  │    │
│  │  │  [+ 新建用户]                                  │  │    │
│  │  └───────────────────────────────────────────────┘  │    │
│  │                                                     │    │
│  │  ┌────────┬────────────┬────────────┬─────────────┐  │    │
│  │  │ 用户名 │ 权限       │ API Token  │ 操作        │  │    │
│  │  ├────────┼────────────┼────────────┼─────────────┤  │    │
│  │  │ admin  │ 全部权限   │ mdtk_xxx.. │ [复制Token] │  │    │
│  │  │ john   │ 查询       │ mdtk_yyy.. │ [复制][🗑️] │  │    │
│  │  └────────┴────────────┴────────────┴─────────────┘  │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

#### 词典管理

**上传词典：**

1. 在上传区域拖拽 `.mdx` 文件
2. 或点击上传区域选择文件
3. 如有配套的 `.mdd` 文件，一起上传
4. 等待上传和解析完成

**启用/禁用词典：**

- 点击词典行的开关按钮切换状态
- 禁用的词典不会出现在查询页面
- 禁用不会删除词典文件

**删除词典：**

1. 点击词典行的删除按钮
2. 在确认对话框中点击"确认删除"
3. 词典文件将从服务器删除

**词典信息：**

| 字段 | 说明 |
|------|------|
| 文件名 | 原始文件名 |
| 大小 | 文件大小（自动格式化） |
| 条目数 | 词典中的词条数量 |
| 日期 | 上传时间 |

#### 用户管理

**新建用户：**

1. 点击"新建用户"按钮
2. 填写用户名（必填）
3. 填写密码（可选，不填则自动生成）
4. 设置权限：
   - `can_use_api` - 是否允许调用 API
   - `is_dict_admin` - 是否为词典管理员
   - `is_user_admin` - 是否为用户管理员
5. 点击"创建"按钮
6. 复制生成的 API Token（只显示一次）

**修改权限：**

1. 在用户列表中勾选/取消勾选权限框
2. 权限立即生效

**复制 API Token：**

- 点击用户行的"复制Token"按钮
- Token 将复制到剪贴板
- Token 格式：`mdtk_xxxxxxxxx...`

**重置 API Token：**

1. 点击用户行的"重置Token"按钮
2. 在确认对话框中点击"确认重置"
3. 复制新的 Token

**删除用户：**

1. 点击用户行的删除按钮
2. 在确认对话框中点击"确认删除"
3. 注意：管理员账户不能删除

---

## API 使用

### 认证

#### 获取 Token

```bash
# 登录获取 Token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-password"
  }'
```

响应：
```json
{
  "code": 0,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 7200
  }
}
```

#### 使用 Token

```bash
# 在请求头中携带 Token
curl http://localhost:8080/api/v1/search?word=hello \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

#### 刷新 Token

```bash
# Access Token 过期后，使用 Refresh Token 刷新
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

### 查询单词

#### 精确查询

```bash
# 查询单个单词
curl "http://localhost:8080/api/v1/search?word=hello" \
  -H "Authorization: Bearer your-token"

# 查询指定词典
curl "http://localhost:8080/api/v1/search?word=hello&dict_id=oxford" \
  -H "Authorization: Bearer your-token"
```

#### 模糊查询

```bash
# 前缀查询
curl "http://localhost:8080/api/v1/search/fuzzy?keyword=hel" \
  -H "Authorization: Bearer your-token"

# 带分页
curl "http://localhost:8080/api/v1/search/fuzzy?keyword=hel&page=1&page_size=10" \
  -H "Authorization: Bearer your-token"
```

### 词典管理

```bash
# 获取词典列表
curl http://localhost:8080/api/v1/dicts \
  -H "Authorization: Bearer your-token"

# 启用/禁用词典
curl -X PATCH http://localhost:8080/api/v1/dicts/oxford/status \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{"is_enabled": false}'

# 上传词典
curl -X POST http://localhost:8080/api/v1/dicts/upload \
  -H "Authorization: Bearer your-token" \
  -F "file=@/path/to/dictionary.mdx"

# 下载词典
curl -O -J http://localhost:8080/api/v1/dicts/oxford/download \
  -H "Authorization: Bearer your-token"

# 删除词典
curl -X DELETE http://localhost:8080/api/v1/dicts/oxford \
  -H "Authorization: Bearer your-token"
```

### 用户管理

```bash
# 获取用户列表
curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer your-token"

# 创建用户
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

# 修改权限
curl -X PUT http://localhost:8080/api/v1/users/user-id/permissions \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "can_use_api": true,
    "is_dict_admin": true,
    "is_user_admin": false
  }'

# 重置 Token
curl -X POST http://localhost:8080/api/v1/users/user-id/reset-token \
  -H "Authorization: Bearer your-token"

# 删除用户
curl -X DELETE http://localhost:8080/api/v1/users/user-id \
  -H "Authorization: Bearer your-token"
```

---

## Agent 集成

### 获取 Skill 配置

访问以下地址获取 Agent Skill 配置文件：

```
http://localhost:8080/api/v1/skill.json
```

该文件遵循 OpenAPI 3.0 规范，可以导入到支持的 AI 智能体平台。

### 在 Dify 中使用

1. 登录 Dify 平台
2. 进入"工具" → "自定义工具"
3. 点击"导入 OpenAPI Schema"
4. 粘贴 `skill.json` 的内容
5. 配置认证信息：
   - 类型：API Key
   - Header：Authorization
   - 值：`Bearer your-api-token`
6. 保存并测试

### 在 Coze 中使用

1. 登录 Coze 平台
2. 进入"插件" → "创建插件"
3. 选择"基于 OpenAPI 创建"
4. 粘贴 `skill.json` 的内容
5. 配置认证方式
6. 发布插件

### 在 GPTs 中使用

1. 登录 ChatGPT
2. 进入 GPTs 编辑界面
3. 在"Actions"部分点击"Create new action"
4. 选择"Import from URL"或手动配置
5. 填入 API 信息：
   - Schema：粘贴 `skill.json`
   - Authentication：API Key
   - API Key：`your-api-token`

### 使用示例

#### Python

```python
import requests

API_URL = "http://localhost:8080/api/v1"
TOKEN = "your-api-token"

headers = {
    "Authorization": f"Bearer {TOKEN}"
}

# 查询单词
response = requests.get(
    f"{API_URL}/search",
    params={"word": "hello"},
    headers=headers
)

result = response.json()
print(result["data"]["results"][0]["html"])
```

#### JavaScript

```javascript
const API_URL = 'http://localhost:8080/api/v1';
const TOKEN = 'your-api-token';

async function searchWord(word) {
    const response = await fetch(
        `${API_URL}/search?word=${encodeURIComponent(word)}`,
        {
            headers: {
                'Authorization': `Bearer ${TOKEN}`
            }
        }
    );
    
    const data = await response.json();
    return data.data.results;
}

// 使用
searchWord('hello').then(results => {
    console.log(results);
});
```

#### Go

```go
package main

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

const (
    APIURL = "http://localhost:8080/api/v1"
    TOKEN  = "your-api-token"
)

type SearchResult struct {
    Code int `json:"code"`
    Data struct {
        Word    string `json:"word"`
        Results []struct {
            DictName string `json:"dict_name"`
            HTML     string `json:"html"`
        } `json:"results"`
    } `json:"data"`
}

func SearchWord(word string) (*SearchResult, error) {
    req, _ := http.NewRequest("GET", APIURL+"/search?word="+word, nil)
    req.Header.Set("Authorization", "Bearer "+TOKEN)
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    body, _ := io.ReadAll(resp.Body)
    var result SearchResult
    json.Unmarshal(body, &result)
    
    return &result, nil
}

func main() {
    result, _ := SearchWord("hello")
    fmt.Println(result.Data.Results[0].HTML)
}
```

---

## 常见问题

### Q: 忘记管理员密码怎么办？

**方案 1：查看启动日志**

如果是首次启动且未设置密码，查看容器日志：

```bash
docker logs mdict-server | grep -i "admin"
```

**方案 2：重置密码**

删除数据库文件，重启服务会自动生成新的管理员账户：

```bash
# 停止服务
docker-compose down

# 删除数据库
rm data/mdict.db

# 重启服务
docker-compose up -d
```

**方案 3：通过环境变量重置**

设置新的环境变量后重启：

```bash
# 修改 .env 文件
ADMIN_PASS=new-password

# 重启服务
docker-compose down
docker-compose up -d
```

### Q: 词典上传失败怎么办？

**检查文件格式：**
- 确保文件是有效的 `.mdx` 格式
- 文件名不能包含特殊字符

**检查文件大小：**
- 默认限制 500MB
- 可通过 `MAX_UPLOAD_SIZE` 环境变量调整

**检查磁盘空间：**
```bash
df -h
```

**检查权限：**
```bash
ls -la dicts/
```

### Q: 查询没有结果？

**检查词典状态：**
1. 登录管理后台
2. 确认词典已启用（状态开关为开）

**检查查询词：**
- 确保查询词存在于词典中
- 尝试模糊查询

**检查日志：**
```bash
docker logs mdict-server | grep -i "search"
```

### Q: API 返回 401 错误？

**检查 Token：**
- Token 是否过期（默认 2 小时）
- 使用 Refresh Token 刷新

**检查权限：**
- 确认用户有 `can_use_api` 权限
- 管理员默认拥有所有权限

**检查请求头：**
```bash
# 正确格式
-H "Authorization: Bearer your-token"

# 错误格式（缺少 Bearer）
-H "Authorization: your-token"
```

### Q: 如何备份数据？

```bash
# 备份数据库
cp data/mdict.db backup/mdict_$(date +%Y%m%d).db

# 备份词典文件（可选）
tar -czf backup/dicts_$(date +%Y%m%d).tar.gz dicts/
```

### Q: 如何升级服务？

```bash
# 拉取最新镜像
docker pull ghcr.io/HW618/mdict-server:latest

# 停止旧服务
docker-compose down

# 启动新服务
docker-compose up -d

# 验证版本
curl http://localhost:8080/api/v1/health | jq '.version'
```

### Q: 如何限制访问？

**方案 1：IP 白名单（Nginx）**

```nginx
location / {
    allow 192.168.1.0/24;
    deny all;
    proxy_pass http://localhost:8080;
}
```

**方案 2：HTTP Basic Auth（Nginx）**

```bash
# 生成密码文件
htpasswd -c /etc/nginx/.htpasswd user1

# Nginx 配置
location / {
    auth_basic "Restricted";
    auth_basic_user_file /etc/nginx/.htpasswd;
    proxy_pass http://localhost:8080;
}
```

**方案 3：禁用公开访问**

不设置 `can_use_api` 权限，用户只能通过 Web 界面访问。

---

## 性能优化

### 1. 启用缓存

在 Nginx 中启用静态文件缓存：

```nginx
location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
    proxy_pass http://localhost:8080;
    expires 1y;
    add_header Cache-Control "public, immutable";
}
```

### 2. 调整词典加载

如果词典文件很大，可以：
- 只加载常用的词典
- 禁用不常用的词典
- 使用 SSD 存储词典文件

### 3. 资源限制

```yaml
# docker-compose.yml
services:
  mdict-server:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 1G
```

---

## 更新日志

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2026-06-12 | 初始版本 |
