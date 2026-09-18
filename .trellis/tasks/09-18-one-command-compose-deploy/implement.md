# 执行计划：一条命令 compose 部署（new-api-own）

## 前置

- 用户已回答 Q1（目标目录）与 Q2（是否真实执行安装）。
- 分支 `main`；提交信息带 `[task:one-command-compose-deploy]`。

## 检查清单

1. [ ] **改造 `docker-compose.yml`**
   - 变量插值 + 保留默认值（见 `design.md` §3）
   - 三个服务加日志轮转上限
   - 验收：不带 `.env` 时 `docker compose config` 与改造前的展开结果**逐字段一致**
     （这是"向后兼容"的判别性验证：`docker compose config` 前后各存一份对比）
2. [ ] **新增 `install-compose.sh`**（仓库根，`chmod +x`）
   - 按 `design.md` §6 的流程实现；`--help` 取脚本头部注释块
   - 验收：`bash -n` 通过；`--help` 输出完整用法
3. [ ] **E2E（临时目录，不碰真实部署）**
   - `--dir /tmp/xxx --name-prefix t- --no-start` → 检查生成物（compose/.env/权限）
   - 去掉 `--no-start` 实起 → 三容器 `healthy`、`GET /api/status` 返回 success
   - 收尾：`docker compose down` + 删除临时目录（**不动命名卷以外的任何既有资源**）
4. [ ] **幂等与备份**
   - 重跑 → 被拒；`--force` → 目录里只有一个 compose 名、备份唯一
   - 判别性：把备份名里的微秒去掉（临时改一份副本）→ 同秒连跑出现覆盖 → 恢复
5. [ ] **口令复用**
   - 重跑后 `.env` 哈希不变、Postgres 认证正常
   - 反向：临时改成每次重生成 → 部署起不来（证明确实测到了这条路径）
6. [ ] **撞名预检**
   - 起一个挂到别的项目目录的同名容器 → 脚本在 `up` 之前 die 并给解法
7. [ ] **文档**
   - `README.md`：快速开始加「一条命令部署」
   - `MAINTENANCE.md`：部署段引用脚本；发版流程补「安装脚本按 ref 取 VERSION，无需同步字面量」；
     「部署安全基线」里补一句脚本已自动收紧 `data`/`logs` 权限
8. [ ] **仓库边界自检**
   - 跑 `MAINTENANCE.md` 里的 `grep -rnE "10\.|192\.168\.|172\.(1[6-9]…"` 自检
   - 确认仓库内无生成的口令、无内网地址、无运维机路径
9. [ ] **质量门禁**
   - `bash -n`；若改动 Go 代码（预期不改）则 `make test`
   - `./scripts/check-trellis-gate.sh` 通过
10. [ ] **收尾**
    - `task.py finish` → `task.py archive` → 写 journal
    - 提交带 `[task:one-command-compose-deploy]`

## 回滚点

| 步骤 | 回滚方式 |
|---|---|
| compose 改造 | `git checkout -- docker-compose.yml` |
| 安装脚本 | 删除文件即回到今天（脚本是新文件，不影响既有部署） |
| E2E 临时环境 | `docker compose -p <临时项目名> down -v`（**只针对临时项目名**） |

## 不做

- 不 `docker compose down -v` 任何既有项目；不碰任何真实部署目录。
- 不改网络形态、不挂 `docker.sock`、不动发版流水线。
