import { invoke } from "@tauri-apps/api/core";

export type ThemePreference = "system" | "light" | "dark";

export interface Settings {
  repo_path: string | null;
  fleetctl_path: string | null;
  gitops_dir: string | null;
  first_run_complete: boolean;
  ngrok: NgrokConfig;
  python_server: PythonConfig;
  fleet_serve: FleetServeConfig;
  theme: ThemePreference;
  /// fleetctl cron names the user has starred — drives the Favorites
  /// section at the top of the Trigger sub-tab.
  favorite_crons: string[];
}

export interface GitopsFile {
  name: string;
  path: string;
  size: number;
  mtime_ms: number;
  subdir: string;
}

export interface GitopsRepo {
  name: string;
  path: string;
  has_default: boolean;
  default_path: string;
  default_size: number;
  default_mtime_ms: number;
  team_files: GitopsFile[];
}

export interface GitopsDirScan {
  root: string;
  single_repo_mode: boolean;
  repos: GitopsRepo[];
  ignored: string[];
}

export interface GitopsTargetCheck {
  path: string;
  exists: boolean;
  file_count: number;
  writable: boolean;
  reason: string | null;
}

export interface PerfTemplate {
  id: string;
  label: string;
  version: string;
  mobile: boolean;
  apple: boolean;
}

/// A saved osquery-perf run configuration. Mirrors the backend struct
/// in src-tauri/src/perf_configs.rs. v1 stores enroll_secret and
/// mdm_scep_challenge as plain text — local dev only, same security
/// posture as the rest of the fleet-hangar settings file.
export interface PerfConfig {
  id: string;
  name: string;
  server_url: string;
  enroll_secret: string;
  os_counts: Record<string, number>;
  mdm_enabled: boolean;
  mdm_prob: number;
  mdm_scep_challenge: string;
  start_period: string;
  query_interval: string;
  config_interval: string;
  /// Server-stamped; 0 on a brand-new config the frontend hasn't saved yet.
  created_at_ms: number;
  updated_at_ms: number;
}

export interface NgrokConfig {
  enabled: boolean;
  yml_path: string | null;
  default_tunnels: string[];
  start_all: boolean;
}

export interface PythonConfig {
  enabled: boolean;
  port: number;
  directory: string | null;
}

export interface EnvVar {
  key: string;
  value: string;
  enabled: boolean;
}

export interface FleetServeConfig {
  /// Path passed to `--config`. null / empty = omit the flag.
  config_path: string | null;
  /// Drives `--dev_license`.
  premium: boolean;
  /// Drives `--debug`.
  debug: boolean;
  /// Drives `--logging_debug`.
  logging_debug: boolean;
  env: EnvVar[];
}

export interface NgrokTunnel {
  name: string;
  proto: string;
  addr: string;
}

export interface NgrokYamlInfo {
  valid: boolean;
  error: string | null;
  resolved_path: string;
  has_authtoken: boolean;
  tunnels: NgrokTunnel[];
}

export const DEFAULT_NGROK_CONFIG: NgrokConfig = {
  enabled: false,
  yml_path: null,
  default_tunnels: [],
  start_all: false,
};

export const DEFAULT_PYTHON_CONFIG: PythonConfig = {
  enabled: false,
  port: 8000,
  directory: null,
};

export const DEFAULT_FLEET_SERVE_CONFIG: FleetServeConfig = {
  config_path: "fleet.yml",
  premium: true,
  debug: true,
  logging_debug: true,
  env: [],
};

export interface RepoProbe {
  path: string;
  valid: boolean;
  reason: string | null;
}

export interface CommitInfo {
  sha: string;
  subject: string;
  author: string;
  time_ago: string;
}

export interface FileChange {
  status: string;
  path: string;
}

export interface BranchStatus {
  branch: string;
  clean: boolean;
  ahead: number;
  behind: number;
  modified: FileChange[];
  last_commit: CommitInfo | null;
}

export interface Branch {
  name: string;
  is_current: boolean;
  is_local: boolean;
  is_remote: boolean;
  last_commit: CommitInfo | null;
}

export interface ProcInfo {
  id: string;
  label: string;
  command: string;
  cwd: string;
  state: "idle" | "running" | "done" | "failed" | "stopping";
  started_at_ms: number | null;
  ended_at_ms: number | null;
  exit_code: number | null;
  exit_signal: number | null;
  recent_log: string[];
  was_user_stopped: boolean;
}

export interface LogLine {
  proc_id: string;
  stream: "stdout" | "stderr";
  line: string;
  ts_ms: number;
}

export interface ProcEvent {
  proc_id: string;
  state: "running" | "done" | "failed" | "stopping";
  exit_code: number | null;
  exit_signal: number | null;
}

export interface LogEntry {
  ts_ms: number;
  stream: "stdout" | "stderr";
  level: "debug" | "info" | "warn" | "error" | null;
  message: string;
  channel: string;
}

export interface LogWindow {
  entries: LogEntry[];
  total_in_window: number;
  warn_count: number;
  error_count: number;
}

export interface BackupEntry {
  name: string;
  path: string;
  size: number;
  mtime_ms: number;
  branch: string | null;
  note: string | null;
  created_at_ms: number | null;
}

export interface BackupNameCheck {
  final_name: string;
  exists: boolean;
  relative_path: string;
}

export interface ResolvedBinary {
  path: string;
  source: "settings" | "build" | "missing";
  exists: boolean;
}

export interface ContextSummary {
  name: string;
  address: string | null;
  email: string | null;
  has_token: boolean;
}

export interface ContextInfo {
  config_path: string;
  exists: boolean;
  current: ContextSummary | null;
  contexts: ContextSummary[];
}

export interface RawConfig {
  path: string;
  exists: boolean;
  contents: string;
}

export interface DetectedProcess {
  pid: number;
  command: string;
}

export interface KillOutcome {
  pid: number;
  gone: boolean;
  used_kill: boolean;
  error: string | null;
}

export interface CapturedRun {
  exit_code: number | null;
  stdout: string;
  stderr: string;
}

export interface ContainerState {
  name: string;
  state: string;
}

export interface DockerStatus {
  running: boolean;
  containers: ContainerState[];
}

export interface DepCheck {
  id: string;
  name: string;
  installed: boolean;
  version: string | null;
  required: string | null;
  version_ok: boolean | null;
  runtime_ok: boolean | null;
  install_command: string;
  doc_url: string | null;
  note: string | null;
}

export interface DepReport {
  checks: DepCheck[];
}

export const api = {
  getSettings: () => invoke<Settings>("get_settings"),
  saveSettings: (settings: Settings) =>
    invoke<void>("save_settings", { settings }),
  probeFleetRepo: (path?: string) =>
    invoke<RepoProbe[]>("probe_fleet_repo", { path }),
  checkDependencies: (repoPath?: string | null, refreshPath?: boolean) =>
    invoke<DepReport>("check_dependencies", {
      repoPath: repoPath ?? null,
      refreshPath: refreshPath ?? false,
    }),
  parseNgrokYml: (path?: string | null) =>
    invoke<NgrokYamlInfo>("parse_ngrok_yml", { path: path ?? null }),
  readTextFile: (path: string) =>
    invoke<string>("read_text_file", { path }),
  writeTextFile: (path: string, contents: string) =>
    invoke<void>("write_text_file", { path, contents }),
  openPath: (path: string, reveal?: boolean) =>
    invoke<void>("open_path", { path, reveal: reveal ?? false }),
  openUrl: (url: string) => invoke<void>("open_url", { url }),

  gitBranchStatus: (repo: string) =>
    invoke<BranchStatus>("git_branch_status", { repo }),
  gitListBranches: (repo: string, filter?: string, limit?: number) =>
    invoke<Branch[]>("git_list_branches", { repo, filter, limit }),
  gitFetch: (repo: string) => invoke<string>("git_fetch", { repo }),
  gitPull: (repo: string) => invoke<string>("git_pull", { repo }),
  gitCheckout: (repo: string, branch: string) =>
    invoke<string>("git_checkout", { repo, branch }),
  gitStashAndCheckout: (repo: string, branch: string) =>
    invoke<string>("git_stash_and_checkout", { repo, branch }),
  gitDiscardAndCheckout: (repo: string, branch: string) =>
    invoke<string>("git_discard_and_checkout", { repo, branch }),

  listProcesses: () => invoke<ProcInfo[]>("list_processes"),
  startProcess: (args: {
    id: string;
    label: string;
    cwd: string;
    program: string;
    args: string[];
    log_channel?: string | null;
    /// KV pairs as tuples — matches Vec<(String, String)> on the
    /// Rust side. Empty-key rows are dropped server-side.
    env?: Array<[string, string]> | null;
  }) => invoke<void>("start_process", args),
  stopProcess: (id: string) => invoke<void>("stop_process", { id }),
  restartProcess: (id: string) => invoke<void>("restart_process", { id }),
  forgetProcess: (id: string) => invoke<void>("forget_process", { id }),

  dockerComposeStatus: (cwd: string) =>
    invoke<DockerStatus>("docker_compose_status", { cwd }),
  dockerComposeDown: (cwd: string) =>
    invoke<string>("docker_compose_down_cmd", { cwd }),
  dockerComposeRestart: (cwd: string) =>
    invoke<string>("docker_compose_restart_cmd", { cwd }),

  serveTcpCheck: (port: number, host?: string) =>
    invoke<boolean>("serve_tcp_check", { host: host ?? null, port }),

  dbBackupsDir: (repo: string) =>
    invoke<string>("db_backups_dir", { repo }),
  dbEnsureBackupsDir: (repo: string) =>
    invoke<string>("db_ensure_backups_dir", { repo }),
  dbListBackups: (repo: string) =>
    invoke<BackupEntry[]>("db_list_backups", { repo }),
  dbSaveBackupMeta: (
    path: string,
    branch: string | null,
    note: string | null,
  ) => invoke<void>("db_save_backup_meta", { path, branch, note }),
  dbDeleteBackup: (repo: string, path: string) =>
    invoke<void>("db_delete_backup", { repo, path }),
  dbCheckBackupName: (repo: string, rawName: string) =>
    invoke<BackupNameCheck>("db_check_backup_name", {
      repo,
      raw_name: rawName,
    }),

  fleetctlResolveBinary: (
    repo: string | null,
    settingsPath: string | null,
  ) =>
    invoke<ResolvedBinary>("fleetctl_resolve_binary", {
      repo,
      settings_path: settingsPath,
    }),
  fleetctlReadContext: () =>
    invoke<ContextInfo>("fleetctl_read_context"),
  fleetctlReadConfigRaw: () =>
    invoke<RawConfig>("fleetctl_read_config_raw"),
  fleetctlSaveConfig: (yaml: string) =>
    invoke<void>("fleetctl_save_config", { yaml }),

  troubleshootScanPort: (port: number) =>
    invoke<DetectedProcess[]>("troubleshoot_scan_port", { port }),
  troubleshootScanPattern: (pattern: string) =>
    invoke<DetectedProcess[]>("troubleshoot_scan_pattern", { pattern }),
  troubleshootKillPid: (pid: number) =>
    invoke<KillOutcome>("troubleshoot_kill_pid", { pid }),
  fleetctlRunCapture: (args: {
    program: string;
    cwd?: string | null;
    args: string[];
    env?: Record<string, string> | null;
    stdinData?: string | null;
    timeoutMs?: number | null;
  }) =>
    invoke<CapturedRun>("fleetctl_run_capture", {
      program: args.program,
      cwd: args.cwd ?? null,
      args: args.args,
      env: args.env ?? null,
      stdin_data: args.stdinData ?? null,
      timeout_ms: args.timeoutMs ?? null,
    }),

  readLogWindow: (args: {
    source: "fleet-serve" | "docker-compose" | "all";
    since_ms: number;
    levels: string[];
    search?: string | null;
    max_lines?: number | null;
  }) => invoke<LogWindow>("read_log_window", args),
  clearLogChannel: (channel: string) =>
    invoke<void>("clear_log_channel", { channel }),
  saveLogSnapshot: (filename: string, contents: string) =>
    invoke<string>("save_log_snapshot", { filename, contents }),
  logsDir: () => invoke<string>("logs_dir_path"),

  perfListTemplates: () => invoke<PerfTemplate[]>("perf_list_templates"),

  perfConfigsList: () => invoke<PerfConfig[]>("perf_configs_list"),
  perfConfigSave: (config: PerfConfig) =>
    invoke<PerfConfig>("perf_config_save", { config }),
  perfConfigDelete: (id: string) =>
    invoke<void>("perf_config_delete", { id }),

  gitopsListRepos: (dir: string) =>
    invoke<GitopsDirScan>("gitops_list_repos", { dir }),
  gitopsCheckTarget: (dir: string, name: string) =>
    invoke<GitopsTargetCheck>("gitops_check_target", { dir, name }),

  updateTray: (state: TrayState) => invoke<void>("update_tray", { state }),
  shutdownNow: (repoPath: string | null) =>
    invoke<void>("shutdown_now", { repo_path: repoPath }),
};

export interface TrayState {
  branch: string | null;
  serve_up: boolean;
  docker_up: boolean;
  ngrok_running: boolean;
  python_running: boolean;
}
