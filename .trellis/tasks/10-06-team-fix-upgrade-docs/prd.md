# 升级路径文档（复用编制）

- 任务：`10-06-team-fix-upgrade-docs`
- 执行：`review-ops`（运维/文档）
- 共享任务：`task-14`（Lead 指派，修复 D）
- 写范围：`README.md`、`README.en.md`、`MAINTENANCE.md`（本任务唯一写者）

## 背景（Lead 已核实的事实）

- 仓库 `VERSION` = `0.0.6`。
- registry `ghcr.io/zhemed/new-api-own` 的 `latest` / `0.0.6` / `v0.0.6` 指向**同一摘要**。
- 本机缓存的 `latest` 标签**曾陈旧指向 0.0.3**：在本地标签陈旧时，`docker run …:latest`
  会直接复用旧镜像、不去拉新——这是"升级后仍是旧版"的头号嫌疑。
- 根因层：用户把"重启容器"当成"升级"，而重启只重启**同一个镜像**；升级必须**重建容器**。

## 目标

把"从旧版本升到最新版本"写成**照做就不会停在旧版**的路径，并让"我到底跑的哪个版本"可自查。

## 要求

1. **README 双语**（`README.md`、`README.en.md`）：部署段补「升级 / 回滚」子节
   - 升级三步：`docker pull <固定 tag>` → `docker rm -f new-api` → `docker run …`（沿用现有参数：`--name new-api --restart always --network host -v ./data:/data`）；
   - 明确写出：**重启容器（`docker restart` / `--restart always`）不会换镜像，必须删容器重建**；
   - 固定 tag 优先（`0.0.6` 之类），并解释 `:latest` 在本地标签陈旧时可能复用旧镜像的坑；
   - 回滚：把 tag 换成上一个版本，同样三步。
2. **MAINTENANCE.md**："版本对齐核查"从原则升级为**可执行步骤**
   - 三处比对：运行实例 `/api/status` 的版本、registry `docker buildx imagetools inspect` 的摘要/标签、
     仓库 `VERSION` 与最新 tag；
   - 给出判据（三者一致才算对齐；不一致时的动作）；
   - "本地 `latest` 陈旧标签"作为**警告**写进该节，并给自查命令（`docker images --digests`）。
3. 重申**唯一部署方式**（`docker run` + 公开镜像；禁 compose / Helm / K8s / systemd），升级/回滚也只能走这条路径。
4. **本机实测**文中每条命令（只读）：`docker pull 0.0.6`、`imagetools inspect`、`docker images`、
   `docker run --rm <image> --version`；**不停/删任何在运行的容器**、不动用户远程实例。

## 硬约束

- 无外部网络，仅允许 `ghcr.io/zhemed/new-api-own` 的**只读拉取/查询**；
- 不改代码、不改 `web/**`、不改 `scripts/**`；
- 不 commit / push；
- 仓库里不得出现主机名、IP、凭据；文中命令一律用占位符与公开镜像名；
- Docker 环境未达标准（29.7.2 + v5.4.0）时不得执行 docker 操作 → 先 `docker version` 校验。

## 验收

1. 改动清单（文件 + 小节）；
2. 每条命令的实测输出（脱敏）；
3. 一句话结论："照这个文档做，用户从 0.5 能否必然升到 0.6"。

## Notes

- 与既有约定保持一致：`.trellis/spec/guides/deployment-single-method.md`、`AGENTS.md`「唯一部署方式」。
- 细节留在本 PRD；给 Lead 的汇报只给结论。

---

# 执行记录（2026-10-06）

## 改动清单（只改 README.md / README.en.md / MAINTENANCE.md）

| 文件 | 位置 | 内容 |
|---|---|---|
| `README.md` | 部署段新增 `### 升级 / 回滚（照做不会停在旧版）` | 升级三步（pull→rm -f→run）、回滚示例（`0.0.5`）、"重启不换镜像"、`:latest` 陈旧标签的坑、升级后 `curl /api/status` 验证、`make check-version` 提示、重申唯一 `docker run` 形态 |
| `README.en.md` | 部署段新增 `### Upgrade / rollback (never left on an old version)` | 同上，英文一一对应 |
| `MAINTENANCE.md` | 新增 `## 升级与回滚（唯一部署方式下）` | 三步命令 + 回滚 + 为什么必须显式 pull + 禁止第二种形态（引用闸门与规则源） |
| `MAINTENANCE.md` | 新增 `### 升级后三处比对（可执行）`（挂在升级节下，**不再另起 `##`**） | 三处比对表 + 落后时的动作 + `make check-version` 快捷方式 + `:latest` 陈旧标签的坑与 `grep` 误报坑 |
| `MAINTENANCE.md` | 发版流程 step 2/3/4 与约定句 | `0.0.3` 硬编码示例 → `<版本>`，避免文档把读者指向旧版本；step 4 增加"发布后按版本对齐核查再对一次" |
| `MAINTENANCE.md` | 与 Lead 新增的 `### 版本对齐核查（每次维护必做）` 对齐 | 按 Lead 要求把"本机 `latest` 还陈旧指向 0.0.3"改为**过去时 + 已刷新 + 任何机器都可能陈旧**（事故记录段同步改） |

## 实测输出（脱敏；本机 Docker 29.7.2 + Compose v5.4.0）

| # | 命令 | 实测结果 |
|---|---|---|
| 1 | `docker pull ghcr.io/zhemed/new-api-own:0.0.6` | `Digest: sha256:a74d0a6c0940…`（下拉成功） |
| 2 | `docker buildx imagetools inspect …:0.0.6 --format '{{.Manifest.Digest}}'` | `sha256:a74d0a6c0940…` |
| 3 | `docker buildx imagetools inspect …:latest --format '{{.Manifest.Digest}}'` | `sha256:a74d0a6c0940…`（与 0.0.6 同摘要） |
| 4 | `docker images ghcr.io/zhemed/new-api-own --digests` | `0.0.6` 与 `latest` 同为 `a74d0a6c0940`；本地另有 `0.0.3/v0.0.3/0.0.2/v0.0.2/v0.0.4`；**过滤带完整仓库名** |
| 5 | `docker images \| grep 0.0.5`（反例） | 6 行命中，含**其它项目**同号 tag 与无关行（`grep` 的 `.` 是通配符）→ 已写进文档警告 |
| 6 | `docker run --rm --network none …:0.0.6 --version` | `v0.0.6` |
| 7 | `docker run --rm --network none …:0.0.3 --version` | `v0.0.3`（陈旧本地标签拿到旧版，即"升级失败"现场复现） |
| 8 | `docker image inspect …:0.0.6 --format '{{index .Config.Labels "org.opencontainers.image.version"}}'` | `v0.0.6` |
| 9 | `curl -s http://127.0.0.1:3020/api/status \| grep -o '"version":"[^"]*"'`（本机裸跑实例） | `"version":"0.0.6"`（源码直跑不带 `v`，判据是去前缀比较） |
| 10 | `make check-version INSTANCE=http://127.0.0.1:3020` | `==> 判定: OK`，退出码 **0**（5 项一致，0 项无法判定） |
| 11 | `cat VERSION` / `git tag --sort=-v:refname \| head -1` | `0.0.6` / `v0.0.6` |

**端到端仿真**（一次性容器 `newapi-sim`，`-p 127.0.0.1:3021:3000`，临时数据目录，跑完即删；不碰用户任何实例）：

1. 起 `0.0.3` → `/api/status` = `"v0.0.3"`
2. `docker restart` → 仍是 `"v0.0.3"`（**证明重启不会换镜像**）
3. 文档三步（`pull` → `rm -f` → `run` 固定 tag）→ `"v0.0.6"`（**升级成功**）
4. 回滚（`pull` → `rm -f` → `run 0.0.5`）→ `"v0.0.5"`（**回滚可用**；本机原本没有 0.0.5，`pull` 后拿到）
5. 清理：容器已删、临时目录已删；运行中的容器仍只有原有三个，未受影响

## 结论

**能。** 照文档做必然从 0.5 升到 0.6：`docker pull` 拿到 0.6 摘要 → `docker rm -f` + `docker run <固定 tag>`
重建后实例自报 `v0.0.6`；文档同时写明了唯一会让人"以为升级了"的两条歧路——**重启容器**（复现：restart 后仍 v0.0.3）
与**本地陈旧的 `:latest` 标签**（复现：`0.0.3` 本地标签直接跑出 v0.0.3），并给出了三处比对/`make check-version` 兜底。

