# AI Agent 指南：如何编写 yamlit YAML 测试文件

> 本文档面向 AI Agent（LLM / 编码助手），指导你为用户生成正确、高质量的 yamlit YAML 测试文件。

## 目录

1. [格式总览](#1-格式总览)
2. [字段详解](#2-字段详解)
3. [断言系统](#3-断言系统)
4. [变量提取与替换](#4-变量提取与替换)
5. [常见模式](#5-常见模式)
6. [最佳实践](#6-最佳实践)
7. [常见错误](#7-常见错误)
8. [完整示例](#8-完整示例)

---

## 1. 格式总览

yamlit 使用 **YAML** 格式。一个文件是一个 **步骤数组**，每个步骤定义一次 HTTP 请求及其验证逻辑。

```yaml
- name: <步骤名>
  method: <HTTP 方法>
  url: <请求 URL>
  params:              # 可选
    key: value
  headers:             # 可选
    Key: Value
  body:                # 可选
    type: json         # json | form | text
    content: "..."     # 请求体内容
  timeout: 30s         # 可选，默认 30s
  retry_count: 0       # 可选，默认 0
  retry_interval: 1s   # 可选，默认 1s
  on_failure: stop     # 可选，stop | continue，默认 stop
  asserts:             # 可选
    - type: status_code
      expect: 200
  extract:             # 可选
    - source: body
      path: $.data.id
      var_name: my_var
  pre_script: ""       # 可选
  post_script: ""      # 可选
```

### 关键规则

| 规则 | 说明 |
|---|---|
| **步骤顺序执行** | 上一步失败默认停止后续步骤 |
| **方法名大写** | 写成 `post` / `POST` / `Post` 都可以，执行器自动转大写 |
| **字段可省略** | `params`、`headers`、`body`、`asserts`、`extract` 等都可省略 |
| **无断言 = 不检查** | 不写 `asserts` 字段表示不做任何检查 |
| **YAML 纯文本注意** | 含 `{}` 的内容必须用引号包裹：`content: '{"a":1}'` |

---

## 2. 字段详解

### `name`（必填）

步骤的唯一标识，在输出报告中显示。

```yaml
- name: 获取用户信息
```

- 长度不限，见名知意
- 支持 `${var}` 变量替换（通常不需要）

### `method`（必填）

支持的 HTTP 方法：

| 方法 | 用途 |
|---|---|
| `GET` | 查询资源 |
| `POST` | 创建资源 |
| `PUT` | 全量更新资源 |
| `PATCH` | 部分更新资源 |
| `DELETE` | 删除资源 |
| `HEAD` | 获取响应头 |
| `OPTIONS` | 获取支持的请求方法 |

**不支持变量替换。**

### `url`（必填）

请求 URL，支持 `${var}` 变量替换。

```yaml
- name: 获取用户
  url: https://api.example.com/users/${user_id}
```

- 如果 URL 中已经包含 `?` 查询参数，`params` 字段会用 `&` 拼接

### `params`（可选）

URL 查询参数，key/value 形式。支持变量替换。

```yaml
- name: 搜索
  url: https://api.example.com/search
  params:
    q: hello
    page: "1"
    limit: "${page_size}"
```

### `headers`（可选）

HTTP 请求头，key/value 形式。支持变量替换。

```yaml
- name: 获取数据
  headers:
    Authorization: "Bearer ${token}"
    Accept: application/json
```

- `Content-Type` 优先级：手动设置 > 根据 `body.type` 自动推断

### `body`（可选）

请求体，支持三种类型：

```yaml
# JSON
- name: 创建用户
  method: POST
  body:
    type: json
    content: '{"name":"Alice","age":30}'

# Form 表单
- name: 登录
  method: POST
  body:
    type: form
    content: "username=alice&password=123456"

# 纯文本
- name: 发送消息
  method: POST
  body:
    type: text
    content: "Hello, World"
```

每种类型自动设置 `Content-Type`：

| type | Content-Type |
|---|---|
| `json` | `application/json` |
| `form` | `application/x-www-form-urlencoded` |
| `text` | `text/plain` |

**注意：** JSON 格式的 `content` 必须用 YAML 引号包裹（单引号 `'` 或双引号 `"`），否则 YAML 解析器会报错。

### `timeout`（可选）

单次 HTTP 请求超时时间。格式为 Go 的 duration 字符串。

```yaml
- name: 慢接口
  timeout: 60s
```

常见值：`5s`、`10s`、`30s`、`60s`、`120s`。不设置则默认 **30s**。

### `retry_count` / `retry_interval`（可选）

失败重试策略：

```yaml
- name: 不稳定接口
  retry_count: 3
  retry_interval: 2s
```

- `retry_count`：重试次数。设为 0 表示不重试。默认 0。
- `retry_interval`：重试间隔。格式与 `timeout` 相同。默认 1s。
- 重试条件：网络错误 **或** 断言失败
- 总请求次数 = `retry_count + 1`（首次 + 重试）

### `on_failure`（可选）

步骤失败时的行为：

| 值 | 行为 |
|---|---|
| `stop`（默认） | 停止执行后续步骤，测试终止 |
| `continue` | 标记步骤失败，继续执行下一步 |

```yaml
- name: 可选步骤
  on_failure: continue
```

### `pre_script` / `post_script`（可选）

在请求前后执行的 Shell 脚本：

```yaml
- name: 需要准备数据
  pre_script: |
    curl -s -X POST http://test-server/setup \
      -H "Content-Type: application/json" \
      -d '{"status":"ready"}'
  method: GET
  url: https://api.example.com/data
  post_script: |
    curl -s -X POST http://test-server/cleanup
```

- 使用 `sh -c` 执行，支持管道、重定向等 shell 特性
- 超时默认 30s
- 前置脚本在**变量替换之后**执行，可读取 `${var}`
- 后置脚本**始终执行**（无论步骤成功失败），适合清理
- 前置脚本失败 → 步骤直接失败（不发送 HTTP 请求）
- 后置脚本失败 → 只记录警告，不影响步骤结果

---

## 3. 断言系统

断言是一个步骤中**验证响应结果**的部分。多条断言是 **AND** 关系——全部通过才算通过。

### `status_code` — 状态码检查

```yaml
asserts:
  - type: status_code
    expect: 200
```

最常见的断言。expect 写数字的字符串形式。

常见值：
- `200` — 查询成功
- `201` — 创建成功
- `204` — 删除成功（无响应体）
- `400` — 请求参数错误
- `401` — 未认证
- `403` — 无权限
- `404` — 资源不存在
- `500` — 服务器内部错误

### `jsonpath` — JSONPath 键值比对

```yaml
asserts:
  - type: jsonpath
    path: $.code
    expect: "0"
  - type: jsonpath
    path: $.data.user.name
    expect: "Alice"
  - type: jsonpath
    path: $.data.items.#     # 数组长度
    expect: "10"
```

支持两种 JSONPath 写法（效果相同）：
- `$.data.id` — 标准 JSONPath
- `data.id` — 简写（gjson 原生格式）

**expect 的值必须是字符串**。数字 `0` 需要写成 `"0"`，布尔值 `true` 写成 `"true"`。

常用 JSONPath 模式：

| 路径 | 含义 | 响应示例 |
|---|---|---|
| `$.code` | 取顶层字段 | `{"code":0}` → `0` |
| `$.data.id` | 取嵌套字段 | `{"data":{"id":42}}` → `42` |
| `$.data.items.#` | 取数组长度 | `{"data":{"items":[1,2,3]}}` → `3` |
| `$.data.items.0` | 取数组第一个元素 | `{"data":{"items":["a","b"]}}` → `a` |
| `$.data.items.#(status=="ok").id` | 条件查询 | 取第一个 status 为 "ok" 的元素的 id |

### `body_match` — 子串匹配

```yaml
asserts:
  - type: body_match
    expect: "success"
```

检查响应体是否**包含**指定字符串。适合检查错误消息、关键词等。

### `body_equals` — 精确比对

```yaml
asserts:
  - type: body_equals
    expect: '{"code":0,"msg":"ok"}'
```

响应体与期望值**完全一致**（去除两端空白后）。适合需要精确响应的场景。

**注意：** JSON 格式的 expect 需用 YAML 单引号 `'` 包裹。

### `none` — 无断言

```yaml
asserts:
  - type: none
```

跳过所有检查。即使返回 500 也标记为通过（但网络错误仍算失败）。适合只触发请求、不关心结果的步骤（如健康检查）。

### 组合使用

```yaml
asserts:
  - type: status_code
    expect: 200
  - type: jsonpath
    path: $.code
    expect: "0"
  - type: body_match
    expect: "操作成功"
```

---

## 4. 变量提取与替换

### 提取变量

从响应中提取值，存为变量供后续步骤使用：

```yaml
- name: 登录
  method: POST
  url: https://api.example.com/login
  body:
    type: json
    content: '{"username":"admin","password":"123456"}'
  extract:
    - source: body
      path: $.data.token
      var_name: auth_token
    - source: body
      path: $.data.user_id
      var_name: user_id
    - source: header
      path: X-Session-Id
      var_name: session_id
```

| source | path 的取值 | 说明 |
|---|---|---|
| `body` | JSONPath | 从响应体提取，如 `$.data.token` |
| `header` | 头部字段名 | 从响应头提取，如 `X-Session-Id`（不区分大小写） |

**提取规则：**
- 变量仅在**断言全部通过后**才提取
- 同一变量名被覆盖（后面的值覆盖前面的）
- 提取路径不存在时静默忽略（不报错，不创建变量）

### 使用变量

在后续步骤中通过 `${var_name}` 使用：

```yaml
- name: 获取用户信息
  method: GET
  url: https://api.example.com/users/${user_id}
  headers:
    Authorization: "Bearer ${auth_token}"
```

**支持变量替换的字段：**
- `url`
- `headers`（key 和 value）
- `body.content`
- `params`（key 和 value）
- `asserts[*].expect`
- `name`
- `pre_script`
- `post_script`

**不支持变量替换的字段：** `method`。

### 注意事项

- **未定义的变量不会报错**——`${undefined_var}` 保持原样。这是一把双刃剑：可以让你看到拼写错误，但也意味着你不会得到警告。
- **变量不跨文件共享**——每个 YAML 文件有独立的变量池。
- **变量在断言通过后才提取**——如果上一步断言失败，变量不会被提取。

---

## 5. 常见模式

### 模式 1：登录 + 后续操作（最常用）

```yaml
- name: 登录
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
      var_name: token

- name: 获取用户信息
  method: GET
  url: https://api.example.com/user/profile
  headers:
    Authorization: "Bearer ${token}"
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.name
      expect: "管理员"

- name: 更新用户信息
  method: PUT
  url: https://api.example.com/user/profile
  headers:
    Authorization: "Bearer ${token}"
    Content-Type: application/json
  body:
    type: json
    content: '{"nickname":"新昵称"}'
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.code
      expect: "0"
```

### 模式 2：创建 → 查询 → 删除（CRUD 流程）

```yaml
- name: 创建资源
  method: POST
  url: https://api.example.com/items
  body:
    type: json
    content: '{"name":"测试文章","content":"内容"}'
  asserts:
    - type: status_code
      expect: 201
  extract:
    - source: body
      path: $.data.id
      var_name: item_id

- name: 查询资源
  method: GET
  url: https://api.example.com/items/${item_id}
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.name
      expect: "测试文章"

- name: 删除资源
  method: DELETE
  url: https://api.example.com/items/${item_id}
  asserts:
    - type: status_code
      expect: 204

- name: 确认已删除
  method: GET
  url: https://api.example.com/items/${item_id}
  asserts:
    - type: status_code
      expect: 404
```

### 模式 3：分页查询

```yaml
- name: 查询第一页
  method: GET
  url: https://api.example.com/users
  params:
    page: "1"
    limit: "20"
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.items.#
      expect: "20"
  extract:
    - source: body
      path: $.data.total
      var_name: total
    - source: body
      path: $.data.has_more
      var_name: has_more
```

### 模式 4：依赖上一步数据的链式调用

```yaml
- name: 获取文章列表
  method: GET
  url: https://api.example.com/articles
  asserts:
    - type: status_code
      expect: 200
  extract:
    - source: body
      path: $.data.items.0.id
      var_name: first_article_id
    - source: body
      path: $.data.items.0.author_id
      var_name: author_id

- name: 查看文章详情
  method: GET
  url: https://api.example.com/articles/${first_article_id}

- name: 查看作者信息
  method: GET
  url: https://api.example.com/users/${author_id}
```

### 模式 5：不关心结果（只触发）

```yaml
- name: 触发异步任务
  method: POST
  url: https://api.example.com/tasks
  body:
    type: json
    content: '{"type":"data_export"}'
  asserts:
    - type: none
  on_failure: continue
```

### 模式 6：带重试的不稳定接口

```yaml
- name: 查询订单状态（可能延迟）
  method: GET
  url: https://api.example.com/orders/${order_id}
  retry_count: 5
  retry_interval: 3s
  timeout: 10s
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.status
      expect: "completed"
```

### 模式 7：表单登录 + 提取 Cookie（通过响应头）

```yaml
- name: 表单登录
  method: POST
  url: https://api.example.com/login
  body:
    type: form
    content: "username=admin&password=123456"
  asserts:
    - type: status_code
      expect: 200
  extract:
    - source: header
      path: Set-Cookie
      var_name: cookie

- name: 访问需要 Cookie 的接口
  method: GET
  url: https://api.example.com/dashboard
  headers:
    Cookie: "${cookie}"
```

### 模式 8：使用脚本准备和清理数据

```yaml
- name: 创建用户
  pre_script: |
    # 确保测试数据存在
    curl -s -X POST http://localhost:8080/api/test/setup \
      -H "Content-Type: application/json" \
      -d '{"scenario":"user-crud"}' > /dev/null
  method: POST
  url: https://api.example.com/users
  body:
    type: json
    content: '{"name":"TestUser"}'
  asserts:
    - type: status_code
      expect: 201
  extract:
    - source: body
      path: $.data.id
      var_name: user_id
  post_script: |
    # 清理：删除测试用户
    curl -s -X DELETE "http://localhost:8080/api/users/${user_id}" > /dev/null
```

### 模式 9：批量运行多个场景（文件拆分）

将一个复杂测试拆分为多个 YAML 文件，每个文件一个独立场景：

```yaml
# tests/auth/login.yaml — 登录测试
- name: 正常登录
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

- name: 错误密码
  method: POST
  url: https://api.example.com/auth/login
  body:
    type: json
    content: '{"username":"admin","password":"wrong"}'
  asserts:
    - type: status_code
      expect: 401
```

```bash
# 批量运行所有测试
./yamlit tests/
```

### 模式 10：验证错误处理

```yaml
- name: 缺少必填字段
  method: POST
  url: https://api.example.com/users
  body:
    type: json
    content: '{"name":""}'
  asserts:
    - type: status_code
      expect: 400
    - type: jsonpath
      path: $.message
      expect: "name 不能为空"

- name: 无效的 ID
  method: GET
  url: https://api.example.com/users/abc
  asserts:
    - type: status_code
      expect: 400
    - type: body_match
      expect: "无效的ID"

- name: 不存在的资源
  method: GET
  url: https://api.example.com/users/999999
  asserts:
    - type: status_code
      expect: 404
```

---

## 6. 最佳实践

### 文件组织

```
tests/
├── auth/
│   ├── login.yaml
│   └── register.yaml
├── users/
│   ├── create.yaml
│   ├── read.yaml
│   ├── update.yaml
│   └── delete.yaml
├── orders/
│   └── full_flow.yaml
└── health.yaml
```

### 步骤命名

- 使用中文或英文，保持一致
- 命名要体现**做什么**而不是**怎么发请求**
- 好的例子：`创建用户`、`查询订单`、`登录失败_密码错误`
- 不好的例子：`step1`、`test`、`请求1`

### 断言策略

- **必做：** 至少检查 `status_code`。不做状态码检查的步骤意义不大。
- **推荐：** 对关键字段加 `jsonpath` 断言，验证响应的业务含义。
- **适度：** 不要断言每一个字段，只断言对测试场景重要的字段。
- **`body_equals` 谨慎使用：** 精确比对很脆弱，响应体多一个字段或少一个空格都会失败。适合固定不变的小响应。

### 变量管理

- 变量名要有意义：`user_id`、`auth_token`、`order_id`
- 不要用名字冲突：`id` 在不同步骤中可能指不同含义，用 `user_id`、`order_id` 区分
- 在流程开始处提取关键变量（如 token），后续步骤直接引用

### 重试策略

- 重试适合**最终一致性**的场景（如订单状态从 `pending` → `completed`）
- 重试不适合**幂等性要求**的场景（如创建资源，重试可能导致重复创建）
- `retry_interval` 不要太短（至少 1s），避免对服务器造成压力

### 脚本使用

- 前置脚本适合数据准备，后置脚本适合数据清理
- 脚本中尽量使用 `> /dev/null` 静默输出，避免干扰测试日志
- 脚本失败会中断测试，所以只放必要的操作
- 如果测试目标就是本地服务，脚本可以直接调用测试服务的 API 做 setup/teardown

---

## 7. 常见错误

### ❌ JSON 内容不用引号包裹

```yaml
# 错误
body:
  type: json
  content: {"name":"Alice"}

# 正确
body:
  type: json
  content: '{"name":"Alice"}'
```

### ❌ expect 值类型错误

```yaml
# 错误 — expect 的值必须是字符串
asserts:
  - type: jsonpath
    path: $.code
    expect: 0    # 数字，不对

# 正确
asserts:
  - type: jsonpath
    path: $.code
    expect: "0"  # 字符串
```

### ❌ 依赖步骤不检查提取是否成功

```yaml
# 不安全的做法 — 如果 login 失败，token 不会被提取
- name: login
  method: POST
  url: ...
  extract:
    - source: body
      path: $.data.token
      var_name: token

- name: get_data  # 这里会使用 ${token}，但可能是未定义
  headers:
    Authorization: "Bearer ${token}"

# 安全的做法 — 在 login 步骤加断言确保提取成功
- name: login
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.token
      expect: "${token_exists_placeholder}"  # 或者省略，只检查 status_code
  extract:
    - source: body
      path: $.data.token
      var_name: token
```

### ❌ 在方法名中使用变量

```yaml
# 错误 — method 不支持变量替换
- name: 动态方法
  method: "${http_method}"
  url: https://api.example.com/data

# 正确 — 直接写固定方法名
- name: 查询数据
  method: GET
```

### ❌ `on_failure: continue` 忘记加

```yaml
# 如果步骤 1 失败，步骤 2 不会执行
- name: 可选步骤
  method: GET
  url: https://api.example.com/optional
  asserts:
    - type: status_code
      expect: 200

- name: 后续步骤  # 不会走到这里
```

### ❌ 提取路径不存在时无感知

```yaml
- name: 登录
  extract:
    - source: body
      path: $.data.token    # 如果响应体里没有 token，静默忽略
      var_name: token
```

变量提取路径不存在时**不会报错**。建议在上下文中做断言确保字段存在：

```yaml
- name: 登录
  asserts:
    - type: status_code
      expect: 200
  extract:
    - source: body
      path: $.data.token
      var_name: token
```

### ❌ duration 格式错误

```yaml
# 正确写法
timeout: 30s
retry_interval: 1s

# 错误写法
timeout: 30        # 数字，不是字符串
timeout: 30秒      # 不支持中文
timeout: 30seconds # 不支持 long form
```

支持格式（Go time.Duration）：`30s`、`5m`、`1h`、`500ms`。

---

## 8. 完整示例

### 示例 1：用户管理完整流程

```yaml
# tests/users/full_flow.yaml — 用户管理全流程测试

- name: 管理员登录
  method: POST
  url: https://api.example.com/auth/login
  body:
    type: json
    content: '{"username":"admin","password":"admin123"}'
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.code
      expect: "0"
  extract:
    - source: body
      path: $.data.token
      var_name: admin_token

- name: 创建新用户
  method: POST
  url: https://api.example.com/users
  headers:
    Authorization: "Bearer ${admin_token}"
  body:
    type: json
    content: '{"name":"测试用户","email":"test@example.com","role":"user"}'
  asserts:
    - type: status_code
      expect: 201
    - type: jsonpath
      path: $.data.name
      expect: "测试用户"
  extract:
    - source: body
      path: $.data.id
      var_name: new_user_id

- name: 查询新用户
  method: GET
  url: https://api.example.com/users/${new_user_id}
  headers:
    Authorization: "Bearer ${admin_token}"
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.email
      expect: "test@example.com"

- name: 更新用户角色
  method: PUT
  url: https://api.example.com/users/${new_user_id}
  headers:
    Authorization: "Bearer ${admin_token}"
  body:
    type: json
    content: '{"role":"editor"}'
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.role
      expect: "editor"

- name: 删除用户
  method: DELETE
  url: https://api.example.com/users/${new_user_id}
  headers:
    Authorization: "Bearer ${admin_token}"
  asserts:
    - type: status_code
      expect: 204

- name: 确认用户已删除
  method: GET
  url: https://api.example.com/users/${new_user_id}
  headers:
    Authorization: "Bearer ${admin_token}"
  asserts:
    - type: status_code
      expect: 404
    - type: body_match
      expect: "用户不存在"
```

### 示例 2：电商订单流程（带重试）

```yaml
# tests/orders/purchase_flow.yaml — 订单购买流程

- name: 获取商品列表
  method: GET
  url: https://api.example.com/products
  params:
    page: "1"
    limit: "10"
  asserts:
    - type: status_code
      expect: 200
  extract:
    - source: body
      path: $.data.items.0.id
      var_name: product_id
    - source: body
      path: $.data.items.0.price
      var_name: product_price

- name: 用户登录
  method: POST
  url: https://api.example.com/auth/login
  body:
    type: json
    content: '{"username":"buyer","password":"buyer123"}'
  asserts:
    - type: status_code
      expect: 200
  extract:
    - source: body
      path: $.data.token
      var_name: user_token

- name: 创建订单
  method: POST
  url: https://api.example.com/orders
  headers:
    Authorization: "Bearer ${user_token}"
  body:
    type: json
    content: '{"product_id":"${product_id}","quantity":1,"price":${product_price}}'
  asserts:
    - type: status_code
      expect: 201
  extract:
    - source: body
      path: $.data.order_id
      var_name: order_id

- name: 支付订单
  method: POST
  url: https://api.example.com/orders/${order_id}/pay
  headers:
    Authorization: "Bearer ${user_token}"
  body:
    type: json
    content: '{"payment_method":"wechat"}'
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.status
      expect: "paid"

- name: 查询订单状态（可能延迟处理）
  method: GET
  url: https://api.example.com/orders/${order_id}
  headers:
    Authorization: "Bearer ${user_token}"
  retry_count: 5
  retry_interval: 2s
  timeout: 15s
  asserts:
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.data.status
      expect: "completed"
```
