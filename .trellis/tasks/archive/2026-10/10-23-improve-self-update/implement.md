# 实施计划（按顺序，可并行处标注）

## 0. 前置
- [ ] 只读复核现状（已由 Lead 完成：5 项均仍在）
- [ ] 确认签名密钥的生成与存放方式（GitHub Secret 名称、公钥如何内嵌）

## 1. 后端（单成员，独占 `service/update_apply.go`、`controller/update_check.go`、`setting/update_check.go`、`router/api-router.go`、`middleware/auth.go` 及测试）
- [ ] P0：两条路由改 `RootAuth`；`middleware/auth.go` 暴露"本次凭据类型"；handler 拒绝 PAT（返回稳定码，如 `pat_not_allowed`）
- [ ] P1：`SHA256SUMS.sig` 下载 + ed25519 验签（内嵌公钥列表）+ 新错误码 `signature_mismatch`
- [ ] P1：响应加 `persistent=false` + `hint`
- [ ] P2：保留 `.prev`；新增 rollback handler + 路由（RootAuth + 非 PAT）
- [ ] P2：drain —— 写评估结论（本轮暂缓）到代码注释与交付报告
- [ ] 测试：权限矩阵（root/admin/PAT × 两个端点）、签名失败（伪造 .sig → 拒绝 + 原二进制哈希不变）、`.prev` 生成与回滚成功、无 `.prev` 时回滚报明确错误

## 2. 运维/CI（可与后端并行，独占 `.github/workflows/release.yml`）
- [ ] 在 release 流程里对 `SHA256SUMS` 用 ed25519 私钥签名，产出并上传 `SHA256SUMS.sig`
- [ ] 私钥只从 GitHub Secret 读取；文档说明轮换方式
- [ ] 本地可复现：给出用 openssl/python 生成密钥与签名的命令（占位符，不含真实材料）

## 3. 前端（依赖 1 的 API 形状；可先做可见性）
- [ ] 更新按钮按 root 角色显隐（非 root 不显示或禁用并给说明），避免"按钮在但 403"
- [ ] 按钮附近常驻"不持久"提示 + 结果提示里再出现一次
- [ ] 展示新错误码（`signature_mismatch`、`pat_not_allowed`）的可读文案（7 语言走脚本）
- [ ] 回滚入口（若 1 完成）：仅 `.prev` 存在时可用

## 4. 验证与门禁
- [ ] Go：`GOWORK=off go vet/build ./...`、`cd relaykit && GOWORK=off go build ./...`、`make test`
- [ ] 前端：七条门（typecheck/lint/test/format/build/knip/i18n:check）
- [ ] 手工验证命令清单（占位符）：apply 成功/失败、签名失败、回滚

## 5. 发布
- [ ] 版本号、tag、Release、镜像；两端实例按需升级（生产用 SSH 密钥）
