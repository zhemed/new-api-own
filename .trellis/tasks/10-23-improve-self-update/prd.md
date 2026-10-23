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

## 交付结果（2026-10-23，v0.0.43）

### 三条线产出

| 线 | 内容 | 证据 |
|---|---|---|
| 后端（task-44）| 两路由 `RootAuth` + 新增 rollback 路由；`RequireSessionCredential` 拒绝 PAT（`pat_not_allowed`）；`SHA256SUMS.sig` ed25519 验签（先于二进制下载，失败 `signature_mismatch` 502 且不触碰现有二进制）；`persistent`/`hint`；`.prev` 回滚点 + `Rollback()`；状态加 `rollback_available`；drain 评估暂缓写入注释 | `make test` exit 0（42 包）；权限矩阵/伪造签名（原二进制哈希不变）/回滚点/`rollback_available` 三分支测试 |
| CI（task-45）| `release.yml` 签名步骤（`-rawin` 签 `SHA256SUMS` 原始字节 → base64 一行 `SHA256SUMS.sig`）+ secret↔公钥自检 + 四件套资产断言；`docs/release-signing.md`；`scripts/print-ed25519-pubkey.sh`；`UPDATE_SIGNING_KEY` 已写入仓库 secret | YAML OK；7 个 run 块 `bash -n` 全过；临时密钥签名/验签成功、改一字节失败；`gh secret list` 见名；两道门禁 + Trellis 闸门 exit 0 |
| 前端（task-46）| 更新入口仅 root 显隐 + 非 root 说明；不持久提示两处（优先渲染后端 `hint`）；6 个新码 + 3 别名文案（7 语言，+14 键）；回滚入口（二次确认、`rollback_available=false` 禁用并解释） | 七条门全绿；**408 pass**（+15 用例）；i18n 键 5304 → 5318/语言 |

### Lead 独立复核

- **公钥两端一致** ✓（CI 常量 `Jpj2LFm38b+…` 44 字符 == Go 内嵌，最关键的发布风险已排除）；
- 复跑 `go build` ✓、`make test` **42 包 ok** ✓、gofmt 干净 ✓；`bunx tsgo -b --force` exit 0 ✓、`bun test` **408 pass** ✓；
- 抽查代码：三条路由均 `RootAuth`（`api-router.go:27-29`）、`CodePATNotAllowed`/`RequireSessionCredential`（`auth.go:139-160`）、`updatePreviousBinarySuffix`、`ed25519.Verify(publicKey, sumsContent, signatureBytes)` 均在位。

### 发布与实例

- **v0.0.43**：Release + 镜像工作流均成功；`latest = v0.0.43`；**Release 资产含 `SHA256SUMS.sig`** ✓（签名链路打通）；
- 两端升级：本地 v0.0.43 ✓、生产 v0.0.43 ✓（`RestartCount=0`）。

### 实机验收（逐条）

1. **权限**：root 会话 `GET /api/status/update` → **200** ✓；普通 admin/PAT 路径由权限矩阵单测覆盖（root 会话 / 普通 admin / root PAT / 匿名 × 3 端点，含状态码与 code）。
   ⚠️ **PAT 实机测试未能完成**：该账号 DB 里 `users.access_token` 为空（未配置系统访问令牌），我用 API 令牌误测时得到 401 `AUTH_UNAUTHORIZED`（未走到新分支）。**结论以单测为准**，属"实测缺口"。
2. **签名失败**：单测覆盖（伪造/缺失/非法 `.sig` → `signature_mismatch` + 原二进制 sha256 前后一致 + 无 `.prev`/无临时文件）✓；**未做真实资产演练**。
3. **不持久提示**：后端字段 `persistent:false` + `hint` 已实现并测试；前端两处文案 + 优先渲染后端 `hint` ✓。实机 apply 响应因 `syscall.Exec` 立刻替换进程，抓不到完整体（预期行为）。
4. **回滚演练**：**未能实机演练** —— `.prev` 是 v0.0.43 新能力，而两端都是从 v0.0.42（旧代码，不写回滚点）升上来的，故 `/new-api.prev` 不存在、`rollback_available=false`、调用返回 `no_rollback_point`（**语义正确**）。要做真实演练需下一次发布（v0.0.44）时进行；单测已覆盖生成/回滚/无回滚点/非 ELF 拒绝。
