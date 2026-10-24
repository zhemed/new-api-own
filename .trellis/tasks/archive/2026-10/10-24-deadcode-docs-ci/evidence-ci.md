# CI 证据：新增 gofmt + shellcheck 守卫工作流（task-45）

- 任务：Trellis `10-24-deadcode-docs-ci`（PRD R3）；共享任务板 `task-45`；执行者 worker-containers
- 执行时间：2026-10-24T05:23:29Z ～ 05:24Z（UTC）
- 事前依据：PRD `prd.md` R3「把 gofmt 与 shellcheck 检查接进 CI」；用户点名
- 约束遵守：**只新增 `.github/workflows/lint.yml`**；未改 `docker-build.yml` / `release.yml` / `trellis-gate.yml`；
  未改业务代码；未提交未推送；未装工具（actionlint 未安装，按 PRD 不装）

## 0. 结论摘要

| 项目 | 结果 |
|---|---|
| 新增文件 | `.github/workflows/lint.yml`（75 行，两个 job：`gofmt` / `shellcheck`） |
| 触发 | `push` 到 `main` + `pull_request` + `workflow_dispatch`（与 `trellis-gate.yml` 风格一致） |
| 权限/并发 | `permissions: contents: read`；`concurrency` 同 ref 取消旧轮 |
| YAML 合法性 | `python3 yaml.safe_load` 通过；jobs = `gofmt,shellcheck` |
| 干净基线 | 本地同款命令全部 PASS：gofmt 0 未格式化（813 个被跟踪 .go 文件）、bash -n exit=0、shellcheck exit=0 |
| 故意违规 | gofmt 步骤 **exit=1**（输出未格式化文件清单 + `::error::` 注解）；shellcheck 步骤 **exit=1**（SC2034 warning）；清理后无残留 |
| 运行时开销 | 本地检查本体：gofmt 79ms、bash -n 1.3ms、shellcheck 217ms；CI 单 job 预估 20-40s（大头是 checkout + setup-go） |
| 既有 workflow | 三个既有文件 `git diff` 均为 0 行，职责未动 |
| 清理后状态 | 无 demo 残留；`git status` 中本任务相关只有 `?? .github/workflows/lint.yml`（另有 `M README.md` 属其它成员 R2 文档改动，非本任务） |

## 1. workflow 全文（新增文件）

```yaml
name: lint (gofmt + shellcheck)

# 独立的轻量守卫：只检查「Go 文件格式」与「Shell 脚本卫生」，不构建、不发版、不碰闸门。
#
# 与既有 workflow 的分工（本文件不改动它们，也不被它们依赖）：
#   docker-build.yml  -> 打多架构镜像并推 ghcr（tag 触发）
#   release.yml       -> 建 GitHub Release + updater 二进制（tag 触发）
#   trellis-gate.yml  -> 审计提交里的 [task:] 锚点（push/pull_request）
#
# 触发条件与 trellis-gate.yml 保持一致（push main + pull_request），另留 workflow_dispatch 便于手动重跑。
# 运行时开销：不做构建、不装依赖；gofmt 作业约 20-40 秒（大头是 checkout + setup-go），
#             shellcheck 作业约 15-25 秒（ubuntu runner 已预装 shellcheck）。

on:
  push:
    branches: [main]
  pull_request:
  workflow_dispatch:

permissions:
  contents: read

# 同一 ref 上并发运行时取消旧的一轮，避免快速连推时排队堆积。
concurrency:
  group: lint-${{ github.ref }}
  cancel-in-progress: true

jobs:
  gofmt:
    name: gofmt (tracked Go files)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # 固定到 go.mod 声明的 Go 版本：gofmt 的规则随 Go 版本演进，用 runner 默认版本会让
      # 「同一次提交在不同时间跑出不同结果」。本作业不构建，因此关掉模块缓存。
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: false

      - name: gofmt check
        run: |
          # 只检查被 git 跟踪的 .go 文件：
          #   * 天然排除 vendor/（本仓库没有）、web/node_modules/（本机存在且体积大）、未跟踪的临时文件；
          #   * relaykit/ 虽是独立 Go 模块（自带 go.mod），但源码在本仓库，且 gofmt 与模块边界无关，
          #     故有意纳入检查，避免独立模块的格式漂移无人守；
          #   * 本仓库当前没有 `// Code generated ... DO NOT EDIT.` 的 Go 生成物；将来若有，
          #     在此处按路径显式排除，而不是放宽整条检查。
          unformatted=$(git ls-files -z '*.go' | xargs -0 -r gofmt -l)
          if [ -n "$unformatted" ]; then
            echo "::error::gofmt 发现未格式化文件，CI 失败。文件清单如下："
            echo "$unformatted"
            echo "本地修复：gofmt -w <上面列出的文件>"
            exit 1
          fi
          echo "gofmt OK：$(git ls-files '*.go' | wc -l) 个被跟踪的 Go 文件全部已格式化"

  shellcheck:
    name: shellcheck (scripts/*.sh)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      # shellcheck 由 ubuntu-latest 镜像预装（这里不联网安装）；打印版本便于排查
      # 「镜像升级导致 shellcheck 版本变化、新旧规则不一致」这类问题。
      # 级别与本地基线一致：-S warning（info/style 报告不导致失败）。
      - name: 记录 shellcheck 版本
        run: shellcheck --version

      - name: bash 语法检查
        run: bash -n scripts/*.sh

      - name: shellcheck -S warning
        run: shellcheck -S warning scripts/*.sh
```

## 2. 设计取舍（为什么这样写）

1. **独立 workflow，不动既有三条**：`docker-build.yml` / `release.yml` 是 tag 触发的发版链路，
   `trellis-gate.yml` 是提交锚点审计；把格式检查塞进任一都会扩大其职责、增加耦合。
   新增 `lint.yml` 只做检查，失败不影响发版（tag 推送不触发它）。
2. **gofmt 检查「被 git 跟踪的 .go 文件」**（用 `git ls-files` 取全部被跟踪 Go 文件，逐批交给 gofmt）而不是 `gofmt -l .`：
   - 天然排除 `web/node_modules/`（本机存在，796 个顶层条目；`gofmt -l .` 会白遍历它，本机实测 170ms vs 79ms）、
     `vendor/`（本仓库无）；
   - 排除未跟踪的临时/草稿文件（不会因为本地残留的 scratch 文件误红）；
   - **relaykit 纳入**：它是独立 Go 模块（`relaykit/go.mod`），但源码在本仓库、gofmt 与模块边界无关；
     不纳入会让这个 94 文件的目录成为格式漂移的盲区。若将来 relaykit 迁出仓库，再把它从这里去掉即可。
   - 无生成物需要排除：本仓库当前没有 `// Code generated ... DO NOT EDIT.` 的 Go 文件（已实测 count=0）。
3. **Go 版本显式固定**（`actions/setup-go@v5` + `go-version-file: go.mod`）：gofmt 规则随 Go 版本演进，
   用 runner 默认版本可能出现「同一提交换天跑出不同结果」。代价是 job 多 ~10-20s（可接受）。
4. **shellcheck 用 runner 预装版本**（不 `apt-get install`，避免联网与版本不可得问题），但**先打印版本**，
   便于排查「镜像升级导致规则变化」。`-S warning` 与本地基线一致；`bash -n` 作为语法层的廉价双保险。
5. **失败可读性**：gofmt 失败输出 `::error::` 注解（GitHub UI 直接标红该行）+ **未格式化文件清单** + 本地修复命令；
   shellcheck 失败输出其原生诊断（文件名/行号/规则码/解释链接）。

## 3. YAML 合法性校验

```console
$ python3 -c "import yaml; d=yaml.safe_load(open('.github/workflows/lint.yml')); print(d)"
  yaml.safe_load: OK
  top-level keys: ['name', True, 'permissions', 'concurrency', 'jobs']
  triggers: {'push': {'branches': ['main']}, 'pull_request': None, 'workflow_dispatch': None}
  jobs: ['gofmt', 'shellcheck']
  permissions: {'contents': 'read'}
  concurrency: {'group': 'lint-${{ github.ref }}', 'cancel-in-progress': True}
```

说明：PyYAML 按 YAML 1.1 把裸 `on` 解析成布尔键 `True`，而 GitHub 用 YAML 1.2 视作字符串 `on`；
这是已知无害的解析器差异（GitHub 侧正常识别触发器）。本机无 `actionlint`，按 PRD「没有不要装」未安装。

## 4. 本地等价验证（用 workflow 里的**同一份 run 文本**执行）

方法：用 python 从 `lint.yml` 抽出每个 job 的 `run:` 步骤原文，写成 `/tmp/ci-verify/<job>-steps.sh`，
再用 `bash -e <file>` 执行 —— 与 GitHub Actions 默认 shell（`bash -e {0}`）一致。

### 4.1 干净基线：通过

```console
--- gofmt（workflow 同款命令）---
  PASS: 0 unformatted (checked 813 tracked .go files)
--- bash -n（workflow 同款）---
  PASS: bash -n exit=0
--- shellcheck -S warning（workflow 同款）---
  PASS: shellcheck exit=0
--- shellcheck 版本 ---
version: 0.10.0

$ bash -e /tmp/ci-verify/gofmt-steps.sh
gofmt OK：813 个被跟踪的 Go 文件全部已格式化
  gofmt_steps_exit=0
$ bash -e /tmp/ci-verify/shellcheck-steps.sh
  shellcheck_steps_exit=0
```

### 4.2 故意违规 #1：gofmt 不干净 → 检查 FAIL（exit=1）

做法：新增一个**语法合法但格式不合法**的 Go 文件（`func  Bad( ) {` 等），用 `git add -N` 让它出现在
`git ls-files`（模拟「已提交的文件」），再执行 workflow 的 gofmt 步骤。

```console
### V3 故意违规 #1：gofmt 不干净文件（模拟 CI 中的被跟踪文件）
  git ls-files 可见该文件: 1
  --- 执行 workflow 的 gofmt 步骤（同一份 run 文本，bash -e）---
::error::gofmt 发现未格式化文件，CI 失败。文件清单如下：
zz-gofmt-violation-demo.go
本地修复：gofmt -w <上面列出的文件>
  => gofmt 步骤 exit=1 (期望 1)
  RESULT: 故意违规确实让检查 FAIL ✓
```

### 4.3 故意违规 #2：shellcheck warning → 检查 FAIL（exit=1）

做法：在 `scripts/` 放一个未跟踪的脚本，触发 `SC2034 (warning)`；`bash -n` 仍通过（证明两层检查互补）。

```console
### V4 故意违规 #2：shellcheck warning 脚本
  --- bash -n（语法层应仍 PASS，证明 shellcheck 检查的是质量而非语法）---
  bash -n exit=0 (PASS)
  --- 执行 workflow 的 shellcheck 步骤（同一份 run 文本，bash -e）---
In scripts/zz-shellcheck-violation-demo.sh line 2:
unused_variable=1
^-------------^ SC2034 (warning): unused_variable appears unused. Verify use (or export if used externally).
  => shellcheck 步骤 exit=1 (期望 1)
  RESULT: 故意违规确实让检查 FAIL ✓
```

### 4.4 清理与最终状态

```console
### V5 清理
  demo 文件残留检查: 0 个（期望 0）
  git 中是否还有 intent-to-add 残留: 0（期望 0）

### V6 清理后状态
--- git status --short ---
 M README.md
?? .github/workflows/lint.yml
?? .trellis/tasks/10-24-deadcode-docs-ci/
--- .github 变更 ---
?? .github/workflows/lint.yml
--- 确认未改既有 workflow ---
  .github/workflows/docker-build.yml         tracked_diff=0
  .github/workflows/release.yml              tracked_diff=0
  .github/workflows/trellis-gate.yml         tracked_diff=0
```

说明：`M README.md` 是**其它成员**在改 R2 文档，不是本任务动作；本任务在 `.github/` 下只新增了 `lint.yml`。
demo 文件与 index 残留均已清除，清理后重新跑干净基线再次 PASS（见 4.1 的第二次执行）。

## 5. 运行时开销

| 检查 | 本机实测（含进程启动） | CI 预估 |
|---|---|---|
| `git ls-files` 取全部被跟踪 Go 文件 + `gofmt -l`（813 个） | 79.2ms | 同量级（I/O 为主） |
| 对照：`gofmt -l .`（会遍历 node_modules） | 170.4ms | — |
| `bash -n` 检查 scripts 下全部 .sh（7 个脚本） | 1.3ms | 同量级 |
| `shellcheck -S warning` 检查 scripts 下全部 .sh（7 个脚本） | 217.3ms | 同量级 |
| **检查本体合计** | **< 0.3 秒** | < 1 秒 |

CI 单个 job 的墙钟时间主要花在 `actions/checkout` 与 `actions/setup-go`（下载/解压 Go 工具链），
预估每个 job **20-40 秒**，两个 job 并行；无构建、无依赖安装、无缓存写入，属「轻量」。

## 6. 失败时的可读报错

- **gofmt**：GitHub 会把 `::error::` 行渲染成红色注解（指向该 step），并打印**未格式化文件逐行清单**，
  以及本地修复命令 `gofmt -w <文件>`；开发者一眼能定位。
- **shellcheck**：原生诊断格式 `文件:行号: 规则码 (级别): 说明` + 官方 wiki 链接，直接可修。
- **bash -n**：语法错误会带文件与行号；它和 shellcheck 是两层互补（语法 vs 质量）。

## 7. 回滚方式

本改动是**纯新增文件**，回滚只需：`rm .github/workflows/lint.yml`（或 `git checkout -- .github/workflows/`
若已提交后要回退）。不影响任何既有 workflow、构建与发版链路。

## 8. 风险与未完成

- 本任务**未提交未推送**：`lint.yml` 目前是工作区新增文件，首次真实运行要等 Lead 提交/推送后观察（届时可核对
  两个 job 是否按预期绿；若 runner 的 shellcheck 版本与本地 0.10.0 差异较大，可能暴露新的 warning——属预期，修脚本即可）。
- GitHub runner 的 `ubuntu-latest` 预装 shellcheck 与 Go 版本会随镜像更新；已用 setup-go 固定 Go，
  shellcheck 仅在日志里打印版本（未固定）。若要完全可复现，可后续讨论用容器镜像固定 shellcheck 版本（需评估成本）。
- `M README.md` 属其它成员改动，本任务未触碰。
- 未做：未装 actionlint（PRD 明确「没有不要装」）；未在真实 GitHub runner 上跑过（需提交推送后才有）。

## 9. 建议下一步

1. Lead 复核 `lint.yml` 内容与第 4 节的 fail 演示，按闸门提交推送后**看一眼首次 CI 运行**（两个 job 应绿）。
2. 若首次运行暴露 runner shellcheck 版本差异导致的 warning，优先修脚本（而不是放宽级别）。
3. 后续可考虑把 `lint.yml` 纳入分支保护必需检查（需仓库管理员设置，本轮未涉及）。
