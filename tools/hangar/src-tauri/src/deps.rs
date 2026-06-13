//! First-run dependency checks. Detects required tooling, compares
//! versions against what the discovered Fleet repo declares, and
//! returns a structured report the UI renders as a checklist.

use regex_lite::Regex;
use semver::{Version, VersionReq};
use serde::Serialize;
use std::path::{Path, PathBuf};
use std::process::Command;

use crate::shellpath;

#[derive(Debug, Serialize)]
pub struct DepCheck {
    pub id: String,
    pub name: String,
    pub installed: bool,
    pub version: Option<String>,
    pub required: Option<String>,
    /// None if there's no version requirement to compare against.
    pub version_ok: Option<bool>,
    /// Daemon / runtime state for tools that need more than just a binary
    /// on disk (Docker). None when not applicable. Some(false) means the
    /// binary exists but the runtime isn't ready — surfaced in the UI as
    /// "stopped", not "not found".
    pub runtime_ok: Option<bool>,
    pub install_command: String,
    pub doc_url: Option<String>,
    pub note: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct DepReport {
    pub checks: Vec<DepCheck>,
}

fn run(path: &str, cmd: &str, args: &[&str]) -> Option<(bool, String)> {
    let out = Command::new(cmd)
        .args(args)
        .env("PATH", path)
        .output()
        .ok()?;
    let stdout = String::from_utf8_lossy(&out.stdout).to_string();
    let stderr = String::from_utf8_lossy(&out.stderr).to_string();
    let combined = if stdout.trim().is_empty() { stderr } else { stdout };
    Some((out.status.success(), combined.trim().to_string()))
}

/// Run a one-off command through the user's login shell. Needed for
/// `nvm`, which isn't a binary — it's a shell function that only
/// exists after `~/.nvm/nvm.sh` is sourced.
fn run_login_shell(script: &str) -> Option<(bool, String)> {
    let shell = std::env::var("SHELL").unwrap_or_else(|_| "/bin/zsh".into());
    let out = Command::new(&shell).args(["-lc", script]).output().ok()?;
    let stdout = String::from_utf8_lossy(&out.stdout).to_string();
    Some((out.status.success(), stdout.trim().to_string()))
}

/// Pull the first SemVer-looking token out of arbitrary CLI output
/// ("Homebrew 4.2.1", "git version 2.39.5", "v24.10.0", "go version
/// go1.26.3 darwin/arm64"). Pads "x.y" to "x.y.0" so the semver crate
/// can always parse the result.
fn extract_version(s: &str) -> Option<String> {
    let re = Regex::new(r"\d+\.\d+(?:\.\d+)?").ok()?;
    let v = re.find(s)?.as_str().to_string();
    if v.matches('.').count() == 1 {
        Some(format!("{v}.0"))
    } else {
        Some(v)
    }
}

fn satisfies(detected: &str, requirement: &str) -> Option<bool> {
    let req = VersionReq::parse(requirement).ok()?;
    let ver = Version::parse(detected).ok()?;
    Some(req.matches(&ver))
}

/// Read `engines.{key}` from package.json.
fn read_engines(path: PathBuf, key: &str) -> Option<String> {
    let s = std::fs::read_to_string(&path).ok()?;
    let v: serde_json::Value = serde_json::from_str(&s).ok()?;
    v.get("engines")?.get(key)?.as_str().map(|x| x.to_string())
}

/// Resolve the Node version requirement. Prefer the discovered repo's
/// engines.node so the check stays aligned with whatever Fleet pins;
/// fall back to a known-good value before a repo is picked. (Go and
/// Yarn intentionally don't get version-gated — see check_go/check_yarn.)
fn required_node_version(repo_path: Option<&Path>) -> Option<String> {
    repo_path
        .and_then(|repo| read_engines(repo.join("package.json"), "node"))
        .or_else(|| Some("^24.10.0".into()))
}

fn check_xcode(path: &str) -> DepCheck {
    let installed = run(path, "xcode-select", &["-p"])
        .map(|(ok, _)| ok)
        .unwrap_or(false);
    DepCheck {
        id: "xcode-clt".into(),
        name: "Xcode Command Line Tools".into(),
        installed,
        version: None,
        required: None,
        version_ok: None,
        runtime_ok: None,
        install_command: "xcode-select --install".into(),
        doc_url: Some("https://developer.apple.com/download/all/?q=command%20line%20tools".into()),
        note: Some(
            "Provides git, make, and the compiler toolchain. Triggers a macOS install dialog — check behind other windows if you don't see it."
                .into(),
        ),
    }
}

fn check_brew(path: &str) -> DepCheck {
    let (installed, version) = match run(path, "brew", &["--version"]) {
        Some((true, out)) => (true, extract_version(&out)),
        _ => (false, None),
    };
    DepCheck {
        id: "brew".into(),
        name: "Homebrew".into(),
        installed,
        version,
        required: None,
        version_ok: None,
        runtime_ok: None,
        install_command: "/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
            .into(),
        doc_url: Some("https://brew.sh".into()),
        note: Some("Package manager. Installs go, node, yarn, and Docker Desktop.".into()),
    }
}

fn check_git(path: &str) -> DepCheck {
    let (installed, version) = match run(path, "git", &["--version"]) {
        Some((true, out)) => (true, extract_version(&out)),
        _ => (false, None),
    };
    DepCheck {
        id: "git".into(),
        name: "git".into(),
        installed,
        version,
        required: None,
        version_ok: None,
        runtime_ok: None,
        install_command: "brew install git".into(),
        doc_url: None,
        note: Some("Clones the Fleet repo and manages branches.".into()),
    }
}

fn check_go(path: &str) -> DepCheck {
    let (installed, version) = match run(path, "go", &["version"]) {
        Some((true, out)) => (true, extract_version(&out)),
        _ => (false, None),
    };
    // No version_ok: Go's toolchain manages its own version from go.mod's
    // `go` directive (since Go 1.21 it'll auto-download the right one),
    // so flagging a "wrong" Go version would create noise.
    DepCheck {
        id: "go".into(),
        name: "Go".into(),
        installed,
        version,
        required: None,
        version_ok: None,
        runtime_ok: None,
        install_command: "brew install go".into(),
        doc_url: None,
        note: Some("Builds the Fleet server.".into()),
    }
}

/// Which version manager (if any) we detected. Drives both the
/// "Node version manager" dep row and the Node install command so the
/// suggested fix matches what the user actually has on their system.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum NodeManager {
    Nvm,
    N,
    None,
}

struct VersionManagerCheck {
    dep: DepCheck,
    detected: NodeManager,
}

fn check_node_version_manager(path: &str) -> VersionManagerCheck {
    // nvm is a shell function (sourced from ~/.nvm/nvm.sh), not a
    // binary — so a PATH probe wouldn't find it. n is a regular binary,
    // so a plain `run` works. Prefer nvm when both are present because
    // it's what Fleet's docs reference.
    let nvm_installed = dirs::home_dir()
        .map(|h| h.join(".nvm/nvm.sh").exists())
        .unwrap_or(false);
    let nvm_version = if nvm_installed {
        run_login_shell("nvm --version")
            .filter(|(ok, _)| *ok)
            .and_then(|(_, s)| extract_version(&s))
    } else {
        None
    };
    let (n_installed, n_version) = match run(path, "n", &["--version"]) {
        Some((true, out)) => (true, extract_version(&out)),
        _ => (false, None),
    };

    let (detected, version) = match (nvm_installed, n_installed) {
        (true, _) => (NodeManager::Nvm, nvm_version.map(|v| format!("nvm {v}"))),
        (false, true) => (NodeManager::N, n_version.map(|v| format!("n {v}"))),
        (false, false) => (NodeManager::None, None),
    };
    let installed = detected != NodeManager::None;

    let dep = DepCheck {
        id: "node-version-manager".into(),
        name: "nvm or n".into(),
        installed,
        version,
        required: None,
        version_ok: None,
        runtime_ok: None,
        install_command:
            "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash"
                .into(),
        doc_url: Some("https://github.com/nvm-sh/nvm#install--update-script".into()),
        note: Some(
            "Lets you install/switch Node versions. Either nvm (default) or `n` (brew install n) works."
                .into(),
        ),
    };
    VersionManagerCheck { dep, detected }
}

fn check_node(path: &str, required: Option<&str>, manager: NodeManager) -> DepCheck {
    let (installed, version) = match run(path, "node", &["-v"]) {
        Some((true, out)) => (true, extract_version(&out)),
        _ => (false, None),
    };
    let version_ok = match (&version, required) {
        (Some(v), Some(r)) => satisfies(v, r),
        _ => None,
    };
    // Build a copy-paste command that pins to the major Fleet requires.
    // We extract the major from the requirement; if anything weird, fall
    // back to "24" since that's what Fleet wants today.
    let major = required
        .and_then(|r| extract_version(r))
        .and_then(|v| v.split('.').next().map(|s| s.to_string()))
        .unwrap_or_else(|| "24".into());
    let (install_command, note) = match manager {
        NodeManager::N => (
            format!("n {major}"),
            "Fleet pins a specific Node major. Use `n` to install/switch.",
        ),
        // Same instructions for Nvm (preferred) and None (we suggest the
        // documented default rather than `n`).
        _ => (
            format!("nvm install {major} && nvm use {major}"),
            "Fleet pins a specific Node major. Use nvm or `n` to install/switch.",
        ),
    };
    DepCheck {
        id: "node".into(),
        name: "Node.js".into(),
        installed,
        version,
        required: required.map(|s| s.to_string()),
        version_ok,
        runtime_ok: None,
        install_command,
        doc_url: None,
        note: Some(note.into()),
    }
}

fn check_yarn(path: &str) -> DepCheck {
    let (installed, version) = match run(path, "yarn", &["-v"]) {
        Some((true, out)) => (true, extract_version(&out)),
        _ => (false, None),
    };
    // No version_ok: yarn's engines floor in package.json is a soft min;
    // any modern yarn works. We only care whether it's installed.
    DepCheck {
        id: "yarn".into(),
        name: "Yarn".into(),
        installed,
        version,
        required: None,
        version_ok: None,
        runtime_ok: None,
        install_command: "brew install yarn".into(),
        doc_url: None,
        note: Some("Bundles Fleet's frontend.".into()),
    }
}

fn check_docker(path: &str) -> DepCheck {
    let (cli_ok, version) = match run(path, "docker", &["--version"]) {
        Some((true, out)) => (true, extract_version(&out)),
        _ => (false, None),
    };
    // `docker version --format` hits the daemon and fails fast when it
    // isn't running (avoids `docker info`'s long stall on a dead socket).
    let daemon_ok = if cli_ok {
        run(path, "docker", &["version", "--format", "{{.Server.Version}}"])
            .map(|(ok, _)| ok)
            .unwrap_or(false)
    } else {
        false
    };
    let note = if !cli_ok {
        Some(
            "Required for `docker compose up` (MySQL/Redis dev infra).".into(),
        )
    } else if !daemon_ok {
        Some("Installed, but the daemon isn't running. Open Docker Desktop.".into())
    } else {
        Some("Runs Fleet's MySQL/Redis dev infra.".into())
    };
    DepCheck {
        id: "docker".into(),
        name: "Docker".into(),
        // `installed` reflects only the CLI binary. Daemon state lives in
        // `runtime_ok` so the UI can distinguish "missing" from "stopped".
        installed: cli_ok,
        version,
        required: None,
        version_ok: None,
        runtime_ok: if cli_ok { Some(daemon_ok) } else { None },
        install_command: "brew install --cask docker".into(),
        doc_url: Some("https://www.docker.com/products/docker-desktop/".into()),
        note,
    }
}

fn check_rosetta() -> Option<DepCheck> {
    if !cfg!(target_arch = "aarch64") {
        return None;
    }
    // oahd is the Rosetta translation daemon; pgrep -q is silent and
    // returns exit 0 iff a matching process exists.
    let installed = Command::new("/usr/bin/pgrep")
        .args(["-q", "oahd"])
        .output()
        .map(|o| o.status.success())
        .unwrap_or(false);
    Some(DepCheck {
        id: "rosetta".into(),
        name: "Rosetta 2".into(),
        installed,
        version: None,
        required: None,
        version_ok: None,
        runtime_ok: None,
        install_command: "softwareupdate --install-rosetta --agree-to-license".into(),
        doc_url: None,
        note: Some("Apple Silicon only. Some `make generate` tools are x86_64.".into()),
    })
}

#[tauri::command]
pub fn check_dependencies(
    repo_path: Option<String>,
    refresh_path: Option<bool>,
) -> Result<DepReport, String> {
    let repo = repo_path.as_deref().map(Path::new);
    let req_node = required_node_version(repo);
    // Re-probe the login-shell PATH only when the user explicitly asked
    // for it (Recheck button). The auto-refresh that fires when the
    // welcome screen mounts or the picked repo changes reuses the
    // cached PATH — repoPath changes affect version requirements, not
    // the toolchain location.
    let path = if refresh_path.unwrap_or(false) {
        shellpath::refresh()
    } else {
        shellpath::shell_path()
    };

    let vm = check_node_version_manager(&path);
    let mut checks = vec![
        check_xcode(&path),
        check_brew(&path),
        check_git(&path),
        check_go(&path),
        vm.dep,
        check_node(&path, req_node.as_deref(), vm.detected),
        check_yarn(&path),
        check_docker(&path),
    ];
    if let Some(r) = check_rosetta() {
        checks.push(r);
    }
    Ok(DepReport { checks })
}
