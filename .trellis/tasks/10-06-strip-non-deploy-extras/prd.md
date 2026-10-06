# 移除部署以外的产物（发布二进制 / Electron / GitCode）

## 用户定调（2026-10-06）

部署 = 这一条命令，其它全部移除：

```bash
docker run -d --name new-api --restart always \
  --network host \
  -v ./data:/data \
  ghcr.io/zhemed/new-api-own:latest
```

## 移除清单（判定：对"拉镜像 → docker run"没有贡献）

| 对象 | 理由 |
|---|---|
| `.github/workflows/release.yml` | 三/单平台二进制 Release，非镜像交付路径 |
| `.github/workflows/electron-build.yml` | 桌面壳打包 |
| `electron/`（整目录）| 桌面壳源码与打包配置 |
| `.github/workflows/sync-release-to-gitcode.yml` | Release 镜像同步，随之失去意义 |
| 文档里的 Electron / 二进制 Release / GitCode 引用 | 与现状不符 |

## 保留清单（删除即影响部署或流程）

- `docker-build.yml`：镜像构建与发布（唯一交付路径）
- `Dockerfile`、源码、`web/`、`relaykit/`、测试、`install-docker.sh`
- `trellis-gate.yml` 与提交闸门、`scripts/forbid-extra-deploy-methods.sh`
- README / MAINTENANCE / `.env.example` / `.trellis/`（改为反映"仅镜像交付"）

## Acceptance Criteria

- [ ] 上述工作流与目录删除，引用同删
- [ ] 镜像流水线未受影响（YAML 校验 + 触发面复核）
- [ ] 文档只剩"镜像 + docker run"一条交付路径
- [ ] 门禁与 CI 通过
