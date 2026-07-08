/**
 * TerminalPage — full-screen web terminal for a Fleet-managed host.
 * Opens in a new browser tab when the user picks Actions → Connect.
 *
 * Flow:
 *  1. POST /api/v1/fleet/hosts/{id}/terminal  →  { session_id }
 *  2. WebSocket  ws[s]://<host>/api/v1/fleet/hosts/{id}/terminal/{session_id}/ws
 *  3. First WS message: { "token": "<fleet-session-token>" }
 *  4. Relay xterm.js keystrokes as  { "type":"input",  "data":"<base64>" }
 *     Receive PTY output as          { "type":"output", "data":"<base64>" }
 *     Send resize events as          { "type":"resize", "cols":N, "rows":N }
 *
 * Install deps once:  yarn add xterm @xterm/addon-fit
 */

import React, { useEffect, useRef, useState } from "react";

// eslint-disable-next-line import/no-extraneous-dependencies
import { Terminal } from "@xterm/xterm";
// eslint-disable-next-line import/no-extraneous-dependencies
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

import hostAPI from "services/entities/hosts";
import authToken from "utilities/auth_token";

type ConnState = "connecting" | "connected" | "error" | "closed";

interface ITerminalPageProps {
  params: { host_id: string };
}

// styles is declared before the component to satisfy the no-use-before-define
// lint rule; React CSSProperties lets TypeScript check the values.
const styles: Record<string, React.CSSProperties> = {
  page: {
    position: "fixed",
    inset: 0,
    display: "flex",
    flexDirection: "column",
    background: "#1e1e1e",
    color: "#d4d4d4",
    fontFamily: '"Courier New", Courier, monospace',
    zIndex: 9999,
  },
  header: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    padding: "8px 16px",
    background: "#252526",
    borderBottom: "1px solid #333",
    fontSize: 13,
    flexShrink: 0,
  },
  headerTitle: { color: "#ccc" },
  headerStatus: {
    display: "flex",
    alignItems: "center",
    gap: 6,
    fontSize: 12,
    color: "#888",
    textTransform: "capitalize",
  },
  dot: {
    width: 8,
    height: 8,
    borderRadius: "50%",
    background: "#4ec94e",
    display: "inline-block",
  },
  banner: {
    padding: "10px 16px",
    background: "#2d2d2d",
    borderBottom: "1px solid #444",
    fontSize: 13,
    color: "#ccc",
    flexShrink: 0,
  },
  bannerError: {
    background: "#3a1a1a",
    color: "#f48771",
    borderColor: "#6b2020",
  },
  closeBtn: {
    marginLeft: 8,
    padding: "2px 10px",
    background: "#444",
    color: "#ccc",
    border: "1px solid #666",
    borderRadius: 3,
    cursor: "pointer",
    fontSize: 12,
  },
  termContainer: {
    flex: 1,
    padding: "8px",
    overflow: "hidden",
  },
};

const TerminalPage = ({ params }: ITerminalPageProps) => {
  const hostId = parseInt(params.host_id, 10);

  const termRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const wasConnectedRef = useRef(false);

  const [connState, setConnState] = useState<ConnState>("connecting");
  const [errorMsg, setErrorMsg] = useState("");
  const [hostName, setHostName] = useState<string>(`Host ${hostId}`);

  useEffect(() => {
    let cancelled = false;

    const init = async () => {
      // Create xterm.js instance and mount it immediately so it sizes itself.
      const term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: '"Courier New", Courier, monospace',
        theme: { background: "#1e1e1e", foreground: "#d4d4d4" },
      });
      const fitAddon = new FitAddon();
      term.loadAddon(fitAddon);
      xtermRef.current = term;
      fitAddonRef.current = fitAddon;

      if (termRef.current) {
        term.open(termRef.current);
        fitAddon.fit();
      }

      // 1. Create terminal session on the Fleet server.
      let sessionId: string;
      try {
        const resp = await hostAPI.createTerminalSession(hostId);
        sessionId = resp.session_id;
      } catch (err: unknown) {
        if (!cancelled) {
          setConnState("error");
          setErrorMsg(
            err instanceof Error
              ? err.message
              : "Failed to create terminal session"
          );
        }
        return;
      }

      if (cancelled) return;

      // 2. Open WebSocket — protocol follows page protocol automatically.
      const proto = window.location.protocol === "https:" ? "wss" : "ws";
      const wsURL = `${proto}://${window.location.host}/api/v1/fleet/hosts/${hostId}/terminal/${sessionId}/ws`;
      const ws = new WebSocket(wsURL);
      wsRef.current = ws;

      ws.onopen = () => {
        if (cancelled) {
          ws.close();
          return;
        }
        // 3. Authenticate using the Fleet session token (cookie-backed).
        const token = authToken.get() ?? "";
        ws.send(JSON.stringify({ token }));
        wasConnectedRef.current = true;
        setConnState("connected");
        term.focus();
      };

      ws.onmessage = (event: MessageEvent) => {
        try {
          const msg = JSON.parse(event.data as string) as {
            type: string;
            data?: string;
          };
          if (msg.type === "output" && msg.data) {
            term.write(atob(msg.data));
          } else if (msg.type === "error" && msg.data) {
            if (!cancelled) {
              setConnState("error");
              setErrorMsg(msg.data);
            }
          }
        } catch {
          /* ignore malformed frames */
        }
      };

      // Don't overwrite an "error" state set by onmessage.
      // Close the tab when the shell exits normally (logout / exit).
      ws.onclose = () => {
        if (!cancelled) {
          setConnState((prev) => (prev === "error" ? prev : "closed"));
          if (wasConnectedRef.current) {
            // Shell exited — close the tab like SSH does.
            window.close();
          }
        }
      };

      ws.onerror = () => {
        if (!cancelled) {
          setConnState("error");
          setErrorMsg("WebSocket connection failed");
        }
      };

      // 4. Forward keystrokes.
      term.onData((data: string) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: "input", data: btoa(data) }));
        }
      });

      // 5. Forward resize events.
      const sendResize = () => {
        fitAddon.fit();
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(
            JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows })
          );
        }
      };
      const ro = new ResizeObserver(sendResize);
      if (termRef.current) ro.observe(termRef.current);
      window.addEventListener("resize", sendResize);

      return () => {
        ro.disconnect();
        window.removeEventListener("resize", sendResize);
      };
    };

    init();

    return () => {
      cancelled = true;
      wsRef.current?.close();
      xtermRef.current?.dispose();
    };
  }, [hostId]);

  // Auto-focus the terminal once it becomes visible.
  // term.focus() in ws.onopen fires before React re-renders the div to
  // display:block, so xterm ignores it.  This effect runs after the render.
  useEffect(() => {
    if (connState === "connected") {
      const t = setTimeout(() => xtermRef.current?.focus(), 30);
      return () => clearTimeout(t);
    }
    return undefined;
  }, [connState]);

  // Fetch host name for the page title.
  useEffect(() => {
    hostAPI
      .loadHostDetails(hostId)
      .then((resp) => {
        if (resp?.host?.display_name) {
          setHostName(resp.host.display_name);
          document.title = `Terminal — ${resp.host.display_name}`;
        }
      })
      .catch(() => {
        /* non-critical */
      });
  }, [hostId]);

  const statusBanner = () => {
    if (connState === "connecting") {
      return (
        <div style={styles.banner}>
          Connecting to <strong>{hostName}</strong>…
        </div>
      );
    }
    if (connState === "error") {
      return (
        <div style={{ ...styles.banner, ...styles.bannerError }}>
          <strong>Connection failed:</strong> {errorMsg} — make sure the host is
          online and the Fleet agent (orbit) is running.
        </div>
      );
    }
    if (connState === "closed") {
      return (
        <div style={styles.banner}>
          Session closed.{" "}
          <button
            onClick={() => window.close()}
            style={styles.closeBtn}
            type="button"
          >
            Close tab
          </button>
        </div>
      );
    }
    return null;
  };

  return (
    <div style={styles.page}>
      <div style={styles.header}>
        <span style={styles.headerTitle}>
          Fleet Terminal — <strong>{hostName}</strong>
        </span>
        <span style={styles.headerStatus}>
          {connState === "connected" && (
            <span style={styles.dot} title="Connected" />
          )}
          {connState}
        </span>
      </div>

      {statusBanner()}

      {/* xterm.js container — always mounted; hidden until connected */}
      <div
        ref={termRef}
        style={{
          ...styles.termContainer,
          display: connState === "connected" ? "block" : "none",
        }}
      />
    </div>
  );
};

export default TerminalPage;
