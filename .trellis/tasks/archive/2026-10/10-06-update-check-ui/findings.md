# 前端：版本/更新面板（task 10-06-update-check-ui）

## 最终形态

系统维护页「更新卡片」= 四态 + 管理员一键更新。**浏览器零外呼**：全部走自家后端。

### UI 交互流程

```
挂载 → GET /api/status/update（自动一次，AdminAuth）
   ├─ enabled=false → 「更新检查已关闭。在服务端设置 UPDATE_CHECK_ENABLED=true 即可重新开启。」
   │                   （绝不显示"无法确定"；无「立即更新」按钮）
   ├─ enabled=true + state=update_available → 「有新版本可用：vX.Y.Z」
   │        ├─ apply_enabled=true  + 管理员 → 显示「立即更新」
   │        └─ apply_enabled=false         → 不显示按钮，改显示
   │                                          「管理员已关闭面板内更新。请改用 README 中的镜像升级步骤。」
   ├─ enabled=true + state=up_to_date → 「已是最新」
   └─ enabled=true + state=unknown|unavailable → 「无法确定最新版本。」（中性，不报错）
      GET 失败/无 payload → 同上中性态（不谎报、不弹"有新版本"）

「立即更新」点击 → ConfirmDialog 二次确认，正文两条代价：
   ① 更新过程中服务会重启，面板将短暂不可用
   ② 若容器被重建，面板会退回镜像内的版本
   → 确认 → POST /api/status/update/apply → 执行中：按钮文案「更新中…」+ 对话框 isLoading
      ├─ 成功 → 关闭对话框 + toast「更新已开始。服务即将重启，请稍后重新加载本页。」
      └─ 失败 → 对话框保持打开（可重试）+ toast.error(可读原因)
```

失败原因映射（后端 `service/update_apply.go` 的 `code` → 面板文案）：

| 后端 code | 面板 reason | 文案 |
|---|---|---|
| `disabled` | disabled | 管理员已关闭面板内更新，请按 README 镜像三步 |
| `busy` | busy | 已有更新正在进行中。 |
| `not_newer` | not_newer | 当前版本已是该版本。 |
| `download_failed` / `source_unavailable` / `asset_missing` / `checksum_download_failed` | download_failed | 新版本下载失败。 |
| `checksum_mismatch` / `checksum_missing` / `checksum_asset_missing` | checksum_failed | 下载的更新未通过校验。 |
| `unsupported_platform` | unsupported_platform | 当前平台不支持面板内更新。 |
| `binary_too_large` | too_large | 更新包超过服务端允许的大小。 |
| `replace_failed` / `temp_file_failed` / `executable_unknown` / `config_invalid` | replace_failed | 服务端无法替换自身文件，请检查容器权限。 |
| 未知 code + 有 message | null | 直接展示服务端 message |
| 无任何细节 | null | 更新失败，请查看服务端日志了解详情。 |

## 调用点（与前端的契约）

| 用途 | 方法/路径 | 读取/发送 | 字段 |
|---|---|---|---|
| 更新状态 | `GET /api/status/update`（AdminAuth） | 读 | `data.enabled`(bool 检查开关)、`data.state`(`disabled`/`unknown`/`up_to_date`/`update_available`/`unavailable`)、`data.latest_version`(string，如 `v0.0.6`)；同响应还含 `current_version`/`has_update`/`release_url` 等，前端当前不用 |
| 执行更新 | `POST /api/status/update/apply`（AdminAuth） | 发 `{}`（后端 `ApplyLatestUpdate` 不接受目标版本） | 成功读 `data.version`；失败体为**非 2xx** + `{success:false, message, code}`，前端从 axios 异常的 `response.data.code` 取 `code` |
| 应用开关 | `GET /api/option/` | 读 | `update_check_setting.apply_enabled`（选项值是字符串 `"true"`/`"false"`） |

**对接缺口（需要后端补）**：`apply_enabled` 不在 `GET /api/status/update` 里（`service.UpdateCheckStatus` 无该字段），
前端只能绕道选项接口读 `update_check_setting.apply_enabled`。若把 `apply_enabled` 加进该响应，
`use-update-status.ts` 里的 `APPLY_ENABLED_OPTION` 依赖即可删除（一处改动）。

**与上一轮口径的偏离（重要）**：浏览器 → `api.github.com` 的直连**已删除**（`grep api.github.com web/src` = 0）。
依据：状态 3「检查被管理员关闭」要求面板遵守 `UPDATE_CHECK_ENABLED`，而浏览器侧直连无法被该开关约束；
且后端已提供 `GET /api/status/update`（服务端检查 + 缓存 1h + 失败退避 30min）。若必须保留 GitHub 直连作为降级源，一句话即可加回。

## 改动文件

**新增**
- `web/src/features/system-settings/maintenance/update-api.ts` — 两个端点 + 失败 code 映射，永不抛异常
- `web/src/features/system-settings/maintenance/update-status.ts` — 纯状态判定（4 态优先级）
- `web/src/features/system-settings/maintenance/use-update-status.ts` — 唯一接入点（挂载取一次 + 手动刷新 + apply 开关）
- `web/src/features/system-settings/maintenance/__tests__/update-api.test.ts`
- `web/src/features/system-settings/maintenance/__tests__/update-status.test.ts`
- `web/src/features/system-settings/maintenance/__tests__/update-checker-section.test.tsx`

**改动**
- `web/src/features/system-settings/maintenance/update-checker-section.tsx`（重写）
- `web/src/i18n/locales/{en,zh,zh-TW,fr,ja,ru,vi}.json`（+24 键，脚本生成）

**删除**
- `web/src/features/system-settings/maintenance/use-latest-version.ts`（被 `use-update-status.ts` 取代）

未触碰 Go / docs / 根 md / `.github` / `scripts` / `.local-instance`；未 commit/push。

## i18n

三批共 **24** 个新键 × 7 语言，全部经临时 `scripts/add-missing-keys.mjs`（自建、**已删除**）+ `node scripts/sync-i18n.mjs`，
**未手改 locale JSON**。新增键全部以 `t('...')` 字面量出现，无需登记 `STATIC_I18N_KEYS`。

```
$ bun run i18n:check
i18n check: 4025 literal t() keys + 455 static keys = 4176 referenced keys across 7 locales.
All referenced keys exist in every locale.
```

## 六条命令

| # | 命令 | 结果 |
|---|---|---|
| 1 | `bun run typecheck` | ✅ 0 错误 |
| 2 | `bun run lint` | ✅ 0 error（21 warning 为仓库既有，非本次引入；基线同样 21） |
| 3 | `bun test` | ✅ 200 pass / 0 fail（基线 167；本次 +33）；act 警告 0 |
| 4 | `bun run build` | ✅ exit 0（Total 57316.9 kB / gzip 16535.4 kB） |
| 5 | `bun run knip` | ⚠️ exit 1 —— **基线即红**（见下） |
| 6 | `bun run i18n:check` | ✅ 全部键在 7 语言齐备 |

### knip 基线证明（非本次引入）

用 `git archive HEAD | tar -x -C /tmp/knip-baseline`（不动工作区/.git）导出纯净 HEAD，
symlink node_modules 后运行 `bun run knip` → **同样 exit 1**，报同一批 `Unused exports (312)` + `Duplicate exports (1)`
（`useDialogState|default`）+ `Configuration hints (2)`（`tailwindcss`/`tw-animate-css`）。
本次改动对 knip 是**净 −1**：仅使 `getBuildRevision` 从 unused 列表消失，未新增任何条目。

## 测试覆盖（33 个新测试）

- `update-status.test.ts`（7）：四态优先级、开关关闭优先于 stale verdict、端点不可达 → unknown、`older` 不谎报
- `update-api.test.ts`（11）：`GET` 解析、开关缺失按关闭处理、apply 成功/异常 code 映射（含 4 个下载码合一）、未知 code 回落 message、永不抛异常
- `update-checker-section.test.tsx`（15）：四态文案、挂载只调一次自家后端、非管理员无按钮、apply 关闭时按钮被说明文案替代、二次确认含重启与镜像两条代价、确认后 POST 到 `/api/status/update/apply`、失败按 code 给可读原因、未知失败回落服务端 message

## 未验证

- 未做浏览器实测（未起 dev server）：真实 `AlertDialog` 焦点管理、toast 呈现、语言切换后文案未跑运行时验证。
- 后端 `GET /api/status/update` / `POST /api/status/update/apply` 的真实响应（字段名/状态值）按代码读取对齐，
  未在真实实例上跑通（只做了单测级 stub）。

---

# 真机端到端（3020 试验机，Lead 授权）

## 构建与重启

```
cd /root/new-api-own
GOWORK=off go build -ldflags "-X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" \
  -o .local-instance/new-api.new .        # 4.7s，139.9 MB，exit 0
```
- 二进制用 `os.replace`（即 `mv` 语义）覆盖 `.local-instance/new-api`，**未用 `cp` 到运行中的路径**，避开 ETXTBSY。
- 重启：先读 `/proc/<pid>/{cmdline,environ}`（**只打印 key 名，从不打印值**），`SIGTERM` 旧进程，
  再 `fork + setsid + execve` 以**原 argv/原 env** 启动（渲染 `-port 3020 -log-dir=`）。
- 前端是 `//go:embed web/dist`，所以改前端必须重建二进制才会生效——已重建。

## 四态断言（真机 UI）

| 状态 | 触发方式 | 真机显示 |
|---|---|---|
| 有更新 | 后端自带 `UPDATE_CHECK_API_BASE_URL` 指向本地 mock 源（tag `v9.9.9`） | 「最新版本 v9.9.9」「更新状态 **有新版本可用：v9.9.9**」+「立即更新」按钮 |
| 已最新 | **真实源**（GitHub releases/latest = `v0.0.5`，构建 `0.0.6`） | 「最新版本 v0.0.5」「更新状态 **已是最新**」 |
| 无法确定 | 更新源不可达（base URL 指向已停止的端口，连接被拒） | 「更新状态 **无法确定最新版本。**」（中性，无报错、无"有新版本"、无按钮） |
| 检查已关闭 | `UPDATE_CHECK_ENABLED=false` | 「更新状态 **更新检查已关闭。在服务端设置 UPDATE_CHECK_ENABLED=true 即可重新开启。**」（**不是**"无法确定"） |
| 应用已关闭 | `UPDATE_APPLY_ENABLED=false` + 有更新 | 按钮消失，改显示「**管理员已关闭面板内更新。请改用 README 中的镜像升级步骤。**」（`apply-disabled-note`，`hasApplyBtn=false`） |

**生产事故 case 已复现并确认根因**：真实源最新 tag 就是 `v0.0.5`（比构建 0.0.6 旧）。
旧代码严格比较 → 谎报"有新版本可用：v0.0.5"；现在按 `older` 处理 → 「已是最新」。

## 「立即更新」失败分支（真机）

1. 点「立即更新」→ 确认框：「**更新到 v9.9.9？**」+「更新过程中服务会重启，面板将短暂不可用。」+「若容器被重建，面板会退回镜像内的版本。」+ 取消/立即更新。
2. 确认 → `POST /api/status/update/apply` → **HTTP 502**（后端 `UpdateApplyError`，该 mock 源 `assets: []` → `asset_missing`）。
3. 面板 toast：**「新版本下载失败。」**（`asset_missing` → `download_failed` 映射；可读，非错误堆栈）；对话框保持打开可重试。
4. **服务未受影响**：`GET /api/status` → **200**，`version` 仍 `0.0.6`，进程未重启（同 pid，etime 继续增长）。

## 截图

- `.local-instance/e2e-1-panel.png` —— 真实源：已是最新（v0.0.5 ≤ 0.0.6）
- `.local-instance/e2e-2-update-available.png` —— 有更新 + 立即更新按钮
- `.local-instance/e2e-3-confirm-dialog.png` —— 二次确认（重启 + 容器重建两条代价）
- `.local-instance/e2e-4-apply-failure.png` —— 失败 toast + 对话框保持打开

## 环境还原（测试后）

- mock 源已停止；3020 已用**原始 env** 重启，当前状态 = 真实源「已是最新」。
- **管理员密码已还原**：为登录管理员，临时把 `users.password` 换成自建 bcrypt hash（用仓库自带 `golang.org/x/crypto/bcrypt` 生成），
  测完已写回**原 hash**（sha256[:8] `53f449a8` 与备份逐字节一致）；临时凭据文件已删除，**全过程未打印任何凭据**。
- 3020 当前跑的是 14:16 构建（含全部状态 UI）；此后仅再改「优先读 `apply_enabled` 响应字段」一处，
  未再重建二进制（重建需再重启并再次使会话失效）。该路径已由单测覆盖（`prefers the new check_enabled/apply_enabled fields when present`）。
- 本次真机验证覆盖的是 **选项接口回落路径**（后端尚未在状态响应里给出 `apply_enabled`/`check_enabled`）；
  响应字段优先路径为单测覆盖，待后端补字段后自动生效。

## 六条命令（最终）

| # | 命令 | 结果 |
|---|---|---|
| 1 | `bun run typecheck` | ✅ 0 错误 |
| 2 | `bun run lint` | ✅ 0 error（21 warning 既有） |
| 3 | `bun test` | ✅ **202 pass / 0 fail** |
| 4 | `bun run build` | ✅ exit 0（Total 57317.1 kB / gzip 16535.5 kB） |
| 5 | `bun run knip` | ⚠️ exit 1 —— 基线即红（`git archive HEAD` 纯净对照同红）；我的文件 0 处出现在 knip 输出 |
| 6 | `bun run i18n:check` | ✅ 4176 键七语言齐备 |

---

# 收尾轮：删掉 options 回落 + 最终代码真机复验

## 改动（本任务线最后一处）

后端已给出 `check_enabled` / `apply_enabled`（并保留兼容别名 `enabled`），因此删除 options 绕道：

- `use-update-status.ts`：移除 `useSystemOptions` import、`APPLY_ENABLED_OPTION`、`readApplyEnabled`、`OptionsPayload`；
  改为 `applyEnabled: status?.applyEnabled ?? true`。**现在面板只请求 `GET /api/status/update` 一个接口**，
  `grep -rn "use-system-options" web/src/features/system-settings/maintenance/` = 0（仅剩一句说明后端设置名的注释）。
- `update-api.ts`：`checkEnabled` 读 `check_enabled` 并回落兼容别名 `enabled`；`applyEnabled` 读 `apply_enabled`（缺失为 `null`）。
- 测试同步：组件测试不再需要 react-query Provider，开关随状态 payload 下发
  （`AVAILABLE = { check_enabled: true, apply_enabled: true, ... }`）。
- 未新增/删除文件；六条命令重跑见下。

## 最终代码在 3020 上的实测（二进制 14:28 重建并 `os.replace` 部署）

| 场景 | 真机结果 |
|---|---|
| **已最新**（真实源，原始 env） | 「最新版本 **v0.0.5**」「更新状态 **已是最新**」 |
| **有更新 + 应用关闭**（mock 源 v9.9.9 + `UPDATE_APPLY_ENABLED=false`） | 「**有新版本可用：v9.9.9**」+ 按钮消失，显示「管理员已关闭面板内更新。请改用 README 中的镜像升级步骤。」（`hasApplyBtn=false`） |
| **无法确定**（源不可达） | 「**无法确定最新版本。**」 |

**关键点**：最终代码已不再查 `GET /api/option/`，而「应用关闭」态在真机上依然正确渲染——
说明该判定现在确实由 `GET /api/status/update` 的 `apply_enabled` 驱动（旧实现会走 options 回落，故此前无法区分）。

## 环境还原

- mock 已停止；3020 已用**原始 env** 重启（pid 531861，真实源，「已是最新」），`/api/status` 200、version 0.0.6。
- admin 密码 hash 已还原（sha256[:8] `53f449a8` 与备份一致）；临时凭据/脚本文件全部删除；未打印任何凭据。

## 六条命令（收尾轮）

| # | 命令 | 结果 |
|---|---|---|
| 1 | `bun run typecheck` | ✅ 0 错误 |
| 2 | `bun run lint` | ✅ 0 error（21 warning 既有） |
| 3 | `bun test` | ✅ **202 pass / 0 fail** |
| 4 | `bun run build` | ✅ exit 0（Total 57326.1 kB / gzip 16539.2 kB） |
| 5 | `bun run knip` | ⚠️ exit 1 —— 基线即红（纯净 HEAD 对照同红）；我的文件 0 处出现在 knip 输出 |
| 6 | `bun run i18n:check` | ✅ 4176 键七语言齐备 |

---

# 格式收尾

`bun run format:check` 曾 exit 1，报 5 个文件不规范（`system-brand.test.tsx`、
`maintenance/__tests__/{update-api,update-checker-section}.test.*`、`maintenance/{update-api,use-update-status}.ts`）。
用项目自带 `bun run format`（protected-header-safe，oxfmt 包装）就地修复：

- **改动范围可控的证据**：格式化前后各取一次 `git status --porcelain -- web/`（25 条），两次**逐字节相同**
  → 没有任何原本干净的文件被改成脏的；结合「`format:check` 修复前只列出这 5 个文件」与
  「修复后 exit 0」，可确认只有这 5 个文件被重写。
- 复跑：`format:check` exit 0、`typecheck` 0 错误、`lint` 0 error、`bun test` 202 pass / 0 fail、
  `build` exit 0、`i18n:check` 4176 键七语言齐备；`knip` 仍为基线红（同前证据）。
