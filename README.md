# HTTP Tester

一个轻量级的 YAML 驱动的 HTTP Restful API 测试工具。用 Go 实现，单二进制分发，面向个人开发者。

## 快速开始

```bash
# 构建
go build -o http_tester ./cmd/http_tester/

# 运行测试
./http_tester test.yaml

# 批量运行目录下所有测试
./http_tester ./tests/
```

## 安装

```bash
# 从源码构建
git clone <repo>
cd http-tester
go build -o http_tester ./cmd/http_tester/

# 或直接 go install
go install github.com/mike/yaml-testing/cmd/http_tester@latest
```

## CLI 用法

```
./http_tester <file.yaml>               # 执行单个 YAML 文件
./http_tester <directory/>              # 批量执行目录下所有 .yaml/.yml
./http_tester "tests/*.yaml"            # 通配符模式
./http_tester -v <file.yaml>            # verbose 模式（输出详细步骤）
./http_tester -o report.json <file.yaml> # 输出 JSON 报告
./http_tester -v -o report.json <file.yaml> # 同时使用多个 flag
```

> **注意：** flags 必须放在文件路径**前面**，这是 Go flag 标准行为。

返回码：全部通过 → 0，有失败 → 1（方便 CI 集成）。

## YAML 格式说明

一个 YAML 文件包含多个步骤，每个步骤都是一个 HTTP 接口调用：

```yaml
- name: login                          # 步骤标识名
  method: POST                         # HTTP 方法
  url: https://api.example.com/login   # 请求 URL（支持 ${var} 模板）
  params:                              # URL 查询参数（可选）
    key1: value1
  headers:                             # 请求头（可选）
    Content-Type: application/json
  body:                                # 请求体（可选）
    type: json                         # json | form | text
    content: '{"user":"test"}'         # 支持 ${var} 模板
  timeout: 30s                         # 单次请求超时（可选，默认 30s）
  retry_count: 2                       # 失败重试次数（可选，默认 0）
  retry_interval: 1s                   # 重试间隔（可选，默认 1s）
  on_failure: stop                     # stop | continue（可选，默认 stop）
  asserts:                             # 断言列表（可选，多项 AND 关系）
    - type: status_code
      expect: 200
    - type: jsonpath
      path: $.code
      expect: "0"
    - type: body_match
      expect: "success"
    - type: body_equals
      expect: '{"code":0,"msg":"ok"}'
    - type: none                       # 不检查，只执行请求
  extract:                             # 变量提取（可选，多项）
    - source: body                     # body | header
      path: $.data.token
      var_name: auth_token
  pre_script: "echo before"            # 前置 Shell 脚本（可选）
  post_script: "echo after"            # 后置 Shell 脚本（可选）
```

## 断言类型

| 类型 | 说明 | 示例 |
|---|---|---|
| `status_code` | 检查 HTTP 状态码 | `expect: 201` |
| `jsonpath` | 用 JSONPath 提取响应体键值比对 | `path: $.data.id`, `expect: "42"` |
| `body_match` | 响应体包含指定字符串（子串匹配） | `expect: "success"` |
| `body_equals` | 响应体与预期字符串精确比对 | `expect: '{"code":0}'` |
| `none` | 跳过所有断言 | — |

JSONPath 支持 `$.data.name` 和 `data.name` 两种写法。

多条断言为 **AND** 关系，全部通过才算断言通过。

## 变量提取与替换

在前面步骤提取的变量，在后续步骤可以通过 `${var_name}` 读取：

```yaml
- name: login
  method: POST
  url: https://api.example.com/login
  extract:
    - source: body
      path: $.data.token
      var_name: auth_token

- name: get_profile
  method: GET
  url: https://api.example.com/profile
  headers:
    Authorization: "Bearer ${auth_token}"
```

变量替换支持以下字段：
- `url`
- `headers`（key 和 value）
- `body.content`
- `params`（key 和 value）
- `asserts[*].expect`
- `pre_script`
- `post_script`

**不支持** `method` 字段替换。

未定义的 `${var}` 保持原样不替换（不报错）。

## 请求体类型

| type | Content-Type | 说明 |
|---|---|---|
| `json` | `application/json` | JSON 格式 |
| `form` | `application/x-www-form-urlencoded` | 表单格式 |
| `text` | `text/plain` | 纯文本 |

如果未设置 `Content-Type` 请求头，会根据 `body.type` 自动设置。

## 执行策略

### 重试

每个步骤可以独立配置重试策略：

```yaml
- name: flaky-api
  method: GET
  url: https://api.example.com/flaky
  retry_count: 3
  retry_interval: 2s
  timeout: 10s
```

重试发生在以下情况：
- **网络错误**（DNS 解析失败、连接被拒绝等）
- **断言失败**（状态码不匹配、JSONPath 值不对等）

### 失败行为

```yaml
- name: step1
  on_failure: stop      # 步骤失败则停止整个测试（默认）

- name: step2
  on_failure: continue  # 步骤失败继续执行后续步骤
```

## 前置/后置脚本

支持在 HTTP 请求前后执行 Shell 脚本：

```yaml
- name: with-scripts
  pre_script: |
    echo "准备数据..."
    curl -s -X POST http://test-server/setup
  method: GET
  url: https://api.example.com/data
  post_script: |
    echo "清理数据..."
    curl -s -X POST http://test-server/teardown
```

- 前置脚本在变量替换之后、HTTP 请求之前执行
- 后置脚本始终执行（无论步骤成功或失败），适合清理
- 脚本超时默认 30s
- 脚本返回非零退出码 → 步骤失败

## 终端输出

### 单文件执行

```
▶ [1/3] login ............................................ POST https://api.example.com/login
  ✓ 200 OK (238ms)

▶ [2/3] get_user_info .................................... GET https://api.example.com/user
  ✓ 200 OK (45ms)

▶ [3/3] create_order ..................................... POST https://api.example.com/orders
  ✗ 500 ASSERT (312ms)
    └─ JSONPath $.code: 期望 "0"，实际 "50001"

══════════════════════════════════════════════════
  总计: 3  |  ✓ 通过: 2  |  ✗ 失败: 1  |  耗时: 1.2s
  失败步骤: create_order
══════════════════════════════════════════════════
```

### 批量执行

```
▶ auth_test.yaml ......... 3/3 ✓ 通过 (450ms)
▶ user_test.yaml ......... 2/3 ✗ 失败 (1.2s)
  └─ 失败步骤: get_profile
══════════════════════════════════════
  文件: 2  |  ✓ 全通过: 1  |  ✗ 有失败: 1
  失败文件: user_test.yaml
══════════════════════════════════════
```

## 项目结构

```
http-tester/
├── cmd/
│   └── http_tester/
│       └── main.go              # CLI 入口
├── pkg/
│   ├── parser/                  # YAML 解析与校验
│   ├── runner/                  # 执行引擎（协调器）
│   ├── step/                    # 步骤定义与 HTTP 执行
│   ├── assert/                  # 断言引擎
│   ├── extract/                 # 变量提取
│   ├── variable/                # 变量池与模板替换
│   └── reporter/                # 终端输出
├── testdata/
│   ├── basic.yaml               # 示例测试文件
├── go.mod
└── README.md
```

## 开发

```bash
# 运行所有测试
go test ./...

# 构建
go build -o http_tester ./cmd/http_tester/

# 代码检查
go vet ./...
```

## 依赖

- `gopkg.in/yaml.v3` — YAML 解析
- `github.com/tidwall/gjson` — JSONPath 提取
- `github.com/fatih/color` — 终端彩色输出
