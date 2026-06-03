use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};
use std::path::{Path, PathBuf};
use tauri::{AppHandle, Manager};

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct Settings {
    pub repo_path: Option<String>,
    pub fleetctl_path: Option<String>,
    /// Root directory for gitops repos. None = unconfigured; tab
    /// shows an empty-state until the user picks a folder. Can point
    /// to a folder containing many gitops repos OR be a single repo
    /// (i.e. contain default.yml directly) — both are detected at
    /// scan time.
    pub gitops_dir: Option<String>,
    #[serde(default)]
    pub first_run_complete: bool,
    #[serde(default)]
    pub ngrok: NgrokConfig,
    #[serde(default)]
    pub python_server: PythonConfig,
    #[serde(default)]
    pub fleet_serve: FleetServeConfig,
    /// "system" follows the OS appearance via prefers-color-scheme;
    /// "light" / "dark" pin to one mode regardless of OS. Stored as a
    /// string so future modes (e.g. high-contrast) don't break old
    /// configs. Defaults to "system" — see ThemePreference::default.
    #[serde(default)]
    pub theme: ThemePreference,
    /// fleetctl cron names the user has starred. Drives the Favorites
    /// section at the top of the Trigger sub-tab. Plain Vec<String> so
    /// the order is preserved; the UI dedupes via a Set on read.
    #[serde(default)]
    pub favorite_crons: Vec<String>,
}

#[derive(Debug, Clone, Copy, Serialize, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
pub enum ThemePreference {
    System,
    Light,
    Dark,
}

impl Default for ThemePreference {
    fn default() -> Self {
        Self::System
    }
}

#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct NgrokConfig {
    /// Hide ngrok from Active processes when false. Default off — opt-in.
    #[serde(default)]
    pub enabled: bool,
    /// Path to ngrok.yml. Empty/None = use ngrok's discovery default
    /// (~/Library/Application Support/ngrok/ngrok.yml on macOS).
    pub yml_path: Option<String>,
    #[serde(default)]
    pub default_tunnels: Vec<String>,
    #[serde(default)]
    pub start_all: bool,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PythonConfig {
    /// Hide python http.server from Active processes when false. Default off.
    #[serde(default)]
    pub enabled: bool,
    pub port: u16,
    /// Relative to repo_path if not absolute. None = repo root.
    pub directory: Option<String>,
}

impl Default for PythonConfig {
    fn default() -> Self {
        Self {
            enabled: false,
            port: 8000,
            directory: None,
        }
    }
}

/// User-tunable bits of `fleet serve --dev`. Everything optional with
/// defaults that match what we shipped before (config: fleet.yml,
/// premium, both debug flags on).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FleetServeConfig {
    /// Path passed to `--config`. None / empty = omit the flag entirely,
    /// which lets fleet pick up env-driven config or its own defaults.
    pub config_path: Option<String>,
    /// Toggle `--dev_license`. Off = "free" build; on = premium.
    #[serde(default = "true_default")]
    pub premium: bool,
    /// Toggle `--debug`.
    #[serde(default = "true_default")]
    pub debug: bool,
    /// Toggle `--logging_debug`.
    #[serde(default = "true_default")]
    pub logging_debug: bool,
    /// Env vars to set on the spawn. Vec (not HashMap) so the user's row
    /// order is preserved across save/load — the Settings UI renders
    /// them in this order, and a stable order also gives the command
    /// preview a stable env-key list.
    #[serde(default)]
    pub env: Vec<EnvVar>,
}

fn true_default() -> bool {
    true
}

impl Default for FleetServeConfig {
    fn default() -> Self {
        Self {
            config_path: Some("fleet.yml".into()),
            premium: true,
            debug: true,
            logging_debug: true,
            env: Vec::new(),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EnvVar {
    pub key: String,
    pub value: String,
    /// Per-row enable toggle so the user can keep a row in the list
    /// without applying it. Disabled rows are skipped by `serveEnvFor`
    /// before the IPC call. Defaults true so rows saved before this
    /// field existed stay applied.
    #[serde(default = "true_default")]
    pub enabled: bool,
}

impl Default for EnvVar {
    fn default() -> Self {
        Self {
            key: String::new(),
            value: String::new(),
            enabled: true,
        }
    }
}

fn settings_path(app: &AppHandle) -> Result<PathBuf> {
    let dir = app
        .path()
        .app_config_dir()
        .context("resolving app config dir")?;
    std::fs::create_dir_all(&dir).context("creating app config dir")?;
    Ok(dir.join("settings.json"))
}

pub fn load(app: &AppHandle) -> Result<Settings> {
    let p = settings_path(app)?;
    if !p.exists() {
        return Ok(Settings::default());
    }
    let raw = std::fs::read_to_string(&p).context("reading settings.json")?;
    let s: Settings = serde_json::from_str(&raw).context("parsing settings.json")?;
    Ok(s)
}

pub fn save(app: &AppHandle, s: &Settings) -> Result<()> {
    let p = settings_path(app)?;
    let raw = serde_json::to_string_pretty(s)?;
    std::fs::write(&p, raw).context("writing settings.json")?;
    Ok(())
}

#[derive(Debug, Serialize)]
pub struct RepoProbe {
    pub path: String,
    pub valid: bool,
    pub reason: Option<String>,
}

fn probe_path(path: &str) -> RepoProbe {
    probe_resolved(&PathBuf::from(shellexpand(path)))
}

fn probe_resolved(p: &Path) -> RepoProbe {
    if !p.exists() {
        return RepoProbe {
            path: p.to_string_lossy().to_string(),
            valid: false,
            reason: Some("path does not exist".into()),
        };
    }
    let go_mod = p.join("go.mod");
    if !go_mod.exists() {
        return RepoProbe {
            path: p.to_string_lossy().to_string(),
            valid: false,
            reason: Some("no go.mod found".into()),
        };
    }
    let contents = match std::fs::read_to_string(&go_mod) {
        Ok(c) => c,
        Err(e) => {
            return RepoProbe {
                path: p.to_string_lossy().to_string(),
                valid: false,
                reason: Some(format!("could not read go.mod: {e}")),
            };
        }
    };
    if !contents.contains("github.com/fleetdm/fleet") {
        return RepoProbe {
            path: p.to_string_lossy().to_string(),
            valid: false,
            reason: Some("go.mod module is not github.com/fleetdm/fleet".into()),
        };
    }
    RepoProbe {
        path: p.to_string_lossy().to_string(),
        valid: true,
        reason: None,
    }
}

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

/// Common dev-root parent dirs we descend into. Each is scanned at
/// depth 2 (catches `<parent>/<org>/fleet` layouts like GitHub Desktop's
/// `~/Documents/GitHub/fleetdm/fleet`); a depth-1 child that already
/// looks like a repo isn't recursed into.
const DEV_ROOTS: &[&str] = &[
    "~/repositories",
    "~/repos",
    "~/code",
    "~/Code",
    "~/src",
    "~/Developer",
    "~/Documents/GitHub",
    "~/Projects",
    "~/projects",
    "~/work",
    "~/dev",
    "~/github",
    "~/git",
];

/// Walk the well-known dev-root parents and return every directory
/// that looks like a Fleet clone (go.mod module = github.com/fleetdm/fleet).
/// Also checks `~/fleet` directly. Deduplicates by canonical path so the
/// same clone reached via a symlink shows up once.
pub fn discover_fleet_repos() -> Vec<RepoProbe> {
    let mut results: Vec<RepoProbe> = Vec::new();
    let mut seen: HashSet<PathBuf> = HashSet::new();

    let maybe_add = |path: &Path, results: &mut Vec<RepoProbe>, seen: &mut HashSet<PathBuf>| {
        let probe = probe_resolved(path);
        if !probe.valid {
            return;
        }
        let key = std::fs::canonicalize(path).unwrap_or_else(|_| path.to_path_buf());
        if seen.insert(key) {
            results.push(probe);
        }
    };

    // ~/fleet as a one-off (we don't scan all of ~ — too noisy).
    if let Some(home) = dirs::home_dir() {
        maybe_add(&home.join("fleet"), &mut results, &mut seen);
    }

    for root in DEV_ROOTS {
        let parent = PathBuf::from(shellexpand(root));
        let entries = match std::fs::read_dir(&parent) {
            Ok(e) => e,
            Err(_) => continue,
        };
        for entry in entries.flatten() {
            let child = entry.path();
            if !child.is_dir() {
                continue;
            }
            if entry
                .file_name()
                .to_string_lossy()
                .starts_with('.')
            {
                continue;
            }
            // Depth-1 hit?
            let probe = probe_resolved(&child);
            if probe.valid {
                let key = std::fs::canonicalize(&child).unwrap_or_else(|_| child.clone());
                if seen.insert(key) {
                    results.push(probe);
                }
                continue; // don't descend into a known repo
            }
            // If the depth-1 child isn't itself a repo (no go.mod),
            // treat it as a potential org folder and scan one level deeper.
            if !child.join("go.mod").exists() {
                if let Ok(grand) = std::fs::read_dir(&child) {
                    for g in grand.flatten() {
                        let gp = g.path();
                        if !gp.is_dir() {
                            continue;
                        }
                        if g.file_name().to_string_lossy().starts_with('.') {
                            continue;
                        }
                        maybe_add(&gp, &mut results, &mut seen);
                    }
                }
            }
        }
    }

    results.sort_by(|a, b| a.path.cmp(&b.path));
    results
}

#[tauri::command]
pub fn get_settings(app: AppHandle) -> Result<Settings, String> {
    load(&app).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn save_settings(app: AppHandle, settings: Settings) -> Result<(), String> {
    save(&app, &settings).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn probe_fleet_repo(path: Option<String>) -> Result<Vec<RepoProbe>, String> {
    match path {
        Some(p) => Ok(vec![probe_path(&p)]),
        None => Ok(discover_fleet_repos()),
    }
}

#[derive(Debug, Serialize)]
pub struct NgrokTunnel {
    pub name: String,
    pub proto: String,
    pub addr: String,
}

#[derive(Debug, Serialize)]
pub struct NgrokYamlInfo {
    pub valid: bool,
    pub error: Option<String>,
    pub resolved_path: String,
    pub has_authtoken: bool,
    pub tunnels: Vec<NgrokTunnel>,
}

#[derive(Deserialize)]
struct NgrokYamlRaw {
    // ngrok v2 nested the authtoken at the top level. ngrok v3 (which
    // bumps `version: "3"`) moved it under an `agent:` block. We accept
    // either so the badge doesn't lie about a token being missing when
    // the file's just using the newer schema.
    #[serde(default)]
    authtoken: Option<String>,
    #[serde(default)]
    agent: Option<NgrokAgentBlock>,
    #[serde(default)]
    tunnels: HashMap<String, serde_yaml::Value>,
}

#[derive(Deserialize, Default)]
struct NgrokAgentBlock {
    #[serde(default)]
    authtoken: Option<String>,
}

pub fn default_ngrok_yml_path() -> String {
    if let Some(home) = dirs::home_dir() {
        home.join("Library/Application Support/ngrok/ngrok.yml")
            .to_string_lossy()
            .to_string()
    } else {
        "~/Library/Application Support/ngrok/ngrok.yml".into()
    }
}

/// Reject paths that would escape a sane location. We're not trying to
/// be a full sandbox — `start_process` and `fleetctl_run_capture` are
/// general-purpose by design — but the generic file/open commands are
/// the easiest "compromised webview overwrites ~/.zshrc" primitive, so
/// require the resolved path to live under $HOME and contain no `..`
/// segments.
fn ensure_under_home(p: &Path) -> Result<(), String> {
    let home = dirs::home_dir().ok_or_else(|| "no home dir".to_string())?;
    if !p.starts_with(&home) {
        return Err(format!("path must be under {}", home.display()));
    }
    if p.components().any(|c| matches!(c, std::path::Component::ParentDir)) {
        return Err("path contains `..` segments".into());
    }
    Ok(())
}

const YAML_EXTS: &[&str] = &["yml", "yaml"];
const OPEN_EXTS: &[&str] = &["yml", "yaml", "log", "json", "txt", "md", "sql", "gz"];

fn has_ext(p: &Path, allowed: &[&str]) -> bool {
    p.extension()
        .and_then(|e| e.to_str())
        .map(|e| allowed.iter().any(|a| a.eq_ignore_ascii_case(e)))
        .unwrap_or(false)
}

#[tauri::command]
pub fn read_text_file(path: String) -> Result<String, String> {
    let resolved = PathBuf::from(shellexpand(&path));
    ensure_under_home(&resolved)?;
    if !has_ext(&resolved, YAML_EXTS) {
        return Err("only .yml/.yaml files supported".into());
    }
    std::fs::read_to_string(&resolved).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn write_text_file(path: String, contents: String) -> Result<(), String> {
    let resolved = PathBuf::from(shellexpand(&path));
    ensure_under_home(&resolved)?;
    if !has_ext(&resolved, YAML_EXTS) {
        return Err("only .yml/.yaml files supported".into());
    }
    std::fs::write(&resolved, contents).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn open_path(path: String, reveal: Option<bool>) -> Result<(), String> {
    let resolved = PathBuf::from(shellexpand(&path));
    ensure_under_home(&resolved)?;
    // Allow opening directories (backups dir, log dir) or files with a
    // narrow set of plain-text/config extensions. Anything executable
    // (.app, .command, .terminal, .workflow, …) is rejected.
    let is_dir = resolved.is_dir();
    if !is_dir && !has_ext(&resolved, OPEN_EXTS) {
        return Err("unsupported file type for open".into());
    }
    let mut cmd = std::process::Command::new("open");
    if reveal.unwrap_or(false) {
        cmd.arg("-R");
    }
    cmd.arg(&resolved)
        .status()
        .map_err(|e| e.to_string())?;
    Ok(())
}

/// Open an http(s) URL in the user's default browser. Kept narrow:
/// only http/https schemes are accepted, so the frontend can't smuggle
/// a `file://` or custom-handler URL through this command to bypass
/// `open_path`'s sandboxing.
#[tauri::command]
pub fn open_url(url: String) -> Result<(), String> {
    if !(url.starts_with("https://") || url.starts_with("http://")) {
        return Err("only http(s) URLs are allowed".into());
    }
    std::process::Command::new("open")
        .arg(&url)
        .status()
        .map_err(|e| e.to_string())?;
    Ok(())
}

#[tauri::command]
pub fn parse_ngrok_yml(path: Option<String>) -> Result<NgrokYamlInfo, String> {
    let resolved = match path.as_deref().filter(|s| !s.is_empty()) {
        Some(p) => shellexpand(p),
        None => default_ngrok_yml_path(),
    };
    let p = PathBuf::from(&resolved);
    if !p.exists() {
        return Ok(NgrokYamlInfo {
            valid: false,
            error: Some("file not found".into()),
            resolved_path: resolved,
            has_authtoken: false,
            tunnels: vec![],
        });
    }
    let raw = match std::fs::read_to_string(&p) {
        Ok(s) => s,
        Err(e) => {
            return Ok(NgrokYamlInfo {
                valid: false,
                error: Some(format!("read error: {e}")),
                resolved_path: resolved,
                has_authtoken: false,
                tunnels: vec![],
            });
        }
    };
    let parsed: NgrokYamlRaw = match serde_yaml::from_str(&raw) {
        Ok(v) => v,
        Err(e) => {
            return Ok(NgrokYamlInfo {
                valid: false,
                error: Some(format!("parse error: {e}")),
                resolved_path: resolved,
                has_authtoken: false,
                tunnels: vec![],
            });
        }
    };
    // Look for the token in both legal positions: top-level (v2) and
    // under agent.authtoken (v3). Either counts as "has a token". Test
    // each independently so an empty top-level field doesn't mask a
    // valid v3 value (Option::or would pick the first Some, including
    // Some("")).
    let has_v2 = parsed
        .authtoken
        .as_deref()
        .map(|s| !s.trim().is_empty())
        .unwrap_or(false);
    let has_v3 = parsed
        .agent
        .as_ref()
        .and_then(|a| a.authtoken.as_deref())
        .map(|s| !s.trim().is_empty())
        .unwrap_or(false);
    let has_authtoken = has_v2 || has_v3;
    let mut tunnels: Vec<NgrokTunnel> = parsed
        .tunnels
        .into_iter()
        .map(|(name, val)| {
            let proto = val
                .get("proto")
                .and_then(|v| v.as_str())
                .unwrap_or("")
                .to_string();
            let addr = val
                .get("addr")
                .map(|v| match v {
                    serde_yaml::Value::Number(n) => n.to_string(),
                    serde_yaml::Value::String(s) => s.clone(),
                    other => serde_yaml::to_string(other).unwrap_or_default().trim().to_string(),
                })
                .unwrap_or_default();
            NgrokTunnel { name, proto, addr }
        })
        .collect();
    tunnels.sort_by(|a, b| a.name.cmp(&b.name));
    Ok(NgrokYamlInfo {
        valid: true,
        error: None,
        resolved_path: resolved,
        has_authtoken,
        tunnels,
    })
}
