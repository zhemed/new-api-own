# 执行计划：一条命令 compose 部署（new-api-own）

## 前置

- 用户已回答 Q1（目标目录）与 Q2（是否真实执行安装）。
- 分支 `main`；提交信息带 `[task:one-command-compose-deploy]`。

## 检查清单

1. [x] **改造 `docker-compose.yml`**
   - 变量插值 + 保留默认值（见 `design.md` §3）
   - 三个服务加日志轮转上限
   - 验收：不带 `.env` 时 `docker compose config` 与改造前的展开结果**逐字段一致**
     （判别性验证已做：前后各存一份 config 对比，差异**只有**新增的三个 logging 块）
2. [x] **新增 `install-compose.sh`**（仓库根，`chmod +x`，321 行）
   - 按 `design.md` §6 的流程实现；`--help` 取脚本头部注释块；额外加了 `--raw-base`（fork/镜像源）
   - 验收：`bash -n` 通过；`shellcheck` 退出码 0；`--help` 在文件形态与 `curl|bash` 管道形态都可用
3. [x] **E2E（临时目录，不碰真实部署）**
   - `--dir /tmp/e2e-deploy --name-prefix e2e- --project-name e2e-napi-test --no-start` → 检查生成物
   - 去掉 `--no-start` 实起 → 三容器 healthy、`GET /api/status` HTTP 200、脚本退出码 0
   - 收尾：`docker compose down -v`（仅该项目名）+ 删除临时目录；已确认无残留容器/卷
4. [x] **幂等与备份**
   - 重跑 → 被拒（退出码 1）；`--force` → 目录里只有一个 compose 名、备份唯一
   - 判别性：顺序 6 次留 6 份；「秒级 + 无重试循环」的缺陷形态只剩 2 份
   - **实测推翻了初稿断言**：单独去掉亚秒精度不丢备份（重试循环兜住了），已改写结论
5. [x] **口令复用**
   - 连续 `--force` 后两个口令逐字节不变；`psql -U root` 与 `redis-cli ping` 均认证通过
   - 判别性成立：口令若被重新生成，Postgres 必然认证失败
6. [x] **撞名预检**
   - 造一个非 compose 的同名容器 → 脚本在 `up` 之前 die，既有容器状态仍为 `created`，输出给两条解法
7. [x] **文档**
   - `README.md`：新增「方式一：一条命令部署」，原方式顺延为二/三；生产部署参考改指 `.env`
   - `MAINTENANCE.md`：自用部署注意事项加一条；发版流程说明版本字面量不必同步；安全基线补权限自动收紧
8. [x] **仓库边界自检**
   - 跑 `MAINTENANCE.md` 里的 `grep -rnE "10\.|192\.168\.|172\.(1[6-9]…"` 自检：命中项全部为
     既有白名单内容（文档示例、`common/utils.go`、`relaykit` 注释），本次改动未新增内网地址
   - 仓库内 grep 不到生成的口令
9. [x] **质量门禁**
   - `bash -n` + `shellcheck` 通过；本次未改 Go/前端代码，未跑 `make test`
   - `./scripts/check-trellis-gate.sh` 通过
10. [ ] **收尾**
    - `task.py finish` → `task.py archive` → 写 journal
    - 提交带 `[task:one-command-compose-deploy]`（已完成：`a79b054`）

## 回滚点

| 步骤 | 回滚方式 |
|---|---|
| compose 改造 | `git checkout -- docker-compose.yml` |
| 安装脚本 | 删除文件即回到今天（脚本是新文件，不影响既有部署） |
| E2E 临时环境 | `docker compose -p <临时项目名> down -v`（**只针对临时项目名**） |

## 不做

- 不 `docker compose down -v` 任何既有项目；不碰任何真实部署目录。
- 不改网络形态、不挂 `docker.sock`、不动发版流水线。
