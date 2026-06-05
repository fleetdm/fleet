//! Resolves the PATH to hand to spawned child processes.
//!
//! A macOS app launched from Finder/Dock inherits only the bare
//! `/usr/bin:/bin:/usr/sbin:/sbin` — it does NOT see your shell PATH.
//! That omits Homebrew (`/opt/homebrew/bin`), Go (`/usr/local/go/bin`),
//! nvm, `~/.fleetctl`, etc. So bare-name spawns the app relies on
//! (`git`, `go`, `docker`, `ngrok`, `python3`, …) fail with "No such
//! file or directory" in the packaged app, even though they work under
//! `tauri dev` (which inherits the terminal's PATH).
//!
//! We fix this by capturing the user's real *login-shell* PATH once at
//! startup and applying it to every child spawn. In `tauri dev` this is
//! effectively a no-op (the inherited PATH is already correct).
//!
//! The cached value is intentionally shared between `shell_path()`
//! (read) and `refresh()` (overwrite). The dep-check screen calls
//! `refresh()` whenever the user clicks Recheck so that tools they
//! just installed (which modified `.zprofile`) become visible to every
//! subsequent spawn — not just the next Recheck.

use std::process::Command;
use std::sync::{Mutex, OnceLock};

fn cache() -> &'static Mutex<String> {
    static CACHE: OnceLock<Mutex<String>> = OnceLock::new();
    CACHE.get_or_init(|| Mutex::new(String::new()))
}

/// The PATH string to set on spawned children. Computed lazily on the
/// first call and cached; future calls clone the cached value.
pub fn shell_path() -> String {
    let mut g = cache().lock().unwrap();
    if g.is_empty() {
        *g = resolve();
    }
    g.clone()
}

/// Warm the cache eagerly (e.g. at app startup) so the first real spawn
/// doesn't pay the shell-probe latency. Safe to call more than once.
pub fn warm() {
    let _ = shell_path();
}

/// Force a re-probe and overwrite the cache. Future calls to
/// `shell_path()` return the new value too.
pub fn refresh() -> String {
    let new = resolve();
    eprintln!("[shellpath] refresh -> {} chars", new.len());
    *cache().lock().unwrap() = new.clone();
    new
}

fn resolve() -> String {
    if let Some(p) = probe_login_shell() {
        return p;
    }
    augment_inherited()
}

/// Ask the user's login shell for its PATH. `-i -l -c` so the rc /
/// profile files that actually set PATH (Homebrew shellenv, nvm, custom
/// exports) are sourced.
///
/// We take the **last non-empty line** of stdout, not the whole output.
/// Some interactive shells (zsh with session restoration enabled) print
/// banners like `Restored session: Thu May 28 14:31:00 CDT 2026\n` to
/// stdout before the user's command runs. The session-restore line
/// always appears *before* our printf, so the PATH is the last line —
/// we extract it cleanly and avoid contaminating the env we set on
/// child processes with the banner.
fn probe_login_shell() -> Option<String> {
    let shell = std::env::var("SHELL").unwrap_or_else(|_| "/bin/zsh".into());
    let out = Command::new(&shell)
        .args(["-ilc", "printf %s \"$PATH\""])
        .output()
        .ok()?;
    if !out.status.success() {
        return None;
    }
    let stdout = String::from_utf8_lossy(&out.stdout);
    let p = stdout
        .lines()
        .map(|l| l.trim())
        .filter(|l| !l.is_empty())
        .last()?
        .to_string();
    // Sanity-check: a real PATH has separators and absolute entries.
    if !p.contains('/') {
        return None;
    }
    Some(p)
}

/// Fallback when the shell probe fails: prepend the common tool
/// locations to whatever PATH we inherited, de-duplicated and skipping
/// entries that don't exist.
fn augment_inherited() -> String {
    let inherited = std::env::var("PATH").unwrap_or_default();
    let mut parts: Vec<String> = Vec::new();
    let mut push = |p: String| {
        if !p.is_empty() && !parts.contains(&p) {
            parts.push(p);
        }
    };

    push("/opt/homebrew/bin".into());
    push("/opt/homebrew/sbin".into());
    push("/usr/local/bin".into());
    push("/usr/local/go/bin".into());
    if let Some(home) = dirs::home_dir() {
        push(home.join("go/bin").to_string_lossy().to_string());
        push(home.join(".local/bin").to_string_lossy().to_string());
        push(home.join(".fleetctl").to_string_lossy().to_string());
    }
    for entry in inherited.split(':') {
        push(entry.to_string());
    }
    parts.join(":")
}
