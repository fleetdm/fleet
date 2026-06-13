use serde::{Deserialize, Serialize};
use std::path::{Path, PathBuf};
use std::time::{SystemTime, UNIX_EPOCH};

const BACKUPS_DIRNAME: &str = "db-backups";
const BACKUP_EXT: &str = ".sql.gz";

#[derive(Debug, Serialize)]
pub struct BackupEntry {
    pub name: String,
    pub path: String,
    pub size: u64,
    pub mtime_ms: u64,
    pub branch: Option<String>,
    pub note: Option<String>,
    pub created_at_ms: Option<u64>,
}

#[derive(Debug, Serialize, Deserialize, Default)]
struct BackupMeta {
    #[serde(default)]
    created_at_ms: Option<u64>,
    #[serde(default)]
    branch: Option<String>,
    #[serde(default)]
    note: Option<String>,
}

fn backups_dir(repo: &str) -> PathBuf {
    Path::new(repo).join(BACKUPS_DIRNAME)
}

fn meta_path_for(backup_path: &Path) -> PathBuf {
    // Sidecar lives next to the .sql.gz with a .json suffix appended to
    // the full filename so list/delete don't have to re-parse anything.
    let mut s = backup_path.as_os_str().to_owned();
    s.push(".json");
    PathBuf::from(s)
}

fn ensure_dir_with_gitignore(dir: &Path) -> Result<(), String> {
    std::fs::create_dir_all(dir).map_err(|e| format!("creating {dir:?}: {e}"))?;
    // Drop a self-contained .gitignore inside the backups folder so the
    // dumps stay out of git without us patching the repo's main
    // .gitignore. `!.gitignore` keeps this file itself tracked-friendly
    // if anyone ever does want to commit it.
    let gi = dir.join(".gitignore");
    if !gi.exists() {
        let body = "# Auto-created by Fleet Hangar.\n# Ignore all backup artifacts here.\n*\n!.gitignore\n";
        std::fs::write(&gi, body).map_err(|e| format!("writing {gi:?}: {e}"))?;
    }
    Ok(())
}

fn read_meta(path: &Path) -> Option<BackupMeta> {
    let raw = std::fs::read_to_string(path).ok()?;
    serde_json::from_str(&raw).ok()
}

fn mtime_ms(path: &Path) -> u64 {
    std::fs::metadata(path)
        .and_then(|m| m.modified())
        .ok()
        .and_then(|t| t.duration_since(UNIX_EPOCH).ok())
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

fn now_ms() -> u64 {
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

#[tauri::command]
pub fn db_backups_dir(repo: String) -> Result<String, String> {
    Ok(backups_dir(&repo).to_string_lossy().to_string())
}

#[tauri::command]
pub fn db_ensure_backups_dir(repo: String) -> Result<String, String> {
    let dir = backups_dir(&repo);
    ensure_dir_with_gitignore(&dir)?;
    Ok(dir.to_string_lossy().to_string())
}

#[tauri::command]
pub fn db_list_backups(repo: String) -> Result<Vec<BackupEntry>, String> {
    let dir = backups_dir(&repo);
    if !dir.exists() {
        return Ok(Vec::new());
    }
    let entries = std::fs::read_dir(&dir).map_err(|e| format!("reading {dir:?}: {e}"))?;
    let mut out: Vec<BackupEntry> = Vec::new();
    for ent in entries.flatten() {
        let path = ent.path();
        let name = match path.file_name().and_then(|n| n.to_str()) {
            Some(n) => n.to_string(),
            None => continue,
        };
        if !name.ends_with(BACKUP_EXT) {
            continue;
        }
        let size = std::fs::metadata(&path).map(|m| m.len()).unwrap_or(0);
        let mtime = mtime_ms(&path);
        let meta = read_meta(&meta_path_for(&path)).unwrap_or_default();
        out.push(BackupEntry {
            name,
            path: path.to_string_lossy().to_string(),
            size,
            mtime_ms: mtime,
            branch: meta.branch,
            note: meta.note,
            created_at_ms: meta.created_at_ms,
        });
    }
    // Newest first — both for the typical "I just made one" workflow and
    // to keep the default backup.sql.gz from drifting to the top if its
    // mtime is older than custom-named ones.
    out.sort_by(|a, b| b.mtime_ms.cmp(&a.mtime_ms));
    Ok(out)
}

#[tauri::command]
pub fn db_save_backup_meta(
    path: String,
    branch: Option<String>,
    note: Option<String>,
) -> Result<(), String> {
    let p = Path::new(&path);
    let meta = BackupMeta {
        created_at_ms: Some(now_ms()),
        branch: branch.and_then(|s| {
            let t = s.trim().to_string();
            if t.is_empty() { None } else { Some(t) }
        }),
        note: note.and_then(|s| {
            let t = s.trim().to_string();
            if t.is_empty() { None } else { Some(t) }
        }),
    };
    let raw = serde_json::to_string_pretty(&meta).map_err(|e| e.to_string())?;
    std::fs::write(meta_path_for(p), raw).map_err(|e| e.to_string())
}

#[tauri::command]
pub fn db_delete_backup(repo: String, path: String) -> Result<(), String> {
    let p = PathBuf::from(&path);
    // Require the path to live under <repo>/db-backups/. We accept a `repo`
    // argument explicitly rather than inferring from the path so we
    // can't be coerced into removing something outside the project
    // (e.g., a symlinked path or a sibling directory).
    let dir = backups_dir(&repo);
    if !p.starts_with(&dir) {
        return Err(format!("refusing to delete outside {}", dir.display()));
    }
    if !has_backup_ext(&p) {
        return Err(format!("refusing to delete non-backup file: {p:?}"));
    }
    if p.exists() {
        std::fs::remove_file(&p).map_err(|e| format!("deleting {p:?}: {e}"))?;
    }
    // Sidecar is best-effort — missing is fine, but a real I/O error
    // should surface so the user sees a half-cleaned state.
    let meta = meta_path_for(&p);
    if meta.exists() {
        std::fs::remove_file(&meta).map_err(|e| format!("deleting {meta:?}: {e}"))?;
    }
    Ok(())
}

fn has_backup_ext(p: &Path) -> bool {
    p.file_name()
        .and_then(|n| n.to_str())
        .map(|n| n.ends_with(BACKUP_EXT))
        .unwrap_or(false)
}

#[derive(Debug, Serialize)]
pub struct BackupNameCheck {
    pub final_name: String,
    pub exists: bool,
    pub relative_path: String,
}

// rename_all = "snake_case" so `raw_name` matches the snake_case key
// the frontend sends. See note on start_process for why we don't rely
// on Tauri v2's default camelCase mapping.
#[tauri::command(rename_all = "snake_case")]
pub fn db_check_backup_name(repo: String, raw_name: String) -> Result<BackupNameCheck, String> {
    let trimmed = raw_name.trim().trim_end_matches('/').to_string();
    // Strip the extension if the user typed it, then re-add — keeps the
    // ".sql.gz" suffix authoritative and prevents "foo.sql.gz.sql.gz".
    let stem = trimmed
        .strip_suffix(BACKUP_EXT)
        .unwrap_or(&trimmed)
        .to_string();
    if stem.is_empty() {
        return Err("backup name cannot be empty".into());
    }
    // Reject anything that isn't a "safe" filename — letters, digits,
    // dot, underscore, dash. Blocks path separators, `..` traversal,
    // control characters, NUL, leading/embedded dots-only names like
    // `.git`, etc. The frontend mirrors this check but the backend is
    // the source of truth.
    if stem.starts_with('.') {
        return Err("backup name cannot start with a dot".into());
    }
    if !stem
        .chars()
        .all(|c| c.is_ascii_alphanumeric() || c == '.' || c == '_' || c == '-')
    {
        return Err(
            "backup name may only contain letters, digits, dot, underscore, and dash"
                .into(),
        );
    }
    let final_name = format!("{stem}{BACKUP_EXT}");
    let full = backups_dir(&repo).join(&final_name);
    let rel = format!("{BACKUPS_DIRNAME}/{final_name}");
    Ok(BackupNameCheck {
        final_name,
        exists: full.exists(),
        relative_path: rel,
    })
}
