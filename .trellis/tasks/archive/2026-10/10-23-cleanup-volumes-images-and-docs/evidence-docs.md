# evidence-docs：对齐 README 部署段 + 固化「数据目录必须在工作区内」

- 任务板：`task-11`（Trellis 任务 `10-23-cleanup-volumes-images-and-docs`，PRD 见同目录 `prd.md` 的 R6）
- 执行者：`worker-docs`；基线 HEAD `e6ec2d8`（2026-10-23）
- 写作用域内改动：`README.md`、`.trellis/spec/guides/deployment-single-method.md`（+ 本证据文件）

## 一、事实核对（先实测/实读，再写文档）

| 事实 | 证据 |
| --- | --- |
| 容器挂载为工作区内 bind mount | `docker inspect -f '{{.Name}} \| {{.Config.Image}} \| {{range .Mounts}}{{.Type}}:{{.Source}}=>{{.Destination}}{{end}} \| running={{.State.Running}}' new-api` → `bind:/root/new-api-own/data=>/data \| running=true`，镜像 `ghcr.io/zhemed/new-api-own:v0.0.45` |
| Docker 环境达标（才允许 docker 操作） | `docker version --format '{{.Server.Version}}'` → `29.7.2`（标准 29.7.2 + Compose v5.4.0） |
| 数据目录就在工作区内且非空 | `ls -ld data` → 目录存在；`ls data` → `one-api.db`、`logs`、若干 `.bak-*` 备份库 |
| `.gitignore` 已忽略它 | `grep -n '^/data/' .gitignore` → `27:/data/`（锚定仓库根，只匹配根下 `data/`） |
| 实例自报版本 | `cat VERSION` → `0.0.45`（与容器镜像 tag `v0.0.45` 一致；HTTP 200 由 Lead 实测，见 PRD AC2） |
| 文档无过期路径引用 | `grep -rn '/root/data' --include=*.md .`（排除 `.trellis/tasks/`）→ 仅命中 `.trellis/workspace/new-api-own/journal-*.md` 的**历史迁移记录**，README / MAINTENANCE / spec 均无引用 |
| 部署形态未被改动 | `sh scripts/forbid-extra-deploy-methods.sh` → `✅ 部署方式唯一：docker run + ghcr.io/zhemed/new-api-own`（exit 0） |

未做的事：没有自行 `curl` 访问实例（网络动作需点名授权），HTTP 200 与 v0.0.45 采用 Lead 在 PRD 中的实测值；
`docker version` / `docker inspect` 为本地只读操作。

## 二、改动清单

### README.md（2 处）

1. 部署命令后的第一条 bullet（原「默认使用 SQLite，数据保存在 `./data` 目录」）补上：
   **该目录必须位于本仓库工作区内**、`.gitignore:27` 已忽略 `/data/`、并链接到新章节。
2. 新增章节 `### 数据目录（必须在工作区内）`（位于部署 bullet 列表与「升级 / 回滚」之间）：
   - 约定与两条理由（忽略规则只覆盖工作区内；维护会话可读写范围只有工作区）；
   - 换挂载 6 步命令块：`docker stop` → `cp -a`（不要 `mv`）→ 文件数与 sha256 校验 →
     `docker rm`（**不带 `-v`**）→ 照抄部署命令重建（只改 `-v` 源）→ `curl` 期望 200；
   - 三条提醒：必须先停容器再复制（SQLite `-wal` / `-shm`）、用 `docker inspect` 确认挂载源、
     旧目录是回滚点（验证通过再删）。
   - 命令形态**未改**：仍是 `docker run -d --name new-api ... -v ./data:/data ...`。

### .trellis/spec/guides/deployment-single-method.md（追加 1 节）

末节追加 `## 数据目录必须位于工作区内（2026-10-23 定调）`，沿用该文件既有风格（`>` 用户定调引语 +
`**规则：**` 粗体行 + 三列表格 + 代码块）：

- 规则：一律 `./data`，禁止工作区外绝对路径；容器内仍是 `/data`，**不构成第二种部署方式**；
- 与 MAINTENANCE「升级 / 回滚」的三处口径一致性要求；
- 理由三条：① 维护动作只能在允许范围内做（工作区外=越界读取，校验降级为人工）；
  ② 忽略规则只覆盖工作区内（备份/迁移漏项）；③ 回滚点删除后即单点；
- 迁移步骤表（7 步）+ 校验命令（`find | wc -l`、`sha256sum` 聚合摘要）+ 验证命令（`docker inspect`、`curl` 200、版本）；
- 反例与风险表（5 行：挂区外、运行中复制、`docker rm -v` / `volume prune`、忘改 `-v`、未验证就删回滚点）；
- 现状注记：旧目录已按裁定删除，`./data` 为唯一副本且**尚无异地备份**（备份策略本轮不做，已知单点风险）。

判断：未追加第二处规范文件。`README`/`MAINTENANCE`/spec 中数据目录口径已一致，另开落点会引入新的同步负担；
`.trellis/spec/ops/` 下无与本约定重复的文档（本次未发现需要新增处）。

## 三、验证

| 检查 | 命令 | 结果 |
| --- | --- | --- |
| 改动范围 | `git diff --stat` | 仅 `README.md`(+30/-1)、`.trellis/spec/guides/deployment-single-method.md`(+65)；`git status` 中除本任务 evidence 目录外无其它新增 |
| 部署形态闸门 | `sh scripts/forbid-extra-deploy-methods.sh` | `✅ 部署方式唯一`（exit 0） |
| 代码块配对 | `grep -c '^```' README.md .trellis/spec/guides/deployment-single-method.md` | `16` / `6`（均为偶数，配对） |
| 成对美元符号 | 新增行逐条检查（`git diff -U0` 输出） | 0 处（新增行不含该字符） |
| 单星号斜体 | 正则扫描新增行中的孤立星号 | 仅命中 fenced code block 内的正则字面量（`[^"]*` 形式），代码块内不参与 Markdown 渲染 |
| 表格列数一致 | `git diff -U0 \| grep '^+\|' \| awk -F'\|' '{print NF-2}'` | 新增两表全部为 3 列，与该文件既有表格一致 |
| 过期路径 | `grep -rn '/root/data' README.md .trellis/spec/guides/deployment-single-method.md` | 无命中 |
| 事实一致性 | `grep -n '^/data/' .gitignore`；`docker inspect …` | `.gitignore:27 = /data/`；挂载源落在工作区内（与本机实际一致） |

### 打回点修复（Lead 复核：第 3 步校验命令有歧义）

问题：原写法 `find . -type f | wc -l` 若在仓库根执行，会把整个仓库算进去，文件数与 sha256 摘要都错。
修复（README 与 spec 同一处命令块，两处都改）：

```bash
( cd <旧数据目录> && find . -type f | wc -l ) ; ( cd ./data && find . -type f | wc -l )   # 3. 文件数须相等
( cd <旧数据目录> && find . -type f -exec sha256sum {} + | sort | sha256sum )
( cd ./data        && find . -type f -exec sha256sum {} + | sort | sha256sum )            # 3. 两条摘要须一致
```

README 追加说明 bullet：「第 3 步的 `find` 必须在**各自数据目录内部**执行（子 shell `cd` 就是为此）；
在仓库根直接跑会把整个仓库算进去」；spec 同步改写为「在旧、新两个数据目录内部各跑一次」并保留同一句警告。

实测对比（只读、未动容器）：

| 命令 | 输出 |
| --- | --- |
| `( cd ./data && find . -type f \| wc -l )` | `55`（数据目录真实文件数） |
| `( cd ./data && find . -type f -exec sha256sum {} + \| sort \| sha256sum )` | `fe7bab8214d5…（截断）` 正常产出摘要 |
| `find . -type f \| wc -l`（仓库根，错误用法） | `125204`（把整个仓库算进去，含 `web/node_modules` 等） |

修复后重跑自查：

| 检查 | 结果 |
| --- | --- |
| `git status --short` | 仍只有 `README.md`、`.trellis/spec/guides/deployment-single-method.md` 与任务 evidence 目录 |
| 代码块围栏 | `grep -c '^```'` → README `16` / spec `6`（均偶，配平） |
| 成对美元符号 | 新增行 0 处（`git diff -U0 \| grep '^+' \| grep -c` 结果为 0） |
| 单星号斜体 | 新增行中仅 fenced code block 内的正则字面量，不参与渲染 |
| 表格列数 | 新增两表仍为 3 列 |
| 部署形态闸门 | `sh scripts/forbid-extra-deploy-methods.sh` → `✅ 部署方式唯一`（exit 0，修复后复跑） |

## 四、风险与未完成

- README 新增章节与 MAINTENANCE「升级 / 回滚」（`./data` 是相对路径，换目录＝换了数据目录）语义一致，
  但**未同步改 MAINTENANCE**（不在写作用域）；若后续要三处显式互链，需再开一次改动。
- 未做端到端迁移演练（不重启/不动容器），迁移步骤来自既有 journal 记录与本次只读核对，属文档级约定。
- 备份仍缺位：spec 已把「无异地备份」写成现状风险，但本轮未提供备份方案（PRD 明确本轮不做）。
- 文档里的 `docker inspect` 示例输出依赖容器名 `new-api`（与 README 部署命令一致）；改名部署者需自行替换。

## 五、建议下一步

1. 提交前请 Lead 复核 3 处口径（README 新章节 / MAINTENANCE 升级段 / spec 新节）是否需显式互链。
2. 备份策略作为独立任务立项（`./data` 单点 + 无异地副本），可与「数据目录」一节联动补一句指引。
3. 若希望迁移步骤可执行化，可考虑把它做成 `make`/脚本目标——但那属于新动作，需先过闸门与用户同意。
