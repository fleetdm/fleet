use serde::Serialize;
use std::process::Command;

#[derive(Debug, Serialize, Clone)]
pub struct DetectedProcess {
    pub pid: u32,
    pub command: String,
}

#[derive(Debug, Serialize)]
pub struct KillOutcome {
    pub pid: u32,
    pub gone: bool,
    pub used_kill: bool,
    pub error: Option<String>,
}

fn run_capture(program: &str, args: &[&str]) -> Result<String, String> {
    let out = Command::new(program)
        .args(args)
        .output()
        .map_err(|e| format!("{program}: {e}"))?;
    if !out.status.success() {
        // lsof returns 1 with empty stdout when nothing matches —
        // that's not really an error for our use case, treat empty
        // as empty rather than bubbling up.
        if out.stdout.is_empty() && out.stderr.is_empty() {
            return Ok(String::new());
        }
        // pgrep exits 1 when no match. Same idea.
        if out.stdout.is_empty() {
            return Ok(String::new());
        }
    }
    Ok(String::from_utf8_lossy(&out.stdout).to_string())
}

fn pid_command(pid: u32) -> String {
    // ps -p <pid> -o command= prints just the command line (no header).
    // Use a sane fallback if the process has disappeared between when we
    // listed it and when we ask for its command.
    Command::new("ps")
        .args(["-p", &pid.to_string(), "-o", "command="])
        .output()
        .ok()
        .and_then(|o| {
            let s = String::from_utf8_lossy(&o.stdout).trim().to_string();
            if s.is_empty() { None } else { Some(s) }
        })
        .unwrap_or_else(|| "(process exited)".into())
}

#[tauri::command(rename_all = "snake_case")]
pub fn troubleshoot_scan_port(port: u16) -> Result<Vec<DetectedProcess>, String> {
    // -iTCP:<port> filters to the port (must be one combined arg — lsof
    // treats `:<port>` as a filename if it's a separate argv);
    // -t outputs just PIDs (one per line); -P avoids slow service-name
    // resolution; -n avoids slow DNS lookups.
    let i_filter = format!("-iTCP:{port}");
    let raw = run_capture(
        "lsof",
        &["-nP", &i_filter, "-sTCP:LISTEN", "-t"],
    )?;
    let mut out: Vec<DetectedProcess> = Vec::new();
    let mut seen: std::collections::HashSet<u32> = std::collections::HashSet::new();
    for line in raw.lines() {
        let pid: u32 = match line.trim().parse() {
            Ok(p) => p,
            Err(_) => continue,
        };
        if !seen.insert(pid) {
            continue;
        }
        out.push(DetectedProcess {
            pid,
            command: pid_command(pid),
        });
    }
    Ok(out)
}

#[tauri::command(rename_all = "snake_case")]
pub fn troubleshoot_scan_pattern(
    pattern: String,
) -> Result<Vec<DetectedProcess>, String> {
    // pgrep -f matches the full command line. -l would include the
    // process name; we fetch command via ps separately for consistency
    // with the port-based scan.
    let raw = run_capture("pgrep", &["-f", &pattern])?;
    let mut out: Vec<DetectedProcess> = Vec::new();
    let self_pid = std::process::id();
    for line in raw.lines() {
        let pid: u32 = match line.trim().parse() {
            Ok(p) => p,
            Err(_) => continue,
        };
        // Don't list ourselves — if the pattern is "fleet" we might
        // match Fleet Hangar's own command line.
        if pid == self_pid {
            continue;
        }
        out.push(DetectedProcess {
            pid,
            command: pid_command(pid),
        });
    }
    Ok(out)
}

#[cfg(unix)]
fn signal_pid(pid: u32, sig: libc::c_int) -> Result<(), String> {
    let r = unsafe { libc::kill(pid as libc::pid_t, sig) };
    if r != 0 {
        // ESRCH (3) means the process is already gone — not an error
        // for our caller; the post-signal poll handles the "did it
        // actually die" question.
        let err = std::io::Error::last_os_error();
        if err.raw_os_error() != Some(libc::ESRCH) {
            return Err(format!("kill({pid}, {sig}): {err}"));
        }
    }
    Ok(())
}

#[cfg(unix)]
fn pid_alive(pid: u32) -> bool {
    // Signal 0 doesn't deliver but checks permission/existence: 0 = exists,
    // ESRCH = gone, EPERM = exists but we can't signal it.
    let r = unsafe { libc::kill(pid as libc::pid_t, 0) };
    if r == 0 {
        return true;
    }
    let err = std::io::Error::last_os_error();
    err.raw_os_error() != Some(libc::ESRCH)
}

#[tauri::command(rename_all = "snake_case")]
pub async fn troubleshoot_kill_pid(pid: u32) -> Result<KillOutcome, String> {
    #[cfg(not(unix))]
    {
        let _ = pid;
        return Err("kill is only implemented on unix platforms".into());
    }
    #[cfg(unix)]
    {
        if let Err(e) = signal_pid(pid, libc::SIGTERM) {
            return Ok(KillOutcome {
                pid,
                gone: !pid_alive(pid),
                used_kill: false,
                error: Some(e),
            });
        }
        // Poll for graceful exit. 2s total, 100ms granularity — enough
        // for well-behaved daemons (fleet serve, ngrok, python) without
        // making the user wait forever if SIGTERM is being ignored.
        for _ in 0..20 {
            tokio::time::sleep(std::time::Duration::from_millis(100)).await;
            if !pid_alive(pid) {
                return Ok(KillOutcome {
                    pid,
                    gone: true,
                    used_kill: false,
                    error: None,
                });
            }
        }
        // Escalate.
        if let Err(e) = signal_pid(pid, libc::SIGKILL) {
            return Ok(KillOutcome {
                pid,
                gone: !pid_alive(pid),
                used_kill: true,
                error: Some(e),
            });
        }
        // Brief settle window after SIGKILL — the kernel cleans up
        // basically instantly, but we poll once more to be honest in
        // the response.
        for _ in 0..5 {
            tokio::time::sleep(std::time::Duration::from_millis(50)).await;
            if !pid_alive(pid) {
                return Ok(KillOutcome {
                    pid,
                    gone: true,
                    used_kill: true,
                    error: None,
                });
            }
        }
        Ok(KillOutcome {
            pid,
            gone: false,
            used_kill: true,
            error: Some("process still alive after SIGKILL".into()),
        })
    }
}
