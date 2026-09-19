# 评审并处理部署脚本的 9 条建议（A1 + B1-B8）

## Goal

外部评审对 `install-compose.sh` 与 `docker-compose.yml` 提了 9 条意见。逐条对着代码核实后，
按「成立 / 部分成立 / 不成立」给出判定与证据，再落地**用户确认的全量范围**（含两处我修正过的改法
与两条评审没提到、我自己补的问题）。

## 判定与证据（结论已定稿）

| # | 判定 | 证据 |
|---|---|---|
| A1 命名反向 | ✅ 成立（严重度中，非高） | 本机 compose 5.4.0 实测优先级 `compose.yaml` > `compose.yml` > `docker-compose.yml` > `docker-compose.yaml`；脚本 214/216 把 `compose.yaml` 当历史名、225–228 教用户往最低优先级名字改 |
| B1 口令兜底默认值 | ✅ 成立（中） | compose 48/49/80/95 四处 `${VAR:-123456}`。补充：pg 只 listen 127.0.0.1、redis 只 bind 127.0.0.1，弱口令不对外可达，风险限定本机 |
| B2 健康检查未过仍退出 0 | ✅ 成立（**真 bug**） | 296–300 仅 warn，302 的 heredoc 是最后一条命令 → 隐式 exit 0 |
| B3 版本预检精确相等 | ⚠️ 部分成立 | 144 行 `!=` 字面比较确认。但 `AGENTS.md`「Docker 环境标准（强制）」就是固定版本且禁未达标操作，`install-docker.sh` 还 apt-mark hold。**按修正版改**：低于→失败，等于→过，高于同主版本→告警放行，跨主版本→仍需 `--allow-docker-mismatch` |
| B4 重跑 = 隐式升级 | ✅ 成立，且 314 行文案与实现矛盾 | 155–160 未给 `--tag` 即从 ref 的 VERSION 推；`.env` 每次重写 + `pull_policy: always` |
| B5 浮动镜像 tag | ⚠️ 一半成立 | `redis:latest` 成立（跨大版本浮动）→ 钉 `redis:7`；**`postgres:15` 不接受** —— 钉主版本、跟随 15.x 补丁流正是官方推荐做法 |
| B6 依赖无健康检查 | ✅ 成立（低） | 61–63 裸 `depends_on` |
| B7 restart 策略不一致 | ✅ 成立 | 41/78/90 均为 `always` |
| B8 数据卷无备份路径 | ✅ 成立（低） | 脚本 8 处 backup 全为配置文件 |

评审之外的补充：

- **C1**：上一轮只改 `README.md`，`README.en.md` 未同步（英文文档里没有「一条命令部署」，
  却仍在教 `Adjust the default passwords before use`）。
- **C2**：脚本 314 行「换版本：改 `.env` 的 `NEW_API_IMAGE_TAG`，或重跑本脚本加 `--tag`」
  与实现矛盾 —— 不加 `--tag` 重跑同样会换版本。

## Requirements（用户确认全量）

- R1 **A1**：仓库 `docker-compose.yml` → `compose.yaml`；脚本改以 `compose.yaml` 为主名，
  历史名列表改为 `compose.yml docker-compose.yml docker-compose.yaml`；全部引用同步。
- R2 **B1**：四处口令改 `${VAR:?<提示>}`（缺变量直接失败）。README 里「不建 `.env` 也能直接起」
  的说法随之失效，必须同步改写。
- R3 **B2**：健康检查未过 → 摘要照印（供排查）但**以退出码 1 结束**；容器保留不自动回滚。
- R4 **B3**：按修正版语义实现；`--allow-docker-mismatch` 保留给跨主版本/降级场景。
- R5 **B4 + C2**：默认**沿用既有 `.env` 的 tag**；新增 `--upgrade` 显式升级到 ref 的 VERSION；
  `--tag` 优先级最高；文案同步改正。
- R6 **B5**：`redis:latest` → `redis:7`；postgres 保持 `postgres:15`（并在注释里写明为何不钉更细）。
- R7 **B6**：postgres 加 `pg_isready`、redis 加 `redis-cli ping` 健康检查；
  `depends_on` 改 `condition: service_healthy`。
- R8 **B7**：三个服务统一 `restart: unless-stopped`。
- R9 **B8**：`MAINTENANCE.md` 补可复制的数据备份/恢复步骤（Postgres 与 `data/`+`logs/`）。
- R10 **C1**：`README.en.md` 与中文 README 对齐。
- R11 本机 `/opt/docker/new-api-own` 迁移到新约定（改名 + 重建），迁移后仍 healthy。
- R12 守闸门：提交带锚点、journal、归档。

## Acceptance Criteria

- [ ] `docker compose config` 在有 `.env` 时通过；**无 `.env` 时以非 0 失败**并打印我们的提示
      （B1 的判别性验证）
- [ ] 脚本语法 `bash -n` + `shellcheck` 通过
- [ ] 健康检查失败路径**真的**返回非 0（判别性：让健康检查必然失败，验退出码）
- [ ] 重跑不再改 tag（判别性：改 `.env` 里 tag 后重跑，tag 保持；加 `--upgrade` 才动）
- [ ] 版本比较：`29.7.3` 同主版本放行告警、`28.x` 失败、`--allow-docker-mismatch` 可越过
- [ ] 仓库内不再有主 compose 的旧名引用残留；`makefile`/`.github` 不受影响
- [ ] 临时目录 E2E 跑通（三容器 healthy + `/api/status` 200）
- [ ] 迁移后 `/opt/docker/new-api-own` healthy，`komari`/`litepan` 未受影响
- [ ] 两个 README 内容对齐；MAINTENANCE 有备份/恢复段
- [ ] journal + 归档完成

## 范围外

- 不改 `docker-compose.dev.yml`（`makefile` 用显式 `-f` 引用，不受改名影响）。
- 不改服务端业务代码、不改发版流水线。
- 不给 `postgres:15` 钉更细的版本（理由见 R6）。
