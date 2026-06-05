use serde::Serialize;
use std::path::{Path, PathBuf};

#[derive(Debug, Clone, Serialize)]
pub struct GitopsFile {
    pub name: String,         // filename incl extension
    pub path: String,         // absolute path
    pub size: u64,            // bytes
    pub mtime_ms: u64,        // modification time
    pub subdir: String,       // "teams" or "fleets" — which folder it came from
}

#[derive(Debug, Clone, Serialize)]
pub struct GitopsRepo {
    pub name: String,         // basename of the repo dir
    pub path: String,         // absolute path to the repo dir
    pub has_default: bool,    // does default.yml exist (always true here — we only emit repos that have it)
    pub default_path: String, // absolute path to default.yml
    pub default_size: u64,
    pub default_mtime_ms: u64,
    /// All team/fleet YAML files we found. Sorted by filename within
    /// each subdir, with teams listed before fleets so the older
    /// convention reads first when a repo has both.
    pub team_files: Vec<GitopsFile>,
}

#[derive(Debug, Clone, Serialize)]
pub struct GitopsDirScan {
    pub root: String,
    /// True when the root itself contains `default.yml` — treat as a
    /// single-repo. The UI hides the repos column in this case.
    pub single_repo_mode: bool,
    /// Always non-empty when single_repo_mode is true (contains the
    /// root itself, named after the root's basename).
    pub repos: Vec<GitopsRepo>,
    /// Direct child dirs that don't contain `default.yml`. Surfaced
    /// for the UI so the user can see what's being ignored.
    pub ignored: Vec<String>,
}

#[derive(Debug, Clone, Serialize)]
pub struct GitopsTargetCheck {
    pub path: String,
    pub exists: bool,
    /// Number of regular files (recursive). Caps at 200 so a huge tree
    /// doesn't stall the UI — for the UI's "force required" message
    /// "200+ files" is enough.
    pub file_count: u32,
    /// True when the target is a directory we can write into. False
    /// for files, permission-denied targets, or invalid paths.
    pub writable: bool,
    /// User-friendly error reason when the target is unusable (e.g.
    /// path is a regular file, parent doesn't exist, etc.).
    pub reason: Option<String>,
}

fn mtime_ms(p: &Path) -> u64 {
    p.metadata()
        .and_then(|m| m.modified())
        .ok()
        .and_then(|t| t.duration_since(std::time::UNIX_EPOCH).ok())
        .map(|d| d.as_millis() as u64)
        .unwrap_or(0)
}

fn file_size(p: &Path) -> u64 {
    p.metadata().map(|m| m.len()).unwrap_or(0)
}

/// Collect all `*.yml` files in `<repo>/<subdir>`. Returns empty when
/// the subdir doesn't exist (most common — many repos only have
/// `default.yml` so far). Hidden files are skipped.
fn collect_team_yamls(repo: &Path, subdir: &str) -> Vec<GitopsFile> {
    let dir = repo.join(subdir);
    let read = match std::fs::read_dir(&dir) {
        Ok(r) => r,
        Err(_) => return Vec::new(),
    };
    let mut out = Vec::new();
    for entry in read.flatten() {
        let path = entry.path();
        if !path.is_file() {
            continue;
        }
        let Some(name) = path.file_name().and_then(|n| n.to_str()) else {
            continue;
        };
        if name.starts_with('.') {
            continue;
        }
        let lower = name.to_ascii_lowercase();
        if !(lower.ends_with(".yml") || lower.ends_with(".yaml")) {
            continue;
        }
        out.push(GitopsFile {
            name: name.to_string(),
            path: path.to_string_lossy().to_string(),
            size: file_size(&path),
            mtime_ms: mtime_ms(&path),
            subdir: subdir.to_string(),
        });
    }
    out.sort_by(|a, b| a.name.cmp(&b.name));
    out
}

fn build_repo(repo_path: PathBuf) -> Option<GitopsRepo> {
    let default_yml = repo_path.join("default.yml");
    if !default_yml.is_file() {
        return None;
    }
    let name = repo_path
        .file_name()
        .and_then(|n| n.to_str())
        .unwrap_or("")
        .to_string();
    let mut team_files = collect_team_yamls(&repo_path, "teams");
    team_files.extend(collect_team_yamls(&repo_path, "fleets"));
    Some(GitopsRepo {
        name,
        path: repo_path.to_string_lossy().to_string(),
        has_default: true,
        default_path: default_yml.to_string_lossy().to_string(),
        default_size: file_size(&default_yml),
        default_mtime_ms: mtime_ms(&default_yml),
        team_files,
    })
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

#[tauri::command]
pub fn gitops_list_repos(dir: String) -> Result<GitopsDirScan, String> {
    let root = PathBuf::from(shellexpand(&dir));
    if !root.is_dir() {
        return Err(format!("not a directory: {}", root.display()));
    }
    // Single-repo: the configured dir IS the repo.
    if root.join("default.yml").is_file() {
        let repo = build_repo(root.clone()).ok_or_else(|| {
            "single-repo mode but default.yml unreadable".to_string()
        })?;
        return Ok(GitopsDirScan {
            root: root.to_string_lossy().to_string(),
            single_repo_mode: true,
            repos: vec![repo],
            ignored: Vec::new(),
        });
    }
    // Multi-repo: scan direct children.
    let mut repos = Vec::new();
    let mut ignored = Vec::new();
    let entries = std::fs::read_dir(&root).map_err(|e| e.to_string())?;
    for entry in entries.flatten() {
        let p = entry.path();
        if !p.is_dir() {
            continue;
        }
        let basename = p
            .file_name()
            .and_then(|n| n.to_str())
            .unwrap_or("")
            .to_string();
        if basename.starts_with('.') {
            continue;
        }
        match build_repo(p) {
            Some(r) => repos.push(r),
            None => ignored.push(basename),
        }
    }
    repos.sort_by(|a, b| a.name.cmp(&b.name));
    ignored.sort();
    Ok(GitopsDirScan {
        root: root.to_string_lossy().to_string(),
        single_repo_mode: false,
        repos,
        ignored,
    })
}

/// Walks `path` and counts regular files, capped at `cap`. Used for
/// the "force required to overwrite — N files" warning in the generate
/// form. Cap keeps the check fast on a huge tree.
fn count_files(path: &Path, cap: u32) -> u32 {
    let mut total: u32 = 0;
    let mut stack: Vec<PathBuf> = vec![path.to_path_buf()];
    while let Some(p) = stack.pop() {
        if total >= cap {
            return cap;
        }
        let Ok(read) = std::fs::read_dir(&p) else {
            continue;
        };
        for entry in read.flatten() {
            if total >= cap {
                return cap;
            }
            let ep = entry.path();
            if ep.is_dir() {
                stack.push(ep);
            } else if ep.is_file() {
                total += 1;
            }
        }
    }
    total
}

/// Validates the target subdirectory the user is about to generate
/// into. We deliberately do NOT create anything — just answer "is this
/// a safe spot." The webview-facing `name` is restricted so the user
/// can't escape the configured gitops dir.
#[tauri::command]
pub fn gitops_check_target(
    dir: String,
    name: String,
) -> Result<GitopsTargetCheck, String> {
    let root = PathBuf::from(shellexpand(&dir));
    if name.is_empty()
        || name.contains('/')
        || name.contains('\\')
        || name.contains("..")
        || name == "."
        || name == "~"
    {
        return Ok(GitopsTargetCheck {
            path: root.join(&name).to_string_lossy().to_string(),
            exists: false,
            file_count: 0,
            writable: false,
            reason: Some("invalid subdirectory name".into()),
        });
    }
    if !root.is_dir() {
        return Ok(GitopsTargetCheck {
            path: root.join(&name).to_string_lossy().to_string(),
            exists: false,
            file_count: 0,
            writable: false,
            reason: Some(format!(
                "parent directory does not exist: {}",
                root.display()
            )),
        });
    }
    let target = root.join(&name);
    let target_str = target.to_string_lossy().to_string();
    if !target.exists() {
        // Parent exists, target doesn't — safest "available" state. We
        // check the *parent* for writability via metadata permissions;
        // a true permission check would need an actual write attempt,
        // which we don't want as a side-effect of typing in a field.
        let writable = root
            .metadata()
            .map(|m| !m.permissions().readonly())
            .unwrap_or(false);
        return Ok(GitopsTargetCheck {
            path: target_str,
            exists: false,
            file_count: 0,
            writable,
            reason: None,
        });
    }
    if target.is_file() {
        return Ok(GitopsTargetCheck {
            path: target_str,
            exists: true,
            file_count: 0,
            writable: false,
            reason: Some("target is a file, not a directory".into()),
        });
    }
    // Existing directory — count files for the force-overwrite hint.
    let file_count = count_files(&target, 200);
    let writable = target
        .metadata()
        .map(|m| !m.permissions().readonly())
        .unwrap_or(false);
    Ok(GitopsTargetCheck {
        path: target_str,
        exists: true,
        file_count,
        writable,
        reason: None,
    })
}
