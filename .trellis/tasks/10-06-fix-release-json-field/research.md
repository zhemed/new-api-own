# 证据：修掉 release.yml 的不可用 `isLatest` 字段

任务 `10-06-fix-release-json-field`。仅改 `.github/workflows/release.yml`。未触发工作流、未动 v0.0.7。

## 1. 本地复现（gh 2.100.0）

```
$ gh --version
gh version 2.100.0 (2026-09-03)

$ gh release view v0.0.6 --json isLatest
Unknown JSON field: "isLatest"
Available fields:
  apiUrl  assets  author  body  createdAt  databaseId  id  isDraft
  isImmutable  isPrerelease  name  publishedAt  tagName  tarballUrl
  targetCommitish  uploadUrl  url  zipballUrl
```

**18 个可用字段里没有 `isLatest`**，与 CI 报错逐字一致。

顺带确认线上状态（只读）：

```
$ gh release view v0.0.7 --json tagName,isPrerelease,isDraft,assets --jq '{tagName,isPrerelease,isDraft,assets:(.assets|map(.name))}'
{"assets":["new-api-linux-amd64","new-api-linux-arm64","SHA256SUMS"],
 "isDraft":false,"isPrerelease":false,"tagName":"v0.0.7"}

$ gh api repos/zhemed/new-api-own/releases/latest --jq .tag_name
v0.0.7
```

→ v0.0.7 三个资产齐备、且**确实已是 latest**，run 判红纯粹是末尾校验字段的问题。

## 2. 修法

把 `gh release view --json isLatest` 换成**面板自己在用的同一个端点**：

```bash
gh api "repos/${GITHUB_REPOSITORY}/releases/latest" --jq .tag_name
```

**为什么不用 `gh release list --limit 1`**：它按 `created_at` 排序，**不等于** latest flag；
而 `/releases/latest` 正是面板 `update-checker-section.tsx` 请求的端点——断言它才等于断言真实契约。

新判定（四种分支）：

| 条件 | 行为 |
|---|---|
| prerelease 且 `releases/latest == TAG` | `::error::` 退出 1（面板会把预发布当升级） |
| prerelease 且不相等 | ok |
| `releases/latest == TAG` | ok |
| 不相等且 `LATEST_FLAG=--latest` | `::error::` 退出 1（最高的版本却没拿到 latest） |
| 不相等且 `--latest=false`（重跑旧 tag） | ok（有意不抢 latest） |
| API 查询失败 / 返回空 | `::error::` 退出 1（**不静默**） |

同时把汇总步骤里的 `isLatest` 换成真实存在的 `isDraft`。

## 3. 同段脚本其它字段审计（R2）

对 `run:` 脚本**去掉注释行**后扫描所有 `--json`：

```
assets                                     all valid
tagName,isPrerelease,isDraft,assets        all valid
invalid in executable code: NONE
isLatest in executable code: False
isLatest mentions are all comments: True
```

结论：**只有 `isLatest` 一个坏字段**，其余 `assets` / `tagName` / `isPrerelease`
与 `--jq '.assets[].name'`、`--jq .tag_name` 均为该版本支持的用法。
（`isLatest` 的 3 处残留全部在**解释性注释**里，说明"为什么不要用它"。）

## 4. gh stub 矩阵（跑的是从 release.yml 抽出的**真实**校验块，46 行）

```
  PASS [target == latest]                exit=0  ok: v0.0.7 is the repository 'latest' (releases/latest -> v0.0.7)
  PASS [old tag rerun (not latest)]      exit=0  ok: v0.0.5 intentionally does not claim 'latest' (highest version tag is 'v0.0.7'); releases/latest -> v0.0.7
  PASS [API query fails]                 exit=1  ::error::could not read repos/zhemed/new-api-own/releases/latest to verify the 'latest' flag…
  PASS [API returns empty]               exit=1  ::error::repos/zhemed/new-api-own/releases/latest returned an empty tag_name…
  PASS [highest but not latest]          exit=1  ::error::v0.0.7 is the highest version tag and was published with --latest, but releases/latest returns 'v0.0.5'.
  PASS [prerelease served as latest]     exit=1  ::error::pre-release v0.0.8-alpha.1 is being served as the repository 'latest'…
  PASS [prerelease normal]               exit=0  ok: v0.0.8-alpha.1 is a pre-release; releases/latest is 'v0.0.7'
```

stub 实现：`gh api … --jq .tag_name` → 输出 `$STUB_LATEST` 并按 `$STUB_API_EXIT` 决定退出码；
`gh release view …` → 返回合法 JSON。三种必需情况（=latest / 非 latest / 查询失败）全部符合预期。

## 5. 语法与范围

```
YAML parse: OK
bash -n:
  PASS Resolve and validate tag      PASS Align VERSION with the tag
  PASS Build frontend                PASS Build Linux binaries (amd64 + arm64)
  PASS Generate SHA256SUMS           PASS Publish release
ALL SCRIPTS: PASS

项目闸门 forbid-extra-deploy-methods.sh → ✅ 部署方式唯一
git status .github/workflows/ → 只有 release.yml 被改
```

## 6. 顺带发现（未处理，供决策）

- `gh release view v0.0.6 --json …` → **`release not found`**：**v0.0.6 至今仍然没有 Release 条目**。
  v0.0.7 发布后 `/releases/latest` 已是 v0.0.7，面板提示恢复正常，所以**症状已解**；
  但若要让 v0.0.6 也能"被面板识别/可回滚"，仍需用 dispatch 补发一次
  （`gh workflow run "Publish release (image + Linux updater binaries)" -f tag=v0.0.6`）——需授权。
- 旧 `v0.0.5` 上的历史命名资产仍按用户要求不动。
