//! Saved osquery-perf configurations. v1 stores everything (including
//! enroll_secret and SCEP challenge) as plain text in
//! <app-config>/perf-configs.json — these are dev-only credentials for
//! local fleet-perf simulation, same security boundary as the user's
//! ~/.fleetctl/config and the rest of the fleet-hangar settings file.

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::path::PathBuf;
use std::time::{SystemTime, UNIX_EPOCH};
use tauri::{AppHandle, Manager};

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PerfConfig {
    pub id: String,
    pub name: String,
    pub server_url: String,
    pub enroll_secret: String,
    /// Per-template host counts. BTreeMap so the JSON output is
    /// key-sorted and diff-friendly (HashMap would shuffle on every
    /// rewrite).
    pub os_counts: BTreeMap<String, u32>,
    pub mdm_enabled: bool,
    pub mdm_prob: f64,
    pub mdm_scep_challenge: String,
    pub start_period: String,
    pub query_interval: String,
    pub config_interval: String,
    /// Server-stamped on first save; preserved across updates.
    #[serde(default)]
    pub created_at_ms: u64,
    /// Bumped on every save.
    #[serde(default)]
    pub updated_at_ms: u64,
}

#[derive(Debug, Default, Serialize, Deserialize)]
struct ConfigsFile {
    #[serde(default)]
    configs: Vec<PerfConfig>,
}

fn configs_path(app: &AppHandle) -> Result<PathBuf> {
    let dir = app
        .path()
        .app_config_dir()
        .context("resolving app config dir")?;
    std::fs::create_dir_all(&dir).context("creating app config dir")?;
    Ok(dir.join("perf-configs.json"))
}

fn now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

fn read_all(app: &AppHandle) -> Result<Vec<PerfConfig>> {
    let p = configs_path(app)?;
    if !p.exists() {
        return Ok(Vec::new());
    }
    let raw = std::fs::read_to_string(&p).context("reading perf-configs.json")?;
    let file: ConfigsFile = serde_json::from_str(&raw).context("parsing perf-configs.json")?;
    Ok(file.configs)
}

fn write_all(app: &AppHandle, configs: &[PerfConfig]) -> Result<()> {
    let p = configs_path(app)?;
    let file = ConfigsFile {
        configs: configs.to_vec(),
    };
    let raw = serde_json::to_string_pretty(&file)?;
    std::fs::write(&p, raw).context("writing perf-configs.json")
}

#[tauri::command]
pub fn perf_configs_list(app: AppHandle) -> Result<Vec<PerfConfig>, String> {
    read_all(&app).map_err(|e| e.to_string())
}

/// Upsert: if `config.id` already exists, overwrite that entry in place
/// (preserving its original created_at_ms); otherwise append. Returns
/// the saved record with server-stamped timestamps so the frontend can
/// reflect the freshly-written state without a separate list-refresh.
#[tauri::command]
pub fn perf_config_save(
    app: AppHandle,
    mut config: PerfConfig,
) -> Result<PerfConfig, String> {
    let now = now_ms();
    let mut all = read_all(&app).map_err(|e| e.to_string())?;
    if let Some(existing) = all.iter_mut().find(|c| c.id == config.id) {
        config.created_at_ms = existing.created_at_ms;
        config.updated_at_ms = now;
        *existing = config.clone();
    } else {
        if config.created_at_ms == 0 {
            config.created_at_ms = now;
        }
        config.updated_at_ms = now;
        all.push(config.clone());
    }
    write_all(&app, &all).map_err(|e| e.to_string())?;
    Ok(config)
}

#[tauri::command]
pub fn perf_config_delete(app: AppHandle, id: String) -> Result<(), String> {
    let all = read_all(&app).map_err(|e| e.to_string())?;
    let filtered: Vec<_> = all.into_iter().filter(|c| c.id != id).collect();
    write_all(&app, &filtered).map_err(|e| e.to_string())
}
