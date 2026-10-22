# 排查：更新有问题（x.miio.cc / zzds.de）

## 用户报告（2026-10-23）

「你这更新的有问题呀」+ 面板凭据（x.miio.cc / 账号 z）+ SSH 凭据（zzds.de / root）。

## 红线遵守

- 凭据**不复述、不写入任何文件（含仓库/任务记录/journal）、不外发**；
- 两个主机由用户**点名**，因此只读访问属授权范围；SSH 写操作需另行判断必要性；
- 排查完成后向用户建议**更换密码**（密码已出现在会话记录里）。

## 排查顺序（只读优先）

1. `x.miio.cc` 是否就是本机 3000 端口的实例（同 IP / 同版本 / 同数据）——决定"是不是我这几版改动造成的"；
2. 本机实例当前版本与关键配置（币种、模型目录、峰谷表达式）快照；
3. 若为同一实例：列出我这几版改动的**全部可回滚点**（数据库备份、option 原值导出、代码 tag）；
4. 询问用户具体症状（哪个页面/哪个功能），再对症处理。

## Acceptance Criteria

- [ ] 明确 x.miio.cc 与本机实例的关系
- [ ] 给出症状定位与回滚路径
- [ ] 凭据零落盘、事后提醒改密

## 用户症状（2026-10-23）

「模型定价和模型数量跟我们本地不一样」——即 `x.miio.cc`（生产，v0.0.41）与本机实例（v0.0.41）的模型目录不一致。

## Lead 判断（基于已查事实）

- 本机实例：我直接写过它的数据库 option → `ModelRatio` 等各 1 条（仅 deepseek-flash）；
- x.miio.cc：解析到另一台机器，**其数据库我从未改动** → 目录仍是它自己保存的那份（完整）；
- 代码默认值（v0.0.40/41 裁到只剩 deepseek）只影响"从未保存过该 option"的实例；
- 因此两边出现差异属**预期内**，不是运行时缺陷。

## 待用户决定

1. 目标状态：生产也对齐到"只剩 deepseek-flash"，还是本地恢复成完整目录？
2. 排查方式：由用户在其服务器上执行只读命令并粘贴结果（推荐，避免凭据扩散），还是授权 Lead 直接使用其凭据登录排查。

## 调查与处理结果（2026-10-23）

### 关系确认

- `x.miio.cc`（生产，解析到另一台机器）与本机实例**不是同一台**：生产日志 5866 条、本机 1 条；
- 两边版本都是 **v0.0.41**（生产容器名为 `ghcr.io/zhemed/new-api-own:v0.0.41`，12 分钟前重建）；
- 生产库位于 `/root/new-api/data/one-api.db`（容器挂载 `/root/new-api/data → /data`）。

### 差异根因（用户报告"模型定价和数量跟本地不一样"）

生产库的定价 option 与本地不同：

| 键 | 生产（改前）| 本地 |
|---|---|---|
| ModelRatio | 4 条 | 1 条 |
| **CacheRatio** | **76 条**（claude/旧模型全在）← 表格多出行数的主因 | 1 条 |
| CompletionRatio | 7 条 | 1 条 |
| 币种 option | 无（走代码默认 CUSTOM/¥/1.0）| 显式设置 |

### 处理（用户授权"生产也对齐"）

1. **备份**：`/root/new-api/data/one-api.db.bak-align-20261022-175404`（6.2MB）；
2. 通过生产本机 `PUT /api/option/` 写入 9 个键：
   `ModelRatio={"deepseek-flash":0.5}`、`CompletionRatio={"deepseek-flash":4}`、`CacheRatio={"deepseek-flash":0.02}`、
   `ModelPrice={}`、`AudioRatio={}`、`AudioCompletionRatio={}`、
   `quota_display_type=CUSTOM`、`custom_currency_symbol=¥`、`custom_currency_exchange_rate=1.0`；
3. **复核**：`/api/status` version=v0.0.41 / CUSTOM / ¥；定价表并集 = **1**（`deepseek-flash`）；abilities 只有 `deepseek-flash`；
4. **会话清理**：删除本次产生的 1 条 curl 会话，保留用户 19 条浏览器会话。

### 过程中的两处 Lead 失误（如实记录）

1. 探测账号角色时用 `head -c 200` 打印了响应 → **误把 access_token 输出到会话记录** ✗ → 已注销该会话并清理 curl 会话；后续改用 python 只取字段、不再打印令牌；
2. 远程脚本对**字符串型** option 用了 `json.loads` → 报"无效的参数" → 改为直接传字符串后成功。

### SSH 主机指纹变更

连接 `zzds.de` 时报 `REMOTE HOST IDENTIFICATION HAS CHANGED`：
- 本机 known_hosts 原有 2 条 `zzds.de` 指纹（ED25519 + RSA），远端现提供第 3 个 ED25519 指纹；
- 用户确认「**重装/换过机器**」后才执行 `ssh-keygen -R zzds.de`（自动备份 `known_hosts.old`）并重连；
- 该地址解析到**私有网段**（不写入任何记录）。

### 待办/建议

- 密码已出现在会话记录中 → 建议改为 **SSH 密钥**登录并更换面板密码；
- 生产面板刷新后应显示「总计 1」。
