# 技术债清扫 · 后端/脚本/仓库（B1–B4、C2、C3）

- 任务：Trellis `10-24-tech-debt-sweep`；共享任务板 `task-41`
- 执行者：worker-script；时间：2026-10-24
- 范围：**后端 Go / scripts / 仓库一致性**。前端 `web/`（W1–W3）与 C1 版本引用一致性不属本任务。
- 未提交、未推送、未发版；未动 `data/`、未动实例。

## B1 格式 / vet / build（有问题，已修）

### 修复前：`gofmt -l .` 有 9 个文件

```console
$ gofmt -l .
relay/channel/cloudflare/relay_cloudflare.go
relay/channel/jimeng/adaptor.go
relay/channel/jimeng/image.go
relay/channel/jimeng/sign.go
relay/channel/minimax/adaptor.go
relay/channel/minimax/tts.go
relay/channel/vertex/service_account.go
relay/channel/volcengine/adaptor.go
relay/channel/volcengine/tts.go
```

性质：**全部是 import 分组内的 1 行位置调整**（`"github.com/QuantumNous/new-api/common"` 未按字典序排列），
无任何语义变化。全量 diff 统计可复核：

```console
$ gofmt -d $(gofmt -l .) | grep -E '^[+-]' | grep -vE '^(\+\+\+|---)' | sort | uniq -c
      9 -	"github.com/QuantumNous/new-api/common"
      9 +	"github.com/QuantumNous/new-api/common"
$ git diff --stat -- relay/
 9 files changed, 9 insertions(+), 9 deletions(-)
```

### 修复后

```console
$ gofmt -w $(gofmt -l .) && gofmt -l .
（无输出，0 行）

$ go vet ./...            # exit 0（无输出）
$ go build ./...          # exit 0（无输出）
$ cd relaykit && GOWORK=off go build ./...   # exit 0（无输出，独立模块隔离构建通过）
$ go test ./... -count=1
exit=0  0 FAIL  36 ok
```

> **写作用域说明（需 Lead 知悉）**：task-41 的 advisory writeScopes 只写了 `scripts`，
> 但任务正文明确要求「格式类问题直接修」，故对上述 9 个 Go 文件做了 `gofmt -w`（纯格式、9 增 9 删）。
> 回滚：`git checkout -- relay/channel/{cloudflare,jimeng,minimax,vertex,volcengine}`。若 Lead 认为
> 本轮不应动 Go 文件，这 9 个文件可单独 revert，不影响其余结论。

## B2 TODO / FIXME / XXX / HACK 清单（**未改**）

`grep -rn "TODO\|FIXME\|XXX\|HACK" --include=*.go .`（排除 `web/node_modules`）共 **120 处**。

**归属证据（关键结论）**：对 120 处逐行 `git blame`，**全部 120 处都来自初始基线提交
`1092e46`（2026-08-01，"chore: 自维护基线（基于 2026-08-01 代码基）"）**：

```console
$ grep -rn ... | while read f l rest; do git blame -L "$l,$l" --porcelain -- "$f" | head -1 | awk '{print $1}' | tr -d '^'; done | sort | uniq -c
    120 1092e46a5af8102e6097482fe5d20673bf48acd3
```

即：**本项目自维护期间没有新增任何 TODO/FIXME**，120 处全是继承的基线代码债。

### B2-1 模板化占位（103 处 / 26 文件）— 建议「留」

全部是 `//TODO implement me`，集中在 `relay/channel/*/adaptor.go`，是渠道适配器实现大接口时
对「本渠道用不到的方法」的占位。按文件计数（前 10）：

| 文件 | 处数 |
| --- | --- |
| `relay/channel/{zhipu,xunfei,tencent,palm,mokaai,mistral,dify,cohere}/adaptor.go` | 各 6 |
| `relay/channel/{jina,claude,baidu_v2,baidu,aws}/adaptor.go` | 各 5 |
| `relay/channel/{perplexity,deepseek}/adaptor.go` | 各 4 |
| 其余 11 个文件 | 1–3 |

建议：**留**。删掉这些注释不减少任何技术债，反而丢失"该渠道未实现此能力"的信号；
若要降噪，可考虑后续统一改为 `// not supported for this channel` 之类语义化注释（需另开任务）。

### B2-2 有实际语义的 17 处 — 建议见「处置」列

| 文件:行 | 摘要 | 建议处置 |
| --- | --- | --- |
| `service/billing_session.go:216` | 模型层应定义哨兵错误，用 `errors.Is` 替代字符串匹配 | **修**（可观测性/健壮性，改动中等） |
| `model/main.go:266` | `ALTER TABLE channels ...` 注释行「等多数用户升级后删除」 | **删**（已无升级价值） |
| `controller/channel-billing.go:465` | 余额查询「support Azure」 | 留（功能缺口，需产品决策） |
| `controller/channel-billing.go:484` | 「make it async」 | 留（性能优化，需评估） |
| `middleware/distributor.go:486` | `api_version` 统一 | 留（协议层改动，影响面大） |
| `relay/claude_handler.go:99` | 「临时处理」无说明 | **修/补注**（最该补说明的一处） |
| `service/sensitive.go:20` | 「check image url」敏感内容检测未覆盖图片 | 留（功能缺口） |
| `common/gin.go:144` | 非 JSON 请求的变体模型未实现 | 留 |
| `relaykit/dto/audio.go:28` | 流式开始后逻辑正确性待确认 | 留（需实测） |
| `relaykit/dto/gemini.go:166` | 与 thinkingbudget 冲突 | 留 |
| `relay/channel/cohere/adaptor.go:85` | `cohereStreamHandler ... // TODO: fix this` | **修**（无说明的可疑点，建议先补原因） |
| `relay/channel/task/gemini/dto.go:14,15` | 支持 referenceImages / lastFrame | 留（功能缺口） |
| `relay/channel/task/gemini/image.go:56` | 支持 HTTP URL 图片转 base64 | 留（功能缺口） |
| `relay/channel/coze/relay-coze.go:31` | 支持更多 content type | 留（功能缺口） |

### B2-3 误报 1 处

`common/totp.go:77`：`// 格式化为 XXXX-XXXX 格式` —— `XXXX` 是格式占位描述，**不是** TODO 标记。

## B3 死代码候选（**未删**，只给证据与影响面）

### 工具结论（可信基线）

本机有 `golangci-lint 2.12.2`；其 `unused` linter（staticcheck U1000）结果：

```console
$ golangci-lint run --no-config --default=none --enable=unused ./...
39 issues:
* unused: 39
```

**39 个全部是未导出（小写）符号，导出符号 0 个** —— 这是该 linter 的已知边界：导出符号被视为
公共 API，不做"仅本模块内未使用"的判定。因此「导出的死代码」只能靠启发式 + 人工确认。

39 个未导出死符号清单（可直接据此另开清理任务，风险低）：

| 包 | 符号 |
| --- | --- |
| common | `fieldReplacer`, `getEnvOrDefault` |
| controller | `updateChannelCloseAIBalance`, `updateChannelOpenAISBBalance`, `upstreamModelsURL`, `upstreamVendorsURL`, `isStripeWebhookConfigured`, `isWaffoPancakeWebhookConfigured`, `generateDefaultSidebarConfig` |
| middleware | `abortWithMidjourneyMessage` |
| model | `commonTrueVal`, `commonFalseVal`, `logKeyCol`, `createRootAccountIfNeed`, `migrateDBFast`, `cacheSetTokenField`, `updateUserUsedQuota`, `updateUserRequestCount`, `getUserStatusCache`, `updateUserStatusCache`, `updateUserEmailCache` |
| pkg/perf_metrics | `redisCounters`, `parseRedisInt`, `mergeRedisActiveBuckets` |
| relay | `syncModels`, `stopReasonClaude2OpenAI`, `cacheCreationTokensForOpenAIUsage`, `getPayloadHash`, `getContentTypeByFormat`, `handleChatCompletionResponse`, `contains`, `containsInt`, `detectImageMimeType`, `tokenData`, `zhipuImageRequest`, `trimModelThinking`, `taskChangeParams` |
| service | `shouldAutoRefreshCodexChannelStatus`, `newProxyHTTPClient` |

> 注意：`model/main.go` 的 `commonTrueVal/commonFalseVal/logKeyCol` 属于跨库方言辅助变量，
> 删除前需确认不是"留作其它数据库分支备用"（README/AGENTS 提到三库兼容），建议低优先。

### 导出符号候选（启发式：全仓按词边界恰好只出现 1 次 = 仅定义处）

方法：收集非 `_test.go` 的导出 `func/method/type/var/const` 声明，统计其标识符在全仓 `.go`
（含测试）出现次数，`==1` 即「无任何调用点」。结果：**105 个候选**，其中应用层包
（controller/service/model/middleware/setting）**48 个**。

应用层候选（完整 48 条）：

| 文件:行 | 符号 | 说明 |
| --- | --- | --- |
| controller/image.go:7 | `GetImage` | 无调用、未注册路由 |
| controller/token.go:194 | `GetTokenStatus` | 无调用 |
| middleware/auth.go:264 | `WssAuth` | 无调用 |
| middleware/rate-limit.go:181/185 | `DownloadRateLimit` / `UploadRateLimit` | 无调用 |
| middleware/recover.go:12 | `RelayPanicRecover` | 无调用 |
| model/ability.go:57 | `GetAllEnableAbilities` | 无调用 |
| model/channel.go:350/892/902/1125/1136/1143 | `SetTag` / `DeleteChannelByStatus` / `GetPaginatedTags` / `GetChannelsByType` / `CountChannelsByType` / `CountChannelsGroupByType` | 无调用 |
| model/channel_satisfy.go:33 | `IsChannelEnabledForAnyGroupModel` | 无调用 |
| model/midjourney.go:147/157 | `GetMjByuId` / `UpdateProgress` | 无调用 |
| model/model_meta.go:107 | `GetAllModels` | 无调用 |
| model/perf_metric.go:81/125 | `GetPerfMetricsSummaryAll` / `PerfMetricStartTime` | 无调用 |
| model/system_task.go:146 | `FindPendingSystemTasks` | 无调用 |
| model/task.go:72/445 | `SetData`(method) / `TaskQuotaUsage`(type) | 无调用 |
| model/token.go:362/382 | `IsModelLimitsEnabled` / `DisableModelLimits` | 无调用 |
| model/topup.go:62 | `GetTopUpById` | 无调用 |
| model/user.go:131/969/1068/1161/1282 | `GetAccessToken` / `FillUserByEmail` / `IsTelegramIdAlreadyTaken` / `GetUserEmail` / `DeltaUpdateUserQuota` | 无调用 |
| model/user_session.go:65 | `AfterFind`(method) | **误报**：GORM 钩子 |
| service/audio.go:34 | `DecodeBase64AudioData` | 无调用 |
| service/auth_session.go:426 | `FormatAuthError` | 无调用 |
| service/authz/adapter.go:34/69 | `SavePolicy` / `RemovePolicy` | **误报**：casbin `persist.Adapter` 接口实现 |
| service/codex_oauth.go:29 | `RefreshCodexOAuthToken` | 无调用 |
| service/error.go:81 | `ClaudeErrorWrapperLocal` | 无调用 |
| service/image.go:119 | `DecodeUrlImageData` | 无调用 |
| service/passkey/user.go:22/40/51 | `WebAuthnID`/`WebAuthnDisplayName`/`WebAuthnCredentials` | **误报**：webauthn 库 `User` 接口实现 |
| service/request_converter.go:38/50 | `ClaudeToOpenAIRequest` / `GeminiToOpenAIRequest` | 无调用 |
| service/sensitive.go:11 | `CheckSensitiveMessages` | 无调用 |
| service/str.go:51 | `RemoveDuplicate` | 无调用 |
| setting/auto_group.go:25 | `ContainsAutoGroup` | 无调用 |
| setting/ratio_setting/model_ratio.go:75/556 | `GetModelPriceMap` / `GetModelRatioOrPrice` | 无调用 |

逐条 grep 取证样例（出现次数 = 1，仅定义处）：

```console
$ grep -rn "\bGetImage\b" --include=*.go .            → controller/image.go:7（仅定义）
$ grep -rn "\bWssAuth\b" --include=*.go .             → middleware/auth.go:264（仅定义）
$ grep -rn "\bDeleteChannelByStatus\b" --include=*.go . → model/channel.go:892（仅定义）
$ grep -rn "\bRemoveDuplicate\b" --include=*.go .     → service/str.go:51（仅定义）
$ grep -rn "\bContainsAutoGroup\b" --include=*.go .   → setting/auto_group.go:25（仅定义）
```

界面/路由可导出但实际未被引用的典型：`controller.GetImage`、`middleware.WssAuth` ——
确认未在 `router/` 注册（若注册过，计数会 ≥2）。

**误报形态（必须在删除前排除）**：
1. **接口/框架钩子**：如 `AfterFind`（GORM）、`SavePolicy/RemovePolicy`（casbin adapter）、
   `WebAuthn*`（webauthn.User）——声明 1 次但由库回调；
2. **反射/JSON 标签/模板**使用；
3. **为外部消费者保留的公共 API**（`common/`、`types/`、`dto/`、`relaykit/` 属此类，57 个候选多为此）。

**建议**：不删。若要真删，先装全程序分析工具（`deadcode` / `staticcheck` 的 whole-program 模式）——
本机未安装，安装需联网，须 Lead 单独授权。低风险且已确认无接口回调的少数（如
`controller.GetImage`、`middleware.WssAuth`、`service.RemoveDuplicate`、`setting.ContainsAutoGroup`）
可另开任务逐个确认后删除，单次 1–3 个、便于回滚。

## B4 脚本检查（1 个低风险项已修，1 个 note 进清单）

### `bash -n`（7 个脚本全过）

```console
$ for f in scripts/*.sh; do bash -n "$f"; done
OK scripts/check-trellis-gate.sh
OK scripts/check-version-drift.sh
OK scripts/forbid-encoding-json.sh
OK scripts/forbid-extra-deploy-methods.sh
OK scripts/install-git-hooks.sh
OK scripts/print-ed25519-pubkey.sh
OK scripts/trellis-wrapup.sh
```

### `shellcheck`（修复前 2 条 → 修复后 1 条 note）

修复前：

```console
scripts/check-trellis-gate.sh:18:1: warning: Use 'cd ... || exit' ... [SC2164]
scripts/trellis-wrapup.sh:93:20: note: Note that A && B || C is not if-then-else ... [SC2015]
```

**已修**（低风险，SC2164）——`scripts/check-trellis-gate.sh`：

```diff
 REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
-cd "${REPO_ROOT}"
+# cd 失败时后续相对路径都会错位：显式终止（exit 1 与"发现违规"同样让 CI 变红）。
+cd "${REPO_ROOT}" || { echo "无法进入仓库根目录: ${REPO_ROOT}" >&2; exit 1; }
```

验证：`bash -n` 通过；实际运行 `./scripts/check-trellis-gate.sh --audit-only` → 通过（exit 0，
输出「提交可追溯：8a991149..HEAD 共核对 85 个改动提交」）。

**未修（进清单）**：`scripts/trellis-wrapup.sh:93` SC2015，代码为

```bash
[ -n "${BRANCH}" ] && [ "${BRANCH}" != "HEAD" ] || refuse "当前处于 detached HEAD，..."
```

该 `A && B || C` 在本语境**语义正确**（A、B 均真时不触发 C；任一为假时触发 C），
属 note 级提示；且该脚本是收尾流程工具，改控制流收益低、风险不为零 → 只登记，不改。

## C2 残留文件（无问题，未改）

```console
$ find . -path ./web/node_modules -prune -o \( -name '*.orig' -o -name '*.rej' -o -name '*.bak' \
    -o -name '.DS_Store' -o -name '*~' -o -name '.#*' -o -name '#*#' -o -name '*.swp' -o -name '*.tmp' \) -print
（无输出，命中 0）
$ git status --porcelain | grep '^??'      # 仅本任务目录，无散落文件
?? .trellis/tasks/10-24-tech-debt-sweep/
$ git ls-files data/ | wc -l              # data/ 未被跟踪（.gitignore 生效）
0
```

结论：仓库无 `*.orig`/`*.rej`/`*.bak`/`.DS_Store`/交换文件等残留；根目录文件清单正常
（无临时调试脚本）。`web/scripts/check-money-format.mjs` 是既有正式脚本（非临时文件）。

## C3 部署形态红线（通过）

```console
$ sh scripts/forbid-extra-deploy-methods.sh
✅ 部署方式唯一：docker run + ghcr.io/zhemed/new-api-own
[exit=0]
```

## 影响面 / 回滚

| 改动 | 文件数 | 性质 | 回滚 |
| --- | --- | --- | --- |
| B1 格式 | 9（`relay/channel/*`） | 纯 import 重排，9 增 9 删，零语义变化 | `git checkout -- relay/channel/` |
| B4 脚本 | 1（`scripts/check-trellis-gate.sh`） | 加 `cd ... || exit` 守卫 | `git checkout -- scripts/check-trellis-gate.sh` |

B2/B3/C2 全部**只出清单未改**。`go build/vet/test`、relaykit 独立构建、shellcheck、部署红线均复跑通过。

## 建议下一步（给 Lead 的取舍）

1. **B2**：103 处 `//TODO implement me` 属"结构性占位"，建议不清理或统一改写注释（另开小任务）；
   17 处有语义的最值得先处理 `relay/claude_handler.go:99`（补原因）与 `model/main.go:266`（删死注释）。
2. **B3**：`unused` 已给出 39 个未导出死符号，风险低，可单开任务分批删（每批跑全测试）。
   导出符号的 48 个候选需先确认接口/反射，建议逐个来，不要批量删。
3. **B4**：SC2015 一条可直接忽略；若要 stricter 门禁，可把 `shellcheck -S warning scripts/*.sh` 接入 CI。
4. 若 Lead 认为 Go 文件不在本轮写作用域，B1 的 9 个文件可单独 revert（结论不受影响）。
