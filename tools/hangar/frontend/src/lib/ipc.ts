// IPC layer: the frontend's single boundary to the Go backend, backed by the
// Wails-generated service bindings and exposed as the `api.*` surface. The
// exported types are aliases to the Go-generated models (so they can't drift
// from the backend). `cast` narrows a binding's $CancellablePromise<Model> to
// the alias type at the boundary (same runtime JSON).
//
// (Historically this was Tauri's `invoke`; the file kept the `api.*` shape
// across the port to Wails so callers were untouched — hence it long outlived
// the name `tauri.ts`.)
import {
  SettingsService,
  ProcessService,
  GitService,
  DBService,
  GitopsService,
  FleetctlService,
  TroubleshootService,
  PerfService,
  PerfConfigService,
  DepsService,
  TrayService,
  DialogService,
  ScepService,
  MdmAssetsService,
  TufService,
} from "../../bindings/github.com/fleetdm/fleet/tools/hangar/services";

import type * as settingsM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/settings/models";
import type * as processesM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/processes/models";
import type * as gitrepoM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/gitrepo/models";
import type * as dbM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/db/models";
import type * as depsM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/deps/models";
import type * as fleetctlM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/fleetctl/models";
import type * as gitopsM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/gitops/models";
import type * as perfM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/perf/models";
import type * as perfconfigM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/perfconfig/models";
import type * as traymenuM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/traymenu/models";
import type * as troubleshootM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/troubleshoot/models";
import type * as scepM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/scep/models";
import type * as mdmassetsM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/mdmassets/models";
import type * as tufM from "../../bindings/github.com/fleetdm/fleet/tools/hangar/internal/tuf/models";

// cast narrows a binding's $CancellablePromise<GeneratedModel> to the alias
// type. Safe: the underlying value is the same JSON.
function cast<T>(p: unknown): Promise<T> {
  return p as Promise<T>;
}

// ThemePreference stays a string union (the generated ThemePreference is a TS
// enum whose values are these strings; the union is the accurate runtime type
// and avoids enum/literal friction in the Settings UI).
export type ThemePreference = "system" | "light" | "dark";

// Settings aliases the generated model but keeps theme as the union above and
// servers as the FleetServeConfig-aliased ServerProfile.
export type Settings = Omit<settingsM.Settings, "theme" | "servers"> & {
  theme: ThemePreference;
  servers: ServerProfile[];
};
export type NgrokConfig = settingsM.NgrokConfig;
export type PythonConfig = settingsM.PythonConfig;
export type FleetServeConfig = settingsM.FleetServeConfig;
export type EnvVar = settingsM.EnvVar;
export type NgrokTunnel = settingsM.NgrokTunnel;
export type NgrokYamlInfo = settingsM.NgrokYamlInfo;
export type NgrokRunningTunnel = settingsM.NgrokRunningTunnel;
export type RepoProbe = settingsM.RepoProbe;

export type GitopsFile = gitopsM.File;
export type GitopsRepo = gitopsM.Repo;
export type GitopsDirScan = gitopsM.DirScan;
export type GitopsTargetCheck = gitopsM.TargetCheck;

export type PerfTemplate = perfM.Template;
// Override os_counts: Wails types Go maps with optional values
// ({ [k]?: number }), but the values are always present at runtime, so keep
// the simpler Record<string, number> the UI expects.
export type PerfConfig = Omit<perfconfigM.Config, "os_counts"> & {
  os_counts: Record<string, number>;
};

export type CommitInfo = gitrepoM.CommitInfo;
export type FileChange = gitrepoM.FileChange;
export type BranchStatus = gitrepoM.BranchStatus;
export type Branch = gitrepoM.Branch;
export type Worktree = gitrepoM.Worktree;

// Multi-server profiles. ServerProfile keeps its generated fleet_serve as the
// FleetServeConfig alias used elsewhere; ServerPorts is a flat numeric record.
export type ServerPorts = settingsM.ServerPorts;
export type ServerProfile = Omit<settingsM.ServerProfile, "fleet_serve"> & {
  fleet_serve: FleetServeConfig;
};
export type ComposeTarget = processesM.ComposeTarget;

export type ProcInfo = processesM.ProcInfo;
export type LogEntry = processesM.LogEntry;
export type LogWindow = processesM.LogWindow;
export type ContainerState = processesM.ContainerState;
export type DockerStatus = processesM.DockerStatus;

export type BackupEntry = dbM.BackupEntry;
export type BackupNameCheck = dbM.BackupNameCheck;

export type ResolvedBinary = fleetctlM.ResolvedBinary;
export type ContextSummary = fleetctlM.ContextSummary;
export type ContextInfo = fleetctlM.ContextInfo;
export type RawConfig = fleetctlM.RawConfig;
export type CapturedRun = fleetctlM.CapturedRun;

export type DetectedProcess = troubleshootM.DetectedProcess;
export type KillOutcome = troubleshootM.KillOutcome;

export type ScepProfile = settingsM.ScepProfile;
export type ScepBinaryInfo = scepM.BinaryInfo;
export type ScepDepotInfo = scepM.DepotInfo;
export type ScepInitCAParams = scepM.InitCAParams;

export type MdmAssetsConfig = mdmassetsM.Config;
export type MdmAssetsExportResult = mdmassetsM.ExportResult;
export type MdmAssetsFile = mdmassetsM.AssetFile;

export type TufConfig = settingsM.TufConfig;
export type TufServerStatus = tufM.ServerStatus;

export type DepCheck = depsM.DepCheck;
export type DepReport = depsM.DepReport;

export type TrayState = traymenuM.State;

// LogLine and ProcEvent are event payloads (emitted via proc:log / proc:state),
// not service return types, so the binding generator doesn't model them — kept
// hand-written here.
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
  config_path: null,
  premium: true,
  debug: true,
  logging_debug: true,
  env: [],
};

export const api = {
  getSettings: () => cast<Settings>(SettingsService.GetSettings()),
  saveSettings: (settings: Settings) =>
    SettingsService.SaveSettings(settings as never),
  probeFleetRepo: (path?: string) =>
    cast<RepoProbe[]>(SettingsService.ProbeFleetRepo(path ?? "")),
  detectFleetConfig: (repo: string) =>
    cast<string | null>(SettingsService.DetectFleetConfig(repo)),
  checkDependencies: (repoPath?: string | null, refreshPath?: boolean) =>
    cast<DepReport>(
      DepsService.CheckDependencies(repoPath ?? "", refreshPath ?? false),
    ),
  parseNgrokYml: (path?: string | null) =>
    cast<NgrokYamlInfo>(SettingsService.ParseNgrokYml(path ?? "")),
  ngrokTunnels: () =>
    cast<NgrokRunningTunnel[]>(SettingsService.NgrokTunnels()),
  readTextFile: (path: string) => SettingsService.ReadTextFile(path),
  writeTextFile: (path: string, contents: string) =>
    SettingsService.WriteTextFile(path, contents),
  openPath: (path: string, reveal?: boolean) =>
    SettingsService.OpenPath(path, reveal ?? false),
  openUrl: (url: string) => SettingsService.OpenURL(url),

  // Native folder/file pickers (via Wails DialogService).
  pickFolder: () => DialogService.PickFolder(),
  pickFile: () => DialogService.PickFile(),
  pickFileWithFilter: (displayName: string, pattern: string) =>
    DialogService.PickFileWithFilter(displayName, pattern),

  gitBranchStatus: (repo: string) =>
    cast<BranchStatus>(GitService.GitBranchStatus(repo)),
  gitListBranches: (
    repo: string,
    filter?: string,
    query?: string,
    limit?: number,
  ) =>
    cast<Branch[]>(
      GitService.GitListBranches(repo, filter ?? "", query ?? "", limit ?? null),
    ),
  gitFetch: (repo: string) => GitService.GitFetch(repo),
  gitPull: (repo: string) => GitService.GitPull(repo),
  gitCheckout: (repo: string, branch: string) =>
    GitService.GitCheckout(repo, branch),
  gitStashAndCheckout: (repo: string, branch: string) =>
    GitService.GitStashAndCheckout(repo, branch),
  gitDiscardAndCheckout: (repo: string, branch: string) =>
    GitService.GitDiscardAndCheckout(repo, branch),

  // Worktrees — back multi-server (each server builds/runs its own tree).
  gitListWorktrees: (repo: string) =>
    cast<Worktree[]>(GitService.GitListWorktrees(repo)),
  gitAddWorktree: (repo: string, path: string, ref: string) =>
    GitService.GitAddWorktree(repo, path, ref),
  gitRemoveWorktree: (repo: string, path: string, force: boolean) =>
    GitService.GitRemoveWorktree(repo, path, force),

  // Server profiles.
  newServerProfile: () =>
    cast<ServerProfile>(SettingsService.NewServerProfile()),

  // SCEP servers (one shared in-repo binary, many depot-based profiles).
  newScepProfile: () =>
    cast<ScepProfile>(SettingsService.NewScepProfile()),
  scepBinaryStatus: () =>
    cast<ScepBinaryInfo>(ScepService.BinaryStatus()),
  scepEnsureBinary: () => cast<ScepBinaryInfo>(ScepService.EnsureBinary()),
  scepRebuildBinary: () => cast<ScepBinaryInfo>(ScepService.RebuildBinary()),
  scepResolveDepot: (profile: ScepProfile) =>
    cast<string>(ScepService.ResolveDepot(profile as never)),
  scepDepotInfo: (depotPath: string) =>
    cast<ScepDepotInfo>(ScepService.DepotInfo(depotPath)),
  scepProfileDepotInfo: (profile: ScepProfile) =>
    cast<ScepDepotInfo>(ScepService.ProfileDepotInfo(profile as never)),
  scepInitCa: (depotPath: string, params: ScepInitCAParams) =>
    cast<ScepDepotInfo>(ScepService.InitCA(depotPath, params as never)),
  scepStartProfile: (profile: ScepProfile) =>
    ScepService.StartProfile(profile as never),
  scepStopProfile: (profileId: string) =>
    ScepService.StopProfile(profileId),
  scepLanIp: () => cast<string>(ScepService.LanIP()),

  // MDM assets export (tools/mdm/assets).
  mdmAssetsConfigsList: () =>
    cast<MdmAssetsConfig[]>(MdmAssetsService.MdmAssetsConfigsList()),
  mdmAssetsConfigSave: (cfg: MdmAssetsConfig) =>
    cast<MdmAssetsConfig>(MdmAssetsService.MdmAssetsConfigSave(cfg as never)),
  mdmAssetsConfigDelete: (id: string) =>
    MdmAssetsService.MdmAssetsConfigDelete(id),
  mdmAssetsDefaultDir: () =>
    cast<string>(MdmAssetsService.MdmAssetsDefaultDir()),
  mdmAssetsExport: (cfg: MdmAssetsConfig) =>
    cast<MdmAssetsExportResult>(MdmAssetsService.MdmAssetsExport(cfg as never)),
  mdmAssetsReadFile: (path: string) =>
    cast<string>(MdmAssetsService.MdmAssetsReadFile(path)),

  // Local TUF server + fleetd package generation (tools/tuf/test).
  tufServerStatus: () => cast<TufServerStatus>(TufService.TufServerStatus()),
  tufStartBuild: (cfg: TufConfig) => TufService.TufStartBuild(cfg as never),
  tufStopBuild: () => TufService.TufStopBuild(),
  tufStartServer: () => TufService.TufStartServer(),
  tufKillServer: () => cast<KillOutcome[]>(TufService.TufKillServer()),
  tufDeleteAssets: () => TufService.TufDeleteAssets(),
  tufAssetsExist: () => cast<boolean>(TufService.TufAssetsExist()),

  listProcesses: () => cast<ProcInfo[]>(ProcessService.ListProcesses()),
  startProcess: (args: {
    id: string;
    label: string;
    cwd: string;
    program: string;
    args: string[];
    log_channel?: string | null;
    env?: Array<[string, string]> | null;
  }) =>
    ProcessService.StartProcess(
      args.id,
      args.label,
      args.cwd,
      args.program,
      args.args,
      args.log_channel ?? "",
      (args.env ?? []).map(([key, value]) => ({ key, value })) as never,
    ),
  stopProcess: (id: string) => ProcessService.StopProcess(id),
  restartProcess: (id: string) => ProcessService.RestartProcess(id),
  forgetProcess: (id: string) => ProcessService.ForgetProcess(id),

  dockerComposeStatus: (cwd: string, project: string) =>
    cast<DockerStatus>(ProcessService.DockerComposeStatus(cwd, project)),
  dockerComposeDown: (id: string, cwd: string, project: string) =>
    ProcessService.DockerComposeDown(id, cwd, project),
  dockerComposeRestart: (cwd: string, project: string) =>
    ProcessService.DockerComposeRestart(cwd, project),

  serveTcpCheck: (port: number, host?: string) =>
    ProcessService.ServeTCPCheck(host ?? "", port),

  dbBackupsDir: (repo: string) => DBService.DBBackupsDir(repo),
  dbEnsureBackupsDir: (repo: string) => DBService.DBEnsureBackupsDir(repo),
  dbListBackups: (repo: string) =>
    cast<BackupEntry[]>(DBService.DBListBackups(repo)),
  dbSaveBackupMeta: (
    path: string,
    branch: string | null,
    note: string | null,
  ) => DBService.DBSaveBackupMeta(path, branch, note),
  dbDeleteBackup: (repo: string, path: string) =>
    DBService.DBDeleteBackup(repo, path),
  dbCheckBackupName: (repo: string, rawName: string) =>
    cast<BackupNameCheck>(DBService.DBCheckBackupName(repo, rawName)),

  // Central per-server backups (app-data), addressed by directory.
  dbServerBackupsDir: (serverId: string) =>
    DBService.DBServerBackupsDir(serverId),
  dbEnsureDir: (dir: string) => DBService.DBEnsureDir(dir),
  dbListBackupsInDir: (dir: string) =>
    cast<BackupEntry[]>(DBService.DBListBackupsInDir(dir)),
  dbDeleteBackupInDir: (dir: string, path: string) =>
    DBService.DBDeleteBackupInDir(dir, path),
  dbCheckBackupNameInDir: (dir: string, rawName: string) =>
    cast<BackupNameCheck>(DBService.DBCheckBackupNameInDir(dir, rawName)),

  fleetctlResolveBinary: (repo: string | null, settingsPath: string | null) =>
    cast<ResolvedBinary>(
      FleetctlService.FleetctlResolveBinary(repo ?? "", settingsPath ?? ""),
    ),
  fleetctlReadContext: () =>
    cast<ContextInfo>(FleetctlService.FleetctlReadContext()),
  fleetctlReadConfigRaw: () =>
    cast<RawConfig>(FleetctlService.FleetctlReadConfigRaw()),
  fleetctlSaveConfig: (yaml: string) =>
    FleetctlService.FleetctlSaveConfig(yaml),

  troubleshootScanPort: (port: number) =>
    cast<DetectedProcess[]>(TroubleshootService.TroubleshootScanPort(port)),
  troubleshootScanPattern: (pattern: string) =>
    cast<DetectedProcess[]>(
      TroubleshootService.TroubleshootScanPattern(pattern),
    ),
  troubleshootKillPid: (pid: number) =>
    cast<KillOutcome>(TroubleshootService.TroubleshootKillPid(pid)),
  fleetctlRunCapture: (args: {
    program: string;
    cwd?: string | null;
    args: string[];
    env?: Record<string, string> | null;
    stdinData?: string | null;
    timeoutMs?: number | null;
  }) =>
    cast<CapturedRun>(
      FleetctlService.FleetctlRunCapture(
        args.program,
        args.cwd ?? "",
        args.args,
        args.env ?? {},
        args.stdinData ?? "",
        args.timeoutMs ?? 0,
      ),
    ),

  readLogWindow: (args: {
    // A log channel name (per-server `fleet-serve-<id>`), or "all".
    source: string;
    since_ms: number;
    levels: string[];
    search?: string | null;
    max_lines?: number | null;
  }) =>
    cast<LogWindow>(
      ProcessService.ReadLogWindow(
        args.source,
        args.since_ms,
        args.levels,
        args.search ?? null,
        args.max_lines ?? null,
      ),
    ),
  clearLogChannel: (channel: string) => ProcessService.ClearLogChannel(channel),
  saveLogSnapshot: (filename: string, contents: string) =>
    ProcessService.SaveLogSnapshot(filename, contents),
  logsDir: () => ProcessService.LogsDirPath(),

  perfListTemplates: () => cast<PerfTemplate[]>(PerfService.PerfListTemplates()),

  perfConfigsList: () => cast<PerfConfig[]>(PerfConfigService.PerfConfigsList()),
  perfConfigSave: (config: PerfConfig) =>
    cast<PerfConfig>(PerfConfigService.PerfConfigSave(config as never)),
  perfConfigDelete: (id: string) => PerfConfigService.PerfConfigDelete(id),

  gitopsListRepos: (dir: string) =>
    cast<GitopsDirScan>(GitopsService.GitopsListRepos(dir)),
  gitopsCheckTarget: (dir: string, name: string) =>
    cast<GitopsTargetCheck>(GitopsService.GitopsCheckTarget(dir, name)),

  updateTray: (state: TrayState) => TrayService.UpdateTray(state as never),
  shutdownNow: (targets: ComposeTarget[]) =>
    ProcessService.ShutdownNow(targets as never),
};
