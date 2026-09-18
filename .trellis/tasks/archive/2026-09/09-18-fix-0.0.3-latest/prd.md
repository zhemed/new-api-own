# 修复 0.0.3 发布：重建镜像使 latest 回到 0.0.3

## 背景（实测）

- 2026-09-18 历史重写后，我执行了 `git push --force origin v0.0.2 v0.0.3` —— **两个 tag 一起推**，
  `docker-build.yml` 被触发两次；它的 `create_manifests` 每次都重建 `:latest`，v0.0.2 那次后完成，
  于是 registry 的 `:latest` 指向 **v0.0.2** 镜像（digest `094bd74d…`，构建时间 2026-09-18T01:09:49Z）。
- 现状实测：`:0.0.3` 与 `:v0.0.3` → `v0.0.3`（正常）；`:latest` → `v0.0.2`（异常）。

## 目标（本次只做这一件事）

重建 v0.0.3 的镜像发布，使 `:latest` 重新指向 0.0.3 的构建，并核验三个标签指向同一份 0.0.3 镜像。

## 范围

**做：**

1. 触发一次 v0.0.3 镜像重建：`gh workflow run docker-build.yml -R zhemed/new-api-own -f tag=v0.0.3`
   （工作流会先校验该 tag 存在于仓库）。
2. 等待该次运行结束，读取其结论与日志摘要。
3. **只读**核验：`:latest`、`:v0.0.3`、`:0.0.3` 三个标签的容器内版本与 digest 是否一致。

**不做（本次一律不碰）：**

- 不改任何代码、工作流文件、`VERSION`、tag、GitHub Release 及其附件；
- 不重建 v0.0.2，不动 compose、部署机、其它系统；
- 不做"latest 只应由最新版本更新"的结构性改造（另开任务）；
- 不访问除 ghcr.io / github.com（本次任务所必需）之外的任何主机。

## 验收标准

- [x] v0.0.3 的 docker-build 运行完成且结论为 success（run `35319659491`，workflow_dispatch）
- [x] `docker run --rm ghcr.io/zhemed/new-api-own:latest --version` → `v0.0.3`
- [x] `:0.0.3` 与 `:v0.0.3` 仍为 `v0.0.3`，且三个标签 digest 一致（均为 `sha256:3d04916fe29a32f89…`）
- [x] GitHub Release `v0.0.3` 与其附件未受影响（非草稿，7 个附件）

## 结果

| 标签 | 容器内版本 | digest |
|---|---|---|
| `:latest` | `v0.0.3` | `sha256:3d04916fe29a32f89…` |
| `:0.0.3` | `v0.0.3` | 同上 |
| `:v0.0.3` | `v0.0.3` | 同上 |

未做（按范围保留给后续任务）：`latest` 只允许由最新版本更新的结构性改造。

## 风险与处置

- 重建会产生**新的镜像 digest**（`:v0.0.3` / `:0.0.3` 会指向新 digest），这是重建的固有结果；
  Release 二进制不受影响。
- 若工作流失败：**保持现状并如实回报**，不自行扩大操作、不改动工作流补救。

## 完成后

`task.py finish` → `task.py archive --skip-branch-validation` → 写 journal（记录含 digest 核验结果）。
