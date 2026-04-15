import { IHost } from "interfaces/host";

import { getShipScale } from "./ships";

interface IShipEntity {
  host: IHost;
  offsetX: number;
  offsetY: number;
  currentX: number;
  currentY: number;
  driftPhase: number;
  driftSpeed: number;
  wanderSeed: number;
  wanderRadius: number;
  strayTimer: number;
  nextStrayTime: number;
  strayDuration: number;
  strayDistance: number;
  strayAngle: number;
  scale: number;
}

export interface IShipWorldPosition {
  x: number;
  y: number;
  heading: number;
  host: IHost;
  entity: IShipEntity;
}

const createShipEntity = (
  host: IHost,
  offsetX: number,
  offsetY: number
): IShipEntity => ({
  host,
  offsetX,
  offsetY,
  currentX: offsetX,
  currentY: offsetY,
  driftPhase: Math.random() * Math.PI * 2,
  driftSpeed: 0.4 + Math.random() * 0.8,
  wanderSeed: Math.random() * 100,
  wanderRadius: 6 + Math.random() * 12,
  strayTimer: Math.random() * 15,
  nextStrayTime: 8 + Math.random() * 25,
  strayDuration: 3 + Math.random() * 5,
  strayDistance: 15 + Math.random() * 40,
  strayAngle: Math.random() * Math.PI * 2,
  scale: getShipScale(host.platform),
});

export default class FleetFormation {
  name: string;

  hosts: IHost[];

  ships: IShipEntity[] = [];

  x = 0;

  y = 0;

  heading = 0;

  speed = 0;

  turnSeed = 0;

  turnStrength = 0;

  speedPhase = 0;

  speedVariation = 0;

  viewportWidth: number;

  viewportHeight: number;

  hasEntered = false;

  entryAlpha = 0;

  targetAlpha = 1;

  displayAlpha = 1;

  entryDelay: number;

  constructor(
    name: string,
    hosts: IHost[],
    viewportWidth: number,
    viewportHeight: number,
    entryDelay = 0
  ) {
    this.name = name;
    this.hosts = hosts;
    this.viewportWidth = viewportWidth;
    this.viewportHeight = viewportHeight;
    this.entryDelay = entryDelay;
    this.arrangeShips();
    this.initTurning();
    this.heading = Math.random() * Math.PI * 2;
    this.speed = 25 + Math.random() * 35;
    this.placeOffScreen(0);
  }

  private arrangeShips() {
    // Sort hosts so heavier ships tend to be wingmen; leader is simply first.
    const sorted = [...this.hosts];
    const count = sorted.length;
    this.ships = [];

    if (count <= 12) {
      const spacing = 48;
      sorted.forEach((host, idx) => {
        let ox: number;
        let oy: number;
        if (idx === 0) {
          ox = 0;
          oy = 0;
        } else {
          const wingPos = Math.ceil(idx / 2);
          const side = idx % 2 === 1 ? -1 : 1;
          ox = -wingPos * spacing;
          oy = side * wingPos * spacing * 0.65;
        }
        this.ships.push(createShipEntity(host, ox, oy));
      });
      return;
    }

    const cols = Math.ceil(Math.sqrt(count * 1.2));
    const spacing = 44;
    const totalRows = Math.ceil(count / cols);
    sorted.forEach((host, i) => {
      const col = i % cols;
      const row = Math.floor(i / cols);
      let ox = -col * spacing;
      const oyBase = (row - (totalRows - 1) / 2) * spacing;
      if (row % 2 === 1) ox -= spacing * 0.4;
      ox += (Math.random() - 0.5) * 10;
      const oy = oyBase + (Math.random() - 0.5) * 10;
      this.ships.push(createShipEntity(host, ox, oy));
    });
  }

  private initTurning() {
    this.turnSeed = Math.random() * 100;
    this.turnStrength = 0.03 + Math.random() * 0.06;
    this.speedPhase = Math.random() * Math.PI * 2;
    this.speedVariation = 5 + Math.random() * 10;
  }

  private getFormationRadius(): number {
    let maxDist = 0;
    this.ships.forEach((s) => {
      const d = Math.sqrt(s.offsetX * s.offsetX + s.offsetY * s.offsetY);
      if (d > maxDist) maxDist = d;
    });
    return Math.max(maxDist, 60);
  }

  private placeOffScreen(jitter: number) {
    const radius = this.getFormationRadius();
    const w = this.viewportWidth;
    const h = this.viewportHeight;
    // Pick a starting point just outside the viewport on the side opposite
    // the heading, so the formation flies in.
    const margin = radius + 120 + jitter;
    const dx = Math.cos(this.heading);
    const dy = Math.sin(this.heading);
    this.x = w / 2 - dx * (w / 2 + margin);
    this.y = h / 2 - dy * (h / 2 + margin);
  }

  private getTurnRate(time: number): number {
    const t = time + this.turnSeed;
    return (
      this.turnStrength *
      (Math.sin(t * 0.15) * 0.6 + Math.sin(t * 0.37 + 2.0) * 0.4)
    );
  }

  setViewport(width: number, height: number) {
    this.viewportWidth = width;
    this.viewportHeight = height;
  }

  setTargetAlpha(alpha: number) {
    this.targetAlpha = alpha;
  }

  update(dt: number, time: number) {
    // Delay entry so formations stagger in.
    if (this.entryDelay > 0) {
      this.entryDelay -= dt;
      return;
    }

    this.heading += this.getTurnRate(time) * dt;

    this.speedPhase += dt * 0.5;
    const currentSpeed =
      this.speed + Math.sin(this.speedPhase) * this.speedVariation;

    this.x += Math.cos(this.heading) * currentSpeed * dt;
    this.y += Math.sin(this.heading) * currentSpeed * dt;

    // Per-ship wander + stray.
    this.ships.forEach((ship) => {
      ship.driftPhase += dt * ship.driftSpeed;
      const wanderX =
        Math.sin(ship.driftPhase * 0.8 + ship.wanderSeed) * ship.wanderRadius +
        Math.sin(ship.driftPhase * 0.3 + ship.wanderSeed * 2.7) *
          ship.wanderRadius *
          0.5;
      const wanderY =
        Math.cos(ship.driftPhase * 0.6 + ship.wanderSeed * 1.3) *
          ship.wanderRadius +
        Math.cos(ship.driftPhase * 0.25 + ship.wanderSeed * 3.1) *
          ship.wanderRadius *
          0.4;

      let strayX = 0;
      let strayY = 0;
      ship.strayTimer += dt;
      if (ship.strayTimer > ship.nextStrayTime) {
        const strayProgress =
          (ship.strayTimer - ship.nextStrayTime) / ship.strayDuration;
        if (strayProgress < 1) {
          const strayAmount =
            Math.sin(strayProgress * Math.PI) * ship.strayDistance;
          strayX = Math.cos(ship.strayAngle) * strayAmount;
          strayY = Math.sin(ship.strayAngle) * strayAmount;
        } else {
          ship.strayTimer = 0;
          ship.nextStrayTime = 8 + Math.random() * 25;
          ship.strayDuration = 3 + Math.random() * 5;
          ship.strayDistance = 15 + Math.random() * 40;
          ship.strayAngle = Math.random() * Math.PI * 2;
        }
      }

      ship.currentX = ship.offsetX + wanderX + strayX;
      ship.currentY = ship.offsetY + wanderY + strayY;
    });

    // Entry fade.
    if (!this.hasEntered) {
      this.entryAlpha = Math.min(1, this.entryAlpha + dt * 0.6);
      if (this.entryAlpha >= 1) this.hasEntered = true;
    }

    // Smoothly approach target alpha (for fleet toggle).
    const alphaDelta = this.targetAlpha - this.displayAlpha;
    this.displayAlpha += alphaDelta * Math.min(1, dt * 3);

    // Recycle when fully off-screen.
    const radius = this.getFormationRadius();
    const margin = radius + 200;
    const outLeft = this.x < -margin;
    const outRight = this.x > this.viewportWidth + margin;
    const outTop = this.y < -margin;
    const outBottom = this.y > this.viewportHeight + margin;
    if (outLeft || outRight || outTop || outBottom) {
      this.heading = Math.random() * Math.PI * 2;
      this.speed = 25 + Math.random() * 35;
      this.initTurning();
      this.placeOffScreen(Math.random() * 400);
      this.hasEntered = false;
      this.entryAlpha = 0;
    }
  }

  getShipWorldPositions(): IShipWorldPosition[] {
    const cosH = Math.cos(this.heading);
    const sinH = Math.sin(this.heading);
    return this.ships.map((e) => ({
      x: this.x + e.currentX * cosH - e.currentY * sinH,
      y: this.y + e.currentX * sinH + e.currentY * cosH,
      heading: this.heading,
      host: e.host,
      entity: e,
    }));
  }

  getEffectiveAlpha(): number {
    return this.entryAlpha * this.displayAlpha;
  }

  getHostCount(): number {
    return this.hosts.length;
  }
}
