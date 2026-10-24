# 孤儿导出类型清理证据（task-57，4 个 + 补充 1 个）

- Trellis 任务：`10-24-orphan-types-and-relaykit-policy`（PRD 见同目录 `prd.md`，R1）
- 共享任务：task-57「清 4 个孤儿导出类型（cachex/ionet）」
- 执行者：worker-script2
- 日期：2026-10-24
- 边界：未提交、未推送、未发版、未联网、未装工具、未动 `web/`（locale 由 worker-docs 处理）、未动 `data/` 与实例
- 基线：HEAD = `47b2321`（上一轮 `10-24-internal-symbols-and-b-wiring` 已由 Lead 提交并归档；`VERSION` = 0.0.49）

## 0. 结论

| 指标 | 本轮前 | 本轮后 | 说明 |
|---|---|---|---|
| 孤儿导出类型 | 5 个（4 个在清单内 + `HardwareAvailability` 经 Lead 追加授权） | **全部删除**（-35 行，2 文件） | `StringCodec`、`ContainerLogs`、`LogEntry`、`LocationAvailability` + `HardwareAvailability`（补充删除见 §7） |
| `deadcode -test ./... \| wc -l` | 53 | **53（与 53 行清单逐字相同）** | 口径：deadcode 只报函数、不报类型，故数字不变 |
| `golangci-lint unused` | 1（`logKeyCol`） | **1（`logKeyCol`）** | 口径：staticcheck `unused` 默认不报导出标识符 |
| 门禁 | 绿 | **绿**（build/vet/全仓测试/relaykit/gofmt） | 见 §3、§7 |

另有 1 个**新暴露**的孤儿类型 `pkg/ionet.HardwareAvailability`：它在本任务之前就已不可达（唯一引用来自同样是死类型的 `LocationAvailability`），初版未删；经 Lead 追加授权后已补删（§4 → §7）。

## 1. 四类核对（含“deadcode 不报类型”的替代口径）

### 1.1 ① 的替代取证方式

`deadcode` 只输出 `unreachable func: <Name>`，**不会**把类型列为不可达，因此类型对象无法直接沿用①。本任务采用三条等价证据链：

1. **引用图全零**：全仓 Go（含 `_test.go` 与 build tag 文件）对类型的 `-w` 精确匹配只剩定义处（见 §1.3）；
2. **类型级探针全零**：`var _ X`、接口断言 `.(X)`、复合字面量 `X{`、切片 `[]X`、指针 `*X`、映射 `map[..]X` 全部 0 命中；
3. **编译与全量测试**：删除后 `go build ./...` 与 `go test ./... -count=1` 全绿（§3），证明没有隐式引用。

```console
# 探针命令（四类型全部 0 命中；唯一命中是 ContainerLogs 自身的 Logs []LogEntry 字段，两者同批删除）
grep -rn --include='*.go' -E "var _ [^=]*\bX\b|\.\(X\)|\bX\{|\[\]X|\*X\b|map\[[^]]*\]X" .
```

### 1.2 逐类型取证表

| # | 类型（删前位置） | ① 类型可达性（探针/引用图） | ② 全仓 Go 引用（含测试与 build tag） | ③ 非 Go 位置引用 | ④ 序列化/反射/GORM 依赖 | 处置 |
|---|---|---|---|---|---|---|
| 1 | `pkg/cachex.StringCodec`（`pkg/cachex/codec.go:29`） | `deadcode` 不报类型；探针 0；引用图仅定义 | HEAD 全仓 1 处（定义）。上一轮删掉 `Encode`/`Decode` 后已无用者 | 0（README/MAINTENANCE/AGENTS.md/CLAUDE.md/.trellis/spec/scripts/.github/web 均无） | 无 codec 名字注册表（codec 均为静态实例化：`IntCodec`/`JSONCodec`）；无 `json.Marshal` 目标、无 GORM、无 `%T`/reflect | **已删** |
| 2 | `pkg/ionet.ContainerLogs`（`pkg/ionet/types.go:215-221`） | 探针 0；引用图仅注释+定义 | HEAD 全仓 2 处（注释 + 定义） | 0 | 无 `json.Marshal/Unmarshal` 调用以其为目标（0 引用）；`pkg/ionet` 不在 model 包、无 GORM 标签/`AutoMigrate`；无 reflect | **已删** |
| 3 | `pkg/ionet.LogEntry`（`pkg/ionet/types.go:223-229`） | 探针仅 1 处：`ContainerLogs.Logs []LogEntry` 字段（`types.go:218`），随 #2 同批删除后归零 | HEAD 全仓 3 处（字段 + 注释 + 定义） | 0 | 同上 | **已删**（与 #2 同批） |
| 4 | `pkg/ionet.LocationAvailability`（`pkg/ionet/types.go:338-345`） | 探针 0；引用图仅注释+定义 | HEAD 全仓 2 处（注释 + 定义） | 0 | 同上 | **已删** |

补充：`LocationAvailability` 有一个**反向**依赖（它的字段引用 `HardwareAvailability`），不影响自身可删性，但会连带暴露下一个孤儿（§4 → 已补删，§7）。

### 1.3 删后残留核对

```console
$ for t in StringCodec ContainerLogs LogEntry LocationAvailability HardwareAvailability; do
    echo "$t: $(grep -rn --include='*.go' -w "$t" . | wc -l)"; done
StringCodec: 0
ContainerLogs: 0
LogEntry: 0
LocationAvailability: 0
HardwareAvailability: 0
```

### 1.4 ③ 与 ④ 的细化说明

- **③ 非 Go 位置**：四类型在契约性位置（README / MAINTENANCE / AGENTS.md / CLAUDE.md / `.trellis/spec/**` / `scripts/` / `.github/` / `web/`）全部 0 命中。唯一命中是 `.trellis/workspace/new-api-own/journal-3.md:93` 的“待办记忆”（`可选收尾：4 个孤儿导出类型…`），属会话记忆而非契约，且本任务正是执行该待办——删除后该待办即闭环。
- **④ 序列化/反射**：
  - `json.Marshal/Unmarshal`：四类型在删前就是 0 引用，不存在以它们为目标的编解码调用；其 `json` tag 只是 DTO 约定，没有调用点。`decodeData`/`decodeWithFlexibleTimes` 的目标类型是仍在用的 `ContainerList`、`Container`、`LocationsResponse` 等，未受影响。
  - GORM：`pkg/ionet` 无 `gorm` tag、无 `AutoMigrate` 涉及（`grep -rn AutoMigrate | grep -iE 'ionet|ContainerLogs|LogEntry|LocationAvailability|StringCodec'` = 0）；这些类型是 HTTP 响应 DTO，不是数据模型。
  - `%T` / `reflect`：`pkg/ionet` 与 `pkg/cachex` 内无 reflect 使用；全仓唯一的 `%T` 在 `pkg/billingexpr/run.go:102`，与本次对象无关。
  - `//go:linkname`：全仓 0 命中。

## 2. 分批删除记录

| 批次 | 删除内容 | 行数 | `go build ./...` | `go test ./... -count=1` |
|---|---|---|---|---|
| B1 | `pkg/cachex.StringCodec` | -2 | exit=0 | exit=0（36 包 ok / FAIL 0） |
| B2 | `pkg/ionet.ContainerLogs` + `LogEntry`（必须同批：前者字段引用后者） | -16 | exit=0 | exit=0（36 包 ok / FAIL 0） |
| B3 | `pkg/ionet.LocationAvailability` | -9 | exit=0 | exit=0（36 包 ok / FAIL 0） |

全程零失败，**未发生回退**。B1-B3 时点的 diff 为 -27；补删第 5 个类型（§7）后最终 diff：

```console
$ git diff --numstat -- pkg
0	2	pkg/cachex/codec.go
0	33	pkg/ionet/types.go
```

## 3. 最终门禁（本轮实测，`GOPROXY=off`）

```console
### go build ./...
exit=0
### go vet ./...
exit=0
### go test ./... -count=1
...（36 个包 ok，FAIL 行 0）
exit=0
### relaykit GOWORK=off go build ./...
exit=0
### relaykit GOWORK=off go test ./... -count=1
exit=0
### gofmt -l .
(无输出)
exit=0
### golangci-lint run --default=none --enable=unused ./...
model/main.go:27:5: var logKeyCol is unused (unused)
1 issues:
* unused: 1
### deadcode -test ./... | wc -l
53
```

`deadcode` 的 53 行清单与本轮前**逐字相同**：

```console
$ diff <(sort /tmp/deadcode-after-r1.txt) <(sort /tmp/deadcode-after-orphans.txt)
(无输出)
```

口径说明（写进证据以便后人核对）：
- 删除的对象都是**类型**（共 5 个，含 §7 的补删项），`deadcode` 只报 `unreachable func`，因此数字必须保持 53；若它变化反而说明删错了对象。
- `unused`（staticcheck U1000 语义）默认只报**未导出**标识符；这些都是导出类型，因此删除前后都恰好 1（`logKeyCol`）。

## 4. 新发现：第 5 个孤儿类型 `pkg/ionet.HardwareAvailability`（已获追加授权并补删，见 §7）

```console
$ grep -rn --include='*.go' -w HardwareAvailability .
./pkg/ionet/types.go:322:// HardwareAvailability represents availability for specific hardware at a location
./pkg/ionet/types.go:323:type HardwareAvailability struct {
```

- **性质**：删前它唯一的引用来自 `LocationAvailability.HardwareAvailability []HardwareAvailability` 字段（`types.go:327`）；而 `LocationAvailability` 本身早已不可达（上个任务删掉了它唯一的使用者 `GetLocationAvailability`）。因此 `HardwareAvailability` **在本任务之前就已不可达**，只是 deadcode 不报类型、上一轮盘点也没识别。
- **处置**：task-57 的对象清单明确是 4 个类型；按“用户指定了目标就只做那一个、不自行扩大范围”的约束，本任务初版**未删**。随后 Lead 依据用户本轮“继续清残留”的指示给予**追加授权**，已补删（记录见 §7）。

## 5. 回滚方式

纯删除、无逻辑改动；逐批或整体回滚（本任务只碰 2 个文件）：

```bash
# B1
git checkout -- pkg/cachex/codec.go
# B2 + B3
git checkout -- pkg/ionet/types.go
# 或整体
git checkout -- pkg/cachex pkg/ionet
```

回滚后重跑 §3 门禁。注意不要用 `git checkout -- pkg` 之外的通配回滚，以免影响其它成员在 `web/` 的 locale 改动。

## 6. 边界自证

```console
$ git status --short
 M pkg/cachex/codec.go                              # 本任务
 M pkg/ionet/types.go                               # 本任务
 M web/src/i18n/locales/{en,zh,zh-TW,fr,ru,ja,vi}.json   # worker-docs 的 locale 任务，本任务未碰
?? .trellis/tasks/10-24-orphan-types-and-relaykit-policy/  # 任务目录（本证据文件）
```

- 本任务只写 `pkg/cachex/codec.go`、`pkg/ionet/types.go` 与本证据文件；未触碰 `web/` 源码、`relaykit/`、`data/`、实例；无 `go.mod`/`go.sum` 改动；无临时文件残留（探针均为只读 grep）。

## 7. 补充：第 5 个孤儿类型 `pkg/ionet.HardwareAvailability`（Lead 追加授权，已删）

- **授权**：Lead 依用户本轮“继续清残留”的指示明确同意补删（同文件、同风险；四类核对已在 §4 预核为全 0 命中）。
- **删前状态**（补删前）：全仓 Go 仅 2 处自引用（注释 `types.go:322` + 定义 `:323-328`）；类型级探针 0；非 Go 契约位置 0；无序列化/反射/GORM 依赖（口径同 §1.4）。
- **删除**：`pkg/ionet/types.go` 末尾删除 7 行（注释 + 结构体）并连带 1 个前置空行，diff 计 -8。
- **删后残留**：`grep -rn --include='*.go' -w HardwareAvailability .` = **0**。

补删后门禁（GOPROXY=off，实测）：

```console
$ go build ./...                                exit=0
$ go vet ./...                                  exit=0
$ go test ./... -count=1                        exit=0（36 包 ok / FAIL 0）
$ (cd relaykit && GOWORK=off go build ./...)    exit=0
$ gofmt -l .                                    (无输出)
$ golangci-lint run --default=none --enable=unused ./...
model/main.go:27:5: var logKeyCol is unused (unused)
1 issues:
* unused: 1
$ /root/go/bin/deadcode -test ./... | wc -l
53
```

`deadcode` 补删后的 53 行清单与补删前**逐字相同**（`diff` 空）——符合“类型不参与 deadcode 统计”的口径。

- **最终累计 diff（本任务 5 个类型）**：

```console
$ git diff --numstat -- pkg
0	2	pkg/cachex/codec.go
0	33	pkg/ionet/types.go
```

- **回滚**：`git checkout -- pkg/ionet/types.go`（或整体 `git checkout -- pkg/cachex pkg/ionet`）。本任务至此共删 **5 个孤儿导出类型、-35 行**，无插入、无逻辑改动。

渲染安全自查：正文无成对美元符号（美元符号只出现在 console 代码块的命令提示符中）；不使用单星号斜体；各表格列数一致；反引号代码片段已配对。
