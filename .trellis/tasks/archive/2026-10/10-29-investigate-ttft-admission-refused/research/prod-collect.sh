#!/usr/bin/env bash
# 生产 429 / ttft_admission_refused 只读取证脚本（一次性执行，输出为 markdown 风格小节）
#
# 用法（在已获授权的操作端执行；主机只放在环境变量里，不写入本文件）：
#   ssh "$ZZDS_HOST" 'bash -s' < prod-collect.sh > prod-raw.txt 2>&1
# 若操作端使用 sshpass（仅在授权允许时）：
#   sshpass -e ssh "$ZZDS_HOST" 'bash -s' < prod-collect.sh > prod-raw.txt 2>&1
#
# 硬性约束：只读。本脚本不写 / 不删任何文件，不改配置，不重启容器，不装包；
#   不读取 docker env 与数据库凭据列（channels.key / tokens.key / users.password 等一律不查）；
#   base_url 与疑似密钥在输出前打码；sqlite 用 mode=ro 只读连接。
#
# 覆盖：docker ps/inspect 摘要（含 Mounts 定位 /data）、ss -lntp、
#   docker logs --since 6h 关键字筛取与聚合、定位 data 目录下 sqlite 库、
#   sqlite 只读查询（schema 摘要、channels、abilities、logs type=5 错误聚合、logs type=2 消费聚合、
#   限流/重试相关 options）。
# 说明：本项目错误日志 type=5（LogTypeError），type=2 为消费日志（LogTypeConsume），两段都查。

set -u
export LC_ALL=C

hr() { printf '\n## %s\n\n' "$1"; }
say() { printf '%s\n' "$*"; }

hr "0. meta"
say "collect_at_utc=$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
say "host_kernel=$(uname -sr 2>/dev/null)"
say "uptime=$(uptime 2>/dev/null | sed -E 's/^ +//')"
say "epoch_now=$(date -u +%s)"
say "note=time values printed as UTC; logs.created_at is epoch seconds"

hr "1. docker ps -a"
if command -v docker >/dev/null 2>&1; then
  docker ps -a --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}' 2>&1
else
  say "docker_not_found"
fi

# 容器名探测（new-api 单容器部署；不打印 env）
C=$(docker ps -a --format '{{.Names}} {{.Image}}' 2>/dev/null | grep -Ei 'new-api|newapi|one-api|oneapi' | head -1 | awk '{print $1}')
if [ -z "${C:-}" ]; then
  C=$(docker ps -a --format '{{.Names}}' 2>/dev/null | head -1)
fi
say "container_detected=${C:-none}"

hr "2. container inspect summary（Image/State/Mounts/Ports，不含 Env）"
if [ -n "${C:-}" ]; then
  docker inspect -f 'name={{.Name}} image={{.Config.Image}} running={{.State.Running}} started={{.State.StartedAt}} finished={{.State.FinishedAt}} restarts={{.RestartCount}} netmode={{.HostConfig.NetworkMode}}' "$C" 2>&1
  say "-- mounts --"
  docker inspect -f '{{range .Mounts}}{{.Type}} {{.Source}} -> {{.Destination}} rw={{.RW}}{{"\n"}}{{end}}' "$C" 2>&1
  say "-- published/listening ports (docker) --"
  docker inspect -f '{{json .NetworkSettings.Ports}}' "$C" 2>&1
else
  say "no_container_found_skip_inspect"
fi

hr "3. ss -lntp"
(ss -lntp 2>&1 || netstat -lntp 2>&1 || true) | head -40

hr "4. docker logs --since 6h 关键字筛取"
if [ -n "${C:-}" ]; then
  say "-- startup / version markers (head 12) --"
  docker logs "$C" 2>&1 | head -12 | sed -E 's/(sk-[A-Za-z0-9_-]{6,})/[REDACTED]/g'
  say "-- total lines since 6h --"
  LOGN=$(docker logs --since 6h "$C" 2>&1 | wc -l)
  say "lines_since_6h=$LOGN"
  say "-- keyword counts since 6h --"
  for kw in 'ttft_admission_refused' 'the cell is at capacity' '429' 'rate limit' 'rate_limit' 'retry' 'channel' 'error' 'auto.?ban' 'disable'; do
    n=$(docker logs --since 6h "$C" 2>&1 | grep -E -c "$kw" 2>/dev/null || true)
    printf '%-28s %s\n' "$kw" "${n:-0}"
  done
  say ""
  say "-- ttft_admission_refused / capacity lines (tail 40) --"
  docker logs --since 6h "$C" 2>&1 | grep -E 'ttft_admission_refused|the cell is at capacity' | tail -40 | sed -E 's/(sk-[A-Za-z0-9_-]{6,})/[REDACTED]/g'
  say ""
  say "-- 429 related lines (tail 60) --"
  docker logs --since 6h "$C" 2>&1 | grep -E '429' | grep -Ei 'error|fail|upstream|rate|status|refus|capacity' | tail -60 | sed -E 's/(sk-[A-Za-z0-9_-]{6,})/[REDACTED]/g'
  say ""
  say "-- 上游错误 / 容量 / 重试相关（tail 60） --"
  docker logs --since 6h "$C" 2>&1 | grep -Ei 'upstream|capacity|retry|refus|rate.?limit|auto.?ban|自动禁用|禁用渠道' | tail -60 | sed -E 's/(sk-[A-Za-z0-9_-]{6,})/[REDACTED]/g'
  say ""
  say "-- 归一化错误行 top 25（去掉行首时间戳后计数） --"
  docker logs --since 6h "$C" 2>&1 | grep -Ei 'error|fail' \
    | sed -E 's/^[0-9]{4}-[0-9]{2}-[0-9]{2}T[^ ]+ //' \
    | sed -E 's/(request[_ ]?id[=:] ?[A-Za-z0-9-]+)/request_id=***/Ig' \
    | sed -E 's/(sk-[A-Za-z0-9_-]{6,})/[REDACTED]/g' \
    | cut -c1-240 | sort | uniq -c | sort -rn | head -25
  say ""
  say "-- 最近 40 行日志（上下文） --"
  docker logs --since 6h "$C" 2>&1 | tail -40 | sed -E 's/(sk-[A-Za-z0-9_-]{6,})/[REDACTED]/g'
else
  say "no_container_found_skip_logs"
fi

hr "5. data 目录与 sqlite 文件定位"
DATA_SRC=""
if [ -n "${C:-}" ]; then
  DATA_SRC=$(docker inspect -f '{{range .Mounts}}{{if eq .Destination "/data"}}{{.Source}}{{end}}{{end}}' "$C" 2>/dev/null)
fi
say "data_mount_source=${DATA_SRC:-none}"
if [ -n "${DATA_SRC:-}" ]; then
  say "-- data dir entries (names/sizes only) --"
  ls -la "$DATA_SRC" 2>&1 | head -30
  DB="$DATA_SRC/one-api.db"
  if [ ! -f "$DB" ]; then
    DB=$(find "$DATA_SRC" -maxdepth 2 -name '*.db' -type f 2>/dev/null | head -1)
  fi
  say "db_path_used=${DB:-none}"
else
  DB=""
  say "no /data mount detected"
fi

hr "6. sqlite 只读查询（mode=ro；type=5 错误日志 / type=2 消费日志）"
if [ -n "${DB:-}" ] && [ -f "$DB" ] && command -v python3 >/dev/null 2>&1; then
  python3 - "$DB" <<'PY'
import sqlite3, sys, re, time, os

db = sys.argv[1]
now = int(time.time())
W6 = now - 6 * 3600
W24 = now - 24 * 3600
U = 'unixepoch'

def out(s=''):
    print(s)

def mask_url(u):
    if u is None:
        return ''
    u = str(u).strip()
    if not u:
        return ''
    scheme = ''
    m = re.match(r'^(https?)://(.*)$', u, re.I)
    if m:
        scheme = m.group(1) + '://'
        rest = m.group(2)
    else:
        rest = u
        if rest.startswith('//'):
            rest = rest[2:]
    rest = rest.split('#')[0].split('?')[0]
    host, _sep, path = rest.partition('/')
    host = host.split('@')[-1]
    port = ''
    if not host.startswith('[') and ':' in host:
        h, p = host.rsplit(':', 1)
        if p.isdigit():
            host, port = h, ':' + p
    parts = [p for p in host.split('.') if p]
    if not parts:
        return scheme + '***'
    if len(parts) >= 3:
        masked = '***.' + '.'.join(parts[-2:])
    elif len(parts) == 2:
        masked = parts[0][:2] + '***.' + parts[1]
    else:
        masked = '***'
    return scheme + masked + port + ('/' + path if path else '')

URL_RE = re.compile(r'https?://[^\s"\'<>\)\]]+', re.I)
KEY_RE = re.compile(r'\b(sk-[A-Za-z0-9_-]{6,}|[A-Za-z0-9_-]{24,})\b')

def scrub(s):
    if s is None:
        return ''
    s = str(s)
    s = URL_RE.sub(lambda m: mask_url(m.group(0)), s)
    s = KEY_RE.sub('[ID]', s)
    return s

try:
    con = sqlite3.connect('file:%s?mode=ro' % db, uri=True)
    con.execute('PRAGMA query_only=ON')
except Exception as e:
    out('db_open_error=%s' % e)
    sys.exit(0)

def cols_of(table):
    try:
        return [r[1] for r in con.execute('PRAGMA table_info(%s)' % table)]
    except Exception:
        return []

# logs 表的渠道列在不同版本可能名为 channel_id 或 channel，动态适配
_LOG_COLS = cols_of('logs')
LOG_CH = 'channel_id' if 'channel_id' in _LOG_COLS else ('channel' if 'channel' in _LOG_COLS else 'channel_id')
out('logs_channel_column=%s' % LOG_CH)

def run(sql, args=()):
    try:
        cur = con.execute(sql, args)
        return [d[0] for d in (cur.description or [])], cur.fetchall()
    except Exception as e:
        out('query_error: %s | sql=%s' % (e, sql[:160]))
        return [], []

def show(title, sql, args=(), limit=80, scrub_cols=()):
    out('')
    out('### ' + title)
    headers, rows = run(sql, args)
    if not headers:
        return []
    out('| ' + ' | '.join(headers) + ' |')
    out('|' + '---|' * len(headers))
    for r in rows[:limit]:
        vals = []
        for i, v in enumerate(r):
            s = '' if v is None else str(v)
            if headers[i] == 'base_url':
                s = mask_url(s)
            elif headers[i] in scrub_cols:
                s = scrub(s)
            else:
                s = KEY_RE.sub('[ID]', URL_RE.sub(lambda m: mask_url(m.group(0)), s)) if 'http' in s else s
            s = s.replace('\n', ' ').replace('|', '/')
            if len(s) > 220:
                s = s[:217] + '...'
            vals.append(s)
        out('| ' + ' | '.join(vals) + ' |')
    if len(rows) > limit:
        out('(truncated, total rows=%d)' % len(rows))
    return rows

out('db_file=%s' % db)
try:
    st = os.stat(db)
    out('db_size_bytes=%d db_mtime_utc=%s' % (st.st_size, time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime(st.st_mtime))))
except Exception as e:
    out('db_stat_error=%s' % e)
h, rows = run("select name from sqlite_master where type='table' order by name")
out('tables=' + ','.join(r[0] for r in rows))
for t in ('channels', 'abilities', 'logs', 'options'):
    c = cols_of(t)
    if c:
        out('cols_%s=%s' % (t, ','.join(c)))
for t in ('logs',):
    try:
        idx = [r[1] for r in con.execute("select name, sql from sqlite_master where type='index' and tbl_name=?", (t,))]
        out('indexes_%s=%s' % (t, '; '.join(str(i) for i in idx)))
    except Exception:
        pass

show('channels（不查 key 列；base_url 打码）',
     'select ' + ','.join('"%s"' % c for c in ['id', 'type', 'name', 'status', 'priority', 'weight',
                                                 'group', 'auto_ban', 'base_url', 'models',
                                                 'model_mapping', 'test_time', 'response_time',
                                                 'created_time', 'setting']
                           if c in cols_of('channels')) + ' from channels order by id',
     limit=40, scrub_cols=('setting', 'model_mapping'))

show('channels 状态分布',
     'select status, count(*) as cnt from channels group by status order by status')

show('abilities：模型 → 渠道（全量，上限 200 行）',
     'select ' + ','.join('"%s"' % c for c in ['model', 'channel_id', 'enabled', 'priority', 'weight', 'tag']
                           if c in cols_of('abilities')) + ' from abilities order by model, channel_id',
     limit=200)

show('abilities：按模型聚合（渠道数 / 启用数）',
     'select model, count(*) as total, sum(enabled) as enabled_cnt from abilities group by model order by model',
     limit=100)

show('logs 总量与最新时间',
     'select count(*) as total, datetime(max(created_at), ?) as latest_utc, datetime(min(created_at), ?) as earliest_utc from logs',
     args=(U, U))

show('logs 近 6h 按类型计数（1=充值 2=消费 3=管理 4=系统 5=错误 6=退款 7=登录）',
     'select type, count(*) as cnt from logs where created_at>=? group by type order by type',
     args=(W6,))

show('logs type=5 错误：近 6h 按模型/渠道聚合',
     'select model_name, ' + LOG_CH + ', count(*) as cnt, datetime(min(created_at), ?) as first_utc, datetime(max(created_at), ?) as last_utc from logs where type=5 and created_at>=? group by model_name, ' + LOG_CH + ' order by cnt desc limit 40',
     args=(U, U, W6), limit=40)

show('logs type=5 错误：近 24h 按模型/渠道聚合',
     'select model_name, ' + LOG_CH + ', count(*) as cnt from logs where type=5 and created_at>=? group by model_name, ' + LOG_CH + ' order by cnt desc limit 40',
     args=(W24,), limit=40)

for kw in ('ttft', 'capacity', '429', 'rate', 'refus'):
    h, rows = run('select count(*) as cnt from logs where type=5 and created_at>=? and lower(content) like ?',
                  (W6, '%' + kw + '%'))
    out('err_logs_6h_like_%s=%s' % (kw, rows[0][0] if rows else 'n/a'))

show('logs type=5：含 ttft/capacity/429 的样本（近 6h，content 打码截断）',
     'select id, datetime(created_at, ?) as utc, model_name, ' + LOG_CH + ', substr(content, 1, 500) as content ' +
     'from logs where type=5 and created_at>=? and (lower(content) like ? or lower(content) like ? or content like ?) ' +
     'order by id desc limit 30',
     args=(U, W6, '%ttft%', '%capacity%', '%429%'), limit=30, scrub_cols=('content',))

show('logs type=5：近 6h 最新错误 40 条',
     'select id, datetime(created_at, ?) as utc, model_name, ' + LOG_CH + ', is_stream, use_time, ' +
     'substr(content, 1, 400) as content from logs where type=5 and created_at>=? order by id desc limit 40',
     args=(U, W6), limit=40, scrub_cols=('content',))

show('logs type=5 错误：近 24h 按小时分布',
     'select datetime(created_at - (created_at % 3600), ?) as hour_utc, count(*) as cnt from logs where type=5 and created_at>=? group by hour_utc order by hour_utc',
     args=(U, W24), limit=30)

show('logs type=2 消费：近 6h 按模型/渠道/流式聚合',
     'select model_name, ' + LOG_CH + ', is_stream, count(*) as cnt, sum(prompt_tokens) as prompt_tok, ' +
     'sum(completion_tokens) as completion_tok, round(avg(use_time), 2) as avg_use_time_s ' +
     'from logs where type=2 and created_at>=? group by model_name, ' + LOG_CH + ', is_stream order by cnt desc limit 40',
     args=(W6,), limit=40)

show('logs type=2 消费：近 24h 按小时分布',
     'select datetime(created_at - (created_at % 3600), ?) as hour_utc, count(*) as cnt from logs where type=2 and created_at>=? group by hour_utc order by hour_utc',
     args=(U, W24), limit=30)

show('logs 近 6h 最新 30 条任意类型（含请求是否到达网关）',
     'select id, datetime(created_at, ?) as utc, type, model_name, ' + LOG_CH + ', is_stream, use_time, substr(content, 1, 300) as content ' +
     'from logs where created_at>=? order by id desc limit 30',
     args=(U, W6), limit=30, scrub_cols=('content',))

show('options：限流/重试/自动禁用/日志开关相关键',
     'select "key", substr(value, 1, 200) as value from options where ' +
     '"key" like ? or "key" like ? or "key" like ? or "key" like ? or "key" like ? or "key" like ? ' +
     'order by "key"',
     args=('%RateLimit%', '%rate_limit%', '%Retry%', '%retry%', '%ChannelDisableThreshold%', '%Log%'),
     limit=60)

show('options：日志开关与重试值精确项',
     'select "key", substr(value, 1, 120) as value from options where "key" in (?, ?, ?, ?, ?) order by "key"',
     args=('ErrorLogEnabled', 'LogConsumeEnabled', 'DataExportEnabled', 'RetryTimes', 'retry_times'))

con.close()
PY
else
  say "skip_sqlite: db_not_found_or_python3_missing db=${DB:-none}"
fi

hr "7. collect done"
say "done_at_utc=$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
say "readonly=yes no_writes=yes no_restart=yes no_package_install=yes"
