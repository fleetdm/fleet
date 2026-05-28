use serde::Serialize;
use std::path::Path;
use std::process::Command;

#[derive(Debug, Serialize)]
pub struct BranchStatus {
    pub branch: String,
    pub clean: bool,
    pub ahead: u32,
    pub behind: u32,
    pub modified: Vec<FileChange>,
    pub last_commit: Option<CommitInfo>,
}

#[derive(Debug, Serialize)]
pub struct FileChange {
    pub status: String,
    pub path: String,
}

#[derive(Debug, Serialize, Clone)]
pub struct CommitInfo {
    pub sha: String,
    pub subject: String,
    pub author: String,
    pub time_ago: String,
}

#[derive(Debug, Serialize)]
pub struct Branch {
    pub name: String,
    pub is_current: bool,
    pub is_local: bool,
    pub is_remote: bool,
    pub last_commit: Option<CommitInfo>,
}

fn run_git(repo: &str, args: &[&str]) -> Result<String, String> {
    // PATH from the login shell so `git` resolves when the app is
    // launched from Finder (git is often /opt/homebrew/bin/git, not on
    // the bare GUI PATH).
    let out = Command::new("git")
        .env("PATH", crate::shellpath::shell_path())
        .arg("-C")
        .arg(repo)
        .args(args)
        .output()
        .map_err(|e| format!("failed to spawn git: {e}"))?;
    if !out.status.success() {
        return Err(String::from_utf8_lossy(&out.stderr).to_string());
    }
    Ok(String::from_utf8_lossy(&out.stdout).to_string())
}

#[tauri::command]
pub fn git_branch_status(repo: String) -> Result<BranchStatus, String> {
    if !Path::new(&repo).join(".git").exists() {
        return Err(format!("not a git repo: {repo}"));
    }

    let branch = run_git(&repo, &["rev-parse", "--abbrev-ref", "HEAD"])?
        .trim()
        .to_string();

    let porcelain = run_git(&repo, &["status", "--porcelain"])?;
    let modified: Vec<FileChange> = porcelain
        .lines()
        .filter_map(|line| {
            if line.len() < 4 {
                return None;
            }
            let status = line[..2].trim().to_string();
            let path = line[3..].to_string();
            Some(FileChange { status, path })
        })
        .collect();
    // "clean" = no tracked changes. Untracked files (??) don't count —
    // they don't block checkouts and shouldn't make the hero indicator warn.
    let clean = modified.iter().all(|f| f.status == "??");

    let (ahead, behind) = match run_git(
        &repo,
        &["rev-list", "--left-right", "--count", "HEAD...@{upstream}"],
    ) {
        Ok(s) => {
            let parts: Vec<&str> = s.split_whitespace().collect();
            let a = parts.first().and_then(|x| x.parse().ok()).unwrap_or(0);
            let b = parts.get(1).and_then(|x| x.parse().ok()).unwrap_or(0);
            (a, b)
        }
        Err(_) => (0, 0),
    };

    let last_commit = read_last_commit(&repo, "HEAD").ok();

    Ok(BranchStatus {
        branch,
        clean,
        ahead,
        behind,
        modified,
        last_commit,
    })
}

fn read_last_commit(repo: &str, refname: &str) -> Result<CommitInfo, String> {
    let raw = run_git(
        repo,
        &[
            "log",
            "-1",
            "--format=%h\x1f%s\x1f%an\x1f%cr",
            refname,
        ],
    )?;
    let parts: Vec<&str> = raw.trim().splitn(4, '\x1f').collect();
    if parts.len() != 4 {
        return Err("unexpected git log output".into());
    }
    Ok(CommitInfo {
        sha: parts[0].to_string(),
        subject: parts[1].to_string(),
        author: parts[2].to_string(),
        time_ago: parts[3].to_string(),
    })
}

/// Extracts the minor-line key from an RC branch name.
/// e.g. "rc-minor-fleet-v4.86.0" -> Some("4.86")
///      "rc-patch-fleet-v4.86.3" -> Some("4.86")
fn parse_rc_minor_key(name: &str) -> Option<String> {
    let s = name
        .strip_prefix("rc-minor-fleet-v")
        .or_else(|| name.strip_prefix("rc-patch-fleet-v"))?;
    let mut parts = s.split('.');
    let major = parts.next()?;
    let minor = parts.next()?;
    major.parse::<u32>().ok()?;
    minor.parse::<u32>().ok()?;
    Some(format!("{}.{}", major, minor))
}

#[tauri::command]
pub fn git_list_branches(
    repo: String,
    filter: Option<String>,
    limit: Option<u32>,
) -> Result<Vec<Branch>, String> {
    let current = run_git(&repo, &["rev-parse", "--abbrev-ref", "HEAD"])?
        .trim()
        .to_string();

    // Build the ref pattern list based on filter.
    let patterns: Vec<String> = match filter.as_deref() {
        Some("rc") => vec![
            "refs/heads/rc-patch-fleet-v*".into(),
            "refs/heads/rc-minor-fleet-v*".into(),
            "refs/remotes/origin/rc-patch-fleet-v*".into(),
            "refs/remotes/origin/rc-minor-fleet-v*".into(),
        ],
        Some("main") => vec![
            "refs/heads/main".into(),
            "refs/heads/master".into(),
            "refs/remotes/origin/main".into(),
            "refs/remotes/origin/master".into(),
        ],
        // None or "all" → every local and remote ref
        _ => vec!["refs/heads".into(), "refs/remotes".into()],
    };

    let mut args: Vec<String> = vec![
        "for-each-ref".into(),
        "--sort=-committerdate".into(),
        "--format=%(refname:short)\x1f%(objectname:short)\x1f%(contents:subject)\x1f%(authorname)\x1f%(committerdate:relative)\x1f%(refname)".into(),
    ];
    // For non-RC filters, apply a count cap on the for-each-ref side.
    // RC handles its limit semantics ("N minor lines") in post-processing
    // below, so we fetch the full set there.
    if filter.as_deref() != Some("rc") {
        if let Some(n) = limit {
            args.push(format!("--count={}", n.saturating_mul(2)));
        }
    }
    args.extend(patterns);
    let arg_refs: Vec<&str> = args.iter().map(String::as_str).collect();
    let raw = run_git(&repo, &arg_refs)?;

    let mut seen = std::collections::HashSet::new();
    let mut branches: Vec<Branch> = Vec::new();

    for line in raw.lines() {
        let parts: Vec<&str> = line.splitn(6, '\x1f').collect();
        if parts.len() != 6 {
            continue;
        }
        let mut name = parts[0].to_string();
        let full_ref = parts[5];
        let is_local = full_ref.starts_with("refs/heads/");
        let is_remote = full_ref.starts_with("refs/remotes/");

        if is_remote {
            if let Some(rest) = name.strip_prefix("origin/") {
                if rest == "HEAD" {
                    continue;
                }
                name = rest.to_string();
            } else {
                continue;
            }
        }

        if !seen.insert(name.clone()) {
            // already added (local takes precedence)
            continue;
        }

        let is_current = is_local && name == current;
        branches.push(Branch {
            name,
            is_current,
            is_local,
            is_remote: is_remote && !is_local,
            last_commit: Some(CommitInfo {
                sha: parts[1].to_string(),
                subject: parts[2].to_string(),
                author: parts[3].to_string(),
                time_ago: parts[4].to_string(),
            }),
        });
    }

    // For RC: group by minor-line key, keep the N most-recently-active
    // minor lines, drop everything else. This way a patch never appears
    // without its minor, and you get full release-line context.
    if filter.as_deref() == Some("rc") {
        let n = limit.unwrap_or(10) as usize;
        let mut minor_order: Vec<String> = Vec::new();
        let mut seen_keys = std::collections::HashSet::new();
        for b in &branches {
            if let Some(key) = parse_rc_minor_key(&b.name) {
                if seen_keys.insert(key.clone()) {
                    minor_order.push(key);
                }
            }
        }
        let kept_keys: std::collections::HashSet<String> =
            minor_order.into_iter().take(n).collect();
        branches.retain(|b| {
            // Always keep the current branch (so users see where they are).
            b.is_current
                || parse_rc_minor_key(&b.name)
                    .map(|k| kept_keys.contains(&k))
                    .unwrap_or(false)
        });
    } else if let Some(n) = limit {
        branches.truncate(n as usize);
    }

    Ok(branches)
}

#[tauri::command]
pub fn git_fetch(repo: String) -> Result<String, String> {
    run_git(&repo, &["fetch", "--all", "--prune"])
}

#[tauri::command]
pub fn git_pull(repo: String) -> Result<String, String> {
    run_git(&repo, &["pull", "--ff-only"])
}

#[tauri::command]
pub fn git_checkout(repo: String, branch: String) -> Result<String, String> {
    run_git(&repo, &["checkout", &branch])
}

#[tauri::command]
pub fn git_stash_and_checkout(repo: String, branch: String) -> Result<String, String> {
    run_git(&repo, &["stash", "push", "-u", "-m", "fleet-hangar auto-stash"])?;
    run_git(&repo, &["checkout", &branch])
}

#[tauri::command]
pub fn git_discard_and_checkout(repo: String, branch: String) -> Result<String, String> {
    run_git(&repo, &["checkout", "--", "."])?;
    run_git(&repo, &["clean", "-fd"])?;
    run_git(&repo, &["checkout", &branch])
}
