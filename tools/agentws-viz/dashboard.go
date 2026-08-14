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
  :root {
    --bg: #101418; --panel: #181e24; --border: #2a323c;
    --text: #d7dee6; --dim: #8494a6; --accent: #34d399;
    --warn: #fbbf24; --err: #f87171; --info: #60a5fa;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; padding: 20px; background: var(--bg); color: var(--text);
    font: 14px/1.45 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  }
  h1 { font-size: 16px; margin: 0; font-weight: 600; }
  h1 .sub { color: var(--dim); font-weight: 400; }
  header { display: flex; align-items: center; gap: 10px; margin-bottom: 16px; }
  #status { width: 10px; height: 10px; border-radius: 50%; background: var(--dim); flex: none; }
  #status.ok { background: var(--accent); }
  #status.err { background: var(--err); }
  #banner { color: var(--err); margin-left: auto; }
  .row { display: flex; gap: 16px; flex-wrap: wrap; align-items: stretch; }
  .panel { background: var(--panel); border: 1px solid var(--border); border-radius: 8px; padding: 14px 16px; }
  .stats { display: flex; gap: 16px; margin-bottom: 16px; }
  .stat .n { font-size: 28px; font-weight: 700; }
  .stat .l { color: var(--dim); font-size: 12px; text-transform: uppercase; letter-spacing: .06em; }
  #spark { display: block; }
  .grow { flex: 1 1 480px; min-width: 0; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 6px 10px; white-space: nowrap; }
  th { color: var(--dim); font-weight: 500; font-size: 12px; border-bottom: 1px solid var(--border); }
  tbody tr { border-bottom: 1px solid var(--border); }
  tbody tr:last-child { border-bottom: none; }
  tr.flash td { animation: flash 1.2s ease-out; }
  @keyframes flash { from { background: rgba(52, 211, 153, .25); } to { background: transparent; } }
  td.num { text-align: right; font-variant-numeric: tabular-nums; }
  td .drop { color: var(--warn); }
  .empty { color: var(--dim); padding: 18px 10px; }
  #log { flex: 0 1 380px; max-height: 480px; overflow-y: auto; }
  #log .e { padding: 2px 0; }
  #log .t { color: var(--dim); margin-right: 8px; }
  #log .connect { color: var(--accent); }
  #log .disconnect { color: var(--err); }
  #log .notify { color: var(--info); }
  .tablewrap { overflow-x: auto; }
</style>
</head>
<body>
<header>
  <div id="status"></div>
  <h1>agentws <span class="sub">— live WebSocket connections (this Fleet instance)</span></h1>
  <div id="banner"></div>
</header>

<div class="stats">
  <div class="panel stat"><div class="n" id="count">–</div><div class="l">connected</div></div>
  <div class="panel stat"><div class="n" id="notified">–</div><div class="l">notifications</div></div>
  <div class="panel stat"><div class="n" id="dropped">–</div><div class="l">dropped</div></div>
  <div class="panel stat"><div class="n" id="bytesin">–</div><div class="l">bytes in</div></div>
  <div class="panel stat"><div class="n" id="bytesout">–</div><div class="l">bytes out</div></div>
  <div class="panel"><canvas id="spark" width="360" height="64"></canvas><div class="l" style="color:var(--dim);font-size:12px">connections, last 5 min</div></div>
</div>

<div class="row">
  <div class="panel grow tablewrap">
    <table>
      <thead><tr>
        <th>host</th><th>remote</th><th>connected</th><th>last notified</th>
        <th style="text-align:right">notified</th><th style="text-align:right">dropped</th>
        <th style="text-align:right">in</th><th style="text-align:right">out</th>
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
let prev = new Map();          // host_id -> last snapshot row
let history = [];              // connection counts for the sparkline
const HISTORY_MAX = 300;       // 5 min at 1s polls
let flashHosts = new Set();

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

function logEvent(cls, text) {
  const e = document.createElement("div");
  e.className = "e";
  const t = document.createElement("span");
  t.className = "t";
  t.textContent = new Date().toLocaleTimeString();
  const m = document.createElement("span");
  m.className = cls;
  m.textContent = text;
  e.append(t, m);
  const log = $("log");
  log.prepend(e);
  while (log.childElementCount > 200) log.lastChild.remove();
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
        logEvent("notify", "host " + id + " notified" + (delta > 1 ? " (×" + delta + ")" : ""));
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

function render(conns) {
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
  $("bytesin").textContent = fmtBytes(bytesIn);
  $("bytesout").textContent = fmtBytes(bytesOut);

  const tbody = $("conns");
  tbody.replaceChildren();
  for (const c of conns) {
    const tr = document.createElement("tr");
    if (flashHosts.has(c.host_id)) tr.className = "flash";
    const cells = [
      c.host_id, c.remote_addr, ago(c.connected_at),
      c.last_notified_at ? ago(c.last_notified_at) + " ago" : "never",
    ];
    for (const v of cells) {
      const td = document.createElement("td");
      td.textContent = v;
      tr.appendChild(td);
    }
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
    tbody.appendChild(tr);
  }
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
  ctx.strokeStyle = "#34d399";
  ctx.lineWidth = 1.5;
  ctx.stroke();
  ctx.fillStyle = "#8494a6";
  ctx.font = "10px ui-monospace, monospace";
  ctx.fillText(String(max), 2, 10);
}

async function poll() {
  try {
    const resp = await fetch("/api/snapshot");
    if (!resp.ok) throw new Error(await resp.text() || resp.status);
    const data = await resp.json();
    if (!data.enabled) {
      $("status").className = "err";
      $("banner").textContent = "websocket transport is disabled on the server (set FLEET_WEBSOCKET_TRANSPORT_ENABLED=1)";
      return;
    }
    const conns = data.connections || [];
    diff(conns);
    render(conns);
    history.push(conns.length);
    if (history.length > HISTORY_MAX) history.shift();
    sparkline();
    $("status").className = "ok";
    $("banner").textContent = "";
  } catch (err) {
    $("status").className = "err";
    $("banner").textContent = String(err).slice(0, 120);
  }
}

poll();
setInterval(poll, 1000);
</script>
</body>
</html>
`
