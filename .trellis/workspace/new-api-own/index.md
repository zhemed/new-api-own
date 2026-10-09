# Workspace Index - new-api-own

> Journal tracking for AI development sessions.

---

## Current Status

<!-- @@@auto:current-status -->
- **Active File**: `journal-1.md`
- **Total Sessions**: 49
- **Last Active**: 2026-10-09
<!-- @@@/auto:current-status -->

---

## Active Documents

<!-- @@@auto:active-documents -->
| File | Lines | Status |
|------|-------|--------|
| `journal-1.md` | ~1601 | Active |
<!-- @@@/auto:active-documents -->

---

## Session History

<!-- @@@auto:session-history -->
| # | Date | Title | Commits | Branch |
|---|------|-------|---------|--------|
| 49 | 2026-10-09 | 剩余三项：依赖/抑制/死代码全清，knip 四类归零；398 条 exports/types 停手留痕 | `b33c1fa` | `main` |
| 48 | 2026-10-09 | 三项收尾并发 v0.0.13；实例一次升级到最新（KPI 上色实测生效） | `550965e` | `main` |
| 47 | 2026-10-09 | 四项收尾（KPI 上色/mock 标注/lint 清理/文档订正）并发 v0.0.12 | `a2ad71b` | `main` |
| 46 | 2026-10-09 | 手机端上色实为已生效（更正）；盘点 6 类未处理问题 | `c1b2b23` | `main` |
| 45 | 2026-10-09 | 核查使用日志上色（通过）并确认维持 50/100 速度段 | `6287c58` | `main` |
| 44 | 2026-10-09 | 核查使用日志上色：已生效；但 50/100 阈值几乎全绿，建议按真实分布校准 | `be63897` | `main` |
| 43 | 2026-10-09 | 用户决定保持内存日志（接受更新清历史） | `a1e4bf3` | `main` |
| 42 | 2026-10-09 | 实例升级 v0.0.11；升级清空内存日志（预期），彩色 t/s 待新请求 | `d9bad5e` | `main` |
| 41 | 2026-10-09 | 使用日志表 t/s 上色（补齐漏掉的渲染点）并发 v0.0.11；更正 lint 归因 | `c8c308a` | `main` |
| 40 | 2026-10-09 | 用面板更新器把实例升到 v0.0.10 并验证（含会话失效观察） | `1f286dd` | `main` |
| 39 | 2026-10-09 | 发 v0.0.10（TPS 上色）；偶发构建失败用重跑解决 | `1cd7307` | `main` |
| 38 | 2026-10-09 | TPS 数值按阈值上色（红<50 / 黄50-100 / 绿≥100，无数据不上色） | `e59b721` | `main` |
| 37 | 2026-10-09 | 更正：截图速度是真实指标；mock 只覆盖图表/可用率等区块 | `b936322` | `main` |
| 36 | 2026-10-07 | 主仓文档一致性：10 处 docker run 命令统一为当前日志形态 | `03e82a8` | `main` |
| 35 | 2026-10-07 | 评估文件日志对内存日志目的的影响；给 docker 日志加上限并同步 README | `6e3effd` | `main` |
| 34 | 2026-10-07 | 核验并解释：内存用量日志 vs /data/logs 文件日志 | `7be4308` | `main` |
| 33 | 2026-10-07 | README 部署命令改为当前日志内存模式（与实例逐项一致） | `4a89762` | `main` |
| 32 | 2026-10-07 | 评估：能否把 opencode 头做进源码（结论：用现成模板+复制，不改码） | `56af210` | `main` |
| 31 | 2026-10-07 | 保存 opencode 渠道请求头覆盖并验证测试通过（含一条自我更正） | `5058c8b` | `main` |
| 30 | 2026-10-07 | 排查 opencode 渠道 400：请求头覆盖写法正确但未保存 | `34a77df` | `main` |
| 29 | 2026-10-07 | 讨论：0.0.3 数据能否直接用 0.0.9 还原（结论：可以） | `379d98f` | `main` |
| 28 | 2026-10-07 | 讨论：0.0.3 数据能否直接用 0.0.9 还原（结论：可以） | `3fb2ded` | `main` |
| 27 | 2026-10-06 | 修复 v0.0.8 回归（zhCN 语言标签致日志页崩溃）并发 v0.0.9 | `96f4881` | `main` |
| 26 | 2026-10-06 | v0.0.8：日志改为字节预算（写入触发），去掉 5 分钟定时清理 | `f2b3577` | `main` |
| 25 | 2026-10-06 | 用户实例改为内存日志（20万/7天/5分钟）并验证 | `3c4243d` | `main` |
| 24 | 2026-10-06 | 维护：双写法拉取验证、补 v0.0.6 Release、清 16 个过时资产 | `2085df9` | `main` |
| 23 | 2026-10-06 | 面板内自更新交付：v0.0.7 发布并端到端验证通过 | `f5b7b9f` | `main` |
| 22 | 2026-10-06 | 被指出：团队没发现线上还是 0.5——立版本对齐核查，刷新本机 latest 标签 | `16f9d8f` | `main` |
| 21 | 2026-10-06 | 本机演示实例真机验证通过（0.0.6）；生产升级经用户选择跳过 | `1addd9f` | `main` |
| 20 | 2026-10-06 | 团队处理开放项：前端恢复功能+补齐 5 个缺键，后端堵住 sora 计费上界绕过 | `90a4f19` | `main` |
| 19 | 2026-10-06 | 团队复查本轮改动：删码安全、消毒逻辑站得住，但我的机制描述错了 | `d04893a` | `main` |
| 18 | 2026-10-06 | 补齐前端工具链：bun+jsdom+knip，抓出消毒测试失效并清 33 个死文件 | `58e028d` | `main` |
| 17 | 2026-10-06 | 收口：删零引用文件、补已知不一致、发版 0.0.6 验证前端 | `c2458ca` | `main` |
| 16 | 2026-10-06 | 团队全面审查与维护（3 成员 + Lead 复核） | `ecc5642` | `main` |
| 15 | 2026-10-06 | 瘦身：部署只剩镜像，其余产物全清 | `35da53c` | `main` |
| 14 | 2026-10-06 | 项目维护轮次：文档对齐 + 陈旧内容清理 | `b39314f` | `main` |
| 13 | 2026-10-06 | systemd unit 删除：部署方式唯一化收口 | `ad0713d` | `main` |
| 12 | 2026-10-06 | compose 清理与唯一部署方式（机械闸门） | `b726ae9` | `main` |
| 11 | 2026-10-06 | 弱盘日志：提交发版 0.0.4 + 本机演示实例 | `7f9a50b` | `main` |
| 10 | 2026-10-06 | 弱盘日志：LOG_SQL_DSN 支持内存库/独立 SQLite + 清理可调度 | - | `main` |
| 9 | 2026-10-06 | 评估：弱盘机器把日志放内存 | - | `main` |
| 8 | 2026-09-18 | Status check: v0.0.3 not released yet | - | `main` |
| 7 | 2026-09-18 | Investigation: x-opencode-session / （实例兜底值已脱敏） channel override | - | `main` |
| 6 | 2026-09-18 | Trellis commit gate enabled (3 layers) | `47c61db` | `main` |
| 5 | 2026-09-18 | Maintainer onboarding flow documented and verified | `2bc0e1a` | `main` |
| 4 | 2026-09-18 | Release v0.0.2: dual image tags, GitHub Release, version-injection fix | `6829e8e` | `main` |
| 3 | 2026-09-18 | Audit: VERSION 0.0.1 origin and the （内网地址已脱敏） reference | - | `main` |
| 2 | 2026-09-18 | Disable global web/API/critical/search rate limiters | `a4e74df` | `main` |
| 1 | 2026-09-18 | Trellis bootstrap: backend spec filled from real code | `8e62ba2` | `main` |
<!-- @@@/auto:session-history -->

---

## Notes

- Sessions are appended to journal files
- New journal file created when current exceeds 2000 lines
- Use `add_session.py` to record sessions