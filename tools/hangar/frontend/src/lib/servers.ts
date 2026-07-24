// Multi-server helpers. A "server" is one independent local Fleet instance:
// its own git worktree (so it can build/run a different branch), host ports,
// docker compose project, and serve config. Server 1 keeps the canonical dev
// ports so it behaves byte-identically to the single-server era; additional
// servers get offset ports + a scoped docker stack.
import type { ServerProfile, ServerPorts, Settings } from "./ipc";

export const MAX_SERVERS = 3;

// --- identity / lookup ---

export function activeServer(settings: Settings): ServerProfile {
  return (
    settings.servers.find((s) => s.id === settings.active_server_id) ??
    settings.servers[0]
  );
}

export function serverById(
  settings: Settings,
  id: string,
): ServerProfile | undefined {
  return settings.servers.find((s) => s.id === id);
}

/// Returns new Settings with server `id` replaced by updater(server). Spreads
/// the rest so legacy/unrelated fields are preserved.
export function updateServer(
  settings: Settings,
  id: string,
  updater: (s: ServerProfile) => ServerProfile,
): Settings {
  return {
    ...settings,
    servers: settings.servers.map((s) => (s.id === id ? updater(s) : s)),
  };
}

export function updateActiveServer(
  settings: Settings,
  updater: (s: ServerProfile) => ServerProfile,
): Settings {
  return updateServer(settings, settings.active_server_id, updater);
}

export function canAddServer(settings: Settings): boolean {
  return settings.servers.length < MAX_SERVERS;
}

// --- process / log namespacing ---
//
// Per-server processes are keyed `<serverID>:<base>` so two servers' chains
// never collide in the process manager. ngrok / python stay global (one each).

export function procId(serverId: string, base: string): string {
  return `${serverId}:${base}`;
}

/// The fleet-serve log channel for a server (one structured-log ring + on-disk
/// file per server).
export function serveChannel(serverId: string): string {
  return `fleet-serve-${serverId}`;
}

// --- docker compose ---

/// The docker network the compose project creates (`<project>_default`). Used
/// by backup.sh / restore.sh, which attach a one-off mysql client container to
/// it via FLEET_COMPOSE_NETWORK.
export function composeNetwork(server: ServerProfile): string {
  return `${server.compose_project}_default`;
}

// One-off client image for db backup/restore. Matches tools/backup_db/*.sh
// (${FLEET_MYSQL_IMAGE:-mysql:8.0.44}).
const DB_CLIENT_IMAGE = "mysql:8.0.44";

/// `bash -c` payload that dumps a server's `fleet` DB, via that server's compose
/// network, to a gzipped file. Hangar builds this itself instead of shelling out
/// to the worktree's tools/backup_db/backup.sh, because secondary worktrees run
/// old/released refs whose scripts predate FLEET_COMPOSE_NETWORK and would
/// silently target the primary (fleet_default) stack. `set -o pipefail` inside
/// the container makes a failed mysqldump fail the whole command instead of
/// writing a "successful" empty archive.
export function dbBackupCommand(server: ServerProfile, absPath: string): string {
  const net = composeNetwork(server);
  return (
    `docker run --rm --network '${net}' ${DB_CLIENT_IMAGE} ` +
    `bash -c 'set -o pipefail; mysqldump -hmysql -uroot -ptoor ` +
    `--default-character-set=utf8mb4 --add-drop-database --databases fleet | gzip -' ` +
    `> '${absPath}'`
  );
}

/// `bash -c` payload that restores a gzipped dump into a server's `fleet` DB via
/// that server's compose network. See dbBackupCommand for why Hangar owns this.
export function dbRestoreCommand(
  server: ServerProfile,
  absPath: string,
): string {
  const net = composeNetwork(server);
  return (
    `docker run --rm -i --network '${net}' ${DB_CLIENT_IMAGE} ` +
    `bash -c 'set -o pipefail; gzip -dc - | MYSQL_PWD=toor mysql -hmysql -uroot fleet' ` +
    `< '${absPath}'`
  );
}

/// docker compose env that maps the parameterized host ports to this server's
/// block. Read by the `${FLEET_*_PORT:-default}` substitutions in
/// docker-compose.yml at `up` time.
export function dockerEnvFor(server: ServerProfile): Array<[string, string]> {
  const p = server.ports;
  return [
    ["FLEET_MYSQL_PORT", String(p.mysql)],
    ["FLEET_REDIS_PORT", String(p.redis)],
    ["FLEET_S3_PORT", String(p.s3)],
    ["FLEET_S3_CONSOLE_PORT", String(p.s3_console)],
  ];
}

/// `docker compose -p <project> up -d [services...]`. The default stack (server
/// 1) brings up everything, matching prior single-server behavior; secondary
/// stacks bring up only what a parallel fleet server needs (mysql/redis/s3),
/// so the optional services (mail, saml, prometheus, …) don't collide on their
/// host ports between projects.
export function dockerUpArgs(server: ServerProfile): string[] {
  const args = ["compose", "-p", server.compose_project, "up", "-d"];
  if (!isDefaultStack(server)) {
    args.push("mysql", "redis", "s3");
  }
  return args;
}

// --- fleet serve / prepare db ---

/// True when the server is on the canonical dev ports. For the default stack we
/// omit the address flags / S3 env so `fleet serve --dev` runs exactly as it
/// did pre-multi-server; non-default stacks get explicit addresses.
export function isDefaultStack(server: ServerProfile): boolean {
  const p = server.ports;
  return (
    p.server === 8080 && p.mysql === 3306 && p.redis === 6379 && p.s3 === 9000
  );
}

export function serveArgsFor(server: ServerProfile): string[] {
  const cfg = server.fleet_serve;
  const args = ["serve", "--dev"];
  const configPath = cfg.config_path?.trim();
  if (configPath) args.push("--config", configPath);
  if (cfg.premium) args.push("--dev_license");
  if (cfg.debug) args.push("--debug");
  if (cfg.logging_debug) args.push("--logging_debug");
  if (!isDefaultStack(server)) {
    const p = server.ports;
    args.push("--server_address", `localhost:${p.server}`);
    args.push("--mysql_address", `localhost:${p.mysql}`);
    args.push("--redis_address", `localhost:${p.redis}`);
  }
  return args;
}

/// fleet-serve env as [key, value] tuples. The user's configured env comes
/// first (enabled, non-empty-key rows only); non-default stacks then get the
/// S3 endpoint overrides so the server talks to its own object store.
export function serveEnvFor(server: ServerProfile): Array<[string, string]> {
  const rows: Array<[string, string]> = server.fleet_serve.env
    .map((e) => ({ ...e, key: e.key.trim() }))
    .filter((e) => e.enabled && e.key.length > 0)
    .map((e) => [e.key, e.value] as [string, string]);
  if (!isDefaultStack(server)) {
    const s3 = `http://localhost:${server.ports.s3}`;
    rows.push(["FLEET_S3_SOFTWARE_INSTALLERS_ENDPOINT_URL", s3]);
    rows.push(["FLEET_S3_CARVES_ENDPOINT_URL", s3]);
  }
  return rows;
}

/// `fleet prepare db --dev`, pointed at the server's MySQL for non-default
/// stacks (dev uses root/toor to create the schema).
export function prepareDbArgsFor(server: ServerProfile): string[] {
  const args = ["prepare", "db", "--dev"];
  if (!isDefaultStack(server)) {
    args.push(`--mysql_address=localhost:${server.ports.mysql}`);
    args.push(
      "--mysql_username=root",
      "--mysql_password=toor",
      "--mysql_database=fleet",
    );
  }
  return args;
}

/// Label for the fleet-serve row/chip: reflects premium/free (the flag that
/// meaningfully changes server behavior).
export function serveLabelFor(server: ServerProfile): string {
  return server.fleet_serve.premium
    ? "fleet serve --dev (premium)"
    : "fleet serve --dev (free)";
}

// --- presentation ---

/// Server accent key → CSS variable. Used by the switcher, status dots, and
/// per-server chrome so each server reads as a distinct color.
export function serverColorVar(color: string): string {
  switch (color) {
    case "purple":
      return "var(--core-fleet-purple)";
    case "blue":
      return "var(--core-vibrant-blue)";
    case "green":
    default:
      return "var(--core-fleet-green)";
  }
}

export type { ServerProfile, ServerPorts };
