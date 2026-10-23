# 评估并改进 UI 自升级二进制

## 用户要求（2026-10-23）

给出一份 5 项问题的改进规格（P0 权限 / P1 资产签名 / P1 不持久提示 / P2 回滚点 / P2 优雅退出），并要求：
「这个你评估一下，和我们的实际情况对比，然后看着维护。还有个事它已经维护一版了，你知道就行」

→ 即：**先核实仓库实际状态**（可能已被维护过、与规格基线不一致），据此评估取舍，再动手。

## 规格中的事实基线（用户提供，需核实）

1. 已做对：临时文件同目录 + 失败清理 + 校验通过才 `os.Rename` 原子替换；linux/arch 限制、体积/超时上限、单飞锁、单调性检查、下载不带凭据；先 Flush 再延迟 500ms 激活；exec 优先、失败退化为 `os.Exit(0)`；稳定错误码（`checksum_mismatch`/`not_newer`/`busy`）。
2. 作者自陈限制：容器部署下替换的是容器内二进制，重建容器会回退镜像版本。
3. 待改进：
   - P0 权限：`GET /status/update` 与 `POST /status/update/apply` 仅 `AdminAuth`（非 RootAuth），且接受 PAT；
   - P1 签名：校验和与二进制同源，只防损坏不防篡改（二进制无 cosign 痕迹，镜像侧 CI 有 cosign）；
   - P1 提示：无"不持久"提示；
   - P2 回滚：无 `.bak`/prev，且 `not_newer` 拒绝降级；
   - P2 优雅退出：`syscall.Exec` 硬中断在途请求。

## 评估步骤（只读，先做）

1. 读 `service/update_apply.go`、`controller/update_check.go`、`setting/update_check.go`、`router/api-router.go`、`middleware/auth.go` 现状，逐条对照上述 5 项**是否仍然存在**；
2. 查 git log/最近提交，确认"已维护过一版"改了什么（避免重复劳动或回退他人改动）；
3. 查前端更新入口（`web/src/features/system-settings/maintenance/**`）现状：按钮可见性控制、文案、错误码展示；
4. 查 CI：release 资产是否已有签名/attestation 可复用（`docker-build.yml` / `release.yml`）；
5. 输出：每项「现状 → 是否仍需改 → 建议方案与取舍 → 验证方式」。

## 目标（按优先级，视评估结果裁剪）

- P0 权限收紧（RootAuth 或禁 PAT）+ 前端入口可见性同步；
- P1 资产签名校验（cosign / GitHub attestation），失败拒绝替换 + 新稳定错误码；
- P1 "不持久"提示（响应结构化字段 + UI 两处文案）；
- P2 回滚点（保留旧二进制 + 回滚途径，明确是否允许显式降级）；
- P2 优雅退出（drain 评估；不做要给理由与残余影响）。

## 约束

- 遵守 Trellis 流程；提交带 `[task:<slug>]`；
- 文档/提交信息不出现真实域名/IP/端口/主机名（用 `<domain>` 等占位符）；
- 只改上述文件及其测试，不顺手重构；
- 每项改动配测试或可复核命令；无法实测的标"待验证"并写验证方法。

## Acceptance Criteria

- [ ] 5 项逐条给出「现状（实测）→ 结论 → 方案」，与用户基线差异明示
- [ ] 按优先级落地并通过 Go/前端门禁
- [ ] 响应字段与 UI 文案有不持久提示
- [ ] 回滚途径可用或有明确不做理由
- [ ] 交付按五小节 + 证据等级（实测/推断/待查）
