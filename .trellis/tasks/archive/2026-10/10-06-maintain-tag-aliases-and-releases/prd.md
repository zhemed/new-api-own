# 维护：双写法拉取 / v0.0.6 Release / v0.0.5 过时资产

## 用户指示（2026-10-06）

「v 和不带改成都能拉取，剩下两个你随便维护一下就行了」

## 要做

1. **证明并保证两种写法都可拉取**：`ghcr.io/zhemed/new-api-own:{v0.0.7,0.0.7,latest}` 三者必须同一摘要；
   实际 `docker pull` 两种写法并比对本机镜像 ID；确认 `docker-build.yml` / `release.yml`
   对 `v<数字>` 与 `<数字>` 两种 tag 拼写都发布同一产物（TAG_ALIAS 逻辑）。
2. **补 v0.0.6 的 Release**（用新工作流的 `workflow_dispatch`，不改写 tag）——同时验证修好的
   `isLatest` 校验逻辑在真实 CI 里跑通。
3. **清掉 v0.0.5 Release 上过时的二进制资产**（旧命名 `new-api-v0.0.5`/`new-api-arm64-v0.0.5`/
   `checksums-linux.txt`，与"唯一部署方式=镜像"冲突且不被更新器使用）；**保留 Release 条目本身**。

## Acceptance Criteria

- [ ] 两种写法的拉取实测证据（摘要/镜像 ID 相同）
- [ ] v0.0.6 Release 存在且带三资产，v0.0.7 仍为 Latest
- [ ] v0.0.5 的过时资产已清、Release 条目保留
- [ ] 全部变更留痕并推送；生产未动

## 执行结果（2026-10-06）

### 1) 双写法拉取：实测通过

```
docker pull ghcr.io/zhemed/new-api-own:0.0.7  → 本机 ID 68c9842d16fb
docker pull ghcr.io/zhemed/new-api-own:v0.0.7 → 本机 ID 68c9842d16fb   （同一镜像）
registry 摘要：latest = 0.0.7 = v0.0.7 = sha256:68c9842d…
              0.0.6 = v0.0.6 = sha256:a74d0a6c…
```
工作流对 `v<数字>` 与 `<数字>` 两种 tag 拼写都会发布（`TAG_ALIAS` 逻辑），所以以后写哪种都能拉。

**顺手修的隐患**：本机 `latest` 标签又陈旧了（`docker pull :0.0.7` **不会**移动 `latest`）→ 已 `docker pull :latest` 刷新到 0.0.7。
结论重申：升级**必须显式 `docker pull`**，这条已写进 README/MAINTENANCE。

### 2) 补发 v0.0.6 Release（不改写 tag）

`gh workflow run "Publish release (image + Linux updater binaries)" -f tag=v0.0.6` → **run 成功**，
v0.0.6 获得三资产（`new-api-linux-amd64` / `-arm64` / `SHA256SUMS`），且 **v0.0.7 仍是 Latest**。
这次真实 CI 运行同时验证了修好的 `/releases/latest` 校验逻辑（v0.0.6 正确判为 `--latest=false`）。

### 3) 清理过时资产（条目保留）

| Release | 删除 | 剩余 |
|---|---|---|
| v0.0.5 | `new-api-v0.0.5` / `new-api-arm64-v0.0.5` / `checksums-linux.txt` | 0 |
| v0.0.4 | `new-api-macos-v0.0.4` / `checksums-macos.txt` | 0 |
| v0.0.3 | macos+windows+旧命名 linux 共 7 个 | 0 |
| v0.0.2 | 同上 7 个 | 0 |

合计 **16 个资产**（含被明令放弃的 macOS/Windows 二进制），释放 700+ MB。
现在只有 v0.0.6 / v0.0.7 带**新命名 Linux 资产**；所有 Release 条目与 tag 全部保留（可回滚识别、可按 tag 重建）。

### 最终状态

`releases/latest = v0.0.7`（面板读的就是它）；镜像 `latest = 0.0.7`；工作树干净。
