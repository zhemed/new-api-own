# Release 正文改为更新日志

## 用户意见（2026-10-19）

截图 v0.0.29 Release 页：正文是「Docker image（唯一支持的部署方式）/ docker pull 命令 / Upgrade 步骤 / 附件 binaries 说明」。
用户：「这些都没必要，**全部移除**，应该写**更新日志**」。

## 只读调查

1. `release.yml` 里 Release 正文（body）是怎么生成的（heredoc / `--notes` / 变量）；
2. 我们的 tag 注解本身就是 changelog（每版 `git tag -a -m "vX\n- ..."`）→ 能否直接取来当 body；
3. 附件（`new-api-linux-amd64` / `-arm64` / `SHA256SUMS`）必须保留（面板内更新器按固定名下载）；
4. 是否回填历史 Release 正文（v0.0.2 … v0.0.29）。

## Acceptance Criteria

- [ ] 给出正文生成位置（文件:行号）
- [ ] 给出改造方案：body = tag 注解原文；附件与 Release 本身不变
- [ ] 给出历史 Release 回填方案与风险（需用户确认）

## 调查结论（2026-10-19，只读）

### 正文生成位置

`.github/workflows/release.yml:210-221` —— 一段硬编码 `BODY="$(printf '%s\n' 'Docker image...' '```' ... 'Attached Linux binaries...')"`，
随后 `gh release edit/create ... --notes "$BODY"`（`:228` / `:232`）。

### 现成的 changelog 来源

我们每个版本都用注解 tag 发布（`git tag -a -m "vX.Y.Z\n- ..."`），**注解正文本身就是要的更新日志**：

```console
$ git tag -l --format='%(contents:body)' v0.0.29
- 消耗分布 / 模型调用分析 两处图表标题：『总计』数值改为 text-foreground + font-medium + tabular-nums
- 标签保持 text-muted-foreground，字号与布局不变（仍 text-xs）
...
```

### 改造方案

1. `BODY="$(git tag -l --format='%(contents:body)' "$TAG")"`；
   若为空（极少数未写注解正文的 tag）→ 回退 `git log -1 --pretty=%s`，避免空正文；
2. **其余全部保留**：附件三件套（`new-api-linux-amd64`/`-arm64`/`SHA256SUMS`）、附件契约断言、`--latest` 逻辑；
3. 可选回填：对既有 Release 执行一次 `gh release edit <tag> --notes "<tag 注解正文>"`（不动 latest 标记、不动附件）。

## 风险

- 低：只改 Release 正文文本；不影响面板更新器（它读 `releases/latest` 与**固定附件名**，不解析正文）；
- 回填历史 Release 会改写公开页面内容（只改正文，不删附件、不动 tag/commit）。

## 执行结果（2026-10-19）

用户选"改工作流 + 回填全部 28 个"：

### 工作流（`.github/workflows/release.yml:213-216`）

```bash
BODY="$(git tag -l --format='%(contents:body)' "$TAG")"
if [ -z "$(printf '%s' "$BODY" | tr -d '[:space:]')" ]; then
  BODY="$(git log -1 --pretty=%s "$TAG")"   # 无注解正文时回退提交标题
fi
```

- 删除原硬编码的部署说明帧（`release.yml:210-221`）；
- **保留**：附件三件套上传与契约断言、`--clobber`、`--latest`/`--prerelease`、幂等 edit/create 分支；
- 验证：`bash -n` 校验全部 `run` 块 **exit 0**；`forbid-extra-deploy-methods.sh` ✓、`forbid-encoding-json.sh` ✓。

### 回填（28 个 Release）

- 对全部 28 个已有 Release 执行 `gh release edit <tag> --notes "<tag 注解正文>"` → **成功 28 / 失败 0**；
- 抽验 v0.0.29 / v0.0.20 / v0.0.5 正文均为 changelog；
- `releases/latest` 仍为 **v0.0.29**；附件仍为 `new-api-linux-amd64, new-api-linux-arm64, SHA256SUMS`（未动）✓。

### 生效时机

工作流改动对**下一个 tag** 的发布生效；面板更新器不受影响（它读 `releases/latest` 与固定附件名，不解析正文）。
