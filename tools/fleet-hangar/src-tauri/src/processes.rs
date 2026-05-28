use serde::{Deserialize, Serialize};
use std::collections::{HashMap, VecDeque};
use std::fs::{File, OpenOptions};
use std::io::{BufWriter, Write};
use std::path::PathBuf;
use std::process::Stdio;
use std::sync::{Arc, Mutex, OnceLock};
use tauri::{AppHandle, Emitter, Manager, State};
use tokio::io::{AsyncBufReadExt, BufReader};
use tokio::net::TcpStream;
use tokio::process::Command;
use tokio_rustls::TlsConnector;

const LOG_TAIL_CAP: usize = 60;
const LOG_CHANNEL_CAP: usize = 50_000;
/// Rotate the on-disk log when it crosses this size. Keeps one previous
/// generation as `<channel>.log.1`. Fleet serve under debug logging can
/// produce ~tens of MB per hour; without this the file grows unbounded.
const LOG_FILE_MAX_BYTES: u64 = 16 * 1024 * 1024;

#[derive(Debug, Clone, Serialize)]
pub struct ProcInfo {
    pub id: String,
    pub label: String,
    pub command: String,
    pub cwd: String,
    pub state: String, // "idle" | "running" | "done" | "failed" | "stopping"
    pub started_at_ms: Option<u64>,
    pub ended_at_ms: Option<u64>,
    pub exit_code: Option<i32>,
    /// Terminating signal number on Unix when the process was killed by
    /// a signal (None = exited normally, regardless of code). 9 =
    /// SIGKILL (external killer), 11 = SIGSEGV, 6 = SIGABRT/panic, etc.
    /// The frontend uses this to surface the real cause instead of
    /// guessing from the last stderr line.
    pub exit_signal: Option<i32>,
    pub recent_log: Vec<String>,
    /// True when the user explicitly asked us to stop this process via the
    /// UI (stop button, stop all, or docker compose down). Used by the
    /// frontend so we don't visually flag intentional stops as failures.
    pub was_user_stopped: bool,
}

#[derive(Debug, Clone, Serialize)]
pub struct LogLine {
    pub proc_id: String,
    pub stream: String,
    pub line: String,
    pub ts_ms: u64,
}

#[derive(Debug, Clone, Serialize)]
pub struct ProcEvent {
    pub proc_id: String,
    pub state: String,
    pub exit_code: Option<i32>,
    pub exit_signal: Option<i32>,
}

/// Translate a signal number to its short name. Just the ones we're
/// likely to see — anything else falls through to the numeric form.
fn signal_name(sig: i32) -> &'static str {
    match sig {
        1 => "SIGHUP",
        2 => "SIGINT",
        3 => "SIGQUIT",
        6 => "SIGABRT",
        9 => "SIGKILL",
        11 => "SIGSEGV",
        13 => "SIGPIPE",
        15 => "SIGTERM",
        _ => "?",
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct LogEntry {
    pub ts_ms: u64,
    pub stream: String,       // "stdout" | "stderr"
    pub level: Option<String>, // "debug" | "info" | "warn" | "error"
    pub message: String,
    pub channel: String,
}

#[derive(Debug, Clone)]
pub struct StartArgs {
    pub label: String,
    pub cwd: String,
    pub program: String,
    pub args: Vec<String>,
    pub log_channel: Option<String>,
    /// Env vars applied on top of (not replacing) the inherited
    /// environment. Restart reuses these.
    pub env: Vec<(String, String)>,
}

#[derive(Debug, Clone, Serialize)]
pub struct ContainerState {
    pub name: String,
    pub state: String,
}

#[derive(Debug, Clone, Serialize)]
pub struct DockerStatus {
    pub running: bool,
    pub containers: Vec<ContainerState>,
}

/// Cached on-disk log writer per channel. Holding a BufWriter avoids the
/// open/close-per-line syscall churn that hits hard under fleet serve
/// debug logging. `bytes_written` tracks size for in-process rotation
/// without a separate metadata stat.
pub struct ChannelWriter {
    pub writer: BufWriter<File>,
    pub bytes_written: u64,
    pub path: PathBuf,
}

pub struct ProcessManager {
    pub procs: Mutex<HashMap<String, ProcInfo>>,
    pub pids: Mutex<HashMap<String, u32>>,
    pub last_args: Mutex<HashMap<String, StartArgs>>,
    pub log_store: Mutex<HashMap<String, VecDeque<LogEntry>>>,
    pub log_writers: Mutex<HashMap<String, ChannelWriter>>,
}

impl ProcessManager {
    pub fn new() -> Self {
        Self {
            procs: Mutex::new(HashMap::new()),
            pids: Mutex::new(HashMap::new()),
            last_args: Mutex::new(HashMap::new()),
            log_store: Mutex::new(HashMap::new()),
            log_writers: Mutex::new(HashMap::new()),
        }
    }
}

fn now_ms() -> u64 {
    std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

/// A `docker` command with the resolved login-shell PATH applied.
/// docker lives in /usr/local/bin or /opt/homebrew/bin — neither is on
/// the PATH a Finder-launched app inherits, so a bare spawn fails in
/// the packaged build. Routing every docker call through here keeps the
/// PATH fix in one place.
fn docker_cmd() -> Command {
    let mut c = Command::new("docker");
    c.env("PATH", crate::shellpath::shell_path());
    c
}

fn append_recent_log(state: &Arc<ProcessManager>, id: &str, line: &str) {
    // Recover from a poisoned guard rather than panicking the log spawn
    // task — losing a tail of recent_log on a previous panic is fine,
    // crashing the whole logger because of it is not.
    let mut map = state
        .procs
        .lock()
        .unwrap_or_else(|p| p.into_inner());
    if let Some(info) = map.get_mut(id) {
        info.recent_log.push(line.to_string());
        let len = info.recent_log.len();
        if len > LOG_TAIL_CAP {
            info.recent_log.drain(0..(len - LOG_TAIL_CAP));
        }
    }
}

/// ASCII-case-insensitive contains. We avoid allocating a lowercased
/// copy of every log line — under fleet serve debug logging this fires
/// thousands of times a second.
fn icontains(hay: &str, needle: &str) -> bool {
    if needle.len() > hay.len() {
        return false;
    }
    let hb = hay.as_bytes();
    let nb = needle.as_bytes();
    'outer: for i in 0..=(hb.len() - nb.len()) {
        for j in 0..nb.len() {
            if !hb[i + j].eq_ignore_ascii_case(&nb[j]) {
                continue 'outer;
            }
        }
        return true;
    }
    false
}

fn detect_level(msg: &str) -> Option<String> {
    // Logrus / slog-style key=value
    if icontains(msg, "level=error") || icontains(msg, "level=err") {
        return Some("error".into());
    }
    if icontains(msg, "level=warn") {
        return Some("warn".into());
    }
    if icontains(msg, "level=debug") {
        return Some("debug".into());
    }
    if icontains(msg, "level=info") {
        return Some("info".into());
    }
    // Token-based — match common prefixed forms in the first 64 bytes
    // (where timestamp+level live; checking the whole line gets noisy).
    let head_len = msg.len().min(64);
    let head = &msg[..head_len];
    let head_lower_starts_with_error = head.len() >= 5
        && head.as_bytes()[..5].eq_ignore_ascii_case(b"error");
    if icontains(head, " error ")
        || icontains(head, "] error ")
        || head_lower_starts_with_error
    {
        return Some("error".into());
    }
    if icontains(head, " warn ") || icontains(head, "] warn ") || icontains(head, " warning") {
        return Some("warn".into());
    }
    if icontains(head, " debug ") || icontains(head, "] debug ") {
        return Some("debug".into());
    }
    if icontains(head, " info ") || icontains(head, "] info ") {
        return Some("info".into());
    }
    None
}

/// Scrub obvious bearer tokens before writing logs to disk. Fleet serve
/// at --logging_debug echoes Authorization headers; we don't need those
/// persisted to ~/Library/Logs where any other local user could read
/// them. Best-effort; not meant as a comprehensive PII filter.
///
/// Implemented via regex to dodge byte-indexed slicing entirely — the
/// previous version landed `i` mid-UTF-8 on certain content and panicked
/// on `&line[i..]`. A panic here killed the spawn_log_reader task,
/// which closed the pipe to fleet, which then died on EPIPE. Catch:
/// fleet died silently with no log trail. Took a while to find.
fn scrub_secrets(line: &str) -> String {
    use regex_lite::Regex;
    static RE: OnceLock<Regex> = OnceLock::new();
    let re = RE.get_or_init(|| {
        // Three patterns ORed:
        //   "Bearer <token>" (case-insensitive)
        //   "(token|password|authtoken|authorization)=<value>"
        //   (token continues until whitespace / quote / & / line end)
        Regex::new(
            r#"(?i)(Bearer\s+[^\s'"\r\n]+)|((?:token|password|authtoken|authorization)=[^\s'"&\r\n]+)"#,
        )
        .expect("scrub_secrets regex compiles")
    });
    // Caps: we want to keep the key/prefix and replace only the value.
    re.replace_all(line, |caps: &regex_lite::Captures| {
        let m = caps.get(0).unwrap().as_str();
        // Find where the value starts: after "Bearer " or after "=".
        if let Some(rest) = strip_prefix_ci(m, "Bearer ") {
            let _ = rest; // silence
            return "Bearer [redacted]".to_string();
        }
        if let Some(eq_idx) = m.find('=') {
            return format!("{}=[redacted]", &m[..eq_idx]);
        }
        m.to_string()
    })
    .into_owned()
}

fn strip_prefix_ci<'a>(s: &'a str, prefix: &str) -> Option<&'a str> {
    if s.len() < prefix.len() {
        return None;
    }
    let (head, tail) = s.split_at(prefix.len());
    if head.eq_ignore_ascii_case(prefix) {
        Some(tail)
    } else {
        None
    }
}

fn logs_dir(app: &AppHandle) -> Result<PathBuf, String> {
    let dir = app.path().app_log_dir().map_err(|e| e.to_string())?;
    std::fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    Ok(dir)
}

fn log_file_path(app: &AppHandle, channel: &str) -> Result<PathBuf, String> {
    Ok(logs_dir(app)?.join(format!("{channel}.log")))
}

// ----- crash-survival PID tracking -----
//
// `kill_on_drop(true)` cleans up children when our `Child` handle is
// dropped — but a hard parent death (SIGKILL from `tauri dev` reloading
// after a crash, force-quit, OS panic) skips all destructors. The
// children outlive us. To recover, we persist every running spawn to
// `app_data_dir/running.json` and, on next startup, look up each pid:
// if it's still alive AND its `ps` command line still matches what we
// recorded, we SIGTERM (then SIGKILL) it. The command-line match is
// what keeps us from killing a recycled pid that now belongs to
// something innocent.

#[derive(Debug, Clone, Serialize, Deserialize)]
struct PidRecord {
    id: String,
    pid: u32,
    program: String,
    args: Vec<String>,
}

fn pid_file_path(app: &AppHandle) -> Result<PathBuf, String> {
    let dir = app.path().app_data_dir().map_err(|e| e.to_string())?;
    std::fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    Ok(dir.join("running.json"))
}

fn collect_pid_records(state: &ProcessManager) -> Vec<PidRecord> {
    let pids = state
        .pids
        .lock()
        .unwrap_or_else(|p| p.into_inner());
    let last_args = state
        .last_args
        .lock()
        .unwrap_or_else(|p| p.into_inner());
    pids.iter()
        .filter_map(|(id, pid)| {
            let a = last_args.get(id)?;
            Some(PidRecord {
                id: id.clone(),
                pid: *pid,
                program: a.program.clone(),
                args: a.args.clone(),
            })
        })
        .collect()
}

fn write_pid_file(app: &AppHandle, state: &ProcessManager) {
    let Ok(path) = pid_file_path(app) else {
        return;
    };
    let records = collect_pid_records(state);
    if records.is_empty() {
        // No live spawns; remove the file rather than leave an empty
        // array. A missing file on next startup means "nothing to
        // clean," which is the same fast-path.
        let _ = std::fs::remove_file(&path);
        return;
    }
    if let Ok(json) = serde_json::to_string(&records) {
        let _ = std::fs::write(&path, json);
    }
}

#[cfg(unix)]
fn pid_is_alive(pid: u32) -> bool {
    // Signal 0 checks existence/perm without delivering anything. 0 =
    // exists, ESRCH = gone, EPERM = exists but unsignalable (still alive).
    let r = unsafe { libc::kill(pid as libc::pid_t, 0) };
    if r == 0 {
        return true;
    }
    std::io::Error::last_os_error().raw_os_error() != Some(libc::ESRCH)
}

#[cfg(not(unix))]
fn pid_is_alive(_pid: u32) -> bool {
    false
}

fn pid_matches_record(pid: u32, program: &str, args: &[String]) -> bool {
    // Read the current command line via `ps -o command= -p <pid>`. If
    // the program basename and the first arg both appear, we're
    // confident the pid is still our spawn. Pid recycling on macOS is
    // common enough that a bare "is pid alive" check would risk
    // SIGTERMing whoever happens to occupy that pid now.
    let output = std::process::Command::new("ps")
        .args(["-p", &pid.to_string(), "-o", "command="])
        .output();
    let Ok(out) = output else {
        return false;
    };
    let line = String::from_utf8_lossy(&out.stdout);
    let line = line.trim();
    if line.is_empty() {
        return false;
    }
    let prog_basename = std::path::Path::new(program)
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or(program);
    if !line.contains(prog_basename) {
        return false;
    }
    if let Some(first) = args.first() {
        if !line.contains(first) {
            return false;
        }
    }
    true
}

/// Read the prior session's running.json and SIGTERM/SIGKILL anything
/// still alive whose command line matches what we recorded. Called once
/// at app startup before the tray and command pipeline come up.
pub fn clean_orphans_from_prior_run(app: &AppHandle) {
    let Ok(path) = pid_file_path(app) else {
        return;
    };
    if !path.exists() {
        return;
    }
    let records: Vec<PidRecord> = std::fs::read_to_string(&path)
        .ok()
        .and_then(|s| serde_json::from_str(&s).ok())
        .unwrap_or_default();
    // Wipe immediately — whether we successfully kill or not, this is
    // stale bookkeeping and we don't want it tripping next startup.
    let _ = std::fs::remove_file(&path);

    for r in records {
        if !pid_is_alive(r.pid) || !pid_matches_record(r.pid, &r.program, &r.args) {
            continue;
        }
        #[cfg(unix)]
        unsafe {
            // Try the process group first (we spawn with
            // process_group(0) so pgid == pid for our children),
            // falling back to the pid itself if that returns ESRCH.
            let pg = -(r.pid as i32);
            if libc::kill(pg, libc::SIGTERM) != 0 {
                libc::kill(r.pid as i32, libc::SIGTERM);
            }
        }
        // Brief grace before escalating. 500ms is plenty for python
        // http.server / ngrok; long-runners like fleet serve usually
        // get killed cleanly via SIGTERM here too.
        std::thread::sleep(std::time::Duration::from_millis(500));
        if pid_is_alive(r.pid) {
            #[cfg(unix)]
            unsafe {
                let pg = -(r.pid as i32);
                if libc::kill(pg, libc::SIGKILL) != 0 {
                    libc::kill(r.pid as i32, libc::SIGKILL);
                }
            }
        }
    }
}

fn open_writer(path: &PathBuf) -> std::io::Result<ChannelWriter> {
    let file = OpenOptions::new().create(true).append(true).open(path)?;
    let bytes_written = file.metadata().map(|m| m.len()).unwrap_or(0);
    Ok(ChannelWriter {
        writer: BufWriter::new(file),
        bytes_written,
        path: path.clone(),
    })
}

fn rotate_if_needed(cw: &mut ChannelWriter) {
    if cw.bytes_written < LOG_FILE_MAX_BYTES {
        return;
    }
    // Flush and drop the writer so the file is closed before we move it.
    let _ = cw.writer.flush();
    let rotated = cw.path.with_extension("log.1");
    let _ = std::fs::rename(&cw.path, &rotated);
    if let Ok(new) = open_writer(&cw.path) {
        *cw = new;
    }
}

fn write_log_disk(
    app: &AppHandle,
    state: &Arc<ProcessManager>,
    channel: &str,
    entry: &LogEntry,
) {
    let Ok(path) = log_file_path(app, channel) else {
        return;
    };
    let Ok(mut writers) = state.log_writers.lock() else {
        return;
    };
    let cw = match writers.get_mut(channel) {
        Some(cw) => cw,
        None => match open_writer(&path) {
            Ok(cw) => writers.entry(channel.to_string()).or_insert(cw),
            Err(_) => return,
        },
    };
    rotate_if_needed(cw);
    // Tab-delimited so the message can contain anything (we strip
    // embedded tabs). Scrub bearer tokens / password= before persist.
    let scrubbed = scrub_secrets(&entry.message);
    let msg = scrubbed.replace('\t', "    ");
    let line = format!("{}\t{}\t{}\n", entry.ts_ms, entry.stream, msg);
    if cw.writer.write_all(line.as_bytes()).is_ok() {
        cw.bytes_written += line.len() as u64;
    }
    // BufWriter auto-flushes at ~8KiB. That's a win for chatty stdout,
    // but it means when a process crashes, the last ~8KiB of its output
    // sits in the buffer forever — exactly the lines you most want to
    // read on disk. Flush stderr immediately so error tails are
    // durable, and rely on lazy flush for stdout.
    if entry.stream == "stderr" {
        let _ = cw.writer.flush();
    }
}

/// Drain any buffered bytes for `channel` to disk, then drop the cached
/// writer. Called when a managed process exits so the on-disk log tail
/// reflects everything that was emitted up to the moment of exit (the
/// lines you actually want when diagnosing a crash).
fn flush_log_writer(state: &Arc<ProcessManager>, channel: &str) {
    let Ok(mut writers) = state.log_writers.lock() else {
        return;
    };
    if let Some(mut cw) = writers.remove(channel) {
        let _ = cw.writer.flush();
    }
}

fn spawn_log_reader<R>(
    app: AppHandle,
    state: Arc<ProcessManager>,
    id: String,
    log_channel: Option<String>,
    stream_name: &'static str,
    reader: R,
) where
    R: tokio::io::AsyncRead + Unpin + Send + 'static,
{
    tokio::spawn(async move {
        let mut lines = BufReader::new(reader).lines();
        while let Ok(Some(line)) = lines.next_line().await {
            append_recent_log(&state, &id, &line);
            let ts = now_ms();
            if let Some(ch) = &log_channel {
                let level = detect_level(&line);
                let entry = LogEntry {
                    ts_ms: ts,
                    stream: stream_name.to_string(),
                    level,
                    message: line.clone(),
                    channel: ch.clone(),
                };
                push_to_log_store(&state, entry.clone());
                write_log_disk(&app, &state, ch, &entry);
            }
            let _ = app.emit(
                "proc:log",
                LogLine {
                    proc_id: id.clone(),
                    stream: stream_name.to_string(),
                    line,
                    ts_ms: ts,
                },
            );
        }
        // Stream closed = the process is on its way out. Flush so the
        // tail (the lines that actually explain why it died) is durable
        // before the BufWriter goes idle.
        if let Some(ch) = &log_channel {
            flush_log_writer(&state, ch);
        }
    });
}

fn push_to_log_store(state: &Arc<ProcessManager>, entry: LogEntry) {
    let mut store = state
        .log_store
        .lock()
        .unwrap_or_else(|p| p.into_inner());
    let buf = store
        .entry(entry.channel.clone())
        .or_insert_with(VecDeque::new);
    buf.push_back(entry);
    while buf.len() > LOG_CHANNEL_CAP {
        buf.pop_front();
    }
}

#[tauri::command]
pub async fn list_processes(
    state: State<'_, Arc<ProcessManager>>,
) -> Result<Vec<ProcInfo>, String> {
    let map = state.procs.lock().map_err(|e| e.to_string())?;
    Ok(map.values().cloned().collect())
}

// rename_all = "snake_case" because the frontend invokes this with
// snake_case keys (id, log_channel, ...). Tauri v2's default is to expect
// camelCase, which silently dropped `log_channel` as None — meaning the
// structured log store was never populated and the Logs tab stayed empty.
#[tauri::command(rename_all = "snake_case")]
pub async fn start_process(
    app: AppHandle,
    state: State<'_, Arc<ProcessManager>>,
    id: String,
    label: String,
    cwd: String,
    program: String,
    args: Vec<String>,
    log_channel: Option<String>,
    env: Option<Vec<(String, String)>>,
) -> Result<(), String> {
    spawn_managed(
        app,
        state.inner().clone(),
        id,
        label,
        cwd,
        program,
        args,
        log_channel,
        env.unwrap_or_default(),
    )
    .await
}

async fn spawn_managed(
    app: AppHandle,
    state: Arc<ProcessManager>,
    id: String,
    label: String,
    cwd: String,
    program: String,
    args: Vec<String>,
    log_channel: Option<String>,
    env: Vec<(String, String)>,
) -> Result<(), String> {
    {
        let pids = state.pids.lock().map_err(|e| e.to_string())?;
        if pids.contains_key(&id) {
            return Err(format!("process {id} is already running"));
        }
    }

    let mut cmd = Command::new(&program);
    cmd.args(&args)
        .current_dir(&cwd)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .stdin(Stdio::null());
    // Use the login-shell PATH so bare-name programs (go, docker, make,
    // python3, ngrok, …) resolve when launched from Finder, not just
    // under `tauri dev`. Set before the caller env loop so an explicit
    // PATH override from the caller (none today) would still win.
    cmd.env("PATH", crate::shellpath::shell_path());
    // Layer caller-supplied env vars on top of inherited env. Empty-key
    // rows would mean nothing to the OS and silently no-op on some
    // platforms — skip them here so the contract is "rows we pass got
    // set." Empty values are kept: `FOO=` is a real, distinct state.
    for (k, v) in &env {
        if k.is_empty() {
            continue;
        }
        cmd.env(k, v);
    }
    // Intentionally NOT setting `kill_on_drop(true)`. With it, every
    // time `tauri dev` rebuilds and restarts the app, tokio runs
    // destructors on the Child handles which SIGKILL fleet (and
    // anything else we've spawned) — fleet dies silently with no log
    // output. Without it, the spawned process becomes parent-less if
    // the app exits without cleanup, and the next startup
    // catches it via `clean_orphans_from_prior_run` (running.json).
    // User-initiated quit still tears everything down explicitly via
    // `shutdown_now`.

    #[cfg(unix)]
    cmd.process_group(0);

    let mut child = cmd
        .spawn()
        .map_err(|e| format!("failed to spawn {program}: {e}"))?;
    let pid = child.id().ok_or_else(|| "child has no pid".to_string())?;

    let display = format!("{program} {}", args.join(" "));
    let info = ProcInfo {
        id: id.clone(),
        label: label.clone(),
        command: display,
        cwd: cwd.clone(),
        state: "running".into(),
        started_at_ms: Some(now_ms()),
        ended_at_ms: None,
        exit_code: None,
        exit_signal: None,
        recent_log: Vec::new(),
        was_user_stopped: false,
    };

    {
        let mut map = state.procs.lock().map_err(|e| e.to_string())?;
        map.insert(id.clone(), info);
        let mut pids = state.pids.lock().map_err(|e| e.to_string())?;
        pids.insert(id.clone(), pid);
        let mut la = state.last_args.lock().map_err(|e| e.to_string())?;
        la.insert(
            id.clone(),
            StartArgs {
                label: label.clone(),
                cwd: cwd.clone(),
                program: program.clone(),
                args: args.clone(),
                log_channel: log_channel.clone(),
                env: env.clone(),
            },
        );
    }
    // Persist running set so the next startup can clean orphans if we
    // die without our wait task reaping the child.
    write_pid_file(&app, &state);

    let _ = app.emit(
        "proc:state",
        ProcEvent {
            proc_id: id.clone(),
            state: "running".into(),
            exit_code: None,
            exit_signal: None,
        },
    );

    if let Some(out) = child.stdout.take() {
        spawn_log_reader(
            app.clone(),
            state.clone(),
            id.clone(),
            log_channel.clone(),
            "stdout",
            out,
        );
    }
    if let Some(err) = child.stderr.take() {
        spawn_log_reader(
            app.clone(),
            state.clone(),
            id.clone(),
            log_channel.clone(),
            "stderr",
            err,
        );
    }

    let app3 = app.clone();
    let state3 = state.clone();
    let id3 = id.clone();
    tokio::spawn(async move {
        let status = child.wait().await;
        let exit_code = status.as_ref().ok().and_then(|s| s.code());
        let ok = status.as_ref().map(|s| s.success()).unwrap_or(false);
        #[cfg(unix)]
        let exit_signal: Option<i32> = {
            use std::os::unix::process::ExitStatusExt;
            status.as_ref().ok().and_then(|s| s.signal())
        };
        #[cfg(not(unix))]
        let exit_signal: Option<i32> = None;

        // If we explicitly stopped this process via signal_stop, its state
        // will be "stopping" at this point. That's not a failure — the user
        // asked for it. Mark as "done" so it shows ✓ in Recent.
        let final_state: &'static str = {
            let map = state3.procs.lock().unwrap_or_else(|p| p.into_inner());
            let was_user_stop = map
                .get(&id3)
                .map(|i| i.state == "stopping")
                .unwrap_or(false);
            if was_user_stop || ok {
                "done"
            } else {
                "failed"
            }
        };

        // Synthesize a recent_log tail line so View error has something
        // meaningful when the process died silently (signal kill with
        // no stderr output). Without this the UI falls back to the last
        // real log line, which is usually a normal request and reads as
        // "obviously not the error."
        let synth = match (exit_signal, exit_code, final_state) {
            (Some(sig), _, _) => Some(format!(
                "[exit: killed by signal {sig} ({})]",
                signal_name(sig),
            )),
            (None, Some(code), "failed") => {
                Some(format!("[exit: code {code}]"))
            }
            (None, None, "failed") => {
                Some("[exit: process gone, no exit code or signal]".to_string())
            }
            _ => None,
        };

        {
            let mut map = state3.procs.lock().unwrap_or_else(|p| p.into_inner());
            if let Some(info) = map.get_mut(&id3) {
                info.state = final_state.into();
                info.exit_code = exit_code;
                info.exit_signal = exit_signal;
                info.ended_at_ms = Some(now_ms());
                if let Some(line) = &synth {
                    info.recent_log.push(line.clone());
                    let len = info.recent_log.len();
                    if len > LOG_TAIL_CAP {
                        info.recent_log.drain(0..(len - LOG_TAIL_CAP));
                    }
                }
            }
            let mut pids = state3.pids.lock().unwrap_or_else(|p| p.into_inner());
            pids.remove(&id3);
        }
        // Update on-disk running set now that this pid is gone, so a
        // crash *between* this child exiting and the next spawn doesn't
        // leave us trying to kill a recycled pid on next startup.
        write_pid_file(&app3, &state3);
        let _ = app3.emit(
            "proc:state",
            ProcEvent {
                proc_id: id3,
                state: final_state.into(),
                exit_code,
                exit_signal,
            },
        );
    });

    Ok(())
}

async fn signal_stop(
    app: AppHandle,
    state: Arc<ProcessManager>,
    id: String,
) -> Result<(), String> {
    let info = {
        let map = state.procs.lock().map_err(|e| e.to_string())?;
        map.get(&id).cloned()
    };
    if let Some(info) = &info {
        if info.command.starts_with("docker compose")
            || info.label.starts_with("docker compose")
        {
            return docker_compose_down_for(app, state, id).await;
        }
    }

    let pid = {
        let pids = state.pids.lock().map_err(|e| e.to_string())?;
        pids.get(&id).copied()
    };
    let Some(pid) = pid else {
        return Ok(());
    };

    {
        let mut map = state.procs.lock().map_err(|e| e.to_string())?;
        if let Some(info) = map.get_mut(&id) {
            info.state = "stopping".into();
            info.was_user_stopped = true;
        }
    }
    let _ = app.emit(
        "proc:state",
        ProcEvent {
            proc_id: id.clone(),
            state: "stopping".into(),
            exit_code: None,
            exit_signal: None,
        },
    );

    #[cfg(unix)]
    unsafe {
        libc::kill(-(pid as i32), libc::SIGTERM);
    }

    tokio::time::sleep(std::time::Duration::from_millis(800)).await;
    let still_alive = state
        .pids
        .lock()
        .map(|m| m.contains_key(&id))
        .unwrap_or(false);
    if still_alive {
        #[cfg(unix)]
        unsafe {
            libc::kill(-(pid as i32), libc::SIGKILL);
        }
    }
    Ok(())
}

#[tauri::command]
pub async fn stop_process(
    app: AppHandle,
    state: State<'_, Arc<ProcessManager>>,
    id: String,
) -> Result<(), String> {
    signal_stop(app, state.inner().clone(), id).await
}

/// Drop a terminated process from tracking entirely. Used by the perf
/// tab's "Dismiss" on a failed run: without this, the entry lingers in
/// the in-memory `procs` map and reappears in `list_processes` (so a
/// dismissed card pops back when the tab remounts). Refuses to forget a
/// process that's still alive — stop it first. Also clears any
/// remembered args and buffered log lines for the id.
#[tauri::command]
pub async fn forget_process(
    state: State<'_, Arc<ProcessManager>>,
    id: String,
) -> Result<(), String> {
    {
        let pids = state.pids.lock().map_err(|e| e.to_string())?;
        if pids.contains_key(&id) {
            return Err(format!("process {id} is still running"));
        }
    }
    state.procs.lock().map_err(|e| e.to_string())?.remove(&id);
    state.last_args.lock().map_err(|e| e.to_string())?.remove(&id);
    state.log_store.lock().map_err(|e| e.to_string())?.remove(&id);
    Ok(())
}

/// Stop every running managed process and tear down docker compose,
/// then exit the app. Runs in the backend (not the frontend) so the
/// app doesn't disappear before SIGTERM has a chance to escalate to
/// SIGKILL. Called by the frontend after the user confirms quit.
#[tauri::command(rename_all = "snake_case")]
pub async fn shutdown_now(
    app: AppHandle,
    state: State<'_, Arc<ProcessManager>>,
    repo_path: Option<String>,
) -> Result<(), String> {
    let arc = state.inner().clone();

    // 1. Collect ids of currently-running managed procs.
    let running_ids: Vec<String> = {
        let map = arc.procs.lock().map_err(|e| e.to_string())?;
        map.iter()
            .filter(|(_, info)| info.state == "running" || info.state == "stopping")
            .map(|(id, _)| id.clone())
            .collect()
    };

    // 2. SIGTERM each (signal_stop already escalates to SIGKILL after 800ms
    //    if the child is stubborn). docker-compose proc is routed to
    //    docker_compose_down_for internally — but see step 3 for the case
    //    where the spawn already exited but containers are still up.
    for id in &running_ids {
        let _ = signal_stop(app.clone(), arc.clone(), id.clone()).await;
    }

    // 3. Tear down docker compose unconditionally. `docker compose up -d`
    //    exits after starting containers, so our managed proc is usually
    //    "done" by the time the user quits — meaning step 2 wouldn't
    //    touch docker. Running `down` here is idempotent: a no-op if
    //    nothing's there, a real shutdown if containers are alive.
    if let Some(repo) = repo_path {
        let _ = docker_cmd()
            .args(["compose", "down"])
            .current_dir(&repo)
            .output()
            .await;
    }

    // 4. Final safety net for any spawn that's still alive — signal_stop's
    //    own escalation timer has already fired, but if something exotic
    //    is holding on, SIGKILL it now.
    for id in &running_ids {
        let alive = {
            let pids = arc.pids.lock().map_err(|e| e.to_string())?;
            pids.get(id).copied()
        };
        if let Some(pid) = alive {
            #[cfg(unix)]
            unsafe {
                libc::kill(-(pid as i32), libc::SIGKILL);
            }
        }
    }
    tokio::time::sleep(std::time::Duration::from_millis(150)).await;

    // Flag this exit as intentional so the RunEvent::ExitRequested
    // handler doesn't intercept and bounce us back to hide-to-tray.
    crate::mark_intentional_quit();
    app.exit(0);
    Ok(())
}

#[tauri::command]
pub async fn restart_process(
    app: AppHandle,
    state: State<'_, Arc<ProcessManager>>,
    id: String,
) -> Result<(), String> {
    let state_arc = state.inner().clone();
    let _ = signal_stop(app.clone(), state_arc.clone(), id.clone()).await;
    // brief wait so the wait task removes the pid before we try to respawn
    tokio::time::sleep(std::time::Duration::from_millis(900)).await;
    let args_opt = {
        let map = state_arc.last_args.lock().map_err(|e| e.to_string())?;
        map.get(&id).cloned()
    };
    let Some(a) = args_opt else {
        return Err(format!("no remembered args for {id}"));
    };
    spawn_managed(
        app,
        state_arc,
        id,
        a.label,
        a.cwd,
        a.program,
        a.args,
        a.log_channel,
        a.env,
    )
    .await
}

#[tauri::command]
pub async fn docker_compose_status(cwd: String) -> Result<DockerStatus, String> {
    let out = docker_cmd()
        .args(["compose", "ps", "--format", "json"])
        .current_dir(&cwd)
        .output()
        .await
        .map_err(|e| e.to_string())?;
    if !out.status.success() {
        return Ok(DockerStatus {
            running: false,
            containers: vec![],
        });
    }
    let stdout = String::from_utf8_lossy(&out.stdout);
    let mut containers = Vec::new();
    for line in stdout.lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        if let Ok(v) = serde_json::from_str::<serde_json::Value>(line) {
            let name = v
                .get("Service")
                .and_then(|x| x.as_str())
                .or_else(|| v.get("Name").and_then(|x| x.as_str()))
                .unwrap_or("")
                .to_string();
            let state = v
                .get("State")
                .and_then(|x| x.as_str())
                .unwrap_or("")
                .to_string();
            if !name.is_empty() {
                containers.push(ContainerState { name, state });
            }
        }
    }
    let running = containers.iter().any(|c| c.state == "running");
    Ok(DockerStatus {
        running,
        containers,
    })
}

#[tauri::command]
pub async fn docker_compose_down_cmd(
    app: AppHandle,
    state: State<'_, Arc<ProcessManager>>,
    cwd: String,
) -> Result<String, String> {
    // Flip to "stopping" + set user-stopped up-front so the chain row hides
    // the play button and shows "stopping…" for the full duration of the
    // down command (1-2s, sometimes longer). Without this the row would
    // either flash red ("not running") as the health probe sees docker go
    // down, or jump straight to idle + ▶ play while compose is still
    // tearing containers down.
    {
        let mut map = state.procs.lock().map_err(|e| e.to_string())?;
        if let Some(info) = map.get_mut("docker-compose-up") {
            info.was_user_stopped = true;
            info.state = "stopping".into();
        }
    }
    let _ = app.emit(
        "proc:state",
        ProcEvent {
            proc_id: "docker-compose-up".into(),
            state: "stopping".into(),
            exit_code: None,
            exit_signal: None,
        },
    );

    let out = docker_cmd()
        .args(["compose", "down"])
        .current_dir(&cwd)
        .output()
        .await
        .map_err(|e| e.to_string())?;

    // Flip back to "done" so the row falls through to the user-stopped
    // idle path (○) and the ▶ play button becomes available.
    {
        let mut map = state.procs.lock().map_err(|e| e.to_string())?;
        if let Some(info) = map.get_mut("docker-compose-up") {
            info.state = "done".into();
            info.ended_at_ms = Some(now_ms());
        }
    }
    let _ = app.emit(
        "proc:state",
        ProcEvent {
            proc_id: "docker-compose-up".into(),
            state: "done".into(),
            exit_code: None,
            exit_signal: None,
        },
    );

    if !out.status.success() {
        return Err(String::from_utf8_lossy(&out.stderr).to_string());
    }
    Ok(String::from_utf8_lossy(&out.stdout).to_string())
}

#[tauri::command]
pub async fn docker_compose_restart_cmd(cwd: String) -> Result<String, String> {
    let out = docker_cmd()
        .args(["compose", "restart"])
        .current_dir(&cwd)
        .output()
        .await
        .map_err(|e| e.to_string())?;
    if !out.status.success() {
        return Err(String::from_utf8_lossy(&out.stderr).to_string());
    }
    Ok(String::from_utf8_lossy(&out.stdout).to_string())
}

async fn docker_compose_down_for(
    app: AppHandle,
    state: Arc<ProcessManager>,
    id: String,
) -> Result<(), String> {
    let cwd = {
        let map = state.procs.lock().map_err(|e| e.to_string())?;
        map.get(&id).map(|i| i.cwd.clone())
    };
    let Some(cwd) = cwd else {
        return Ok(());
    };

    let out = docker_cmd()
        .args(["compose", "down"])
        .current_dir(&cwd)
        .output()
        .await
        .map_err(|e| e.to_string())?;

    {
        let mut map = state.procs.lock().map_err(|e| e.to_string())?;
        if let Some(info) = map.get_mut(&id) {
            let stdout = String::from_utf8_lossy(&out.stdout);
            let stderr = String::from_utf8_lossy(&out.stderr);
            for line in stdout.lines().chain(stderr.lines()) {
                info.recent_log.push(line.to_string());
            }
            let len = info.recent_log.len();
            if len > LOG_TAIL_CAP {
                info.recent_log.drain(0..(len - LOG_TAIL_CAP));
            }
            info.state = if out.status.success() { "done" } else { "failed" }.into();
            info.exit_code = out.status.code();
            info.ended_at_ms = Some(now_ms());
            // docker_compose_down_for is only called via signal_stop or
            // dockerComposeDown — both are explicit user-stop intents.
            info.was_user_stopped = true;
        }
        let mut pids = state.pids.lock().map_err(|e| e.to_string())?;
        pids.remove(&id);
    }
    let _ = app.emit(
        "proc:state",
        ProcEvent {
            proc_id: id,
            state: if out.status.success() { "done" } else { "failed" }.into(),
            exit_code: out.status.code(),
            exit_signal: None,
        },
    );

    Ok(())
}

/// Rustls verifier that accepts any server certificate. Fleet's dev
/// server uses a self-signed cert and the probe just needs to reach
/// the TLS handshake stage — we don't care who's on the other end, only
/// that something is listening and able to speak TLS.
#[derive(Debug)]
struct AcceptAnyCert;

impl rustls::client::danger::ServerCertVerifier for AcceptAnyCert {
    fn verify_server_cert(
        &self,
        _end_entity: &rustls::pki_types::CertificateDer<'_>,
        _intermediates: &[rustls::pki_types::CertificateDer<'_>],
        _server_name: &rustls::pki_types::ServerName<'_>,
        _ocsp_response: &[u8],
        _now: rustls::pki_types::UnixTime,
    ) -> Result<rustls::client::danger::ServerCertVerified, rustls::Error> {
        Ok(rustls::client::danger::ServerCertVerified::assertion())
    }

    fn verify_tls12_signature(
        &self,
        _message: &[u8],
        _cert: &rustls::pki_types::CertificateDer<'_>,
        _dss: &rustls::DigitallySignedStruct,
    ) -> Result<rustls::client::danger::HandshakeSignatureValid, rustls::Error> {
        Ok(rustls::client::danger::HandshakeSignatureValid::assertion())
    }

    fn verify_tls13_signature(
        &self,
        _message: &[u8],
        _cert: &rustls::pki_types::CertificateDer<'_>,
        _dss: &rustls::DigitallySignedStruct,
    ) -> Result<rustls::client::danger::HandshakeSignatureValid, rustls::Error> {
        Ok(rustls::client::danger::HandshakeSignatureValid::assertion())
    }

    fn supported_verify_schemes(&self) -> Vec<rustls::SignatureScheme> {
        use rustls::SignatureScheme as S;
        vec![
            S::RSA_PKCS1_SHA256,
            S::RSA_PKCS1_SHA384,
            S::RSA_PKCS1_SHA512,
            S::ECDSA_NISTP256_SHA256,
            S::ECDSA_NISTP384_SHA384,
            S::ECDSA_NISTP521_SHA512,
            S::RSA_PSS_SHA256,
            S::RSA_PSS_SHA384,
            S::RSA_PSS_SHA512,
            S::ED25519,
        ]
    }
}

fn tls_connector() -> &'static TlsConnector {
    // Build once, reuse forever. ClientConfig is cheap to clone (Arc
    // internally) but rebuilding it per probe — including allocating
    // the verifier and crypto provider — is wasted work.
    static CONNECTOR: OnceLock<TlsConnector> = OnceLock::new();
    CONNECTOR.get_or_init(|| {
        let config = rustls::ClientConfig::builder_with_provider(
            std::sync::Arc::new(rustls::crypto::ring::default_provider()),
        )
        .with_safe_default_protocol_versions()
        .expect("safe defaults")
        .dangerous()
        .with_custom_certificate_verifier(std::sync::Arc::new(AcceptAnyCert))
        .with_no_client_auth();
        TlsConnector::from(std::sync::Arc::new(config))
    })
}

#[tauri::command]
pub async fn serve_tcp_check(host: Option<String>, port: u16) -> bool {
    let h = host.unwrap_or_else(|| "127.0.0.1".into());
    // 1.5s budget for the full handshake (TCP connect + TLS). 500ms was
    // fine for a plain TCP probe but a real handshake on a loaded dev
    // machine can occasionally tip past that.
    let deadline = std::time::Duration::from_millis(1500);
    let probe = async {
        let stream = TcpStream::connect((h.as_str(), port)).await.ok()?;
        let server_name =
            rustls::pki_types::ServerName::try_from(h.clone()).ok()?;
        // Hand the stream over to rustls and run the handshake. We don't
        // care about the resulting TlsStream — dropping it sends a
        // close_notify and frees everything.
        tls_connector()
            .connect(server_name, stream)
            .await
            .ok()
            .map(|_| ())
    };
    matches!(tokio::time::timeout(deadline, probe).await, Ok(Some(())))
}

#[derive(Debug, Serialize)]
pub struct LogWindow {
    pub entries: Vec<LogEntry>,
    pub total_in_window: usize,
    pub warn_count: usize,
    pub error_count: usize,
}

// Same camelCase pitfall as start_process — frontend passes since_ms /
// max_lines, so we opt into snake_case explicitly.
#[tauri::command(rename_all = "snake_case")]
pub async fn read_log_window(
    state: State<'_, Arc<ProcessManager>>,
    source: String,           // "fleet-serve" | "docker-compose" | "all"
    since_ms: u64,            // 0 = no lower bound
    levels: Vec<String>,      // levels to include
    search: Option<String>,   // substring; if wrapped in /.../, treat as regex
    max_lines: Option<usize>, // cap on returned entries (newest first)
) -> Result<LogWindow, String> {
    let store = state.log_store.lock().map_err(|e| e.to_string())?;
    let level_set: std::collections::HashSet<&str> =
        levels.iter().map(String::as_str).collect();

    let (pattern, is_regex): (Option<regex_lite::Regex>, bool) = match &search {
        Some(s) if s.starts_with('/') && s.ends_with('/') && s.len() >= 3 => {
            let inner = &s[1..s.len() - 1];
            (regex_lite::Regex::new(inner).ok(), true)
        }
        _ => (None, false),
    };
    let search_lower: Option<String> = search
        .as_ref()
        .filter(|_| !is_regex)
        .map(|s| s.to_ascii_lowercase());

    let channels: Vec<&String> = match source.as_str() {
        "all" => store.keys().collect(),
        s => store.keys().filter(|k| k.as_str() == s).collect(),
    };

    let mut total = 0usize;
    let mut warn_count = 0usize;
    let mut error_count = 0usize;
    let mut entries: Vec<LogEntry> = Vec::new();

    for ch in channels {
        let Some(buf) = store.get(ch) else { continue };
        for e in buf.iter() {
            if e.ts_ms < since_ms {
                continue;
            }
            // Level filtering — entries without a detected level are
            // treated as "info". An empty set means the user has toggled
            // every chip off, so we show nothing (don't fall back to
            // "show all" — that confused the chip semantics).
            let lvl = e.level.as_deref().unwrap_or("info");
            if !level_set.contains(lvl) {
                continue;
            }
            // Search filtering
            if let Some(ref re) = pattern {
                if !re.is_match(&e.message) {
                    continue;
                }
            } else if let Some(ref needle) = search_lower {
                if !e.message.to_ascii_lowercase().contains(needle) {
                    continue;
                }
            }
            total += 1;
            if lvl == "warn" {
                warn_count += 1;
            }
            if lvl == "error" {
                error_count += 1;
            }
            entries.push(e.clone());
        }
    }

    // Sort by ts ascending so the body reads top-to-bottom in time order.
    entries.sort_by_key(|e| e.ts_ms);
    if let Some(cap) = max_lines {
        if entries.len() > cap {
            let drop_n = entries.len() - cap;
            entries.drain(0..drop_n);
        }
    }

    Ok(LogWindow {
        entries,
        total_in_window: total,
        warn_count,
        error_count,
    })
}

/// Writes a pre-formatted text snapshot of the current log view to
/// `<app_log_dir>/snapshots/<filename>`. Frontend handles formatting
/// (so the on-disk view matches exactly what the user sees with their
/// current filter/search/window) and supplies a basename — we reject
/// anything containing path separators so a hostile webview can't
/// escape the snapshots dir.
#[tauri::command(rename_all = "snake_case")]
pub async fn save_log_snapshot(
    app: AppHandle,
    filename: String,
    contents: String,
) -> Result<String, String> {
    if filename.contains('/')
        || filename.contains('\\')
        || filename.contains("..")
        || filename.is_empty()
    {
        return Err("invalid filename".into());
    }
    let dir = logs_dir(&app)?.join("snapshots");
    std::fs::create_dir_all(&dir).map_err(|e| e.to_string())?;
    let path = dir.join(&filename);
    std::fs::write(&path, contents).map_err(|e| e.to_string())?;
    Ok(path.to_string_lossy().to_string())
}

/// Absolute path to the directory where channel logs and snapshots
/// live (`app_log_dir`). Surfaced so the Logs tab can reveal it in the
/// system file manager.
#[tauri::command]
pub fn logs_dir_path(app: AppHandle) -> Result<String, String> {
    Ok(logs_dir(&app)?.to_string_lossy().to_string())
}

#[tauri::command]
pub async fn clear_log_channel(
    app: AppHandle,
    state: State<'_, Arc<ProcessManager>>,
    channel: String,
) -> Result<(), String> {
    // Clear in-memory ring.
    {
        let mut store = state.log_store.lock().map_err(|e| e.to_string())?;
        if channel == "all" {
            store.clear();
        } else {
            store.remove(&channel);
        }
    }
    // Truncate the on-disk file(s) too — otherwise the UI says "cleared"
    // while disk keeps growing, and a subsequent scroll-back still finds
    // the old content. Drop any cached writer first so the truncate
    // isn't shadowed by buffered bytes.
    let channels: Vec<String> = if channel == "all" {
        let writers = state.log_writers.lock().map_err(|e| e.to_string())?;
        writers.keys().cloned().collect()
    } else {
        vec![channel.clone()]
    };
    {
        let mut writers = state.log_writers.lock().map_err(|e| e.to_string())?;
        for ch in &channels {
            writers.remove(ch);
        }
    }
    for ch in channels {
        if let Ok(path) = log_file_path(&app, &ch) {
            if path.exists() {
                let _ = std::fs::write(&path, b"");
            }
            let rotated = path.with_extension("log.1");
            if rotated.exists() {
                let _ = std::fs::remove_file(&rotated);
            }
        }
    }
    Ok(())
}
