use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::path::{Path, PathBuf};
use std::process::Stdio;
use tokio::io::AsyncWriteExt;
use tokio::process::Command;

fn shellexpand(s: &str) -> String {
    if let Some(rest) = s.strip_prefix("~/") {
        if let Some(home) = dirs::home_dir() {
            return home.join(rest).to_string_lossy().to_string();
        }
    }
    if s == "~" {
        if let Some(home) = dirs::home_dir() {
            return home.to_string_lossy().to_string();
        }
    }
    s.to_string()
}

#[derive(Debug, Serialize)]
pub struct ResolvedBinary {
    pub path: String,
    pub source: &'static str, // "settings" | "build" | "missing"
    pub exists: bool,
}

#[tauri::command(rename_all = "snake_case")]
pub fn fleetctl_resolve_binary(
    repo: Option<String>,
    settings_path: Option<String>,
) -> Result<ResolvedBinary, String> {
    // Prefer the explicit settings path if the user picked one — they
    // may be testing a release binary outside the repo. Fall back to
    // the conventional <repo>/build/fleetctl so the common dev path
    // just works.
    if let Some(p) = settings_path.as_deref().filter(|s| !s.is_empty()) {
        let expanded = shellexpand(p);
        let exists = Path::new(&expanded).exists();
        return Ok(ResolvedBinary {
            path: expanded,
            source: "settings",
            exists,
        });
    }
    if let Some(r) = repo.as_deref().filter(|s| !s.is_empty()) {
        let candidate = Path::new(r).join("build").join("fleetctl");
        let exists = candidate.exists();
        return Ok(ResolvedBinary {
            path: candidate.to_string_lossy().to_string(),
            source: "build",
            exists,
        });
    }
    Ok(ResolvedBinary {
        path: String::new(),
        source: "missing",
        exists: false,
    })
}

// ----- ~/.fleet/config parsing -----

#[derive(Debug, Deserialize)]
struct FleetConfigFile {
    // serde_yaml::Mapping preserves insertion order (it's backed by
    // indexmap internally). We use it instead of HashMap so the list
    // we return mirrors the order of contexts in the YAML file —
    // which is what the user sees in the editor and expects to match
    // the parsed list panel.
    #[serde(default)]
    contexts: serde_yaml::Mapping,
}

#[derive(Debug, Deserialize, Default)]
struct RawContext {
    #[serde(default)]
    address: Option<String>,
    #[serde(default)]
    email: Option<String>,
    #[serde(default)]
    token: Option<String>,
}

#[derive(Debug, Serialize)]
pub struct ContextSummary {
    pub name: String,
    pub address: Option<String>,
    pub email: Option<String>,
    pub has_token: bool,
}

#[derive(Debug, Serialize)]
pub struct ContextInfo {
    pub config_path: String,
    pub exists: bool,
    pub current: Option<ContextSummary>,
    pub contexts: Vec<ContextSummary>,
}

fn default_config_path() -> PathBuf {
    if let Some(home) = dirs::home_dir() {
        home.join(".fleet").join("config")
    } else {
        PathBuf::from("~/.fleet/config")
    }
}

#[derive(Debug, Serialize)]
pub struct RawConfig {
    pub path: String,
    pub exists: bool,
    pub contents: String,
}

#[tauri::command]
pub fn fleetctl_read_config_raw() -> Result<RawConfig, String> {
    let path = default_config_path();
    let path_str = path.to_string_lossy().to_string();
    if !path.exists() {
        return Ok(RawConfig {
            path: path_str,
            exists: false,
            contents: String::new(),
        });
    }
    let raw = std::fs::read_to_string(&path).map_err(|e| e.to_string())?;
    Ok(RawConfig {
        path: path_str,
        exists: true,
        contents: raw,
    })
}

#[tauri::command]
pub fn fleetctl_save_config(yaml: String) -> Result<(), String> {
    // Validate parse first so users see "line 12: unexpected mapping"
    // rather than fleetctl crashing later. We just need the file to be
    // valid YAML — don't enforce structure beyond that; the user may
    // be doing something unusual with custom-headers or similar.
    serde_yaml::from_str::<serde_yaml::Value>(&yaml)
        .map_err(|e| format!("YAML parse error: {e}"))?;
    let path = default_config_path();
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent)
            .map_err(|e| format!("creating {parent:?}: {e}"))?;
    }
    std::fs::write(&path, yaml).map_err(|e| format!("writing {path:?}: {e}"))
}

#[tauri::command]
pub fn fleetctl_read_context() -> Result<ContextInfo, String> {
    // We always read the *default* config path because fleetctl itself
    // does the same unless --config is passed. If the user runs the
    // binary with a custom CONFIG env elsewhere, this won't reflect
    // it — but Fleet Hangar always launches fleetctl with no overrides,
    // so the two stay aligned.
    let path = default_config_path();
    let path_str = path.to_string_lossy().to_string();
    if !path.exists() {
        return Ok(ContextInfo {
            config_path: path_str,
            exists: false,
            current: None,
            contexts: Vec::new(),
        });
    }
    let raw = std::fs::read_to_string(&path).map_err(|e| e.to_string())?;
    let parsed: FleetConfigFile = serde_yaml::from_str(&raw)
        .map_err(|e| format!("parse {path_str}: {e}"))?;
    // Walk the Mapping in iteration order — which is file order. We
    // skip entries whose key isn't a string (malformed config) and
    // skip values that can't be deserialized into a RawContext rather
    // than failing the whole read.
    let contexts: Vec<ContextSummary> = parsed
        .contexts
        .into_iter()
        .filter_map(|(k, v)| {
            let name = k.as_str()?.to_string();
            let raw: RawContext = serde_yaml::from_value(v).unwrap_or_default();
            Some(ContextSummary {
                name,
                address: raw.address,
                email: raw.email,
                has_token: raw
                    .token
                    .as_deref()
                    .map(|t| !t.is_empty())
                    .unwrap_or(false),
            })
        })
        .collect();
    // fleetctl's default context name is "default" unless the user
    // selected something else with `fleetctl config switch-context` —
    // which writes to a different file we don't currently track. For
    // Fleet Hangar dev, "default" is right.
    let current = contexts.iter().find(|c| c.name == "default").cloned();
    Ok(ContextInfo {
        config_path: path_str,
        exists: true,
        current,
        contexts,
    })
}

impl Clone for ContextSummary {
    fn clone(&self) -> Self {
        Self {
            name: self.name.clone(),
            address: self.address.clone(),
            email: self.email.clone(),
            has_token: self.has_token,
        }
    }
}

// ----- one-shot capture runner -----

#[derive(Debug, Serialize)]
pub struct CapturedRun {
    pub exit_code: Option<i32>,
    pub stdout: String,
    pub stderr: String,
}

/// Runs a fleetctl invocation synchronously and returns the captured
/// output. Use this for short, finite commands where the caller wants
/// the result inline (login, get, status checks). For long-running or
/// streamy commands, use the existing start_process pipeline instead so
/// output shows up in the Logs tab.
#[tauri::command(rename_all = "snake_case")]
pub async fn fleetctl_run_capture(
    program: String,
    cwd: Option<String>,
    args: Vec<String>,
    env: Option<HashMap<String, String>>,
    stdin_data: Option<String>,
    timeout_ms: Option<u64>,
) -> Result<CapturedRun, String> {
    let mut cmd = Command::new(&program);
    cmd.args(&args)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .stdin(if stdin_data.is_some() {
            Stdio::piped()
        } else {
            Stdio::null()
        });
    // Login-shell PATH so fleetctl (and anything it shells out to)
    // resolves when launched from Finder. Set before caller env so an
    // explicit PATH from the caller would still take precedence.
    cmd.env("PATH", crate::shellpath::shell_path());
    if let Some(dir) = cwd.as_deref().filter(|s| !s.is_empty()) {
        cmd.current_dir(dir);
    }
    if let Some(envs) = env {
        for (k, v) in envs {
            cmd.env(k, v);
        }
    }

    let mut child = cmd
        .spawn()
        .map_err(|e| format!("spawn {program}: {e}"))?;

    if let Some(data) = stdin_data {
        if let Some(mut stdin) = child.stdin.take() {
            let _ = stdin.write_all(data.as_bytes()).await;
            let _ = stdin.shutdown().await;
        }
    }

    let timeout = std::time::Duration::from_millis(timeout_ms.unwrap_or(60_000));
    let waited = tokio::time::timeout(timeout, child.wait_with_output()).await;
    let out = match waited {
        Ok(Ok(o)) => o,
        Ok(Err(e)) => return Err(format!("wait {program}: {e}")),
        Err(_) => {
            // Best-effort cleanup; the kill_on_drop wasn't set here.
            return Err(format!("timed out after {}ms", timeout.as_millis()));
        }
    };
    Ok(CapturedRun {
        exit_code: out.status.code(),
        stdout: String::from_utf8_lossy(&out.stdout).to_string(),
        stderr: String::from_utf8_lossy(&out.stderr).to_string(),
    })
}
