<p align="center">
  <h1 align="center">yamlit</h1>
  <p align="center"><strong>YAML Integration Testing Toolkit</strong></p>
  <p align="center">轻量级 YAML 驱动的 HTTP API 测试工具</p>
  <p align="center">
    <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go" alt="Go version">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <img src="https://img.shields.io/badge/stability-beta-yellow" alt="Stability">
  </p>
</p>

---

**yamlit** 是一个用 Go 实现的单二进制 HTTP API 测试工具。你只需写一个 YAML 文件定义请求步骤、断言和变量，执行器就会自动运行并输出清晰的测试报告。

适合：**个人开发者**、**本地调试**、**CI 集成**。

---

## 📦 目录

- [功能亮点](#-功能亮点)
- [快速开始](#-快速开始)
- [教程：5 分钟上手](#-教程5-分钟上手)
- [CLI 参考](#-cli-参考)
- [YAML 参考](#-yaml-参考)
- [断言系统](#-断言系统)
- [变量系统](#-变量系统)
- [执行策略](#-执行策略)
- [脚本钩子](#-脚本钩子)
- [输出格式](#-输出格式)
- [项目结构](#-项目结构)
- [开发指南](#-开发指南)
- [相关文档](#-相关文档)

---

## ✨ 功能亮点

- **YAML 驱动** — 零代码，写 YAML 即完成 API 测试
- **变量传递** — 前一步的响应值自动注入后续请求
- **5 种断言** — 状态码、JSONPath、文本匹配、精确比对、无断言
- **重试机制** — 网络错误或断言失败自动重试
- **Shell 钩子** — 请求前后执行任意脚本
- **彩色输出** — 终端实时显示测试结果
- **JSON 报告** — `-o` 输出结构化报告，适合 CI 解析
- **批量执行** — 支持目录递归和通配符模式
- **3 种请求体** — JSON / Form / Text，自动设置 Content-Type
- **单二进制** — 无运行时依赖，下载即用

---

## 🚀 快速开始

```bash
# 1. 安装
git clone git@github.com:zend/yamlit.git && cd yamlit
go build -o yamlit ./cmd/yamlit/

# 2. 写一个测试文件
cat > test.yaml << 'EOF'
- name: 检查服务健康
  method: GET
  url: https://httpbin.org/get
  asserts:
    - type: status_code
      expect: 200
EOF

# 3. 运行
./yamlit test.yaml
```

输出示例：

```
▶ [1/1] 检查服务健康 ................................. GET https://httpbin.org/get
  ✓ 200 OK (1.2s)

══════════════════════════════════════════════════
  总计: 1  |  ✓ 通过: 1  |  ✗ 失败: 0  |  耗时: 1.2s
══════════════════════════════════════════════════
```

---

## 📖 教程：5 分钟上手

用一个真实的 API 流程演示：登录 → 获取用户信息 → 验证结果。

### 第 1 步：创建测试文件

```yaml
# tutorial.yaml
- name: 登录获取 Token
  method: POST
  url: https://api.example.com/auth/login
  body:
    type: json
    content: '{"username":"admin","password":"pass123"}'
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.code
      expect: "0"
  extract:
    - source: body
      path: $.data.token
      var_name: auth_token

- name: 获取用户信息
  method: GET
  url: https://api.example.com/user/profile
  headers:
    Authorization: "Bearer ${auth_token}"
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.role
      expect: "admin"

- name: 获取订单列表
  method: GET
  url: https://api.example.com/orders
  params:
    page: "1"
    limit: "20"
  headers:
    Authorization: "Bearer ${auth_token}"
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.items.#
      expect: "20"
```

### 第 2 步：运行

```bash
./yamlit -v tutorial.yaml
```

### 第 3 步：查看结果

```
▶ [1/3] 登录获取 Token ................................. POST https://api.example.com/auth/login
  ✓ 200 OK (412ms)

▶ [2/3] 获取用户信息 ................................... GET https://api.example.com/user/profile
  ✓ 200 OK (45ms)

▶ [3/3] 获取订单列表 ................................... GET https://api.example.com/orders?page=1&limit=20
  ✓ 200 OK (89ms)

══════════════════════════════════════════════════
  总计: 3  |  ✓ 通过: 3  |  ✗ 失败: 0  |  耗时: 546ms
══════════════════════════════════════════════════
```

---

## 🛠 CLI 参考

### 用法

```bash
./yamlit [flags] <input>
```

### 参数

| 参数 | 说明 |
|---|---|
| `<file.yaml>` | 执行单个 YAML 文件 |
| `<directory/>` | 批量执行目录下所有 `.yaml` / `.yml` 文件 |
| `"<pattern>"` | 通配符模式，如 `"tests/*.yaml"` |

### Flags

| Flag | 默认值 | 说明 |
|---|---|---|
| `-v` | `false` | verbose 模式：输出每个步骤的详细结果（含请求/响应体） |
| `-o <file>` | `""` | 输出 JSON 格式测试报告到指定文件 |

### 示例

```bash
# 基础用法
./yamlit test.yaml

# 详细输出
./yamlit -v test.yaml

# 输出 JSON 报告
./yamlit -o report.json test.yaml

# 详细输出 + JSON 报告
./yamlit -v -o report.json test.yaml

# 批量运行目录
./yamlit ./tests/

# 通配符
./yamlit "tests/*.yaml"
```

### 返回码

| 返回码 | 含义 |
|---|---|
| `0` | 全部测试通过 |
| `1` | 存在失败的测试 |

> **注意：** flags 必须放在 `<input>` **前面**，这是 Go flag 的标准行为。

---

## 📝 YAML 参考

### 完整格式

```yaml
- name: <步骤名>                      # [必填] 步骤标识
  method: <HTTP 方法>                 # [必填] GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS
  url: <请求 URL>                     # [必填] 支持 ${var}

  # --- 请求配置（可选）---
  params:                            # URL 查询参数
    key: value
  headers:                           # 请求头
    Key: Value
  body:                              # 请求体
    type: json                       # json | form | text
    content: '{"key":"value"}'       # 支持 ${var}
  timeout: 30s                       # 超时（默认 30s）

  # --- 重试策略（可选）---
  retry_count: 2                     # 失败重试次数（默认 0）
  retry_interval: 1s                 # 重试间隔（默认 1s）

  # --- 失败行为（可选）---
  on_failure: stop                   # stop | continue（默认 stop）

  # --- 断言（可选）---
  asserts:
    - type: status_code              # 断言类型
      expect: 200                    # 期望值
      path: $.data.name              # JSONPath（仅 jsonpath 类型需要）

  # --- 变量提取（可选）---
  extract:
    - source: body                   # body | header
      path: $.data.token
      var_name: auth_token

  # --- Shell 钩子（可选）---
  pre_script: "echo before"          # 前置脚本
  post_script: "echo after"          # 后置脚本
```

### 字段速查表

| 字段 | 必填 | 类型 | 变量替换 | 说明 |
|---|---|---|---|---|
| `name` | ✅ | string | ✅ | 步骤名称 |
| `method` | ✅ | string | ❌ | HTTP 方法 |
| `url` | ✅ | string | ✅ | 请求 URL |
| `params` | ❌ | map | ✅ (value) | URL 查询参数 |
| `headers` | ❌ | map | ✅ (key+value) | 请求头 |
| `body.type` | ❌ | string | ❌ | 请求体类型 |
| `body.content` | ❌ | string | ✅ | 请求体内容 |
| `timeout` | ❌ | duration | ❌ | 超时时间 |
| `retry_count` | ❌ | int | ❌ | 重试次数 |
| `retry_interval` | ❌ | duration | ❌ | 重试间隔 |
| `on_failure` | ❌ | string | ❌ | 失败行为 |
| `asserts` | ❌ | array | ✅ (expect) | 断言列表 |
| `extract` | ❌ | array | ❌ | 变量提取 |
| `pre_script` | ❌ | string | ✅ | 前置 Shell 脚本 |
| `post_script` | ❌ | string | ✅ | 后置 Shell 脚本 |

### 方法

| 方法 | 用途 |
|---|---|
| `GET` | 查询资源 |
| `POST` | 创建资源 |
| `PUT` | 全量更新资源 |
| `PATCH` | 部分更新资源 |
| `DELETE` | 删除资源 |
| `HEAD` | 仅获取响应头 |
| `OPTIONS` | 获取支持的请求方法 |

方法名不区分大小写（`post` / `POST` / `Post` 都合法），执行器自动转大写。

### 请求体类型

| `body.type` | 自动设置的 `Content-Type` | 示例 content |
|---|---|---|
| `json` | `application/json` | `'{"name":"Alice"}'` |
| `form` | `application/x-www-form-urlencoded` | `"name=Alice&age=30"` |
| `text` | `text/plain` | `"Hello, World"` |

如果手动设置了 `Content-Type` 请求头，则不会自动覆盖。

### 超时与重试

`timeout` 和 `retry_interval` 使用 Go duration 格式：

| 写法 | 含义 |
|---|---|
| `500ms` | 500 毫秒 |
| `1s` | 1 秒 |
| `30s` | 30 秒 |
| `5m` | 5 分钟 |

### YAML 书写注意事项

- **JSON 内容必须用引号包裹：** `content: '{"key":"value"}'` — 否则 YAML 解析器会把 `{}` 当作内联映射
- **不要用 tab 缩进：** YAML 只允许空格缩进
- **布尔值/数字要写为字符串：** `expect: "0"` 而不是 `expect: 0`

---

## ✅ 断言系统

断言验证 HTTP 响应的结果。多条断言是 **AND** 关系——全部通过才算通过。

### 断言类型

#### `status_code` — 状态码检查

```yaml
asserts:
  - type: status_code
    expect: 200
```

最常见的基础断言。`expect` 写数字的字符串形式。

常用值：`200`（成功）、`201`（创建成功）、`204`（无内容）、`400`（参数错误）、`401`（未认证）、`403`（无权限）、`404`（不存在）、`500`（服务器错误）。

#### `jsonpath` — JSONPath 键值比对

```yaml
asserts:
  - type: jsonpath
    path: $.data.user.name
    expect: "Alice"
  - type: jsonpath
    path: $.data.items.#
    expect: "10"
```

使用 [gjson](https://github.com/tidwall/gjson) 从响应体中提取值。支持两种写法：

| 写法 | 示例 | 说明 |
|---|---|---|
| 标准 JSONPath | `$.data.id` | 带 `$.` 前缀 |
| 简写 | `data.id` | 不带前缀，效果相同 |

常用路径模式：

| 路径 | 含义 | 响应 `{"data":{"items":[{"id":1},{"id":2}]}}` |
|---|---|---|
| `$.data.items.#` | 数组长度 | `2` |
| `$.data.items.0.id` | 第一个元素的 id | `1` |
| `$.data.items.#(id==1)` | 过滤查询 | `{"id":1}` |

> **注意：** `expect` 的值必须是字符串。数字 `0` 写成 `"0"`，布尔 `true` 写成 `"true"`。

#### `body_match` — 子串匹配

```yaml
asserts:
  - type: body_match
    expect: "操作成功"
```

检查响应体是否包含指定字符串。适合检查错误消息、关键词、状态标记。

#### `body_equals` — 精确比对

```yaml
asserts:
  - type: body_equals
    expect: '{"code":0,"msg":"ok"}'
```

响应体与期望值完全一致（去除两端空白后）。适合固定不变的小响应。

> ⚠️ 精确比对很脆弱——多一个字段或少一个空格都会失败。只在响应体稳定时使用。

#### `none` — 无断言

```yaml
asserts:
  - type: none
```

跳过所有断言。即使 HTTP 500 也标记为通过（网络错误仍算失败）。适合只触发不关心结果的步骤。

### 断言示例组合

```yaml
asserts:
  - type: status_code
    expect: 200
  - type: jsonpath
    path: $.code
    expect: "0"
  - type: body_match
    expect: "数据获取成功"
```

---

## 🔗 变量系统

### 提取变量

从上一步的响应中提取值，供后续步骤使用：

```yaml
- name: 登录
  method: POST
  url: https://api.example.com/login
  asserts:
    - type: status_code
      expect: 200
  extract:
    - source: body                   # 从响应体提取（JSONPath）
      path: $.data.token
      var_name: auth_token
    - source: header                 # 从响应头提取
      path: Set-Cookie
      var_name: session_cookie
```

| `source` | `path` 取值 | 说明 |
|---|---|---|
| `body` | JSONPath（如 `$.data.id`） | 从响应 JSON 体提取 |
| `header` | 头部字段名（如 `X-Session-Id`） | 从响应头提取，不区分大小写 |

### 使用变量

在后续步骤中通过 `${var_name}` 引用：

```yaml
- name: 获取用户信息
  method: GET
  url: https://api.example.com/users/${user_id}
  headers:
    Authorization: "Bearer ${auth_token}"
```

### 变量替换范围

| 字段 | 是否替换 |
|---|---|
| `url` | ✅ |
| `headers` (key + value) | ✅ |
| `body.content` | ✅ |
| `params` (key + value) | ✅ |
| `asserts[*].expect` | ✅ |
| `name` | ✅ |
| `pre_script` / `post_script` | ✅ |
| `method` | ❌ |

### 变量规则

1. **断言通过后才提取** — 步骤断言失败，不提取变量
2. **后值覆盖前值** — 同名变量，后面的值覆盖前面的
3. **路径不存在则静默忽略** — 不报错，不创建变量
4. **未定义变量保留原样** — `${undefined_var}` 保持原样，不报错
5. **跨文件不共享** — 每个 YAML 文件独立的变量池

---

## ⚙️ 执行策略

### 执行流程

每个步骤的执行顺序：

```
前置脚本 → 变量替换 → HTTP 请求（带重试） → 断言 → 变量提取 → 后置脚本
```

### 重试

```yaml
- name: 查询订单状态
  method: GET
  url: https://api.example.com/orders/${order_id}
  retry_count: 5
  retry_interval: 2s
  timeout: 10s
  asserts:
    - type: jsonpath
      path: $.data.status
      expect: "completed"
```

触发重试的条件：
- **网络错误** — DNS 解析失败、连接被拒绝、TLS 握手失败
- **断言失败** — 状态码不匹配、JSONPath 值不对等

不触发重试的情况：
- **前置脚本失败** — 直接终止步骤
- **请求超时** — 超时算网络错误，会触发重试

### 失败行为

```yaml
steps:
  - name: 可选步骤
    on_failure: continue    # 失败后继续执行

  - name: 关键步骤
    on_failure: stop        # 失败后终止（默认行为）
```

| `on_failure` | 行为 |
|---|---|
| `stop`（默认） | 步骤失败，停止执行后续所有步骤 |
| `continue` | 步骤标记为失败，继续执行下一步 |

---

## 📜 脚本钩子

在 HTTP 请求前后执行 Shell 脚本：

```yaml
- name: 创建用户
  pre_script: |
    echo "准备测试数据..."
    curl -s -X POST http://localhost:3000/test/setup \
      -H "Content-Type: application/json" \
      -d '{"scenario":"user-create"}'
  method: POST
  url: https://api.example.com/users
  body:
    type: json
    content: '{"name":"测试用户"}'
  asserts:
    - type: status_code
      expect: 201
  post_script: |
    echo "清理测试数据..."
    curl -s -X DELETE http://localhost:3000/test/cleanup > /dev/null
```

### 规则

| 规则 | 说明 |
|---|---|
| **执行环境** | 使用 `sh -c` 执行，支持管道、重定向 |
| **超时** | 默认 30s |
| **变量替换** | 脚本内容支持 `${var}` 替换 |
| **前置脚本失败** | 步骤直接失败，不发送 HTTP 请求 |
| **后置脚本** | 无论步骤成功/失败**始终执行**，适合清理 |
| **后置脚本失败** | 仅记录警告，不影响步骤结果 |

---

## 📊 输出格式

### 单文件执行（默认模式）

```
▶ [1/3] 登录获取 Token ................................. POST https://api.example.com/auth/login
  ✓ 200 OK (412ms)

▶ [2/3] 获取用户信息 ................................... GET https://api.example.com/user/profile
  ✓ 200 OK (45ms)

▶ [3/3] 创建订单 ....................................... POST https://api.example.com/orders
  ✗ 500 ASSERT (312ms)
    └─ JSONPath $.code: 期望 "0"，实际 "50001"

══════════════════════════════════════════════════
  总计: 3  |  ✓ 通过: 2  |  ✗ 失败: 1  |  耗时: 769ms
  失败步骤: 创建订单
══════════════════════════════════════════════════
```

### 批量执行模式（目录 / 通配符）

```
▶ auth_test.yaml ......... 3/3 ✓ 通过 (450ms)
▶ user_test.yaml ......... 2/3 ✗ 失败 (1.2s)
  └─ 失败步骤: 获取用户信息
▶ order_test.yaml ........ 4/4 ✓ 通过 (890ms)
══════════════════════════════════════
  文件: 3  |  ✓ 全通过: 2  |  ✗ 有失败: 1
  失败文件: user_test.yaml
══════════════════════════════════════
```

### JSON 报告（`-o report.json`）

```json
[
  {
    "file": "user_test.yaml",
    "total": 3,
    "passed": 2,
    "failed": 1,
    "elapsed": "1.2s",
    "steps": [
      {
        "name": "登录",
        "number": 1,
        "method": "POST",
        "url": "https://api.example.com/auth/login",
        "status_code": 200,
        "duration": "412ms",
        "passed": true
      },
      {
        "name": "获取用户信息",
        "number": 2,
        "method": "GET",
        "url": "https://api.example.com/user/profile",
        "status_code": 500,
        "duration": "45ms",
        "passed": false,
        "error": "assertion failed",
        "failures": [
          {
            "type": "jsonpath",
            "path": "$.data.role",
            "expected": "admin",
            "actual": "<path not found>",
            "passed": false
          }
        ]
      }
    ]
  }
]
```

---

## 📁 项目结构

```
yamlit/
├── cmd/
│   └── yamlit/
│       └── main.go              # CLI 入口：参数解析、文件发现、执行调度
├── pkg/
│   ├── parser/                  # YAML 解析与校验
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── runner/                  # 执行引擎：步骤协调、重试、脚本
│   │   ├── runner.go
│   │   └── runner_test.go
│   ├── step/                    # 步骤定义与 HTTP 执行
│   │   ├── types.go             # 步骤数据结构
│   │   ├── step_result.go       # 执行结果数据结构
│   │   ├── executor.go          # HTTP 请求执行器
│   │   └── executor_test.go
│   ├── assert/                  # 断言引擎
│   │   ├── assert.go
│   │   └── assert_test.go
│   ├── extract/                 # 变量提取
│   │   ├── extract.go
│   │   └── extract_test.go
│   ├── variable/                # 变量池与模板替换
│   │   ├── pool.go
│   │   └── pool_test.go
│   └── reporter/                # 终端彩色输出
│       ├── reporter.go
│       └── reporter_test.go
├── testdata/
│   └── basic.yaml               # 示例测试文件
├── docs/
│   ├── plans/
│   │   ├── 2026-05-01-yamlit-design.md
│   │   └── 2026-05-01-yamlit-implementation-plan.md
│   └── ai-agent-yaml-guide.md   # AI Agent 编写 YAML 指南
├── go.mod
├── go.sum
└── README.md
```

---

## 🛠 开发指南

### 环境要求

- Go 1.26+
- Linux / macOS

### 常用命令

```bash
# 构建
go build -o yamlit ./cmd/yamlit/

# 运行所有测试
go test ./... -v

# 仅运行特定包的测试
go test ./pkg/assert/ -v

# 代码静态检查
go vet ./...

# 测试覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 清理
go clean
```

### 依赖

| 包 | 用途 |
|---|---|
| `gopkg.in/yaml.v3` | YAML 反序列化 |
| `github.com/tidwall/gjson` | JSONPath 提取 |
| `github.com/fatih/color` | 终端彩色输出 |

---

## 📚 相关文档

| 文档 | 说明 |
|---|---|
| [AI Agent YAML 编写指南](docs/ai-agent-yaml-guide.md) | 面向 AI Agent 的详细 YAML 编写教程，含 10 种常见模式和 2 个完整示例 |
| [设计文档](docs/plans/2026-05-01-yamlit-design.md) | 项目架构设计决策与组件说明 |
| [实现计划](docs/plans/2026-05-01-yamlit-implementation-plan.md) | TDD 实现步骤与测试策略 |
