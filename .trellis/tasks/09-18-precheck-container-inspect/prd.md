# 修 install-compose.sh 预检误报：docker inspect 同时匹配镜像

## Goal

实装到 `/opt/docker/new-api-own` 时，脚本在**启动前预检**这一步误报：

```
[错误] 已有同名容器 "redis"（项目目录：未知/非 compose）
```

但 `docker ps -a` 里根本没有名为 `redis` 的容器。根因：`docker inspect <名字>` **同时匹配容器与镜像**，
而本项目的容器名 `redis` / `postgres` 恰好与 `docker-compose.yml` 自己要拉的镜像
（`redis:latest`、`postgres:15`）同名 —— 于是**镜像一旦存在**（第一次部署后必然如此），
预检就把镜像当成容器，脚本从第二次起永远拒绝运行。

带 `--name-prefix e2e-` 的 E2E 恰好绕开了这个组合（`e2e-redis` 与镜像名 `redis` 不同），所以没测出来。

## Requirements

- R1 容器名预检必须只查**容器**：`docker inspect` → `docker container inspect`（三处调用 + 提示文案里的示例命令）。
- R2 行为不倒退：真有同名容器时仍要在 `compose up` **之前**拦下，并给两条可操作解法。
- R3 补判别性验证（三条同时成立才算过）：
  1. 只有同名**镜像**、无同名容器 → 放行（旧命令在此误报）；
  2. 真有同名**容器** → 拦下；
  3. 冲突容器不被脚本删改。
- R4 范围克制：本次只修这一个缺陷，不顺带重构安装脚本。
- R5 走闸门：建任务 → start → 提交带锚点 → journal → 归档。

## Acceptance Criteria

- [x] `docker inspect redis` 回显 exit=0（误报可复现）/ `docker container inspect redis` exit≠0（放行）
- [x] 造一个真实同名容器 → `docker container inspect` exit=0（仍能拦下）
- [x] 脚本内不再有裸 `docker inspect`（4 处全部改为 `docker container inspect`，含提示文案）
- [x] `bash -n` + `shellcheck` 通过
- [x] 用**发布的**一行命令在 `/opt/docker/new-api-own` 实装成功，且**第二次重跑不再误报**
- [x] journal + 归档完成

## 范围外

- 不改部署形态、不改 `.env` 口令策略、不动任何既有容器（komari / litepan）。
- 不重写 E2E 为"用默认容器名"（那会与真实部署抢名字）；本缺陷的回归靠上面 R3 的三条直接验证覆盖。
