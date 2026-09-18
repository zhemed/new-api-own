# 设计：一条命令 compose 部署（new-api-own）

## 1. 与 komari 的逐项对照

| 维度 | komari `install-compose.sh` | new-api-own 的适配 | 为什么不能照抄 |
|---|---|---|---|
| 服务数 | 1 个容器 | **3 个**（new-api / redis / postgres） | 撞名预检、健康等待、日志上限都要覆盖三个服务 |
| 状态 | `./data`（SQLite 文件） | `./data`、`./logs` + **`pg_data` 命名卷** | 命名卷的归属由 compose 项目名决定，搬家会"看起来数据没了" |
| 口令 | 没有 | Postgres / Redis 两个口令 | 必须生成、必须复用、必须不进仓库 |
| 镜像 tag | 脚本内 `DEFAULT_TAG` 字面量 + `check-repo` 校验 | **不写字面量**：从 ref 上的 `VERSION` 推导 | 我们已经有 `VERSION`，写死第二份必然漂移 |
| 升级 | 面板一键升级（挂 docker.sock） | `docker compose pull && up -d` | 挂 socket ≈ 宿主 root，与本仓库安全基线冲突 |
| 网络 | `host` + `KOMARI_LISTEN` | `host`（沿用现状） | 改网络要连 redis/postgres 一起迁，v1 不做 |

komari 里**值得原样搬过来**的四条工程细节：① 拒绝覆盖 + `--force` 才重写；② 备份名精确到微秒并
带"占用就继续抖"的兜底，否则同一秒连跑会互相覆盖、"只保留最近 N 份"形同虚设；③ 容器名冲突必须是
**启动前**的预检（起一半再报 Conflict 极难收拾）；④ `--no-start` 干跑，便于生成文件做对比。

## 2. 目标目录布局

```
/opt/docker/new-api-own/        # 750，root:root
├── docker-compose.yml          # 从 ref 拉取（单一事实来源），--force 时备份后重写
├── .env                        # 600，本机生成：两个口令 + 镜像 tag + 项目名；gitignore 已覆盖
├── data/                       # 700，绑定挂载到容器 /data（含明文上游 Key）
├── logs/                       # 700，绑定挂载到容器 /app/logs
└── <历史 compose>.bak-<时间戳>-<微秒>   # 只保留最近 3 份
```

`pg_data` 不在目录里 —— 它是 Docker 命名卷，名字形如 `<项目名>_pg_data`。

## 3. 单一事实来源：compose 用变量插值

改造仓库根 `docker-compose.yml`，全部保留**向后兼容的默认值**（单独 clone 后不建 `.env`
仍可 `docker compose up -d`，与今天行为一致）：

| 位置 | 改成 |
|---|---|
| 镜像 | `ghcr.io/zhemed/new-api-own:${NEW_API_IMAGE_TAG:-latest}` |
| `SQL_DSN` | `postgresql://root:${POSTGRES_PASSWORD:-123456}@127.0.0.1:5432/new-api` |
| `REDIS_CONN_STRING` | `redis://:${REDIS_PASSWORD:-123456}@127.0.0.1:6379` |
| redis `command` | `--requirepass ${REDIS_PASSWORD:-123456}` |
| postgres `POSTGRES_PASSWORD` | `${POSTGRES_PASSWORD:-123456}` |
| 三个 `container_name` | `${NEW_API_CONTAINER_NAME:-new-api}` / `${REDIS_CONTAINER_NAME:-redis}` / `${POSTGRES_CONTAINER_NAME:-postgres}` |
| 三个服务 | 新增 `logging: json-file, max-size 10m, max-file 3` |

安装脚本**不内嵌 compose 内容**，而是把同一 ref 上的 `docker-compose.yml` 拉到目标目录。
理由：内嵌会产生"脚本里的 compose"与"仓库里的 compose"两份事实，必然漂移（komari 用
`check-repo` 校验 §15.2 定稿来对抗这一点，我们有更省事的做法——直接复用仓库那一份）。

`.env` 由安装脚本生成，compose 自动读取项目目录下的 `.env`。同时写入
`COMPOSE_PROJECT_NAME=new-api-own`：**固定项目名**，这样即使 `--dir` 换了目录，命名卷
`new-api-own_pg_data` 不变，不会出现"换了目录数据就没了"。

## 4. 版本：不写字面量

```
REF   := ${NEW_API_COMPOSE_REF:-main}          # --ref 可覆盖（如 v0.0.3）
TAG   := ${NEW_API_IMAGE_TAG:-}                 # --tag 显式覆盖
若 TAG 为空：TAG := v$(curl -fsSL .../<REF>/VERSION)
```

拉到脚本时必然可达 GitHub，所以多一次 `VERSION` 请求是廉价的；换来的是**永不漂移**：
`VERSION` 改一次，引用 `<ref>` 的安装就跟着走。`--tag` 用于钉死到某个历史版本。

## 5. 口令与幂等（本设计最硬的不变量）

1. 生成：`LC_ALL=C tr -dc 'a-f0-9' < /dev/urandom | head -c 32` —— **只用十六进制**。
   理由：口令要拼进 `postgresql://` 与 `redis://` 形式的 DSN，非 URL 安全字符会造成
   解析歧义；纯 hex 规避整类问题。
2. 落盘：目标目录 `.env`，写后 `chmod 600`；目录 `chmod 750`。
3. **重跑绝不重新生成**：`.env` 已存在则读出来沿用，只在摘要里写"沿用既有口令"。
   违反它的后果是确定的：`POSTGRES_PASSWORD` 只在数据目录为空时生效，改口令后应用连不上
   自己的库，而错误表现是"新装的面板里数据全丢"这种极难自查的形态。
4. **不打印口令值**：输出里只出现变量名与"已生成/已沿用"，不出现值；仓库内不存在任何生成值。

## 6. 流程

```
解析参数 → root 检查 → Docker/Compose 版本预检 → 解析 REF/TAG
  → 目标目录不存在则 mkdir -p {,data,logs} 并 chmod 750
  → 已有部署？(--force 才继续，重写前备份并清理超过 3 份的 compose 备份)
  → 写/复用 .env（600）
  → 拉 docker-compose.yml 到目标目录
  → --no-start 则到此为止
  → 容器名预检（三个名字，属于别的项目目录即 die 并给解法）
  → docker compose pull（可选）→ docker compose up -d
  → 轮询 new-api 健康检查至 healthy（上限 ~120s）
  → chmod 700 data logs
  → 打印面板地址 / 目录 / 常用命令 / 升级 / 回滚 / 安全基线提示
```

失败处理：任何一步 `die` 都以非 0 退出并打印**下一步该敲什么**；`up -d` 之后才失败时，
容器留在原地不动（不自动 `down`），由操作者看日志决定——自动回滚会把数据卷卷入风险。

## 7. 安全边界（写进脚本输出与文档）

- 沿用 `network_mode: host` ⇒ new-api 监听 `*:3000`，`/api/status` 无需认证即可读。
  脚本**不自动改网络形态**（改了要连 redis/postgres 一起迁），而是打印 `MAINTENANCE.md`
  「部署安全基线」的两条加固选项，由操作者决定。
- `data/` 含明文上游 Key：脚本把 `data`/`logs` 置 700 并提示备份文件的权限同一要求。
- 容器仍以 root 运行（`Dockerfile` 未设 `USER`）——沿用现状，不在本任务改。
- 脚本以 root 运行且来自 `<ref>`：文档里明确建议**钉 ref**（`--ref vX.Y.Z`）而不是长期用
  `main`，并说明 `curl | bash` 的固有风险。

## 8. 验证方式与实测结果

E2E 全部在**临时目录 + 非默认项目名**（`/tmp/e2e-*`、项目名 `e2e-napi-test`）里跑，未触碰任何真实部署。

| 验收项 | 怎么验 | 实测结果 |
|---|---|---|
| E2E | 临时目录实起三个容器 | 镜像拉取成功 → 三容器 Created/Started → `new-api` **healthy** → `GET /api/status` HTTP 200；脚本退出码 0 |
| 幂等 | 重跑 | 退出码 1，打印「docker-compose.yml 已存在……要覆盖请加 `--force`」 |
| 备份唯一（顺序） | 同一目录连跑 6 次 `--force --no-start` | **6 份备份、6 个唯一文件名** |
| 判别性 | 把备份名退化到秒级**并**去掉占位重试（= 原始缺陷形态） | 同样 6 次连跑只剩 **2** 份 —— 防覆盖不是装饰 |
| 并发 | 同一秒并发 6 次 `--force` | 只剩 1 份备份，但**哨兵内容校验证明它含原文件** —— 原文件没丢；其余 5 个进程是在文件已被移走之后才检查，本就不需要再备份 |
| 口令复用 | 连续 `--force` 后比对两个口令值 | 逐字节不变；用 `.env` 里的口令 `psql -U root` 与 `redis-cli ping` **都认证通过**（口令若被重生成必然失败，故为判别性） |
| 口令样式 | 长度与字符集 | 32 位、纯 hex |
| 口令不外泄 | 全仓库 grep | 仓库内无该口令 |
| 撞名预检 | 造一个非 compose 的同名容器 | 退出码 1，容器状态仍为 `created`（未被删改），输出给出 `--name-prefix` 与 `rm -f` 两条解法 |
| 数据不变量 | `--force` 前后比对 | `data/`、`logs/` 文件哈希集合不变；`pg_data` 卷文件清单不变且卷仍在 |
| 权限 | 创建后立即检查 | 目录 750、`data/` 与 `logs/` 700、`.env` 600（**干跑路径也生效**） |
| 语法 | `bash -n` + `shellcheck` | 均通过（shellcheck 退出码 0） |
| 向后兼容 | 无 `.env` 时 `docker compose config` 与改造前逐行 diff | 差异**只有**新增的三个 `logging` 块，其余逐字段一致 |

> **一条被实测推翻的初始假设，记在这里以免重犯**：设计初稿写的是「去掉亚秒精度即出现覆盖」，
> 实测**不成立** —— 脚本里"名字被占用就继续抖"的重试循环单独就能防住。真正的缺陷形态是
> **亚秒精度与重试循环同时缺席**，这时 6 次连跑只剩 2 份。断言要跟着证据改，不是反过来。
