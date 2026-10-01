package main

// dashboardHTML is the self-contained live dashboard. It polls /api/snapshot
// once per second and renders the held connections, a connection-count
// sparkline, and an event log derived from snapshot-to-snapshot diffs.
const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>agentws-viz</title>
<style>
  /* Fleet design tokens (frontend/styles/var/colors.scss, 2025 branding). */
  :root {
    --bg: #f9fafc;            /* ui-off-white */
    --card: #ffffff;          /* core-fleet-white */
    --border: #e2e4ea;        /* ui-fleet-black-10 */
    --row-hover: #f9fafc;     /* ui-fleet-blue-10 */
    --text: #192147;          /* core-fleet-black */
    --text-secondary: #515774;/* ui-fleet-black-75 */
    --dim: #8b8fa2;           /* ui-fleet-black-50 */
    --green: #009a7d;         /* core-fleet-green */
    --blue: #6a67fe;          /* core-vibrant-blue */
    --blue-10: #f1f0ff;       /* ui-vibrant-blue-10 */
    --success: #3db67b;       /* ui-success */
    --warning: #a87b1f;       /* readable take on ui-warning for text */
    --error: #d66c7b;         /* ui-error */
  }
  @media (prefers-color-scheme: dark) {
    :root {
      --bg: #181a1f; --card: #1e2128; --border: #474c58; --row-hover: #252830;
      --text: #e2e4ea; --text-secondary: #bebebf; --dim: #87888b;
      --green: #00c28b; --blue: #7b79ff; --blue-10: #2d2e4d;
      --success: #4dc98b; --warning: #f0ca5e; --error: #e07888;
    }
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; padding: 24px; background: var(--bg); color: var(--text);
    font: 400 14px/1.5 Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  }
  h1 { font-size: 20px; margin: 0; font-weight: 600; letter-spacing: -0.01em; }
  h1 .sub { color: var(--dim); font-weight: 400; font-size: 14px; }
  header { display: flex; align-items: center; gap: 12px; margin-bottom: 24px; }
  #status { width: 10px; height: 10px; border-radius: 50%; background: var(--dim); flex: none; }
  #status.ok { background: var(--success); }
  #status.err { background: var(--error); }
  #banner { color: var(--error); margin-left: auto; font-size: 13px; }
  .row { display: flex; gap: 16px; flex-wrap: wrap; align-items: stretch; }
  .panel { background: var(--card); border: 1px solid var(--border); border-radius: 8px; padding: 16px 20px; }
  .stats { display: flex; gap: 16px; margin-bottom: 16px; flex-wrap: wrap; }
  .stat .n { font-size: 24px; font-weight: 600; font-variant-numeric: tabular-nums; }
  .stat .l { color: var(--dim); font-size: 12px; font-weight: 400; text-transform: uppercase; letter-spacing: .04em; margin-top: 2px; }
  #online { color: var(--green); }
  #count { color: var(--blue); }
  #legacyreads { color: var(--warning); }
  #spark { display: block; }
  .grow { flex: 1 1 520px; min-width: 0; }
  table { width: 100%; border-collapse: collapse; font-size: 14px; }
  th, td { text-align: left; padding: 8px 12px; white-space: nowrap; }
  th { color: var(--text-secondary); font-weight: 600; font-size: 12px; border-bottom: 1px solid var(--border); }
  tbody tr { border-bottom: 1px solid var(--border); }
  tbody tr:last-child { border-bottom: none; }
  tbody tr:hover { background: var(--row-hover); }
  tr.flash td { animation: flash 1.2s ease-out; }
  @keyframes flash { from { background: var(--blue-10); } to { background: transparent; } }
  td.num { text-align: right; font-variant-numeric: tabular-nums; }
  td .drop { color: var(--warning); font-weight: 600; }
  .empty { color: var(--dim); padding: 20px 12px; }
  #log { flex: 0 1 400px; max-height: 480px; overflow-y: auto; font-size: 13px; }
  #log .e { padding: 3px 0; }
  #log .t { color: var(--dim); margin-right: 8px; font-variant-numeric: tabular-nums; }
  #log .connect { color: var(--success); }
  #log .disconnect { color: var(--error); }
  #log .notify { color: var(--blue); }
  .tablewrap { overflow-x: auto; }
  td.os svg { width: 15px; height: 15px; vertical-align: -2px; fill: var(--text-secondary); }
  td.os { width: 24px; }
  td.mono { font-family: "SF Mono", SFMono-Regular, Menlo, Consolas, monospace; font-size: 13px; }
  #tabs { display: flex; gap: 4px; margin-bottom: 16px; border-bottom: 1px solid var(--border); flex-wrap: wrap; }
  #tabs button {
    background: none; border: none; border-bottom: 2px solid transparent; margin-bottom: -1px;
    padding: 8px 14px; color: var(--text-secondary); font: inherit; cursor: pointer;
  }
  #tabs button:hover { color: var(--text); }
  #tabs button.active { color: var(--blue); border-bottom-color: var(--blue); font-weight: 600; }
  #tabs button .n { color: var(--dim); font-size: 12px; margin-left: 6px; font-variant-numeric: tabular-nums; }
  #tabs button.stale { color: var(--dim); font-style: italic; }
  #tabs .pending { color: var(--dim); font-size: 12px; padding: 8px 14px; align-self: center; }
</style>
</head>
<body>
<header>
  <div id="status"></div>
  <h1>agentws <span class="sub">— live WebSocket connections per Fleet instance</span></h1>
  <div id="banner"></div>
</header>

<div id="tabs"></div>

<div class="stats">
  <div class="panel stat"><div class="n" id="count">–</div><div class="l">ws connected</div></div>
  <div class="panel stat"><div class="n" id="nextsync">–</div><div class="l">next sync</div></div>
  <div class="panel stat"><div class="n" id="notified">–</div><div class="l">notifications</div></div>
  <div class="panel stat"><div class="n" id="dropped">–</div><div class="l">dropped</div></div>
  <div class="panel stat"><div class="n" id="bytesin">–</div><div class="l">bytes in</div></div>
  <div class="panel stat"><div class="n" id="bytesout">–</div><div class="l">bytes out</div></div>
  <div class="panel stat"><div class="n" id="orbitreads">–</div><div class="l">reads (orbit)</div></div>
  <div class="panel stat"><div class="n" id="legacyreads">–</div><div class="l">reads (v1 legacy)</div></div>
  <div class="panel"><canvas id="spark" width="360" height="64"></canvas><div class="l" style="color:var(--dim);font-size:12px" id="sparklabel">connections</div></div>
</div>

<div class="row">
  <div class="panel grow tablewrap">
    <table>
      <thead><tr>
        <th>os</th><th>host</th><th>hostname</th><th>remote</th><th>connected</th><th>last notified</th>
        <th style="text-align:right">notified</th><th style="text-align:right">dropped</th>
        <th style="text-align:right">in</th><th style="text-align:right">out</th>
        <th style="text-align:right" title="distributed/read via /api/osquery/ (orbit)">reads</th>
        <th style="text-align:right" title="distributed/read via /api/v1/osquery/ (osqueryd tls plugin)">v1 reads</th>
      </tr></thead>
      <tbody id="conns"></tbody>
    </table>
    <div class="empty" id="empty" hidden>no connections</div>
  </div>
  <div class="panel" id="log"></div>
</div>

<script>
"use strict";
const $ = id => document.getElementById(id);
const HISTORY_MAX = 300;       // samples; window = HISTORY_MAX * poll interval
let intervalMs = 0;            // poll interval, reported by the tool on the first poll

// Per-instance state, keyed by instance id. Every instance is diffed on every
// poll (so event logs stay complete in the background); only the active tab
// is rendered.
const states = new Map();
let active = null;             // active instance id
// Aliases into the active instance's state, rebound by setActive().
let prev = new Map();          // host_id -> last snapshot row
let history = [];              // connection counts for the sparkline
let flashHosts = new Set();

function stateFor(id) {
  let st = states.get(id);
  if (!st) {
    st = { prev: new Map(), history: [], flashHosts: new Set(), nextSyncAt: null, log: [], data: null };
    states.set(id, st);
  }
  return st;
}

function bind(st) {
  prev = st.prev; history = st.history; flashHosts = st.flashHosts; nextSyncAt = st.nextSyncAt;
}

function unbind(st) {
  st.prev = prev; st.flashHosts = flashHosts; st.nextSyncAt = nextSyncAt;
}

function setActive(id) {
  if (id === active) return;
  active = id;
  const st = stateFor(id);
  // Rebuild the log panel from this instance's stored entries.
  const log = $("log");
  log.replaceChildren();
  for (const e of st.log) log.appendChild(logNode(e));
  bind(st);
  if (st.data) {
    updateNextSync();
    render(st.data.connections || [], st.data.read_stats || [], st.data);
    sparkline();
  }
  renderTabs(lastInstances);
}

let lastInstances = { want: 1, instances: [] };

function renderTabs(info) {
  lastInstances = info;
  const tabs = $("tabs");
  tabs.replaceChildren();
  for (const inst of info.instances) {
    const b = document.createElement("button");
    const id = inst.id;
    b.textContent = inst.label;
    b.title = "instance " + inst.id;
    const st = states.get(id);
    if (st && st.data && st.data.connections) {
      const n = document.createElement("span");
      n.className = "n";
      n.textContent = st.data.connections.length;
      b.appendChild(n);
    }
    if (id === active) b.classList.add("active");
    if (inst.stale) { b.classList.add("stale"); b.title += " — not seen recently"; }
    b.onclick = () => setActive(id);
    tabs.appendChild(b);
  }
  if (info.instances.length < info.want) {
    const p = document.createElement("span");
    p.className = "pending";
    p.textContent = "discovering instances… " + info.instances.length + "/" + info.want;
    tabs.appendChild(p);
  }
}

// Inline SVG OS icons keyed by icon class; viewBox 0 0 16 16.
const OS_ICONS = {
  apple: '<path d="M11.6 8.5c0-1.5 1.2-2.2 1.3-2.3-.7-1-1.8-1.2-2.2-1.2-.9-.1-1.8.6-2.3.6-.5 0-1.2-.6-2-.5-1 0-2 .6-2.5 1.5-1.1 1.9-.3 4.7.8 6.2.5.8 1.1 1.6 1.9 1.6.8 0 1.1-.5 2-.5s1.2.5 2 .5 1.4-.8 1.9-1.5c.6-.9.8-1.7.9-1.8-.1 0-1.8-.7-1.8-2.6zM10.1 3.9c.4-.5.7-1.2.6-1.9-.6 0-1.4.4-1.8.9-.4.5-.7 1.2-.6 1.9.7 0 1.4-.4 1.8-.9z"/>',
  windows: '<path d="M1 3.5 7 2.7v5H1v-4.2zM8 2.5 15 1.5v6.2H8V2.5zM1 8.5h6v5L1 12.7V8.5zM8 8.5h7v6.2l-7-1V8.5z"/>',
  linux: '<path d="M8 1.5c-1.6 0-2.5 1.2-2.5 2.8 0 .9.1 1.6-.3 2.4-.5 1-1.4 1.9-1.7 3.2-.2.9 0 1.8.4 2.4l-.7.4c-.3.2-.3.6 0 .8.8.5 2 .9 3 .9h3.6c1 0 2.2-.4 3-.9.3-.2.3-.6 0-.8l-.7-.4c.4-.6.6-1.5.4-2.4-.3-1.3-1.2-2.2-1.7-3.2-.4-.8-.3-1.5-.3-2.4 0-1.6-.9-2.8-2.5-2.8zM6.7 5.2c.3 0 .5.3.5.6s-.2.6-.5.6-.5-.3-.5-.6.2-.6.5-.6zm2.6 0c.3 0 .5.3.5.6s-.2.6-.5.6-.5-.3-.5-.6.2-.6.5-.6zM6.6 6.9c.4.3.9.5 1.4.5s1-.2 1.4-.5c.2.4.3.9.3 1.4 0 1.7-.8 3.2-1.7 3.2s-1.7-1.5-1.7-3.2c0-.5.1-1 .3-1.4z"/>',
};

// osIcon maps a fleet platform string to one of the OS_ICONS keys.
function osIcon(platform) {
  const p = (platform || "").toLowerCase();
  if (p === "darwin" || p === "macos") return "apple";
  if (p === "windows") return "windows";
  // Fleet reports Linux hosts by distro (ubuntu, debian, rhel, fedora, ...).
  const linux = ["linux", "ubuntu", "debian", "rhel", "centos", "fedora", "arch",
    "gentoo", "amzn", "pop", "kali", "linuxmint", "opensuse", "sles", "nixos", "void", "tuxedo"];
  if (linux.some(d => p.includes(d))) return "linux";
  return null;
}

// nextSyncAt is the local-clock time of the server's next interval-check
// tick, derived from the server-computed remaining time (next_check_in_ms) so
// clock skew between server and browser doesn't matter.
let nextSyncAt = null;

function updateNextSync() {
  const el = $("nextsync");
  if (nextSyncAt == null) {
    el.textContent = "–";
    return;
  }
  el.textContent = Math.max(0, Math.ceil((nextSyncAt - Date.now()) / 1000)) + "s";
}

function fmtBytes(n) {
  if (n < 1024) return n + " B";
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + " KB";
  if (n < 1024 * 1024 * 1024) return (n / (1024 * 1024)).toFixed(1) + " MB";
  return (n / (1024 * 1024 * 1024)).toFixed(2) + " GB";
}

function ago(iso) {
  if (!iso) return "–";
  const s = Math.max(0, (Date.now() - new Date(iso)) / 1000);
  if (s < 60) return Math.floor(s) + "s";
  if (s < 3600) return Math.floor(s / 60) + "m" + String(Math.floor(s % 60)).padStart(2, "0") + "s";
  return Math.floor(s / 3600) + "h" + String(Math.floor((s % 3600) / 60)).padStart(2, "0") + "m";
}

function logNode(entry) {
  const e = document.createElement("div");
  e.className = "e";
  const t = document.createElement("span");
  t.className = "t";
  t.textContent = entry.time;
  const m = document.createElement("span");
  m.className = entry.cls;
  m.textContent = entry.text;
  e.append(t, m);
  return e;
}

// logEvent records an event for the instance currently being diffed
// (diffing) and, if that is the active tab, shows it immediately.
let diffing = null;

function logEvent(cls, text) {
  const entry = { cls, text, time: new Date().toLocaleTimeString() };
  const st = stateFor(diffing);
  st.log.unshift(entry);
  if (st.log.length > 200) st.log.pop();
  if (diffing === active) {
    const log = $("log");
    log.prepend(logNode(entry));
    while (log.childElementCount > 200) log.lastChild.remove();
  }
}

function diff(conns) {
  const cur = new Map(conns.map(c => [c.host_id, c]));
  for (const [id, c] of cur) {
    const p = prev.get(id);
    if (!p) {
      logEvent("connect", "host " + id + " connected (" + c.remote_addr + ")");
      flashHosts.add(id);
    } else {
      const delta = c.notified_count - p.notified_count;
      if (delta > 0) {
        const reason = c.last_notify_reason ? " [" + c.last_notify_reason + "]" : "";
        logEvent("notify", "host " + id + " notified" + reason + (delta > 1 ? " (×" + delta + ")" : ""));
        flashHosts.add(id);
      }
      if (c.dropped_count > p.dropped_count) {
        logEvent("disconnect", "host " + id + " dropped " + (c.dropped_count - p.dropped_count) + " notification(s) (buffer full)");
      }
      // Reconnection shows up as a newer connected_at.
      if (c.connected_at !== p.connected_at) {
        logEvent("connect", "host " + id + " reconnected");
        flashHosts.add(id);
      }
    }
  }
  for (const id of prev.keys()) {
    if (!cur.has(id)) logEvent("disconnect", "host " + id + " disconnected");
  }
  prev = cur;
}

function readCells(tr, stats) {
  const tdR = document.createElement("td");
  tdR.className = "num";
  tdR.textContent = stats ? stats.orbit_reads : 0;
  const tdV1 = document.createElement("td");
  tdV1.className = "num";
  // Legacy v1 reads mean the host's osqueryd is still polling with the tls
  // plugin — the exact thing the transport should eliminate. Flag them.
  if (stats && stats.legacy_reads > 0) {
    const s = document.createElement("span");
    s.className = "drop";
    s.textContent = stats.legacy_reads;
    tdV1.appendChild(s);
  } else {
    tdV1.textContent = "0";
  }
  tr.append(tdR, tdV1);
}

function render(conns, readStats, data) {
  $("count").textContent = conns.length;

  let notified = 0, dropped = 0, bytesIn = 0, bytesOut = 0;
  for (const c of conns) {
    notified += c.notified_count;
    dropped += c.dropped_count;
    bytesIn += c.bytes_in || 0;
    bytesOut += c.bytes_out || 0;
  }
  $("notified").textContent = notified;
  $("dropped").textContent = dropped;

  const statsByHost = new Map(readStats.map(s => [s.host_id, s]));
  let orbitReads = 0, legacyReads = 0;
  for (const s of readStats) {
    orbitReads += s.orbit_reads;
    legacyReads += s.legacy_reads;
  }

  // Byte/read metrics are only collected when the server runs with debug
  // logging; make that state visible instead of showing misleading zeros.
  if (data.metrics_enabled === false) {
    for (const id of ["bytesin", "bytesout", "orbitreads", "legacyreads"]) {
      const el = $(id);
      el.textContent = "off";
      el.title = "metrics require the server to run with debug logging (FLEET_LOGGING_DEBUG=1)";
    }
  } else {
    $("bytesin").textContent = fmtBytes(bytesIn);
    $("bytesout").textContent = fmtBytes(bytesOut);
    $("orbitreads").textContent = orbitReads;
    $("legacyreads").textContent = legacyReads;
  }

  const tbody = $("conns");
  tbody.replaceChildren();
  for (const c of conns) {
    const tr = document.createElement("tr");
    if (flashHosts.has(c.host_id)) tr.className = "flash";

    const tdOS = document.createElement("td");
    tdOS.className = "os";
    tdOS.title = c.platform || "unknown";
    const icon = osIcon(c.platform);
    if (icon) {
      // Trusted, hardcoded SVG paths only; the platform string never goes
      // through innerHTML (it goes into title/textContent above/below).
      tdOS.innerHTML = '<svg viewBox="0 0 16 16" aria-label="' + icon + '">' + OS_ICONS[icon] + "</svg>";
    } else {
      tdOS.textContent = c.platform || "?";
    }
    tr.appendChild(tdOS);

    const cells = [
      c.host_id, c.hostname || "–", c.remote_addr, ago(c.connected_at),
      c.last_notified_at
        ? ago(c.last_notified_at) + " ago" + (c.last_notify_reason ? " · " + c.last_notify_reason : "")
        : "never",
    ];
    cells.forEach((v, i) => {
      const td = document.createElement("td");
      td.textContent = v;
      if (i === 2) td.className = "mono"; // remote address
      tr.appendChild(td);
    });
    const tdN = document.createElement("td");
    tdN.className = "num";
    tdN.textContent = c.notified_count;
    const tdD = document.createElement("td");
    tdD.className = "num";
    if (c.dropped_count > 0) {
      const s = document.createElement("span");
      s.className = "drop";
      s.textContent = c.dropped_count;
      tdD.appendChild(s);
    } else {
      tdD.textContent = "0";
    }
    const tdIn = document.createElement("td");
    tdIn.className = "num";
    tdIn.textContent = fmtBytes(c.bytes_in || 0);
    const tdOut = document.createElement("td");
    tdOut.className = "num";
    tdOut.textContent = fmtBytes(c.bytes_out || 0);
    tr.append(tdN, tdD, tdIn, tdOut);
    readCells(tr, statsByHost.get(c.host_id));
    tbody.appendChild(tr);
  }

  // Read stats for hosts with no WebSocket connection on this instance are
  // not listed: those hosts are either connected to another instance (the
  // load balancer routes their reads anywhere) or legacy pollers, which only
  // show up in the aggregate read counters above.
  $("empty").hidden = conns.length > 0;
  flashHosts = new Set();
}

function sparkline() {
  const c = $("spark"), ctx = c.getContext("2d");
  ctx.clearRect(0, 0, c.width, c.height);
  if (history.length < 2) return;
  const max = Math.max(1, ...history);
  const stepX = c.width / (HISTORY_MAX - 1);
  const x0 = c.width - (history.length - 1) * stepX;
  ctx.beginPath();
  history.forEach((v, i) => {
    const x = x0 + i * stepX;
    const y = c.height - 4 - (v / max) * (c.height - 10);
    i === 0 ? ctx.moveTo(x, y) : ctx.lineTo(x, y);
  });
  const styles = getComputedStyle(document.documentElement);
  ctx.strokeStyle = styles.getPropertyValue("--blue").trim() || "#6a67fe";
  ctx.lineWidth = 1.5;
  ctx.stroke();
  ctx.fillStyle = styles.getPropertyValue("--dim").trim() || "#8b8fa2";
  ctx.font = "10px Inter, sans-serif";
  ctx.fillText(String(max), 2, 10);
}

// pollInstance fetches one instance's snapshot and updates its state,
// rendering only if it is the active tab.
async function pollInstance(id) {
  const resp = await fetch("/api/snapshot?instance=" + encodeURIComponent(id));
  if (!resp.ok) throw new Error(await resp.text() || resp.status);
  const data = await resp.json();
  if (!data.enabled) {
    throw new Error("websocket transport is disabled on the server (set FLEET_WEBSOCKET_TRANSPORT_ENABLED=1)");
  }
  // Swap the module-level aliases to this instance's state for the
  // synchronous diff below, then swap the active tab's state back.
  const st = stateFor(id), act = stateFor(active);
  unbind(act);
  bind(st);
  diffing = id;
  const conns = data.connections || [];
  nextSyncAt = data.next_check_in_ms != null ? Date.now() + data.next_check_in_ms : null;
  diff(conns);
  history.push(conns.length);
  if (history.length > HISTORY_MAX) history.shift();
  st.data = data;
  if (id === active) {
    updateNextSync();
    render(conns, data.read_stats || [], data);
    sparkline();
  } else {
    flashHosts = new Set(); // flashes are only meaningful when rendered live
  }
  unbind(st);
  bind(act);
}

async function poll() {
  try {
    const resp = await fetch("/api/instances");
    if (!resp.ok) throw new Error(await resp.text() || resp.status);
    const info = await resp.json();
    const ids = info.instances.map(i => i.id);
    // Drop state for instances that expired so they don't come back stale.
    for (const id of states.keys()) if (!ids.includes(id)) states.delete(id);
    if ((active == null || !ids.includes(active)) && ids.length > 0) { active = null; setActive(ids[0]); }
    renderTabs(info);
    if (info.interval_ms && info.interval_ms !== intervalMs) {
      intervalMs = info.interval_ms;
      const mins = HISTORY_MAX * intervalMs / 60000;
      $("sparklabel").textContent = "connections, last " + (mins >= 1 ? Math.round(mins) + " min" : Math.round(mins * 60) + "s");
    }
    if (info.disabled) {
      $("status").className = "err";
      $("banner").textContent = "websocket transport is disabled on the server (set FLEET_WEBSOCKET_TRANSPORT_ENABLED=1)";
      return;
    }
    if (ids.length === 0) {
      $("status").className = info.error ? "err" : "";
      $("banner").textContent = info.error ? String(info.error).slice(0, 120) : "discovering instances…";
      return;
    }
    const results = await Promise.allSettled(ids.map(pollInstance));
    const errs = results.filter(r => r.status === "rejected");
    if (errs.length > 0) {
      $("status").className = errs.length === ids.length ? "err" : "ok";
      $("banner").textContent = String(errs[0].reason).slice(0, 120);
    } else if (info.error) {
      $("status").className = "ok";
      $("banner").textContent = String(info.error).slice(0, 120);
    } else if (info.no_instance_id) {
      $("status").className = "ok";
      $("banner").textContent = "server reports no instance_id; all instances are merged into one tab (upgrade the server)";
    } else {
      $("status").className = "ok";
      $("banner").textContent = "";
    }
  } catch (err) {
    $("status").className = "err";
    $("banner").textContent = String(err).slice(0, 120);
  }
}

// Chain polls on a timer so the cadence follows the tool's -interval.
(async function loop() {
  await poll();
  setTimeout(loop, intervalMs || 1000);
})();
// Tick the countdown between polls so it never appears to stall.
setInterval(updateNextSync, 250);
</script>
</body>
</html>
`
