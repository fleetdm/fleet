import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import ReactDOM from "react-dom";
import classnames from "classnames";

import { IHost } from "interfaces/host";
import hostsAPI from "services/entities/hosts";

import {
  buildFleetColorMap,
  getFleetColor,
  IFleetColor,
  NO_FLEET_LABEL,
} from "./colors";
import FleetFormation from "./formations";
import { drawShip, getShipHitRadius } from "./ships";
import Starfield from "./starfield";

const baseClass = "fleet-arcade";

// Max pages to fetch so a huge fleet doesn't block the animation for too long.
const MAX_PAGES = 20;
const HOSTS_PER_PAGE = 100;

interface IFleetArcadeProps {
  onClose: () => void;
}

type LoadState = "loading" | "ready" | "error";

const groupByFleet = (hosts: IHost[]): Record<string, IHost[]> => {
  const groups: Record<string, IHost[]> = {};
  hosts.forEach((h) => {
    const key = h.team_name || NO_FLEET_LABEL;
    if (!groups[key]) groups[key] = [];
    groups[key].push(h);
  });
  return groups;
};

const FleetArcade = ({ onClose }: IFleetArcadeProps): JSX.Element => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);

  const formationsRef = useRef<FleetFormation[]>([]);
  const starfieldRef = useRef<Starfield | null>(null);
  const colorMapRef = useRef<Record<string, IFleetColor>>({});
  const visibilityRef = useRef<Record<string, boolean>>({});
  const hoveredHostRef = useRef<IHost | null>(null);

  const [loadState, setLoadState] = useState<LoadState>("loading");
  const [loadMessage, setLoadMessage] = useState("Initializing fleet...");
  const [fleetNames, setFleetNames] = useState<string[]>([]);
  const [fleetVisibility, setFleetVisibility] = useState<
    Record<string, boolean>
  >({});
  const [selectedHost, setSelectedHost] = useState<IHost | null>(null);

  // Keep the visibility ref in sync with state so the render loop can read it
  // without retriggering on every toggle.
  useEffect(() => {
    visibilityRef.current = fleetVisibility;
    formationsRef.current.forEach((f) => {
      const visible = fleetVisibility[f.name] !== false;
      f.setTargetAlpha(visible ? 1 : 0.06);
    });
  }, [fleetVisibility]);

  // Fetch all hosts (paginated) using the current session's credentials.
  useEffect(() => {
    let cancelled = false;

    const fetchAll = async () => {
      const collected: IHost[] = [];
      try {
        for (let page = 0; page < MAX_PAGES; page += 1) {
          if (cancelled) return;
          setLoadMessage(
            `Scanning sector ${page + 1}... (${collected.length} vessels)`
          );
          // eslint-disable-next-line no-await-in-loop
          const res = await hostsAPI.loadHosts({
            page,
            perPage: HOSTS_PER_PAGE,
          });
          const hosts = res?.hosts || [];
          collected.push(...hosts);
          if (hosts.length < HOSTS_PER_PAGE) break;
        }

        if (cancelled) return;

        const names = Array.from(
          new Set(collected.map((h) => h.team_name || NO_FLEET_LABEL))
        );
        colorMapRef.current = buildFleetColorMap(names);
        setFleetNames(names);

        const initialVisibility: Record<string, boolean> = {};
        names.forEach((n) => {
          initialVisibility[n] = true;
        });
        setFleetVisibility(initialVisibility);

        const canvas = canvasRef.current;
        const width = canvas?.clientWidth || window.innerWidth;
        const height = canvas?.clientHeight || window.innerHeight;

        const groups = groupByFleet(collected);
        const formationEntries = Object.entries(groups);
        formationsRef.current = formationEntries.map(
          ([name, hosts], idx) =>
            new FleetFormation(name, hosts, width, height, idx * 0.8)
        );

        if (collected.length === 0) {
          setLoadMessage("No vessels detected in this sector.");
        }
        setLoadState("ready");
      } catch (err) {
        console.error("FleetArcade: failed to load hosts", err);
        if (!cancelled) {
          setLoadState("error");
          setLoadMessage("Long-range sensors offline. Press ESC to retreat.");
        }
      }
    };

    fetchAll();
    return () => {
      cancelled = true;
    };
  }, []);

  // Main render loop. We use refs for all animation state so re-renders don't
  // tear the canvas.
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return undefined;
    const ctx = canvas.getContext("2d");
    if (!ctx) return undefined;

    let animationId = 0;
    let lastFrame = performance.now();
    let gameTime = 0;

    const resize = () => {
      const dpr = window.devicePixelRatio || 1;
      const width = window.innerWidth;
      const height = window.innerHeight;
      canvas.width = width * dpr;
      canvas.height = height * dpr;
      canvas.style.width = `${width}px`;
      canvas.style.height = `${height}px`;
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
      if (!starfieldRef.current) {
        starfieldRef.current = new Starfield(width, height);
      } else {
        starfieldRef.current.resize(width, height);
      }
      formationsRef.current.forEach((f) => f.setViewport(width, height));
    };

    resize();
    window.addEventListener("resize", resize);

    const frame = (now: number) => {
      const dtRaw = (now - lastFrame) / 1000;
      const dt = Math.min(dtRaw, 0.1);
      lastFrame = now;
      gameTime += dt;

      starfieldRef.current?.update(dt);
      formationsRef.current.forEach((f) => f.update(dt, gameTime));

      const width = window.innerWidth;
      const height = window.innerHeight;
      ctx.fillStyle = "#000408";
      ctx.fillRect(0, 0, width, height);

      starfieldRef.current?.draw(ctx);

      // Draw each formation.
      formationsRef.current.forEach((formation) => {
        const alpha = formation.getEffectiveAlpha();
        if (alpha <= 0.001) return;
        const color = getFleetColor(colorMapRef.current, formation.name);
        const positions = formation.getShipWorldPositions();
        if (positions.length === 0) return;

        ctx.save();
        ctx.globalAlpha = alpha;

        // Formation label above the topmost ship.
        const top = positions.reduce((acc, p) => (p.y < acc.y ? p : acc));
        ctx.fillStyle = color.primary;
        ctx.shadowColor = color.primary;
        ctx.shadowBlur = 8;
        ctx.font = "bold 11px 'Press Start 2P', 'Courier New', monospace";
        ctx.textAlign = "center";
        const label = `${formation.name.toUpperCase()} · ${formation.getHostCount()}`;
        ctx.fillText(label, top.x, top.y - 22);
        ctx.shadowBlur = 0;

        positions.forEach((p) => {
          drawShip(ctx, p.x, p.y, p.host, color, gameTime, p.heading);
        });

        ctx.restore();
      });

      animationId = requestAnimationFrame(frame);
    };

    animationId = requestAnimationFrame(frame);

    return () => {
      cancelAnimationFrame(animationId);
      window.removeEventListener("resize", resize);
    };
  }, []);

  // Hover + click handling to pick ships.
  const handlePointerMove = useCallback((e: React.PointerEvent) => {
    const x = e.clientX;
    const y = e.clientY;
    let closest: { host: IHost; dist: number } | null = null;
    formationsRef.current.forEach((formation) => {
      if (visibilityRef.current[formation.name] === false) return;
      formation.getShipWorldPositions().forEach((p) => {
        const dx = p.x - x;
        const dy = p.y - y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        const r = getShipHitRadius(p.host.platform);
        if (dist < r && (!closest || dist < closest.dist)) {
          closest = { host: p.host, dist };
        }
      });
    });
    hoveredHostRef.current = closest ? closest.host : null;
    if (canvasRef.current) {
      canvasRef.current.style.cursor = closest ? "pointer" : "crosshair";
    }
  }, []);

  const handleClick = useCallback(() => {
    if (hoveredHostRef.current) {
      setSelectedHost(hoveredHostRef.current);
    }
  }, []);

  // ESC to close (and close the detail panel first if open).
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        if (selectedHost) {
          setSelectedHost(null);
        } else {
          onClose();
        }
      }
    };
    window.addEventListener("keydown", handleKeyDown, true);
    return () => window.removeEventListener("keydown", handleKeyDown, true);
  }, [onClose, selectedHost]);

  const totalHosts = useMemo(
    () => formationsRef.current.reduce((acc, f) => acc + f.getHostCount(), 0),
    // Recompute whenever the fleet list changes.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [fleetNames]
  );

  const toggleFleet = (name: string) => {
    setFleetVisibility((prev) => ({ ...prev, [name]: prev[name] === false }));
  };

  const sortedFleetNames = useMemo(
    () =>
      [...fleetNames].sort((a, b) => {
        if (a === NO_FLEET_LABEL) return 1;
        if (b === NO_FLEET_LABEL) return -1;
        return a.toLowerCase().localeCompare(b.toLowerCase());
      }),
    [fleetNames]
  );

  const overlay = (
    <div className={baseClass} ref={containerRef}>
      <canvas
        ref={canvasRef}
        className={`${baseClass}__canvas`}
        onPointerMove={handlePointerMove}
        onClick={handleClick}
      />
      <div className={`${baseClass}__crt`} aria-hidden="true" />

      <div className={`${baseClass}__hud`}>
        <div className={`${baseClass}__title`}>FLEET ARCADE</div>
        <div className={`${baseClass}__subtitle`}>
          {loadState !== "ready" || totalHosts === 0
            ? loadMessage
            : `${totalHosts} VESSELS · ${sortedFleetNames.length} FLEETS`}
        </div>
      </div>

      {loadState === "ready" && sortedFleetNames.length > 0 && (
        <div className={`${baseClass}__legend`}>
          {sortedFleetNames.map((name) => {
            const color = getFleetColor(colorMapRef.current, name);
            const active = fleetVisibility[name] !== false;
            return (
              <button
                type="button"
                key={name}
                className={classnames(`${baseClass}__legend-item`, {
                  [`${baseClass}__legend-item--dim`]: !active,
                })}
                onClick={() => toggleFleet(name)}
                style={{ color: color.primary }}
              >
                <span
                  className={`${baseClass}__legend-swatch`}
                  style={{ background: color.primary }}
                />
                {name}
              </button>
            );
          })}
        </div>
      )}

      <button
        type="button"
        className={`${baseClass}__exit`}
        onClick={onClose}
        aria-label="Exit arcade"
      >
        [ESC] EXIT
      </button>

      {selectedHost && (
        <>
          <div
            className={`${baseClass}__panel-backdrop`}
            onClick={() => setSelectedHost(null)}
            role="presentation"
          />
          <div
            className={`${baseClass}__panel`}
            style={{
              borderColor: getFleetColor(
                colorMapRef.current,
                selectedHost.team_name
              ).primary,
            }}
          >
            <div className={`${baseClass}__panel-header`}>
              <span>VESSEL LOG</span>
              <button
                type="button"
                className={`${baseClass}__panel-close`}
                onClick={() => setSelectedHost(null)}
              >
                [X]
              </button>
            </div>
            <h3 className={`${baseClass}__panel-name`}>
              {selectedHost.display_name || selectedHost.hostname}
            </h3>
            <dl className={`${baseClass}__panel-stats`}>
              <dt>FLEET</dt>
              <dd>{selectedHost.team_name || NO_FLEET_LABEL}</dd>
              <dt>CLASS</dt>
              <dd>{selectedHost.platform || "unknown"}</dd>
              <dt>OS</dt>
              <dd>{selectedHost.os_version || "unknown"}</dd>
              <dt>STATUS</dt>
              <dd>{(selectedHost.status || "unknown").toUpperCase()}</dd>
              <dt>ISSUES</dt>
              <dd>{selectedHost.issues?.total_issues_count ?? 0}</dd>
              <dt>UUID</dt>
              <dd className={`${baseClass}__panel-mono`}>
                {selectedHost.uuid || "—"}
              </dd>
            </dl>
            <a
              className={`${baseClass}__panel-link`}
              href={`/hosts/${selectedHost.id}`}
            >
              OPEN HOST DETAILS →
            </a>
          </div>
        </>
      )}
    </div>
  );

  return ReactDOM.createPortal(overlay, document.body);
};

export default FleetArcade;
