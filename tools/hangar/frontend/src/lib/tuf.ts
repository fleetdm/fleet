// Local TUF helpers. The tab drives tools/tuf/test/main.sh from a saved config;
// process/log ids mirror the Go side (internal/tuf) so the frontend can address
// the build run and tail its output.
import type { NgrokRunningTunnel } from "./ipc";

export const TUF_PORT = 8081;
export const FLEET_PORT = 8080;
export const TUF_PROC_ID = "tuf:build"; // must match internal/tuf.ProcID
export const TUF_CHANNEL = "tuf-build"; // must match internal/tuf.LogChannel

// Platform keys must match internal/tuf.Platforms.
export const TUF_PLATFORMS: { key: string; label: string }[] = [
  { key: "macos", label: "macOS (.pkg)" },
  { key: "windows", label: "Windows (.msi)" },
  { key: "windows-arm64", label: "Windows ARM64 (.msi)" },
  { key: "linux", label: "Linux (.deb + .rpm)" },
  { key: "linux-arm64", label: "Linux ARM64 (.deb + .rpm)" },
];

// tunnelForPort finds a running ngrok tunnel forwarding to a given local port
// (Addr is the local target, e.g. "http://localhost:8081").
export function tunnelForPort(
  tunnels: NgrokRunningTunnel[],
  port: number,
): NgrokRunningTunnel | undefined {
  return tunnels.find((t) => {
    const addr = t.addr || "";
    return addr.endsWith(`:${port}`) || addr.endsWith(`:${port}/`);
  });
}

// domainOf strips the scheme (and trailing slash) from a URL for the ngrok
// --domain flag, e.g. https://tuf.andrey.ngrok.app → tuf.andrey.ngrok.app.
export function domainOf(url: string): string {
  return url.replace(/^https?:\/\//, "").replace(/\/+$/, "");
}
