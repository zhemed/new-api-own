# 证据：docker-build 触发条件修复 + 探针标签清理

任务 `10-06-fix-docker-build-trigger`。只读网络，未 push、未触发 workflow。

## R1 触发条件前后对照

```diff
 on:
   push:
+    # Only version tags may publish. ...
     tags:
-      - '*'
-      - '!nightly*'
+      - 'v[0-9]*'
+      - '[0-9]*'
   workflow_dispatch:
```

- 改前 `'*'`：**任何** tag 都触发；workflow 又**无条件**用 `imagetools create` 重建
  `:latest-{arch}` → `:latest`，于是非发布 tag（如 `pre-rollback-backup-20260921`）会把
  `:latest` 指到非发布快照。`'!nightly*'` 只挡 `nightly*` 前缀，挡不住其它 tag。
- 改后 `'v[0-9]*'` + `'[0-9]*'`：只接受版本 tag。
  **保留 `[0-9]*` 的理由**：workflow 自身设计（`TAG_ALIAS` 步骤 + "alias without v" 清单步骤）
  就是"一次运行同时发布 `v0.0.6` 与 `0.0.6`"；只留 `v*` 会静默丢掉"不带 v 也能触发"这一既有能力，
  属于超出本次缺陷的回归。
- `'!nightly*'` 在新条件下已是死规则（`nightly*` 不匹配任一正模式），故移除并在注释中说明。

## R2 校验输出

```
YAML parse: OK (safe_load succeeded)
triggers keys : ['push', 'workflow_dispatch']
push.tags     : ['v[0-9]*', '[0-9]*']
workflow_dispatch present: True

tag                              expect   actual   verdict
v0.0.6                           True     True     PASS
0.0.6                            True     True     PASS
v1.2.3-alpha.1                   True     True     PASS
pre-rollback-backup-20260921     False    False    PASS
nightly-2026                     False    False    PASS
release-notes                    False    False    PASS
main                             False    False    PASS

ALL GLOB CASES: PASS

--- downstream regression guards ---
PASS provenance: mode=max
PASS sbom: true
PASS cosign sign (build)
PASS cosign sign manifests
PASS manifest order TAG->latest->alias
PASS TAG_ALIAS logic kept
```

项目自带闸门：`bash scripts/forbid-extra-deploy-methods.sh` → `✅ 部署方式唯一` exit 0。

diffstat：`1 file changed, 13 insertions(+), 2 deletions(-)`（全部位于 `on:` 块内）。

## R3 探针标签清理

```
=== BEFORE ===
ghcr.io/zhemed/new-api-own:zz-local-stale-probe   5e2309035332   25.9kB   9.49kB
=== RMI ===
Untagged: ghcr.io/zhemed/new-api-own:zz-local-stale-probe
=== AFTER ===
(empty - OK)
=== hello-world body intact? ===
hello-world:latest 5e2309035332
```

只解除别名，`hello-world` 本体（同一 ID `5e2309035332`）未被连带删除。

## R4 附：固定 tag 也会陈旧（实测）

本机本地标签记录的 RepoDigest 与远端当前 index 摘要对比：

| tag | 远端 index | 本地 RepoDigest | 判定 |
|---|---|---|---|
| `0.0.2` | `094bd74d…` | `e16437fd…` | **陈旧** |
| `v0.0.4` | `d159a4df…` | `e850cb5d…` | **陈旧** |
| `0.0.5` | `79f4bdcb…` | `79f4bdcb…` | 一致 |
| `0.0.6` / `latest` | `a74d0a6c…` | `a74d0a6c…` | 一致 |

原因：workflow 重跑时 `docker buildx imagetools create -t :${TAG}` 会**重新生成同名版本 tag**
（新摘要）——版本 tag 在 registry 上是**可变**的。
结论：改用固定 tag 能防 `:latest` 漂移，但**不能**免除升级第一步的 `docker pull <tag>`。

## R5 遗留（未修，需决策）

失效模式 #5 仍在：`workflow_dispatch` 手动重建**旧** tag 时，`build_single_arch` 仍会覆盖全局
`:latest-{arch}`，`create_manifests` 随即用它们重建 `:latest` → `:latest` 静默**回退**到旧版本。
需要额外的"只有最新版本才允许移动 `:latest`"保护；本轮未实现以免把改动面扩大到发布语义。
