# 3020 真机验证：初始化 + 部署新前端 + 截图取证（2026-10-06）

## 结论

3020 已完成初始化、已部署新前端，**认证后顶栏与系统维护页都显示版本 `0.0.6`**，
与 `/api/status` 的 `version` **一致**；构建标识从 `rv.0000.2k6e8r7p` 变为 **`rv.0.0.6.2k6e8r7p`**。

## 执行记录

1. **侦察**：pid `476111`，`./new-api -port 3020 -log-dir=`，cwd/exe 都在 `.local-instance/`；
   cgroup 显示它由 `dsh-web.service` 拉起（**没有**独立 unit），故按 Lead 指示手工重启。
   启动前环境：30 个变量（含 `LOG_SQL_DSN` / `LOG_CLEANUP_INTERVAL` / `LOG_MEMORY_MAX_ROWS`），
   从 `/proc/476111/environ` 读进 shell，**未打印任何值**。
2. **初始化**：`bw` 走完 4 步向导（数据库检查 → 管理员账户 → 使用模式 → 审核并初始化）→
   "系统初始化成功"。**管理员凭据仅存在于浏览器 sessionStorage，未写入任何文件、未出现在任何汇报里。**
   使用模式选了「对外运营」（全功能，便于验证；如需演示站点模式请告知）。
3. **部署**：`bun run build`（前端）→ `go build -ldflags "-s -w -X .../common.Version=$(cat VERSION)"`
   → 先 `cp` 到 `new-api.new` 再 `mv` 覆盖 `.local-instance/new-api`（避开对运行中二进制的 ETXTBSY）
   → `kill 476111` → 以相同 cwd/参数/环境 `setsid nohup ./new-api -port 3020 -log-dir=` 启动。
   新 pid `500527`，`/` 返回 200。
4. **重建后登录**：重启导致会话失效（符合预期），用 sessionStorage 里的凭据重新登录成功。

## 证据

| 项 | 值 |
| --- | --- |
| `/api/status` → `version` | `0.0.6` |
| `/api/status` → `setup` | `true`（初始化前为 `false`） |
| 服务端入口 chunk | `index.7297671873.js`（新构建） |
| `window.__APP_BUILD__`（**改前**） | `rv.0000.2k6e8r7p` |
| `window.__APP_BUILD__`（**改后**） | `rv.0.0.6.2k6e8r7p` |
| 顶栏 DOM | `[data-testid=system-brand-version]` = `"0.0.6"`；header 首行 `["Toggle Sidebar","New API","0.0.6","主页"]` |
| 系统维护页 | 当前版本 `0.0.6`；构建 ID `rv.0.0.6.2k6e8r7p`；运行时间始于 `2026-10-06 13:57:41`（即本次重启） |

**截图**（本目录）：
- `evidence/before-header.png` —— 改前：顶栏只有「New API」，**无版本**，`rv.0000.2k6e8r7p`
- `evidence/after-header-version.png` —— 改后：顶栏「New API **0.0.6**」
- `evidence/after-system-maintenance.png` —— 系统维护：当前版本 `0.0.6` + 构建 ID `rv.0.0.6.2k6e8r7p`

前后对比同源同页（`/dashboard/overview`），可排除环境差异。

## 两条缺陷的复现步骤（各 ≤3 行）

**E. HTTP 部署下复制按钮失效**
1. 用 `http://<局域网IP>:<端口>` 打开面板（非 localhost、非 HTTPS）；控制台确认
   `window.isSecureContext === false` 且 `navigator.clipboard === undefined`（本机实测：`http://10.0.0.91:3020`）。
2. 进「日志」，打开一条音频/语音日志的预览弹窗，点「复制链接」。
3. 结果：无提示、无复制（`audio-preview-dialog.tsx:123` 无 try/catch、无 await，下一行却 `toast.success`）；
   对比「渠道」页复制正常（走 `useCopyToClipboard` → 有 `execCommand` 兜底）。

**D. 初始化守卫被 localStorage 永久绕过**
1. 任一已初始化的实例先用该浏览器正常访问一次（浏览器写入 `localStorage['setup_status_checked']='true'`）。
2. 让后端回到未初始化状态：清空/重建数据卷后重启（`/api/setup` 返回 `{"status":false}`）。
3. 结果：该浏览器**永远**不再跳转 `/setup`，只会停在登录页并报「用户名或密码错误」，无法自愈；
   而同一实例换新浏览器（或换 IP 访问）会正确跳到 `/setup` —— 本机实测 127.0.0.1 与 10.0.0.91 行为不一致。

## 未做 / 交接

- **未动**「检查更新」直连 `api.github.com`（`update-checker-section.tsx:59-67`）——按 Lead 指示留给后端方案；
  前端语义已修正（`unknown` 不再谎报有新版本），后端一旦在 `/api/status` 给出 `latest_version` 即可直接接入。
- 3020 现已是**已初始化**实例（`setup:true`），`/setup` 向导不会再出现；管理员凭据未落盘。
