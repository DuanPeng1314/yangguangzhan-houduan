# SEO 推送模块开发文档

> 开发时间：2026-03-05
> 开发目的：在不影响原作者代码的情况下，实现搜索引擎自动推送功能

---

## 一、需求分析

### 1.1 功能需求

| 功能 | 说明 |
|------|------|
| 百度推送 | 文章发布/更新时自动推送到百度搜索 |
| Bing 推送 | 文章发布/更新时自动推送到 Bing 搜索 |
| Google 推送 | 文章发布/更新时自动推送到 Google Indexing API |
| 重试机制 | 推送失败时自动重试 |

### 1.2 配置项

| 配置键 | 类型 | 说明 |
|--------|------|------|
| `seo.auto_submit` | boolean | 是否启用自动推送 |
| `seo.retry_times` | number | 重试次数 |
| `seo.retry_interval` | number | 重试间隔（毫秒） |
| `seo.baidu.enable` | boolean | 启用百度推送 |
| `seo.baidu.site` | string | 百度站点地址 |
| `seo.baidu.token` | password | 百度推送 Token |
| `seo.bing.enable` | boolean | 启用 Bing 推送 |
| `seo.bing.api_key` | password | Bing API Key |
| `seo.bing.site_url` | string | Bing 站点 URL |
| `seo.google.enable` | boolean | 启用 Google 推送 |
| `seo.google.credential` | json | Google Service Account 凭证 |

---

## 二、目录结构

```
阳光栈后端/
├── modules/                          # 独立模块目录（新增）
│   ├── module.go                     # 模块接口定义
│   ├── registry.go                   # 模块注册表
│   │
│   └── seo/                          # SEO 推送模块
│       ├── module.go                 # 模块定义
│       ├── service.go                # 推送服务接口
│       ├── baidu.go                  # 百度推送实现
│       ├── bing.go                   # Bing 推送实现
│       ├── google.go                 # Google 推送实现
│       └── listener.go               # 事件监听器
│
├── pkg/                              # 原作者代码（不修改）
├── internal/                         # 原作者代码（最小修改）
│   └── configdef/
│       └── definition.go             # 添加 SEO 配置键定义
│
└── cmd/server/app.go                 # 添加模块注册入口
```

---

## 三、需要修改的原作者代码

### 3.1 pkg/constant/setting.go

添加 SEO 配置键常量（约 15 行）

### 3.2 internal/configdef/definition.go

添加 SEO 配置默认值（约 15 行）

### 3.3 cmd/server/app.go

添加模块注册入口（约 20 行）

---

## 四、开发步骤

| 步骤 | 内容 | 状态 |
|------|------|------|
| 1 | 创建模块目录结构 | ✅ |
| 2 | 实现模块接口和注册表 | ✅ |
| 3 | 实现 SEO 推送服务 | ✅ |
| 4 | 添加配置键定义 | ✅ |
| 5 | 在主应用中注册模块 | ✅ |
| 6 | 测试验证 | ⏳ |

---

## 五、已创建的文件

| 文件路径 | 说明 |
|----------|------|
| `modules/module.go` | 模块接口定义 |
| `modules/registry.go` | 模块注册表 |
| `modules/seo/module.go` | SEO 推送模块实现 |
| `internal/app/listener/seo_module_listener.go` | SEO 模块事件监听器 |
| `docs/开发流程文档/SEO推送模块开发.md` | 开发文档 |

---

## 六、修改的原作者文件

| 文件路径 | 改动内容 | 行数 |
|----------|----------|------|
| `pkg/constant/setting.go` | 添加 SEO 配置键常量 | ~12行 |
| `internal/configdef/definition.go` | 添加 SEO 配置默认值 | ~12行 |
| `cmd/server/app.go` | 添加模块监听器注册 | ~3行 |

---

## 五、API 参考

### 百度链接提交 API

```
POST http://data.zz.baidu.com/urls?site={site}&token={token}
Content-Type: text/plain

{url1}
{url2}
```

### Bing IndexNow API

```
GET https://www.bing.com/indexnow?url={url}&key={api_key}
```

### Google Indexing API

```json
POST https://indexing.googleapis.com/v3/urlNotifications:publish
Authorization: Bearer {access_token}

{
  "url": "{url}",
  "type": "URL_UPDATED"
}
```

---

## 六、更新日志

| 日期 | 内容 |
|------|------|
| 2026-03-05 | 创建开发文档 |
