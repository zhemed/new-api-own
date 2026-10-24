# 死代码删除 + 文档版本号 + CI 检查：独立验证证据（verifier）

- 任务：共享任务板 `task-46`（Trellis 任务 `10-24-deadcode-docs-ci` 的独立验证）
- 验证者：verifier（只读验证；除下述已披露并已还原的 go.mod 意外，未改仓库文件）
- 验证时间：2026-10-24（UTC 05:39 起）
- 基线：`HEAD = 0a91163`；最终 `git status --porcelain` 行数 = **31**（28 个已跟踪改动 + 3 个未跟踪条目），与任务开始一致
- 最终 diff 指纹：`git diff | sha256sum = 9e5ba7acff7914fc5f787599b560afc7472d81d4879924a72b27457bcb4ae72c`
- `go.mod` / `go.sum` 干净：`go.mod` blob = `b0642f162db2b5f5c1649d69d46b52956265619d` = `HEAD:go.mod`
- 新增未跟踪文件：`.github/workflows/lint.yml` = `cacf85b77119e2e7c697d9662e97ab7a2a1aaece`、
  `model/twofa_test.go` = `cd2295ebd8ce2e80045a6cf2f72e418aff13f2a5`

## 结论

**接受（Accept）**。死代码删除经**全量**（不是抽样）独立核对：36 个被删符号在 HEAD 与工作区的所有出现点都能被删除行或已知同
名不同物解释，**无任何功能性残留引用**；25 个纯删除文件共 487 行删除、0 行新增，与实现者报告一致；
build / vet / relaykit 独立构建 / 全仓测试 / gofmt 全绿；`unused` 精确复现 **1 issue（logKeyCol）** 且经三方口径复核
「无处可用、保留有 spec 依据」；twofa 三行替换语义等价且新测试真跑真断言；文档 22 行改动全部为示例版本号、功能门槛与历史记录
逐字节未动；CI YAML 合法、命令与本地等价、既有三个 workflow 未改、违规复现非零退出；边界 31 行与基线一致。
另有 2 条非阻断观察（见第九节）与 1 条验证事故披露（第八节，已由 Lead 还原）。

## 一、删除安全（最重要）

### 1.1 删除规模：独立提取，与报告一致

- 从 `git diff -U0 -- '*.go'` 自动解析删除行，独立提取出 **36 个唯一未导出符号**（全为包级函数/变量/类型，**无方法**）；
- 其中 **25 个文件为纯删除**：合计 **487 行删除、0 行新增**（与实现者报告「25 个文件、487 行删除、0 行新增」完全一致）；
- `model/twofa.go` 单独 `3 增 3 删`（见第四节）；文档 `22 增 22 删`（见第五节）。
- 提取脚本同时得到每个符号在 HEAD 的定义行与所在文件的删除行区间，用于下述引用过滤。

### 1.2 逐符号引用核对（36/36 全量，非抽样）

方法：把 `git archive HEAD` 解出的 HEAD 全树逐文件扫描，对每个符号做标识符边界匹配；命中若落在该文件的「删除行」区间内即视为
随本次删除一并消失，剩下的才算「残留引用」。结果：

- **36 个符号在 HEAD 的 Go 代码中，落在删除行之外的引用 = 0**；
- 工作区 Go 代码全量扫描：31 个符号 **0 命中**，仅 5 个有命中，全部可解释（见 1.3）；
- 非 Go 功能位（`scripts/`、`.github/`、`web/src/`、`web/scripts/`、`makefile`、`Dockerfile`）对 36 个符号的命中 = **0**；
  唯一非 Go 命中是历史归档任务里的说明文档（`archive/2026-10/10-24-tech-debt-sweep/evidence-backend.md`，只是把符号名列为候选清单，
  不构成引用）。

### 1.3 工作区 5 处残留命中逐条解释

| 符号 | 命中 | 性质 | 判定 |
|---|---|---|---|
| contains | 25 处（service/rankings.go 注释、relay/common/override.go 的 JSON 值等） | 通用英文单词/别包同名概念，与 hailuo 包内被删函数无关（该包内已无调用） | 无关 |
| detectImageMimeType | 6 处 | 2 处是 volcengine 内**注释掉**的旧代码；4 处是 `relay/channel/openai/adaptor.go` **自己的**同名函数（:589 定义、:521/:550 调用） | 别包同名 |
| tokenData | 3 处 | `relay/channel/zhipu/relay-zhipu.go` 的**局部变量名**；被删的是 `zhipu_4v` 包内的类型 | 无关 |
| cacheCreationTokensForOpenAIUsage | 2 处 | `relaykit/`（独立模块 `github.com/QuantumNous/new-api/relaykit`）自己的同名函数：定义 + 调用；`internal` 路径且跨模块，root 模块不可能引用它 | 别模块同名 |
| updateChannelOpenAISBBalance | 1 处 | `controller/channel-billing.go:335` 的**注释行**（`// return updateChannelOpenAISBBalance(channel)`），无编译影响 | 死注释（见观察 2） |

即：没有任何仍然生效的代码引用被删符号；`go build` / `go vet` / `go test` 全绿是第二重佐证（被删符号若仍被默认构建或测试引用会直接编译失败）。

### 1.4 构建标签与「非 Go 功能位」

- 仓库仅有 2 个带 `//go:build` 的文件（`!windows` / `windows` 各 1 个），已包含在上述 HEAD 全树扫描中，无对被删符号的引用；
- 跨 OS 编译尝试：`GOOS=windows` / `GOOS=darwin go build ./...` 因本机模块缓存缺少对应平台依赖（`GOPROXY=off` 禁止下载）而无法完成，
  属**离线环境限制**、非代码问题；标签文件已被静态全量扫描覆盖。

### 1.5 抽样 10 个符号独立取证（全部 0 引用）

| 符号 | 位置 | HEAD 残留引用（删除行之外） | 工作区引用 |
|---|---|---|---|
| fieldReplacer | common/custom-event.go:42 | 0 | 0 |
| getEnvOrDefault | common/totp.go:120 | 0 | 0 |
| updateChannelCloseAIBalance | controller/channel-billing.go:168 | 0 | 0 |
| isStripeWebhookConfigured | controller/payment_webhook_availability.go:23 | 0 | 0 |
| createRootAccountIfNeed | model/main.go:57 | 0 | 0 |
| migrateDBFast | model/main.go:393 | 0 | 0 |
| cacheSetTokenField | model/token_cache.go:43 | 0 | 0 |
| updateUserEmailCache | model/user_cache.go:256 | 0 | 0 |
| stopReasonClaude2OpenAI | relay/channel/claude/relay-claude.go:22 | 0 | 0 |
| newProxyHTTPClient | service/http_client.go:354 | 0 | 0 |

## 二、删除完整性 / 门禁复跑（真实工作区，Go 全部 `GOPROXY=off`）

| 门禁 | 命令 | 结果 |
|---|---|---|
| Go 构建 | `go build ./...` | rc=0 |
| Go 静态检查 | `go vet ./...` | rc=0 |
| relaykit 独立构建 | `cd relaykit && GOWORK=off go build ./...` | rc=0 |
| 全仓测试 | `go test ./... -count=1` | rc=0，无 FAIL 行 |
| 格式 | `gofmt -l .` | 输出为空 |
| twofa 重复性 | `go test ./model/ -run TestTwoFA -count=3` | ok |

## 三、`unused` 与 `logKeyCol` 复核

- 独立复跑：`golangci-lint run --default=none --enable=unused ./...`（v2.12.2，`GOFLAGS=-mod=readonly`）
  → 输出 **恰好 1 issue**：`model/main.go:27:5: var logKeyCol is unused (unused)`，与任务预期一致。
- 「无处可用」三方口径独立复核：
  1. `Log` 结构体（model/log.go:59-81）**无 `key` 字段**（只有 Id/UserId/CreatedAt/Type/Content/.../Other）；
  2. logs 表列：GORM 由结构体推导无 key 列；ClickHouse 建表 DDL（model/main.go:409-433）**18 列无 key**；
  3. 全仓 raw SQL 中 logs 与 key 同现 = **0**；`logKeyCol` 自身出现 4 处 = 定义 + 2 处方言赋值 + 1 处 twofa 测试注释，
     **没有任何读取点**。
- 保留是否有依据：`.trellis/spec/backend/database-guidelines.md:28-30` 明确把 `logGroupCol` / `logKeyCol` 列为日志库保留字列适配机制，
  删除会造成 spec 腐化；且该变量在 HEAD 时已是 write-only（本次未引入）。**保留合理，1 issue 属预期状态。**
- `commonTrueVal` / `commonFalseVal` 已因 twofa 改动变为「被读取」，不再出现在 unused 名单（与 twofa_test.go 注释一致）。

## 四、twofa 语义等价

改动仅 `model/twofa.go` 三行（`3 增 3 删`）：

| 原 SQL 片段 | 新 SQL 片段 |
|---|---|
| `"user_id = ? AND is_used = false"`（2 处） | `"user_id = ? AND is_used = " + commonFalseVal` |
| `"is_enabled = true"` | `"is_enabled = " + commonTrueVal` |

| 核对项 | 结论 |
|---|---|
| 常量定义 | `model/main.go:24-25` 声明、`initCol()` 赋值：PostgreSQL 分支 `"true"/"false"`；其它（MySQL/SQLite）分支 `"1"/"0"` |
| 三方言语义等价 | PostgreSQL：拼出 `is_used = false` / `is_enabled = true`，与原字面量**逐字节相同**；MySQL/SQLite：`is_used = 0` / `is_enabled = 1`，对 TINYINT/INTEGER 布尔列与 `false`/`true` 比较等价（SQLite 自 3.23 起也识别 true/false，两者同为 0/1） |
| 查询结构/参数绑定 | 仅替换字符串字面量；`user_id = ?` 占位符与参数不变，无新增拼接用户输入（常量来自方言初始化，非外部输入） |
| `UsingMainDatabase` 分支 | 未改动：`initCol()` 与 `commonTrueVal/commonFalseVal` 定义均不在本次 diff 内（model/main.go 的两个 hunk 只在 :57 与 :393，删除的是死函数） |
| 新测试有效性 | `TestTwoFABooleanLiteralsPerDialect` 断言两分支常量取值 + 在 SQLite 上真执行拼接条件（含「真值不得等于假值」反例）；`TestTwoFABooleanQueriesBehavior` 端到端跑三处查询（未使用备用码列表/计数、已启用 2FA 计数），并断言「用过的备用码不得再次通过」以证明条件真的生效；`-count=3` 通过（测试内保存/恢复主库类型并重跑 initCol，无全局污染） |

## 五、文档：21/22 处示例版本号更新

- 实测改动：`README.md 9 增 9 删` + `MAINTENANCE.md 13 增 13 删` = **22 行 / 17 个 hunk**（实现者称「21 处」，属计数口径差异；
  行数与 `+22/-22` 一致）。
- 逐 hunk 结构核对：把版本号替换为占位符后，**20 行结构完全相同**；另 2 行仍只是版本相关变化
  （`grep 0.0.5 会匹配 …09035…` → `…09048…`；`VERSION 递增（0.0.3、0.0.4 …）` → `（0.0.48 → 0.0.49 …）`，
  后者是示例序列写法调整），**均属示例值，不涉及功能规则**。
- **功能门槛未被改动**（与 HEAD 逐字节比对 = 相同）：
  README 124-125 / 127-130 / 167（LOG_* 变量所属版本、面板自更新受影响版本 v0.0.43–v0.0.45）；
  MAINTENANCE 454（tag 发版必须性）、507-508（LOG_* 变量所属版本）。
- **历史记录未被改动**（逐字节相同）：README 135（2026-10-06 latest 停在 0.0.3）；MAINTENANCE 71（v0.0.2 GHCR 推送）、
  466-467（事故记录 0.0.5/0.0.6）、513（latest 停在 0.0.3/0.0.6）、543（2026-10-06 0.0.3）。
- 程序化核对：改动行集合与「不得改」行集合**交集为空**；指定行 HEAD 与工作区逐字节比对无差异。
- 示例值一致性：`VERSION` 文件当前为 `0.0.48`，文档示例已同步到 0.0.48 / 0.0.49（"下一个"）。

## 六、CI 检查

| 核对项 | 结果 |
|---|---|
| YAML 合法性 | `python3 -c yaml.safe_load` 解析成功；jobs = gofmt、shellcheck；triggers = push main / pull_request / workflow_dispatch（与 trellis-gate 一致）；`permissions: contents: read`；并发组取消旧任务 |
| gofmt job 命令 | `git ls-files -z '*.go'` 的输出交给 `xargs -0 -r gofmt -l`，输出非空即 `::error::` 加 `exit 1`（只查被 git 跟踪的 go 文件，明确纳入 relaykit、排除未跟踪临时文件） |
| shellcheck job 命令 | `shellcheck --version`；随后对 scripts 目录下全部 .sh 执行 `bash -n` 与 `shellcheck -S warning` |
| 与本地等价 | 在真实仓库按原命令复跑：gofmt 输出为空（813 个被跟踪 Go 文件全格式化）；bash 语法检查全过；shellcheck -S warning 全过（shellcheck 0.10.0） |
| 既有 workflow 未改 | `docker-build.yml` / `release.yml` / `trellis-gate.yml` 与 HEAD 一致（`git diff --quiet` 通过；新文件只有 lint.yml） |
| 违规复现（临时目录，已清理） | ① gofmt：加入未格式化文件 → 输出非空 → 按 job 逻辑 **exit 1**；合规对照组输出为空 → 通过。② `bash -n`：缺 `fi` 的脚本 → **rc=2**。③ shellcheck：含未引用位置参数的写法规避脚本 → **rc=1**。演示文件只写在 /tmp，已删除，真实仓库未受影响 |
| 互不依赖 | 注释明确分工：docker-build（tag 构建镜像）、release（tag 发布）、trellis-gate（提交锚点）、lint（格式/脚本卫生），未改动彼此 |

## 七、边界

- `git status --porcelain | wc -l = 31`：28 个已跟踪改动（26 个 Go 文件 + README + MAINTENANCE）+ 3 个未跟踪条目
  （`.github/workflows/lint.yml`、`model/twofa_test.go`、任务目录）；
- 无 `go.mod` / `go.sum` 改动（已复核 blob 与 HEAD 相同）；无其它意外已跟踪改动、无意外未跟踪文件（`-uall` 下仅上述新增 + 任务目录的 6 个文件）。

## 八、事故披露（验证过程中的意外写入，已还原）

- **事实**：我在跑 `golangci-lint run` 时误带 `GOFLAGS=-mod=mod`，golangci 在包加载时**自动改写了真实工作区的 `go.mod`**
  （mtime 05:41:46）：把 `github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.8 // indirect` 的 `// indirect` 去掉（1 行，`go.sum` 未变）。
- **归属与处置**：属我造成的意外，不是实现者改动。发现后立即停止使用任何带 `-mod=mod` 的命令、**未执行** checkout/stash/restore，
  第一时间上报 Lead；Lead 已在真实工作区执行 `git checkout -- go.mod` 还原，复核 `go.mod`/`go.sum` 干净。
- **复核**：还原后我用 `GOFLAGS=-mod=readonly` 重跑 golangci，结果仍为 **1 issue（logKeyCol）**，且 go.mod blob 运行前后均为
  `b0642f16…`（= HEAD），未再发生写入。最终 `git status` 31 行与 Lead 给的基线一致。
- **教训**：即便只读验证，`-mod=mod` 也会触发工作区写入；后续此类命令一律用默认 flags 或 `-mod=readonly`。

## 九、非阻断观察与不确定项

1. （报告同步，非代码问题）实现者证据 `.trellis/tasks/10-24-deadcode-docs-ci/evidence-deadcode.md` 第 17/92/124 行仍写
   「最终 unused **3 issues**（commonTrueVal/commonFalseVal/logKeyCol）」并请 Lead 裁决三选一；但最终状态已按方案 (b) 落地
   （twofa 读取 commonTrueVal/commonFalseVal），实际为 **1 issue（logKeyCol）**。建议归档前把该证据的结论行同步为 1 issue
   （logKeyCol 保留有 spec 依据），否则与 task-46 的验收口径不一致。
2. （死注释，非功能）删除后仍有两处注释引用被删符号：`controller/channel-billing.go:335`
   （`// return updateChannelOpenAISBBalance(channel)`）与 `relay/channel/volcengine/adaptor.go` 内两处
   `// mimeType := detectImageMimeType(...)`。无编译影响，可在后续清扫中顺手删。
3. （不确定项）跨 OS 构建（windows/darwin）未能在离线环境完成（模块缓存缺平台依赖）；已用 HEAD 全树静态扫描覆盖所有
   `//go:build` 文件作为替代证据。
4. （不确定项）twofa 的 SQL 等价性未在真实 MySQL/PostgreSQL 上执行（本机无实例）；结论基于「PostgreSQL 分支拼出与原字面量相同的
   true/false、MySQL/SQLite 分支为 0/1 且与布尔列比较等价」的代码事实 + 新测试的常量断言与 SQLite 实跑。

## 十、复跑命令清单

```bash
cd /root/new-api-own
# 删除安全：提取被删声明 + 引用过滤（脚本思路见本证据 §1）
git diff -U0 -- '*.go'            # 36 个未导出符号，25 个文件纯删除 487 行
git status --porcelain | wc -l    # 31
# 门禁
GOPROXY=off go build ./... && GOPROXY=off go vet ./...
(cd relaykit && GOWORK=off GOPROXY=off go build ./...)
GOPROXY=off go test ./... -count=1
gofmt -l .
# unused（只读 flags，避免写 go.mod）
GOPROXY=off GOFLAGS=-mod=readonly golangci-lint run --default=none --enable=unused ./...
# twofa
GOPROXY=off go test ./model/ -run TestTwoFA -count=3 -v
# 文档与 CI
git diff -U0 -- README.md MAINTENANCE.md
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/lint.yml'))"
git ls-files -z '*.go' | xargs -0 -r gofmt -l
find scripts -name '*.sh' -exec bash -n {} \;
find scripts -name '*.sh' -exec shellcheck -S warning {} \;
```
