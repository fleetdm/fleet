import { IFleetColor } from "./colors";

export type ShipType =
  | "fighter"
  | "cruiser"
  | "stealth"
  | "interceptor"
  | "scout"
  | "drone";

// Map osquery/fleet host platforms to ship silhouettes.
const PLATFORM_SHIPS: Record<string, ShipType> = {
  darwin: "fighter",
  windows: "cruiser",
  ubuntu: "stealth",
  rhel: "stealth",
  amzn: "stealth",
  linux: "stealth",
  chrome: "drone",
  ios: "interceptor",
  ipados: "interceptor",
  android: "scout",
};

export const getShipType = (platform: string | null | undefined): ShipType => {
  if (!platform) return "scout";
  return PLATFORM_SHIPS[platform] || "scout";
};

export const getShipScale = (platform: string | null | undefined): number => {
  switch (getShipType(platform)) {
    case "cruiser":
      return 0.85;
    case "fighter":
      return 0.7;
    case "stealth":
      return 0.75;
    case "interceptor":
      return 0.6;
    case "scout":
      return 0.65;
    case "drone":
      return 0.45;
    default:
      return 0.7;
  }
};

export const getShipHitRadius = (platform: string | null | undefined): number =>
  getShipScale(platform) * 20;

interface IDrawableHost {
  platform: string | null | undefined;
  status?: string;
  issues?: {
    total_issues_count?: number;
    critical_vulnerabilities_count?: number;
  } | null;
}

const drawEngine = (
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  color: string,
  isOnline: boolean,
  time: number,
  size = 1
) => {
  if (!isOnline) return;
  const flicker = 0.7 + 0.3 * Math.sin(time * 8 + x);
  const len = (8 + 4 * Math.sin(time * 12)) * size;

  ctx.save();
  ctx.globalAlpha *= flicker;

  const grad = ctx.createLinearGradient(x, y, x - len, y);
  grad.addColorStop(0, color);
  grad.addColorStop(0.3, color);
  grad.addColorStop(1, "transparent");
  ctx.fillStyle = grad;
  ctx.beginPath();
  ctx.moveTo(x, y - 2 * size);
  ctx.lineTo(x - len, y);
  ctx.lineTo(x, y + 2 * size);
  ctx.fill();

  ctx.fillStyle = color;
  ctx.globalAlpha *= 0.3;
  ctx.beginPath();
  ctx.arc(x, y, 4 * size, 0, Math.PI * 2);
  ctx.fill();

  ctx.restore();
};

const drawFighter = (
  ctx: CanvasRenderingContext2D,
  colors: IFleetColor,
  isOnline: boolean,
  time: number
) => {
  ctx.strokeStyle = colors.primary;
  ctx.fillStyle = colors.secondary;
  ctx.lineWidth = 1.5;

  ctx.beginPath();
  ctx.moveTo(18, 0);
  ctx.lineTo(-6, -10);
  ctx.lineTo(-12, -8);
  ctx.lineTo(-8, 0);
  ctx.lineTo(-12, 8);
  ctx.lineTo(-6, 10);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  ctx.save();
  ctx.fillStyle = colors.primary;
  ctx.globalAlpha *= 0.6;
  ctx.beginPath();
  ctx.ellipse(6, 0, 5, 2.5, 0, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();

  ctx.strokeStyle = colors.primary;
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(4, -3);
  ctx.lineTo(-6, -10);
  ctx.moveTo(4, 3);
  ctx.lineTo(-6, 10);
  ctx.stroke();

  drawEngine(ctx, -12, -5, colors.engine, isOnline, time);
  drawEngine(ctx, -12, 5, colors.engine, isOnline, time);
};

const drawCruiser = (
  ctx: CanvasRenderingContext2D,
  colors: IFleetColor,
  isOnline: boolean,
  time: number
) => {
  ctx.strokeStyle = colors.primary;
  ctx.fillStyle = colors.secondary;
  ctx.lineWidth = 1.5;

  ctx.beginPath();
  ctx.moveTo(16, 0);
  ctx.lineTo(8, -6);
  ctx.lineTo(-4, -8);
  ctx.lineTo(-14, -12);
  ctx.lineTo(-16, -10);
  ctx.lineTo(-14, 0);
  ctx.lineTo(-16, 10);
  ctx.lineTo(-14, 12);
  ctx.lineTo(-4, 8);
  ctx.lineTo(8, 6);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  ctx.save();
  ctx.fillStyle = colors.primary;
  ctx.globalAlpha *= 0.5;
  ctx.fillRect(2, -3, 8, 6);
  ctx.restore();

  ctx.strokeStyle = colors.primary;
  ctx.lineWidth = 0.8;
  ctx.beginPath();
  ctx.moveTo(-4, -8);
  ctx.lineTo(-4, -14);
  ctx.lineTo(-10, -14);
  ctx.lineTo(-14, -12);
  ctx.moveTo(-4, 8);
  ctx.lineTo(-4, 14);
  ctx.lineTo(-10, 14);
  ctx.lineTo(-14, 12);
  ctx.stroke();

  drawEngine(ctx, -16, -10, colors.engine, isOnline, time, 0.8);
  drawEngine(ctx, -14, 0, colors.engine, isOnline, time, 1.2);
  drawEngine(ctx, -16, 10, colors.engine, isOnline, time, 0.8);
};

const drawStealth = (
  ctx: CanvasRenderingContext2D,
  colors: IFleetColor,
  isOnline: boolean,
  time: number
) => {
  ctx.strokeStyle = colors.primary;
  ctx.fillStyle = colors.secondary;
  ctx.lineWidth = 1.5;

  ctx.beginPath();
  ctx.moveTo(16, 0);
  ctx.lineTo(0, -12);
  ctx.lineTo(-12, -10);
  ctx.lineTo(-10, 0);
  ctx.lineTo(-12, 10);
  ctx.lineTo(0, 12);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  ctx.strokeStyle = colors.primary;
  ctx.lineWidth = 0.8;
  ctx.beginPath();
  ctx.moveTo(16, 0);
  ctx.lineTo(-2, 0);
  ctx.moveTo(6, -5);
  ctx.lineTo(-6, -8);
  ctx.moveTo(6, 5);
  ctx.lineTo(-6, 8);
  ctx.stroke();

  drawEngine(ctx, -12, -5, colors.engine, isOnline, time, 0.7);
  drawEngine(ctx, -12, 5, colors.engine, isOnline, time, 0.7);
};

const drawInterceptor = (
  ctx: CanvasRenderingContext2D,
  colors: IFleetColor,
  isOnline: boolean,
  time: number
) => {
  ctx.strokeStyle = colors.primary;
  ctx.fillStyle = colors.secondary;
  ctx.lineWidth = 1.2;

  ctx.beginPath();
  ctx.moveTo(12, 0);
  ctx.lineTo(2, -6);
  ctx.lineTo(-8, -4);
  ctx.lineTo(-6, 0);
  ctx.lineTo(-8, 4);
  ctx.lineTo(2, 6);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  ctx.fillStyle = colors.primary;
  ctx.beginPath();
  ctx.arc(4, 0, 2, 0, Math.PI * 2);
  ctx.fill();

  drawEngine(ctx, -8, 0, colors.engine, isOnline, time, 0.6);
};

const drawScout = (
  ctx: CanvasRenderingContext2D,
  colors: IFleetColor,
  isOnline: boolean,
  time: number
) => {
  ctx.strokeStyle = colors.primary;
  ctx.fillStyle = colors.secondary;
  ctx.lineWidth = 1.2;

  ctx.beginPath();
  ctx.moveTo(10, 0);
  ctx.lineTo(4, -7);
  ctx.lineTo(-6, -6);
  ctx.lineTo(-8, 0);
  ctx.lineTo(-6, 6);
  ctx.lineTo(4, 7);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  ctx.beginPath();
  ctx.moveTo(4, -7);
  ctx.lineTo(6, -11);
  ctx.stroke();
  ctx.fillStyle = colors.primary;
  ctx.beginPath();
  ctx.arc(6, -11, 1, 0, Math.PI * 2);
  ctx.fill();

  drawEngine(ctx, -8, 0, colors.engine, isOnline, time, 0.5);
};

const drawDrone = (
  ctx: CanvasRenderingContext2D,
  colors: IFleetColor,
  isOnline: boolean,
  time: number
) => {
  ctx.strokeStyle = colors.primary;
  ctx.fillStyle = colors.secondary;
  ctx.lineWidth = 1;

  ctx.beginPath();
  ctx.moveTo(8, 0);
  ctx.lineTo(4, -6);
  ctx.lineTo(-4, -6);
  ctx.lineTo(-8, 0);
  ctx.lineTo(-4, 6);
  ctx.lineTo(4, 6);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();

  ctx.save();
  const pulse = 0.5 + 0.5 * Math.sin(time * 4);
  ctx.fillStyle = colors.primary;
  ctx.globalAlpha *= pulse;
  ctx.beginPath();
  ctx.arc(0, 0, 2, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();

  drawEngine(ctx, -8, 0, colors.engine, isOnline, time, 0.4);
};

const drawIssueIndicators = (
  ctx: CanvasRenderingContext2D,
  totalIssues: number,
  criticalVulns: number,
  time: number
) => {
  if (criticalVulns > 0) {
    const intensity = Math.min(criticalVulns / 20, 1);
    const pulse = 0.5 + 0.5 * Math.sin(time * 6);

    ctx.fillStyle = `rgba(255, 50, 50, ${0.3 * intensity * pulse})`;
    ctx.beginPath();
    ctx.arc(0, 0, 20 + intensity * 10, 0, Math.PI * 2);
    ctx.fill();

    const sparkCount = Math.min(Math.floor(criticalVulns / 5) + 1, 4);
    for (let i = 0; i < sparkCount; i += 1) {
      const angle = time * 3 + (i * Math.PI * 2) / sparkCount;
      const dist = 12 + 4 * Math.sin(time * 5 + i);
      const sx = Math.cos(angle) * dist;
      const sy = Math.sin(angle) * dist;
      ctx.fillStyle = `rgba(255, ${100 + Math.random() * 100}, 50, ${
        0.6 * pulse
      })`;
      ctx.beginPath();
      ctx.arc(sx, sy, 1 + Math.random(), 0, Math.PI * 2);
      ctx.fill();
    }
  }

  if (totalIssues > criticalVulns && totalIssues > 0) {
    const blink = Math.sin(time * 4) > 0.3 ? 1 : 0;
    if (blink) {
      ctx.fillStyle = "rgba(255, 255, 0, 0.7)";
      ctx.beginPath();
      ctx.moveTo(-18, -4);
      ctx.lineTo(-15, -9);
      ctx.lineTo(-12, -4);
      ctx.closePath();
      ctx.fill();

      ctx.fillStyle = "#000";
      ctx.font = "5px sans-serif";
      ctx.textAlign = "center";
      ctx.fillText("!", -15, -5);
    }
  }
};

export const drawShip = (
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  host: IDrawableHost,
  colors: IFleetColor,
  time: number,
  heading = 0
) => {
  const type = getShipType(host.platform);
  const isOnline = host.status === "online";
  const scale = getShipScale(host.platform);
  const issues = host.issues?.total_issues_count || 0;
  const critical = host.issues?.critical_vulnerabilities_count || 0;

  ctx.save();
  ctx.translate(x, y);
  ctx.rotate(heading);
  ctx.scale(scale, scale);

  if (!isOnline) {
    ctx.globalAlpha *= 0.45;
  }

  switch (type) {
    case "fighter":
      drawFighter(ctx, colors, isOnline, time);
      break;
    case "cruiser":
      drawCruiser(ctx, colors, isOnline, time);
      break;
    case "stealth":
      drawStealth(ctx, colors, isOnline, time);
      break;
    case "interceptor":
      drawInterceptor(ctx, colors, isOnline, time);
      break;
    case "drone":
      drawDrone(ctx, colors, isOnline, time);
      break;
    case "scout":
    default:
      drawScout(ctx, colors, isOnline, time);
      break;
  }

  if (issues > 0 && isOnline) {
    drawIssueIndicators(ctx, issues, critical, time);
  }

  ctx.restore();
};
