interface IStar {
  x: number;
  y: number;
  size: number;
  opacity: number;
  twinkle: number;
  twinkleSpeed: number;
  tint: string | null;
}

interface INebula {
  x: number;
  y: number;
  radius: number;
  hue: number;
  alpha: number;
}

const LAYERS = [
  { count: 200, sizeMin: 0.3, sizeMax: 0.8, opacity: 0.4 },
  { count: 120, sizeMin: 0.5, sizeMax: 1.2, opacity: 0.6 },
  { count: 60, sizeMin: 1.0, sizeMax: 2.0, opacity: 0.85 },
];

const NEBULA_HUES = [200, 280, 320, 180, 260];

const rand = (min: number, max: number) => min + Math.random() * (max - min);

export default class Starfield {
  private stars: IStar[] = [];

  private nebulae: INebula[] = [];

  private time = 0;

  private width = 0;

  private height = 0;

  constructor(width: number, height: number) {
    this.resize(width, height);
  }

  resize(width: number, height: number) {
    this.width = width;
    this.height = height;
    this.stars = [];
    LAYERS.forEach((layer) => {
      for (let i = 0; i < layer.count; i += 1) {
        const tinted = Math.random() < 0.15;
        const hue = Math.random() < 0.5 ? 200 : 30;
        this.stars.push({
          x: Math.random() * width,
          y: Math.random() * height,
          size: rand(layer.sizeMin, layer.sizeMax),
          opacity: layer.opacity,
          twinkle: Math.random() * Math.PI * 2,
          twinkleSpeed: 0.5 + Math.random() * 2,
          tint: tinted ? `hsl(${hue}, 80%, 80%)` : null,
        });
      }
    });

    this.nebulae = [];
    for (let i = 0; i < 5; i += 1) {
      this.nebulae.push({
        x: Math.random() * width,
        y: Math.random() * height,
        radius: rand(width * 0.15, width * 0.4),
        hue: NEBULA_HUES[i % NEBULA_HUES.length],
        alpha: rand(0.02, 0.06),
      });
    }
  }

  update(dt: number) {
    this.time += dt;
  }

  draw(ctx: CanvasRenderingContext2D) {
    // Nebulae first (behind stars).
    this.nebulae.forEach((n) => {
      const grad = ctx.createRadialGradient(n.x, n.y, 0, n.x, n.y, n.radius);
      grad.addColorStop(0, `hsla(${n.hue}, 80%, 50%, ${n.alpha})`);
      grad.addColorStop(1, "transparent");
      ctx.fillStyle = grad;
      ctx.fillRect(0, 0, this.width, this.height);
    });

    this.stars.forEach((s) => {
      const tw = 0.6 + 0.4 * Math.sin(this.time * s.twinkleSpeed + s.twinkle);
      const alpha = s.opacity * tw;
      ctx.fillStyle = s.tint || `rgba(255, 255, 255, ${alpha})`;
      if (s.tint) ctx.globalAlpha = alpha;
      ctx.beginPath();
      ctx.arc(s.x, s.y, s.size, 0, Math.PI * 2);
      ctx.fill();

      if (s.size > 1.2) {
        ctx.save();
        ctx.globalAlpha = alpha * 0.3;
        ctx.fillStyle = s.tint || "#ffffff";
        ctx.beginPath();
        ctx.arc(s.x, s.y, s.size * 3, 0, Math.PI * 2);
        ctx.fill();
        ctx.restore();
      }
      if (s.tint) ctx.globalAlpha = 1;
    });
  }
}
