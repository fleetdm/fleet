import { promises as fs } from "node:fs";
import path from "node:path";
import { FLEET_REPO_PATH } from "./config";

function resolveSafe(relPath: string): string {
  const abs = path.resolve(FLEET_REPO_PATH, relPath);
  if (!abs.startsWith(path.resolve(FLEET_REPO_PATH) + path.sep)) {
    throw new Error(`refusing to read outside Fleet repo: ${relPath}`);
  }
  return abs;
}

export async function readFleetFile(relPath: string): Promise<string> {
  return fs.readFile(resolveSafe(relPath), "utf8");
}

export async function readFleetLines(
  relPath: string,
  start: number,
  end: number,
): Promise<{ content: string; startLine: number }> {
  const full = await readFleetFile(relPath);
  const lines = full.split("\n");
  const s = Math.max(1, start);
  const e = Math.min(lines.length, end);
  return { content: lines.slice(s - 1, e).join("\n"), startLine: s };
}

export async function fleetFileExists(relPath: string): Promise<boolean> {
  try {
    await fs.access(resolveSafe(relPath));
    return true;
  } catch {
    return false;
  }
}

export async function listFleetDir(relPath: string): Promise<string[]> {
  return fs.readdir(resolveSafe(relPath));
}
