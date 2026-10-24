# 技术债清扫：独立验证证据（verifier）

- 任务：共享任务板 `task-42`（Trellis 任务 `10-24-tech-debt-sweep` 的独立验证）
- 验证者：verifier（只读验证；未改任何仓库文件）
- 验证时间：2026-10-24（UTC 05:17 起）
- 基线：`HEAD = cd01c0d`；改动恰 17 个文件 = 9 relay + 1 脚本 + 7 locale（11 增 24 删）
- 基线指纹：`git diff | sha256sum = 5b56949c0a8cb41194e2357c27abfd4ee619f375c4ab9399221b21ebe90e8020`（验证后复算一致）
- 被验证文件 blob 哈希：

| 文件 | blob |
|---|---|
| relay/channel/cloudflare/relay_cloudflare.go | f834c3be1675219277bdd17feb66a0b644c2b0c2 |
| relay/channel/jimeng/adaptor.go | f0c4f3fb5b332f1a9593d589a4d923a1ea7d8769 |
| relay/channel/jimeng/image.go | be4a02c0d2bee11bebbafeaae0f128de5aab173c |
| relay/channel/jimeng/sign.go | a41e26fbf167941cbbbe88b8fff859e6c2e0294a |
| relay/channel/minimax/adaptor.go | 66f84e65e1b01e58aa1d46d513dbdf80a4f95f00 |
| relay/channel/minimax/tts.go | 8740fef91008a9c5239a447f6b43e3dd905b9c7a |
| relay/channel/vertex/service_account.go | e2cd3ab7d85255ec886d484c426c773f8237241e |
| relay/channel/volcengine/adaptor.go | 5f4ff57a020fe3f2ee36e5a7bdc20ad82c80e634 |
| relay/channel/volcengine/tts.go | aa76cc2b73d9a7cb315c13f5d76e33265f5ac23e |
| scripts/check-trellis-gate.sh | 6543d72ff97283c39db2f97e8239d735fb582e79 |
| web/src/i18n/locales/en.json | 2cfb656c526893e2ac702210d207759006f3c8f8 |
| web/src/i18n/locales/zh.json | 31d7b39a2bbdb67050c85e24e72beec44f34ce38 |
| web/src/i18n/locales/zh-TW.json | 7e9c950c56a4a8ac0ca68af450aca65436cf0365 |
| web/src/i18n/locales/fr.json | 1dd775d536bc130548aafef8d6b7760131312a55 |
| web/src/i18n/locales/ja.json | ec915850ea8a66aba22b8e6487f7448bf4d77de2 |
| web/src/i18n/locales/ru.json | ad37bf18c53ada0a0404814cded0696bdecc3a17 |
| web/src/i18n/locales/vi.json | 135ab1f3f6e996accf474c8defa0941b6a0235d4 |

## 结论

**接受（Accept）**。三类改动均为零运行时语义的低风险清扫，逐项核对全部成立：9 个 relay 文件每个只有 1 行 import
搬到 gofmt 规范位置（9 增 9 删完全相同内容）；两个死键在 HEAD 时已无任何代码引用、7 语言删净且键序与 HEAD 逐项一致
（无重排）；脚本守卫 1 个 hunk、语义正确；全部门禁（含全仓测试、relaykit 独立构建、前端 typecheck/i18n/oxlint）通过；
改动边界恰为 17 个文件，无意外残留。

## 一、relay 格式改动：逐文件确认零语义

| 核对项 | 证据 | 判定 |
|---|---|---|
| 每文件改动量 | `git diff --numstat -- relay/`：9 个文件全部 `1 1`（1 增 1 删） | 通过 |
| 改动内容只有 import | 全量 diff 的 18 行变动（9 增 + 9 删）**内容完全相同**，均为 `"github.com/QuantumNous/new-api/common"`；`git diff -U0 -- relay/` 中没有任何其它 +/- 行 | 通过 |
| 改动性质 | 该 import 从 import 块首行搬到 gofmt 排序位置（stdlib 与第三方同组时按字典序，`bufio` < `github...` < `io`）；无新增/删除依赖、无函数体改动 | 通过 |
| 依赖仍被使用 | 9 个文件均仍引用 `common.`（grep 计数 9/9），不存在「搬了 import 却变成未使用」的情况 | 通过 |
| 编译与静态检查 | `go build ./...`、`go vet ./...` 均 rc=0；全仓测试绿 | 通过 |
| 全仓格式 | `gofmt -l .`（仓库根，含 relaykit）输出为空 | 通过 |

## 二、死 i18n key 清理核对

| 核对项 | 证据 | 判定 |
|---|---|---|
| 源码已无引用 | `grep -rn` 两个 key 于 `web/src` 的 ts/tsx/json：**0 命中** | 通过 |
| HEAD 时即为真死键 | `git grep` 两个 key 于 HEAD 的 web/src 下 ts 与 tsx 文件：0 命中（HEAD 时只存在于 locale 文件） | 通过 |
| 7 语言删净 | 7 个 locale 文件对两个 key 的 grep 计数均为 0；各文件 `0 增 / 2 删`（合计 14 deletions / 0 insertions） | 通过 |
| 无重排 | 用脚本对比「HEAD 键序去掉这两个键」与工作区键序：7 个文件全部 `identicalOrder=true`，键数 5319 = HEAD 的 5321 - 2 | 通过 |
| JSON 合法且各语言一致 | 7 个文件 `jq` 解析 OK，`.translation` 键数均为 5319 | 通过 |
| 混淆键完好 | `footer.new\u0061pi` 在 7 个 locale 中各出现 1 次（与 HEAD 相同） | 通过 |
| i18n 门禁 | `bun run i18n:check` rc=0：4187 个引用键 × 7 locale 全部存在；引用键数与清理前一致（4187），反证被删的 2 个键确实无人引用 | 通过 |
| 陈旧产物说明 | `web/dist/static/js/index.*.js`（gitignored 构建产物）仍含旧 key 文案；属未重建的产物，不影响源码与服务，下次前端构建自动消失 | 已记录 |

## 三、脚本守卫核对

| 核对项 | 证据 | 判定 |
|---|---|---|
| 改动范围 | `git diff -U0 -- scripts/check-trellis-gate.sh`：1 个 hunk，+2/-1（1 行注释 + 把原来的无条件进入仓库根目录改为带失败守卫的一行） | 通过 |
| 守卫语义 | 模拟 `REPO_ROOT` 不存在：输出「无法进入仓库根目录: …」到 stderr 并 **exit 1**，不继续执行后续审计 | 通过 |
| 语法 | `bash -n scripts/*.sh`：7 个脚本全部 OK | 通过 |
| shellcheck | `shellcheck -S warning scripts/check-trellis-gate.sh`：0 发现（SC2164 已消除） | 通过 |
| 正常路径不变 | `./scripts/check-trellis-gate.sh --audit-only`：rc=0，输出「提交可追溯：8a991149..HEAD 共核对 85 个改动提交，全部带 [task:<slug>]」「Trellis 闸门通过」 | 通过 |
| 部署红线 | `sh scripts/forbid-extra-deploy-methods.sh`：rc=0，输出「部署方式唯一」 | 通过 |

## 四、门禁复跑（真实工作区，Go 全部 `GOPROXY=off`）

| 门禁 | 命令 | 结果 |
|---|---|---|
| Go 构建 | `go build ./...` | rc=0 |
| Go 静态检查 | `go vet ./...` | rc=0 |
| relaykit 独立构建 | `cd relaykit && GOWORK=off go build ./...` | rc=0 |
| 全仓测试 | `go test ./... -count=1` | rc=0，无 FAIL 行 |
| 前端类型 | `cd web && bun run typecheck` | rc=0 |
| 前端 i18n | `bun run i18n:check` | rc=0（4187 引用键 × 7 locale 全存在） |
| 前端 lint | 本地 `./node_modules/.bin/oxlint -c .oxlintrc.json .` | 0 warnings / 0 errors（960 文件、186 规则） |

> lint 说明：为遵守「不装工具、不访问外部主机」红线，我执行的是仓库内已安装的本地 oxlint 二进制（结果与 `bunx oxlint` 等价），
> 避免 bunx 在本地缺失时触网拉取。

## 五、边界核对

| 核对项 | 结果 |
|---|---|
| 改动文件清单 | `git diff --name-only` 共 17 个，恰为 9 relay + 1 脚本 + 7 locale，无其它已跟踪改动 |
| 未跟踪文件 | 仅任务目录 `.trellis/tasks/10-24-tech-debt-sweep/`；`--untracked-files=all` 下无其它残留 |
| 验证前后一致性 | 所有门禁与脚本跑完后复算 `git diff` 指纹，与基线一致；仓库源码未被本次验证修改 |
| 回滚性（R2） | 三类改动均可由 git 直接回滚；无运行时行为变化（import 位置、死键、脚本失败分支） |

## 六、不确定项与附注（均非阻断）

1. `web/dist` 是陈旧构建产物（含旧 key 文案），属 gitignored 产物；本轮未重建（不在低风险清扫范围）。若之后要用
   `go build` 出的二进制验证页面文案，需先 `cd web && bun run build`。
2. 本轮只验证了 PRD 中「代码类改动」涉及的门禁；PRD 中 W2/B2/B3/C1/C2 的报告类产出（TODO/死代码/版本引用/残留文件清单）
   属实现者证据文件内容，不在本次改动 diff 内，未逐条复核其清单完整性（如需要可另派只读审计）。
3. `check-trellis-gate.sh` 的历史审计覆盖 `8a991149..HEAD` 共 85 个提交；守卫改动本身不改变审计判定逻辑。

## 七、复跑命令清单

```bash
cd /root/new-api-own
# 格式零语义
git diff --numstat -- relay/            # 9 文件均应为 1 1
gofmt -l .                              # 应为空
go build ./... && go vet ./...
(cd relaykit && GOWORK=off go build ./...)
go test ./... -count=1

# 死键与 locale
grep -rn 'Model prices reset successfully\|This will clear custom pricing ratios and revert to upstream defaults\.' web/src
git diff --numstat -- web/src/i18n/locales/    # 7 文件均应为 0 2
jq -e . web/src/i18n/locales/*.json >/dev/null && echo "all locale JSON valid"
grep -c 'footer.new\\u0061pi' web/src/i18n/locales/en.json   # 应为 1

# 脚本
find scripts -maxdepth 1 -name "*.sh" -exec bash -n {} \; && echo "all scripts syntax OK"
shellcheck -S warning scripts/check-trellis-gate.sh
sh scripts/forbid-extra-deploy-methods.sh
./scripts/check-trellis-gate.sh --audit-only

# 前端
cd web && bun run typecheck && bun run i18n:check && ./node_modules/.bin/oxlint -c .oxlintrc.json .
```
