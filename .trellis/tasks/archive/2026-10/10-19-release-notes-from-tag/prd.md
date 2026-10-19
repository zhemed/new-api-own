# Release 正文改为 tag 注解 changelog

- 任务：`10-19-release-notes-from-tag`
- 执行：`review-ops`（写范围：仅 `.github/workflows/release.yml`）
- 背景：用户要求 Release 页正文**只写更新日志**，移除部署说明（docker pull / Upgrade 步骤 / 附件说明）。

## 现状

`release.yml` 在发布步骤里把正文硬编码为一段部署说明：

- `BODY="$(printf '%s\n' 'Docker image …' … 'SHA256SUMS.')"`（约 210–221 行），
- 随后 `gh release edit … --notes "$BODY"`（约 228 行）与 `gh release create … --notes "$BODY"`（约 232 行）。

而每个版本的**注解 tag 正文本身就是 changelog**（实测 28/28 个 tag 都有正文）。

## 要求

1. `BODY` 改为取 **tag 注解正文**：`git tag -l --format='%(contents:body)' "$TAG"`；
   正文为空（轻量 tag / 无正文）时回退到提交标题 `git log -1 --pretty=%s "$TAG"`，避免空正文。
2. **其余全部保留**：三个附件上传、附件契约断言循环（`SHA256SUMS` / `new-api-linux-amd64` / `new-api-linux-arm64`）、
   `--clobber`、`--latest` / `--prerelease` 逻辑、幂等的 `edit` / `create` 分支。
3. 不改 `docker-build.yml`；不 commit/push。
4. 删干净旧 `printf` 硬编码正文，不留死代码。
5. shell 语法过 `bash -n`（并说明验证方式）；跑 `./scripts/forbid-extra-deploy-methods.sh` 与
   `./scripts/forbid-encoding-json.sh`，两者必须通过。

## 验收

- diff 摘要；
- `bash -n` 的验证方式与结果；
- 两个门禁命令的实际输出。

## 前置核对

- checkout 使用 `fetch-depth: 0`（release.yml:76），所以 `git tag -l` 能取到完整 tag 与注解正文；
- `release.yml` 已被 git 跟踪（`f5b7b9f` 起），改动可用 `git diff` 展示。

## Notes

- 「附件是给面板更新器用的，不是部署方式」这层说明随正文一起移除，但**附件本身与契约断言保留**
  （面板更新器依赖它们，见 `MAINTENANCE.md`「面板内更新」）。

---

# 执行记录（2026-10-19）

## 改动（`release.yml` 唯一一处块，7 增 13 删）

删除 `BODY="$(printf '%s\n' 'Docker image …' … 'SHA256SUMS.')"`（原 210–222 行），替换为：

```bash
# Release notes ARE the annotated tag's own message body — our tags
# carry the changelog. Fall back to the commit subject when the tag has
# no annotation body (lightweight/empty tag) so the body is never blank.
BODY="$(git tag -l --format='%(contents:body)' "$TAG")"
if [ -z "$(printf '%s' "$BODY" | tr -d '[:space:]')" ]; then
  BODY="$(git log -1 --pretty=%s "$TAG")"
fi
```

**未改动**（`git diff` 只含这一处 hunk，已核对）：三附件上传 `gh release upload "$TAG" $ASSETS --clobber`、
附件契约断言循环（`SHA256SUMS` / `new-api-linux-amd64` / `new-api-linux-arm64`）、
`LATEST_FLAG` / `PRERELEASE_FLAG`、`edit` / `create` 幂等分支、`--verify-tag`、`docker-build.yml`。

## 验证

1. **残留检查**：`grep -n "Docker image (multi-arch\|only supported deployment\|Attached Linux binaries"` → 空
   （`docker pull` 也不再出现在 release.yml 里）。
2. **YAML**：`python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"` → `OK`。
3. **shell 语法**：用 YAML 解析取出 release.yml 里**全部 6 个 `run:` 块**，逐个写临时文件跑 `bash -n`
   → `run[0..5] 全部 OK`（被改的那个 113 行块也 OK）。这比 `sed -n '/run:/,$p'` 更严格，
   因为后者会把 YAML 键混进 shell 里导致假失败。
4. **机制实测**（真实 tag）：
   - `git tag -l --format='%(contents:body)' v0.0.6` → 3 条 changelog 正文；`v0.0.5` → 2 条；
   - 全量统计：`29` 个 tag 里只有 1 个（轻量 tag `pre-rollback-backup-20260921`）`contents:body` 为空，
     回退分支实测输出其提交标题；**28 个发布 tag 全部有注解正文**（与 Lead 给的事实一致）。
5. **仓库门禁**：
   - `./scripts/forbid-extra-deploy-methods.sh` → `✅ 部署方式唯一：docker run + ghcr.io/zhemed/new-api-own`（exit 0）
   - `./scripts/forbid-encoding-json.sh` → `✅ No forbidden encoding/json usage`（exit 0）

## 说明

- 正文只从 tag 注解取，因此**发版流程不变**（`git tag -a v<版本> -m "…"` 的 `-m` 就是 changelog）；
  没有注解正文的轻量 tag 会退化成提交标题，属"极少见"的兜底。
- 未 commit/push。

