#!/usr/bin/env node
// ttracker/server.js
//
// Web dashboard for terminal session tracking.
// Replaces the daemon: takes periodic snapshots and serves a UI.
//
// Usage:
//   node server.js              # Start on port 3847
//   node server.js --open       # Start and open browser
//   TT_PORT=8080 node server.js # Custom port
//   TT_INTERVAL=5 node server.js # Snapshot every 5 minutes

const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const { execFile, exec } = require('node:child_process');
const os = require('node:os');

// ─── Constants ───────────────────────────────────────────────────────────────

const PORT = parseInt(process.env.TT_PORT || '3847');
const INTERVAL_MIN = parseInt(process.env.TT_INTERVAL || '10');
const SAFE_MODE = process.env.TT_SAFE_MODE === '1';
const SCRIPT_DIR = __dirname;
const SNAPSHOT_DIR = path.join(SCRIPT_DIR, 'snapshots');
const STATE_FILE = path.join(SNAPSHOT_DIR, 'state.json');
const PID_FILE = path.join(SNAPSHOT_DIR, 'daemon.pid');
const CLAUDE_SESSIONS_DIR = path.join(os.homedir(), '.claude', 'sessions');

// ─── Utilities ───────────────────────────────────────────────────────────────

function runOsascript(script) {
  return new Promise((resolve, reject) => {
    execFile('osascript', ['-e', script], { timeout: 15000 }, (err, stdout) => {
      if (err) return reject(err);
      resolve(stdout.trim());
    });
  });
}

function runOsascriptFile(filepath) {
  return new Promise((resolve, reject) => {
    execFile('osascript', [filepath], { timeout: 30000 }, (err, stdout) => {
      if (err) return reject(err);
      resolve(stdout.trim());
    });
  });
}

function runCommand(cmd, args) {
  return new Promise((resolve) => {
    execFile(cmd, args, { timeout: 5000 }, (err, stdout) => {
      resolve(err ? '' : stdout.trim());
    });
  });
}

function processIsRunning(pid) {
  try {
    process.kill(parseInt(pid), 0);
    return true;
  } catch {
    return false;
  }
}

function readJSON(filepath) {
  try {
    return JSON.parse(fs.readFileSync(filepath, 'utf8'));
  } catch {
    return null;
  }
}

function writeJSON(filepath, data) {
  fs.writeFileSync(filepath, JSON.stringify(data, null, 2));
}

function now() {
  return new Date().toISOString().replace('T', ' ').slice(0, 19);
}

// ─── State Management ────────────────────────────────────────────────────────
// All persistent data lives in a single state.json file:
//   { snapshot: { timestamp, sessions[] }, history: [], notes: {} }

function loadState() {
  const state = readJSON(STATE_FILE);
  if (state && state.snapshot) return state;

  // Migrate from old separate files if they exist
  const migrated = {
    snapshot: readJSON(path.join(SNAPSHOT_DIR, 'latest.json')) || { timestamp: '', session_count: 0, sessions: [] },
    history: readJSON(path.join(SNAPSHOT_DIR, 'history.json')) || [],
    notes: readJSON(path.join(SNAPSHOT_DIR, 'notes.json')) || {}
  };
  saveState(migrated);
  return migrated;
}

function saveState(state) {
  fs.mkdirSync(SNAPSHOT_DIR, { recursive: true });
  writeJSON(STATE_FILE, state);
}

// ─── Snapshot Logic ──────────────────────────────────────────────────────────

const ITERM_APPLESCRIPT = `
tell application "iTerm2"
    set output to ""
    repeat with w from 1 to (count of windows)
        set win to window w
        repeat with t from 1 to (count of tabs of win)
            set theTab to tab t of win
            repeat with s from 1 to (count of sessions of theTab)
                set sess to session s of theTab
                set b to ""
                try
                    tell sess
                        set b to (variable named "badge")
                    end tell
                end try
                set output to output & (id of win) & "\t" & t & "\t" & s & "\t" & (unique ID of sess) & "\t" & (tty of sess) & "\t" & b & "\t" & (name of sess) & linefeed
            end repeat
        end repeat
    end repeat
    return output
end tell`;

let snapshotInProgress = false;

async function takeSnapshot() {
  if (snapshotInProgress) return null;
  snapshotInProgress = true;

  try {
    const raw = await runOsascript(ITERM_APPLESCRIPT).catch(() => '');
    if (!raw) {
      console.log(`[${now()}] Warning: could not reach iTerm2`);
      snapshotInProgress = false;
      return null;
    }

    const lines = raw.split('\n').filter(l => l.trim());
    const sessions = [];

    for (const line of lines) {
      const parts = line.split('\t');
      if (parts.length < 7) continue;

      const [winId, tab, sess, itermUuid, tty, badge, ...nameParts] = parts;
      const sessName = nameParts.join('\t');
      const shortTty = path.basename(tty);

      // Find Claude process on this TTY
      let claudeSessionId = '';
      let cwd = '';
      const psOut = await runCommand('ps', ['-t', shortTty, '-o', 'pid,command']);
      const claudeMatch = psOut.split('\n').find(l => l.includes('claude') && !l.includes('awk'));
      if (claudeMatch) {
        const claudePid = claudeMatch.trim().split(/\s+/)[0];
        const sessFile = path.join(CLAUDE_SESSIONS_DIR, `${claudePid}.json`);
        const sessData = readJSON(sessFile);
        if (sessData) {
          claudeSessionId = sessData.sessionId || '';
          cwd = sessData.cwd || '';
        }
      }

      // Get cwd from the shell process if not already set (non-Claude terminals)
      if (!cwd) {
        const shellMatch = psOut.split('\n').find(l => l.includes('-zsh') || l.includes('bash'));
        if (shellMatch) {
          const shellPid = shellMatch.trim().split(/\s+/)[0];
          cwd = await runCommand('lsof', ['-a', '-p', shellPid, '-d', 'cwd', '-Fn']).then(out => {
            const line = out.split('\n').find(l => l.startsWith('n'));
            return line ? line.slice(1) : '';
          });
        }
      }

      // Find foreground process
      const statOut = await runCommand('ps', ['-t', shortTty, '-o', 'stat,command']);
      let procName = 'unknown';
      const fgLine = statOut.split('\n').find(l => l.match(/^\s*\S*\+/));
      if (fgLine) {
        const cmd = fgLine.trim().split(/\s+/).slice(1)[0] || '';
        procName = path.basename(cmd);
      }

      sessions.push({
        window_id: parseInt(winId) || 0,
        tab: parseInt(tab) || 1,
        session: parseInt(sess) || 1,
        iterm_uuid: itermUuid,
        tty,
        badge: badge || '',
        session_name: sessName || '',
        process: procName,
        claude_session_id: claudeSessionId,
        cwd
      });
    }

    const snapshot = {
      timestamp: now(),
      session_count: sessions.length,
      sessions
    };

    // Update state, preserving missing Claude sessions from previous snapshot
    const state = loadState();
    const liveUuids = new Set(sessions.map(s => s.iterm_uuid));
    const parkedIds = new Set(state.history.map(h => h.claude_session_id));
    if (state.snapshot && state.snapshot.sessions) {
      for (const prev of state.snapshot.sessions) {
        // Keep sessions that: had a Claude session, aren't in the live snapshot, and aren't parked
        if (prev.claude_session_id && !liveUuids.has(prev.iterm_uuid) && !parkedIds.has(prev.claude_session_id)) {
          sessions.push(prev);
        }
      }
      snapshot.sessions = sessions;
      snapshot.session_count = sessions.length;
    }
    state.snapshot = snapshot;
    saveState(state);

    // Timestamped backup
    const backupName = `snapshot-${new Date().toISOString().replace(/[-:T]/g, '').slice(0, 15)}.json`;
    writeJSON(path.join(SNAPSHOT_DIR, backupName), snapshot);

    // Prune old backups (keep last 50)
    const backups = fs.readdirSync(SNAPSHOT_DIR)
      .filter(f => f.startsWith('snapshot-') && f.endsWith('.json'))
      .sort()
      .reverse();
    for (const old of backups.slice(50)) {
      fs.unlinkSync(path.join(SNAPSHOT_DIR, old));
    }

    const claudeCount = sessions.filter(s => s.claude_session_id).length;
    console.log(`[${now()}] Snapshot: ${sessions.length} sessions (${claudeCount} Claude)`);

    return snapshot;
  } finally {
    snapshotInProgress = false;
  }
}

// ─── Liveness Check ──────────────────────────────────────────────────────────

function getRunningSessionIds() {
  const running = new Set();
  try {
    const files = fs.readdirSync(CLAUDE_SESSIONS_DIR).filter(f => f.endsWith('.json'));
    for (const f of files) {
      const pid = f.replace('.json', '');
      if (processIsRunning(pid)) {
        const data = readJSON(path.join(CLAUDE_SESSIONS_DIR, f));
        if (data && data.sessionId) {
          running.add(data.sessionId);
        }
      }
    }
  } catch { /* no sessions dir */ }
  return running;
}

// ─── Focus / Park / Restore ─────────────────────────────────────────────────

async function focusSession(itermUuid) {
  try {
    await runOsascript(`
tell application "iTerm2"
    repeat with w from 1 to (count of windows)
        set win to window w
        repeat with t from 1 to (count of tabs of win)
            repeat with s from 1 to (count of sessions of tab t of win)
                set sess to session s of tab t of win
                if (unique ID of sess) is "${itermUuid}" then
                    select win
                    return "focused"
                end if
            end repeat
        end repeat
    end repeat
    return "not found"
end tell`);
    return { ok: true };
  } catch (err) {
    return { ok: false, error: err.message };
  }
}

async function parkSession(itermUuid) {
  const state = loadState();
  const session = state.snapshot.sessions.find(s => s.iterm_uuid === itermUuid);
  if (!session) return { ok: false, error: 'Session not found' };
  if (!session.claude_session_id) return { ok: false, error: 'Not a Claude session' };

  // Add to history
  if (!state.history.some(h => h.claude_session_id === session.claude_session_id)) {
    state.history.push({
      ...session,
      parked_at: new Date().toISOString().replace('T', ' ').slice(0, 16)
    });
    saveState(state);
  }

  // Close the iTerm2 window
  try {
    await runOsascript(`
tell application "iTerm2"
    repeat with w from 1 to (count of windows)
        set win to window w
        repeat with t from 1 to (count of tabs of win)
            repeat with s from 1 to (count of sessions of tab t of win)
                set sess to session s of tab t of win
                if (unique ID of sess) is "${itermUuid}" then
                    close win
                    return "closed"
                end if
            end repeat
        end repeat
    end repeat
    return "not found"
end tell`);
  } catch { /* window may already be closed */ }

  // Wait for iTerm2 to finish processing the close, then snapshot
  await new Promise(r => setTimeout(r, 2000));
  await takeSnapshot();
  return { ok: true, badge: session.badge };
}

async function restoreSession(claudeSessionId, fromHistory) {
  const state = loadState();
  let session;

  if (fromHistory) {
    const idx = state.history.findIndex(h => h.claude_session_id === claudeSessionId);
    if (idx === -1) return { ok: false, error: 'Not found in history' };
    session = state.history[idx];

    // Remove from history
    state.history.splice(idx, 1);
    saveState(state);
  } else {
    session = state.snapshot.sessions.find(s => s.claude_session_id === claudeSessionId);
    if (!session) return { ok: false, error: 'Session not found' };
  }

  const cwd = session.cwd || os.homedir();
  const badgeB64 = Buffer.from(session.badge || '').toString('base64');
  const claudeCmd = SAFE_MODE
    ? `claude --resume ${claudeSessionId}`
    : `claude --dangerously-skip-permissions --resume ${claudeSessionId}`;

  // Write temp AppleScript file (avoids escaping issues)
  const tmpFile = path.join(os.tmpdir(), `tt-restore-${Date.now()}.applescript`);
  fs.writeFileSync(tmpFile, `tell application "iTerm2"
    set newWindow to (create window with default profile)
    tell current session of current tab of newWindow
        write text "cd ${cwd}"
        delay 1
        write text "printf '\\\\e]1337;SetBadgeFormat=%s\\\\a' '${badgeB64}'"
        delay 2
        write text "${claudeCmd}"
    end tell
end tell`);

  try {
    await runOsascriptFile(tmpFile);
  } finally {
    try { fs.unlinkSync(tmpFile); } catch {}
  }

  // Take fresh snapshot after a delay (let the window open)
  setTimeout(() => takeSnapshot(), 5000);
  return { ok: true, badge: session.badge };
}

// ─── API Handler ─────────────────────────────────────────────────────────────

async function handleAPI(req, res) {
  const url = new URL(req.url, `http://localhost:${PORT}`);
  const pathParts = url.pathname.split('/').filter(Boolean);

  // GET /api/sessions
  if (req.method === 'GET' && url.pathname === '/api/sessions') {
    const state = loadState();
    const running = getRunningSessionIds();
    const parkedIds = new Set(state.history.filter(h => h.claude_session_id).map(h => h.claude_session_id));

    const sessions = state.snapshot.sessions.map(s => ({
      ...s,
      note: state.notes[s.claude_session_id] || state.notes[s.iterm_uuid] || '',
      status: !s.claude_session_id ? 'no-claude'
        : parkedIds.has(s.claude_session_id) ? 'parked'
        : running.has(s.claude_session_id) ? 'running'
        : 'missing'
    }));

    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ ...state.snapshot, sessions }));
    return;
  }

  // GET /api/history
  if (req.method === 'GET' && url.pathname === '/api/history') {
    const state = loadState();
    const running = getRunningSessionIds();

    const entries = state.history.map(h => ({
      ...h,
      note: (h.claude_session_id && state.notes[h.claude_session_id]) || '',
      status: running.has(h.claude_session_id) ? 'running' : 'parked'
    }));

    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(entries));
    return;
  }

  // POST /api/snapshot
  if (req.method === 'POST' && url.pathname === '/api/snapshot') {
    await takeSnapshot();
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ ok: true }));
    return;
  }

  // POST /api/focus/:iterm_uuid
  if (req.method === 'POST' && pathParts[0] === 'api' && pathParts[1] === 'focus' && pathParts[2]) {
    const result = await focusSession(decodeURIComponent(pathParts[2]));
    res.writeHead(result.ok ? 200 : 400, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(result));
    return;
  }

  // POST /api/park/:iterm_uuid
  if (req.method === 'POST' && pathParts[0] === 'api' && pathParts[1] === 'park' && pathParts[2]) {
    const result = await parkSession(decodeURIComponent(pathParts[2]));
    res.writeHead(result.ok ? 200 : 400, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(result));
    return;
  }

  // POST /api/park-missing/:claude_session_id (park a session that's already closed)
  if (req.method === 'POST' && pathParts[0] === 'api' && pathParts[1] === 'park-missing' && pathParts[2]) {
    const sid = decodeURIComponent(pathParts[2]);
    const state = loadState();
    const session = state.snapshot.sessions.find(s => s.claude_session_id === sid);
    if (!session) {
      res.writeHead(400, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ ok: false, error: 'Session not found' }));
      return;
    }
    if (!state.history.some(h => h.claude_session_id === sid)) {
      state.history.push({
        ...session,
        parked_at: new Date().toISOString().replace('T', ' ').slice(0, 16)
      });
      saveState(state);
    }
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ ok: true }));
    return;
  }

  // POST /api/restore/:claude_session_id
  if (req.method === 'POST' && pathParts[0] === 'api' && pathParts[1] === 'restore' && pathParts[2]) {
    const result = await restoreSession(decodeURIComponent(pathParts[2]), false);
    res.writeHead(result.ok ? 200 : 400, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(result));
    return;
  }

  // POST /api/restore-history/:claude_session_id
  if (req.method === 'POST' && pathParts[0] === 'api' && pathParts[1] === 'restore-history' && pathParts[2]) {
    const result = await restoreSession(decodeURIComponent(pathParts[2]), true);
    res.writeHead(result.ok ? 200 : 400, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(result));
    return;
  }

  // PUT /api/note/:session_id
  if (req.method === 'PUT' && pathParts[0] === 'api' && pathParts[1] === 'note' && pathParts[2]) {
    const body = await new Promise((resolve) => {
      let data = '';
      req.on('data', c => data += c);
      req.on('end', () => resolve(data));
    });
    const { note } = JSON.parse(body);
    const sid = decodeURIComponent(pathParts[2]);
    const state = loadState();
    if (note) {
      state.notes[sid] = note;
    } else {
      delete state.notes[sid];
    }
    saveState(state);
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ ok: true }));
    return;
  }

  res.writeHead(404, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify({ error: 'Not found' }));
}

// ─── HTML Dashboard ──────────────────────────────────────────────────────────

function getDashboardHTML() {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>ttracker</title>
<style>
  /* Solarized Light - full palette
     base03  #002b36   base3  #fdf6e3
     base02  #073642   base2  #eee8d5
     base01  #586e75   base1  #93a1a1
     base00  #657b83   base0  #839496
     yellow  #b58900   orange #cb4b16
     red     #dc322f   magenta #d33682
     violet  #6c71c4   blue   #268bd2
     cyan    #2aa198   green  #859900
  */
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: 'SF Mono', 'Menlo', 'Monaco', monospace;
    background: #fdf6e3;
    color: #586e75;
    padding: 24px;
    font-size: 13px;
  }
  h1 {
    color: #268bd2;
    font-size: 22px;
    font-weight: 700;
    letter-spacing: -0.5px;
  }
  h1 span { color: #2aa198; }
  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
    padding-bottom: 16px;
    border-bottom: 2px solid #eee8d5;
  }
  .header-info { color: #839496; font-size: 12px; }
  .header-info span { margin-left: 16px; }
  .refresh-btn {
    background: #2aa198;
    color: #fdf6e3;
    border: none;
    border-radius: 6px;
    padding: 6px 14px;
    cursor: pointer;
    font-family: inherit;
    font-size: 12px;
    font-weight: 600;
  }
  .refresh-btn:hover { opacity: 0.85; }
  .refresh-btn:disabled { background: #93a1a1; }
  h2 {
    color: #268bd2;
    font-size: 15px;
    font-weight: 600;
    margin: 28px 0 12px;
    padding-left: 8px;
    border-left: 3px solid #268bd2;
  }
  .history-heading { color: #6c71c4; border-left-color: #6c71c4; }
  .count { color: #93a1a1; font-weight: 400; margin-left: 8px; }
  table {
    width: 100%;
    border-collapse: collapse;
    margin-bottom: 8px;
    border-radius: 6px;
    overflow: hidden;
  }
  th {
    text-align: left;
    padding: 10px 12px;
    background: #eee8d5;
    color: #586e75;
    font-weight: 600;
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-bottom: 2px solid #d3cdb8;
  }
  td {
    padding: 9px 12px;
    border-bottom: 1px solid #eee8d5;
    vertical-align: middle;
  }
  tr:hover { background: #eee8d5; }
  .badge-cell {
    color: #d33682;
    font-weight: 700;
  }
  .session-name { color: #586e75; }
  .process-cell { color: #2aa198; font-weight: 500; }
  .folder-cell { color: #b58900; font-size: 12px; }
  .session-id {
    color: #93a1a1;
    font-size: 11px;
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .status {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 500;
  }
  .dot {
    width: 9px;
    height: 9px;
    border-radius: 50%;
    display: inline-block;
  }
  .dot-running { background: #859900; }
  .dot-missing { background: #dc322f; }
  .dot-parked { background: #6c71c4; }
  .dot-no-claude { background: #93a1a1; }
  .status-running { color: #859900; }
  .status-missing { color: #dc322f; }
  .status-parked { color: #6c71c4; }
  .status-no-claude { color: #93a1a1; }
  .btn {
    border: none;
    border-radius: 5px;
    padding: 4px 10px;
    cursor: pointer;
    font-family: inherit;
    font-size: 11px;
    font-weight: 600;
    transition: all 0.15s;
    color: #fdf6e3;
  }
  .btn:disabled { opacity: 0.4; cursor: not-allowed; }
  .btn-focus {
    background: #859900;
  }
  .btn-focus:hover:not(:disabled) { opacity: 0.85; }
  .btn-park {
    background: #cb4b16;
  }
  .btn-park:hover:not(:disabled) { opacity: 0.85; }
  .btn-restore {
    background: #268bd2;
  }
  .btn-restore:hover:not(:disabled) { opacity: 0.85; }
  .empty-state {
    color: #93a1a1;
    padding: 24px;
    text-align: center;
    font-style: italic;
  }
  .parked-at { color: #6c71c4; font-size: 12px; font-weight: 500; }
  .actions { white-space: nowrap; }
  .note-input {
    background: transparent;
    border: 1px solid transparent;
    border-radius: 4px;
    color: #586e75;
    font-family: inherit;
    font-size: 12px;
    padding: 3px 6px;
    width: 100%;
    min-width: 120px;
  }
  .note-input:hover { border-color: #2aa198; }
  .note-input:focus {
    outline: none;
    border-color: #268bd2;
    background: #eee8d5;
  }
</style>
</head>
<body>

<div class="header">
  <div>
    <h1>t<span>tracker</span></h1>
  </div>
  <div class="header-info">
    <span id="snapshot-time"></span>
    <span id="session-counts"></span>
    <button class="refresh-btn" onclick="forceSnapshot()">Snapshot Now</button>
  </div>
</div>

<h2>Active Sessions <span class="count" id="active-count"></span></h2>
<table>
  <thead>
    <tr>
      <th style="width:50px">#</th>
      <th>Badge</th>
      <th>Session Name</th>
      <th>Folder</th>
      <th>Note</th>
      <th>Process</th>
      <th>Claude Session</th>
      <th>Status</th>
      <th style="width:200px">Action</th>
    </tr>
  </thead>
  <tbody id="active-body"></tbody>
</table>

<h2 class="history-heading">Parked Sessions <span class="count" id="history-count"></span></h2>
<table>
  <thead>
    <tr>
      <th style="width:50px">#</th>
      <th>Badge</th>
      <th>Session Name</th>
      <th>Folder</th>
      <th>Note</th>
      <th>Claude Session</th>
      <th>Parked At</th>
      <th>Status</th>
      <th style="width:100px">Action</th>
    </tr>
  </thead>
  <tbody id="history-body"></tbody>
</table>

<script>
const API = '';
let refreshTimer;

function escapeHtml(s) {
  const d = document.createElement('div');
  d.textContent = s || '';
  return d.innerHTML;
}

function statusDot(status) {
  const labels = { running: 'running', missing: 'missing', parked: 'parked', 'no-claude': 'idle' };
  return '<span class="status status-' + status + '"><span class="dot dot-' + status + '"></span>' + (labels[status] || status) + '</span>';
}

async function fetchSessions() {
  try {
    const res = await fetch(API + '/api/sessions');
    return await res.json();
  } catch { return { timestamp: '?', sessions: [], session_count: 0 }; }
}

async function fetchHistory() {
  try {
    const res = await fetch(API + '/api/history');
    return await res.json();
  } catch { return []; }
}

function renderActive(data) {
  const el = document.getElementById('active-body');
  const sessions = data.sessions;
  document.getElementById('active-count').textContent = '(' + sessions.length + ')';
  document.getElementById('snapshot-time').textContent = 'Snapshot: ' + (data.timestamp || '?');

  const running = sessions.filter(s => s.status === 'running').length;
  const missing = sessions.filter(s => s.status === 'missing').length;
  document.getElementById('session-counts').textContent = running + ' running, ' + missing + ' missing';

  if (sessions.length === 0) {
    el.innerHTML = '<tr><td colspan="9" class="empty-state">No sessions found</td></tr>';
    return;
  }

  el.innerHTML = sessions.map((s, i) => {
    let action = '<button class="btn btn-focus" onclick="focusSession(\\'' + s.iterm_uuid + '\\')">Focus</button>';
    if (s.status === 'running') {
      action += ' <button class="btn btn-park" onclick="parkSession(\\'' + s.iterm_uuid + '\\')">Park</button>';
    } else if (s.status === 'missing') {
      action = '<button class="btn btn-restore" onclick="restoreSession(\\'' + s.claude_session_id + '\\')">Restore</button>'
        + ' <button class="btn btn-park" onclick="parkMissing(\\'' + s.claude_session_id + '\\')">Park</button>';
    }
    const noteKey = s.claude_session_id || s.iterm_uuid;
    const noteVal = escapeHtml(s.note);
    const folder = s.cwd ? s.cwd.replace(/^\\/Users\\/[^\\/]+\\//, '~/') : '';
    return '<tr>'
      + '<td>' + (i + 1) + '</td>'
      + '<td class="badge-cell">' + escapeHtml(s.badge) + '</td>'
      + '<td>' + escapeHtml(s.session_name) + '</td>'
      + '<td class="folder-cell">' + escapeHtml(folder) + '</td>'
      + '<td><input class="note-input" value="' + noteVal + '" placeholder="..." '
      + 'onblur="saveNote(\\'' + noteKey + '\\', this.value)" '
      + 'onkeydown="if(event.key===\\'Enter\\')this.blur()" /></td>'
      + '<td class="process-cell">' + escapeHtml(s.process) + '</td>'
      + '<td class="session-id">' + escapeHtml(s.claude_session_id) + '</td>'
      + '<td>' + statusDot(s.status) + '</td>'
      + '<td class="actions">' + action + '</td>'
      + '</tr>';
  }).join('');
}

function renderHistory(entries) {
  const el = document.getElementById('history-body');
  document.getElementById('history-count').textContent = '(' + entries.length + ')';

  if (entries.length === 0) {
    el.innerHTML = '<tr><td colspan="9" class="empty-state">No parked sessions</td></tr>';
    return;
  }

  el.innerHTML = entries.map((h, i) => {
    const action = h.status === 'running'
      ? '<span style="color:#859900;font-size:12px">open</span>'
      : '<button class="btn btn-restore" onclick="restoreFromHistory(\\'' + h.claude_session_id + '\\')">Restore</button>';
    const noteVal = escapeHtml(h.note);
    const folder = h.cwd ? h.cwd.replace(/^\\/Users\\/[^\\/]+\\//, '~/') : '';
    return '<tr>'
      + '<td>' + (i + 1) + '</td>'
      + '<td class="badge-cell">' + escapeHtml(h.badge) + '</td>'
      + '<td>' + escapeHtml(h.session_name) + '</td>'
      + '<td class="folder-cell">' + escapeHtml(folder) + '</td>'
      + '<td><input class="note-input" value="' + noteVal + '" placeholder="..." '
      + 'onblur="saveNote(\\'' + h.claude_session_id + '\\', this.value)" '
      + 'onkeydown="if(event.key===\\'Enter\\')this.blur()" /></td>'
      + '<td class="session-id">' + escapeHtml(h.claude_session_id) + '</td>'
      + '<td class="parked-at">' + escapeHtml(h.parked_at) + '</td>'
      + '<td>' + statusDot(h.status) + '</td>'
      + '<td class="actions">' + action + '</td>'
      + '</tr>';
  }).join('');
}

async function refresh() {
  // Skip refresh if user is typing in a note field
  if (document.activeElement && document.activeElement.classList.contains('note-input')) return;
  const [sessions, history] = await Promise.all([fetchSessions(), fetchHistory()]);
  renderActive(sessions);
  renderHistory(history);
}

async function forceSnapshot() {
  const btn = document.querySelector('.refresh-btn');
  btn.textContent = 'Snapshotting...';
  btn.disabled = true;
  await fetch(API + '/api/snapshot', { method: 'POST' });
  await refresh();
  btn.textContent = 'Snapshot Now';
  btn.disabled = false;
}

async function saveNote(sessionId, note) {
  await fetch(API + '/api/note/' + encodeURIComponent(sessionId), {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ note: note.trim() })
  });
}

async function focusSession(itermUuid) {
  await fetch(API + '/api/focus/' + encodeURIComponent(itermUuid), { method: 'POST' });
}

async function parkSession(itermUuid) {
  if (!confirm('Park this session and close the terminal?')) return;
  await fetch(API + '/api/park/' + encodeURIComponent(itermUuid), { method: 'POST' });
  await refresh();
}

async function parkMissing(sessionId) {
  await fetch(API + '/api/park-missing/' + encodeURIComponent(sessionId), { method: 'POST' });
  await refresh();
}

async function restoreSession(sessionId) {
  await fetch(API + '/api/restore/' + encodeURIComponent(sessionId), { method: 'POST' });
  setTimeout(refresh, 3000);
}

async function restoreFromHistory(sessionId) {
  await fetch(API + '/api/restore-history/' + encodeURIComponent(sessionId), { method: 'POST' });
  await refresh();
}

// Initial load + auto-refresh
refresh();
refreshTimer = setInterval(refresh, 5000);
</script>

</body>
</html>`;
}

// ─── HTTP Server ─────────────────────────────────────────────────────────────

async function handleRequest(req, res) {
  try {
    if (req.url === '/' || req.url === '/index.html') {
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end(getDashboardHTML());
      return;
    }

    if (req.url.startsWith('/api/')) {
      await handleAPI(req, res);
      return;
    }

    res.writeHead(404);
    res.end('Not found');
  } catch (err) {
    console.error('Request error:', err.message);
    res.writeHead(500, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ error: err.message }));
  }
}

// ─── Startup ─────────────────────────────────────────────────────────────────

async function main() {
  fs.mkdirSync(SNAPSHOT_DIR, { recursive: true });

  // Kill any existing process on our port
  try {
    const { stdout } = await new Promise((resolve) => {
      exec(`lsof -ti:${PORT}`, (err, stdout) => resolve({ stdout: (stdout || '').trim() }));
    });
    if (stdout) {
      for (const pid of stdout.split('\n').filter(Boolean)) {
        try { process.kill(parseInt(pid)); } catch {}
      }
      console.log('Stopped previous server');
      await new Promise(r => setTimeout(r, 500));
    }
  } catch {}

  // Stop existing shell daemon if running
  try {
    if (fs.existsSync(PID_FILE)) {
      const pid = fs.readFileSync(PID_FILE, 'utf8').trim();
      if (processIsRunning(pid)) {
        process.kill(parseInt(pid));
        console.log(`Stopped existing daemon (PID ${pid})`);
      }
      fs.unlinkSync(PID_FILE);
    }
  } catch {}

  // Take initial snapshot
  console.log('Taking initial snapshot...');
  await takeSnapshot();

  // Start periodic snapshots
  setInterval(() => takeSnapshot(), INTERVAL_MIN * 60 * 1000);

  // Start HTTP server
  const server = http.createServer(handleRequest);
  server.listen(PORT, '127.0.0.1', () => {
    console.log(`\nttracker dashboard: http://localhost:${PORT}`);
    console.log(`Snapshot interval: ${INTERVAL_MIN} minutes`);
    console.log(`Permission mode: ${SAFE_MODE ? 'safe' : 'dangerously-skip-permissions'}`);
    console.log('');
  });

  // Open browser if --open flag
  if (process.argv.includes('--open')) {
    exec(`open http://localhost:${PORT}`);
  }
}

main().catch(err => {
  console.error('Fatal:', err);
  process.exit(1);
});
