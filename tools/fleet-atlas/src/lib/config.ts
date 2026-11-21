// Optional. Set FLEET_REPO_PATH (absolute path to a local fleetdm/fleet checkout)
// to turn file citations into clickable editor links. Unset in hosted deploys,
// in which case editorLink() returns a plain GitHub link instead.
export const FLEET_REPO_PATH = process.env.FLEET_REPO_PATH ?? "";

export const EDITOR_URL_SCHEME =
  process.env.EDITOR_URL_SCHEME ?? "vscode://file";

const FLEET_GITHUB_BLOB = "https://github.com/fleetdm/fleet/blob/main";

export function editorLink(relPath: string, line?: number) {
  if (!FLEET_REPO_PATH) {
    return line
      ? `${FLEET_GITHUB_BLOB}/${relPath}#L${line}`
      : `${FLEET_GITHUB_BLOB}/${relPath}`;
  }
  const abs = `${FLEET_REPO_PATH}/${relPath}`;
  return line ? `${EDITOR_URL_SCHEME}/${abs}:${line}` : `${EDITOR_URL_SCHEME}/${abs}`;
}
