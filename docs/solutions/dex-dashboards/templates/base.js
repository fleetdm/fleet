let DATA = window.__DATA__;

// Build DOM without innerHTML: data values are untrusted text.
function h(tag, attrs, ...kids) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs || {})) {
    if (v == null || v === false) continue;
    if (k === "class") el.className = v;
    else if (k === "text") el.textContent = v;
    else if (k === "style") el.style.cssText = v;
    else if (k.startsWith("on")) el.addEventListener(k.slice(2), v);
    else el.setAttribute(k, v === true ? "" : v);
  }
  for (const kid of kids.flat(Infinity)) {
    if (kid == null || kid === false) continue;
    el.append(kid instanceof Node ? kid : document.createTextNode(String(kid)));
  }
  return el;
}
function svg(tag, attrs, ...kids) {
  const el = document.createElementNS("http://www.w3.org/2000/svg", tag);
  for (const [k, v] of Object.entries(attrs || {})) if (v != null) el.setAttribute(k, v);
  for (const kid of kids.flat(Infinity)) if (kid != null) el.append(kid instanceof Node ? kid : document.createTextNode(String(kid)));
  return el;
}
function mount(id, ...kids) {
  const el = document.getElementById(id);
  el.replaceChildren(...kids.flat().filter(Boolean));
  return el;
}

// On a big fleet the page's data is split into part files next to the page.
// build.py wraps the page script in dexReady() only then, so it draws after they load.
function dexReady(draw) {
  const parts = DATA.__parts__ || [];
  const wrap = document.querySelector(".wrap");
  const note = h("p", { class: "hint", role: "status" }, `Loading ${parts.length} data files…`);
  wrap.prepend(note);
  Promise.all(parts.map((p) => fetch(p).then((r) => {
    if (!r.ok) throw new Error(`${p} returned ${r.status}`);
    return r.json();
  }))).then((loaded) => {
    for (const part of loaded) {
      const { key, merge, value } = part;
      if (merge === "list") DATA[key] = (DATA[key] || []).concat(value);
      else if (merge === "dict") DATA[key] = Object.assign(DATA[key] || {}, value);
      else DATA[key] = value;
    }
    delete DATA.__parts__;
    note.remove();
    draw();
  }).catch((e) => {
    note.replaceWith(empty("This dashboard's data didn't load", `${e.message}. Reload the page; if it keeps happening, check that the last run published every file under data/.`));
  });
}

const fmt = {
  n: (x) => (x == null || Number.isNaN(x) ? "---" : Number(x).toLocaleString("en-US")),
  pct: (x) => (x == null || Number.isNaN(x) ? "---" : `${Math.round(x)}%`),
  days: (d) => {
    if (d == null || Number.isNaN(d)) return "---";
    if (d < 1) return "same day";
    return `${d < 10 ? (Math.round(d * 10) / 10).toString() : Math.round(d)} ${Math.round(d) === 1 ? "day" : "days"}`;
  },
  date: (iso) => {
    if (!iso) return "---";
    const d = new Date(iso.length === 10 ? iso + "T00:00:00Z" : iso);
    return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric", timeZone: "UTC" });
  },
  ago: (days) => {
    if (days == null) return "never";
    if (days < 1 / 24) return "just now";
    if (days < 1) return `${Math.round(days * 24)}h ago`;
    if (days < 60) return `${Math.round(days)}d ago`;
    return `${Math.round(days / 30.4)}mo ago`;
  },
  plural: (n, one, many) => `${fmt.n(n)} ${n === 1 ? one : many || one + "s"}`,
};
const pct = (n, d) => (d ? (100 * n) / d : null);
function quantile(xs, q) {
  const s = xs.filter((x) => Number.isFinite(x)).sort((a, b) => a - b);
  if (!s.length) return null;
  const i = (s.length - 1) * q, lo = Math.floor(i), hi = Math.min(lo + 1, s.length - 1);
  return s[lo] + (s[hi] - s[lo]) * (i - lo);
}
const mean = (xs) => (xs.length ? xs.reduce((a, b) => a + b, 0) / xs.length : null);

function vkey(v) {
  return (String(v || "").match(/\d+|[A-Za-z]+/g) || []).map((p) => (/^\d+$/.test(p) ? [0, +p, ""] : [1, 0, p.toLowerCase()]));
}
function vcmp(a, b) {
  const ka = vkey(a), kb = vkey(b);
  for (let i = 0; i < Math.max(ka.length, kb.length); i++) {
    if (!ka[i]) return -1;
    if (!kb[i]) return 1;
    for (let j = 0; j < 3; j++) if (ka[i][j] !== kb[i][j]) return ka[i][j] < kb[i][j] ? -1 : 1;
  }
  return 0;
}

const hostUrl = (id) => `${DATA.fleet_url}/hosts/${id}`;
function hostLink(id, name) {
  return h("a", { href: hostUrl(id), target: "_blank", rel: "noopener", class: "cell-main", onclick: (e) => e.stopPropagation() }, name);
}

// Fleet and platform filters, remembered per viewer.
const store = {
  get(k) { try { return localStorage.getItem(k); } catch { return null; } },
  set(k, v) { try { localStorage.setItem(k, v); } catch { /* storage unavailable */ } },
};
const PLATFORMS = [["macos", "macOS"], ["windows", "Windows"], ["linux", "Linux"]];
const platformLabel = (p) => (PLATFORMS.find(([k]) => k === p) || [p, p])[1];
let FLEET = "";
let PLATFORM = "";
// counts: {macos: n, ...}. unsupported: {linux: "why"} greys a platform out.
function renderTop(fleets, onChange, counts = {}, unsupported = {}) {
  const saved = store.get("dex-fleet");
  FLEET = fleets.includes(saved) ? saved : "";
  const savedP = store.get("dex-platform") || "";
  PLATFORM = savedP && counts[savedP] && !unsupported[savedP] ? savedP : "";
  const toggle = h("div", { class: "seg-toggle", role: "group", "aria-label": "Platform" });
  const drawToggle = () => toggle.replaceChildren(...[["", "All platforms"], ...PLATFORMS].map(([k, label]) => {
    const why = unsupported[k];
    const n = k ? counts[k] || 0 : Object.entries(counts).filter(([pk]) => !unsupported[pk]).reduce((a, [, c]) => a + c, 0);
    const btn = h("button", { type: "button", "aria-pressed": PLATFORM === k ? "true" : "false", disabled: Boolean(why || (k && !n)),
      title: why || (k && !n ? `No ${label} hosts` : null) || null,
      onclick: () => { PLATFORM = k; store.set("dex-platform", k); drawToggle(); onChange(); } },
      label, why ? "" : h("b", {}, fmt.n(n)));
    return why ? tip(btn, why) : btn;
  }));
  drawToggle();
  const sel = h("select", { id: "fleet-filter", "aria-label": "Filter by fleet", onchange: (e) => {
    FLEET = e.target.value; store.set("dex-fleet", FLEET); onChange();
  } }, h("option", { value: "" }, "All fleets"), fleets.map((f) => h("option", { value: f, selected: f === FLEET }, f)));
  const host = DATA.fleet_url.replace(/^https?:\/\//, "");
  const overview = DATA.links && DATA.links.overview;
  mount("top",
    h("div", { class: "top-row" }, toggle, h("div", { class: "controls" }, sel, themeToggle())),
    h("div", { class: "meta" },
      overview ? h("a", { href: overview, target: "_blank", rel: "noopener" }, "All dashboards") : null,
      h("span", {}, `Updated ${fmt.date(DATA.generated_at)}, ${DATA.generated_at.slice(11, 16)} UTC · `, h("a", { href: DATA.fleet_url, target: "_blank", rel: "noopener" }, host))));
}
const inPlatform = (p) => !PLATFORM || p === PLATFORM;

// Light, dark, or the viewer's system setting. "System" leaves the theme to
// the viewer's own setting and prefers-color-scheme. The viewer owns
// data-theme and re-stamps it, so the choice goes on data-dex-theme. The
// viewer can also block or clear browser storage, so the choice is also kept
// in the viewer's private record in the artifact's store (db + user
// capabilities; Viewers can't write it and fall back to browser storage).
const THEMES = [["light", "Light"], ["dark", "Dark"], ["", "System"]];
function applyTheme(t) {
  if (t) document.documentElement.setAttribute("data-dex-theme", t);
  else document.documentElement.removeAttribute("data-dex-theme");
}
async function themeDoc() {
  const cl = window.claude;
  if (!cl || !cl.use) return null;
  const [db, user] = await Promise.all([cl.use("db"), cl.use("user")]);
  const id = db && user ? await user.id() : null;
  return id ? db.doc(`data/users/${id}/prefs`) : null;
}
function themeToggle() {
  let theme = store.get("dex-theme") || "";
  let picked = false;
  if (theme) applyTheme(theme);
  const doc = themeDoc().catch(() => null);
  let saving = Promise.resolve();
  const set = (k) => { theme = k; store.set("dex-theme", k); applyTheme(k); draw(); };
  const el = h("div", { class: "seg-toggle small", role: "group", "aria-label": "Theme" });
  const draw = () => el.replaceChildren(...THEMES.map(([k, label]) => h("button", { type: "button", "aria-pressed": theme === k ? "true" : "false",
    onclick: () => {
      if (k === theme) return;
      picked = true;
      set(k);
      saving = saving.then(() => doc).then((d) => d && d.set({ theme: k })).catch(() => { /* read-only viewer: browser storage only */ });
    } }, label)));
  draw();
  doc.then((d) => d && d.get()).then((s) => {
    const t = s && s.exists ? s.data().theme : null;
    if (!picked && t !== theme && THEMES.some(([k]) => k === t)) set(t);
  }).catch(() => {});
  return el;
}
const inFleet = (f) => !FLEET || f === FLEET;

// Tooltip shared by every chart.
const tipEl = h("div", { class: "tip", role: "tooltip", hidden: true });
document.body.append(tipEl);
function showTip(text, x, y) {
  tipEl.textContent = text;
  tipEl.hidden = false;
  const r = tipEl.getBoundingClientRect();
  tipEl.style.left = `${Math.min(Math.max(8, x + 12), innerWidth - r.width - 8)}px`;
  tipEl.style.top = `${Math.max(8, y - r.height - 12)}px`;
}
const hideTip = () => (tipEl.hidden = true);
function tip(el, text) {
  const show = (x, y) => showTip(typeof text === "function" ? text() : text, x, y);
  el.addEventListener("mousemove", (e) => show(e.clientX, e.clientY));
  el.addEventListener("mouseleave", hideTip);
  el.addEventListener("focus", () => { const r = el.getBoundingClientRect(); show(r.left + r.width / 2, r.top); });
  el.addEventListener("blur", hideTip);
  if (!el.hasAttribute("tabindex")) el.setAttribute("tabindex", "0");
  return el;
}

// Daily trends. DATA.trend is [[date, {"<fleet>|<platform>": {field: n}}], ...],
// so each field can be summed over the fleets and platforms the filters select.
const RANGES = [[30, "30 days"], [90, "90 days"], [366, "1 year"]];
let RANGE = Number(store.get("dex-range")) || 90;
if (!RANGES.some(([d]) => d === RANGE)) RANGE = 90;
function trendRows(fields) {
  const cutoff = new Date(Date.parse(DATA.generated_at) - RANGE * 864e5).toISOString().slice(0, 10);
  return (DATA.trend || []).filter(([date]) => date > cutoff).map(([date, byKey]) => {
    const row = { date };
    for (const f of fields) row[f] = 0;
    for (const [k, counts] of Object.entries(byKey)) {
      const i = k.lastIndexOf("|");
      if (!inFleet(k.slice(0, i)) || !inPlatform(k.slice(i + 1))) continue;
      for (const f of fields) row[f] += counts[f] || 0;
    }
    return row;
  });
}

// Charts draw at their container's width, so they redraw on resize. Detached
// charts from earlier renders are dropped once they've drawn.
const trendCharts = new Set();
const trendRO = new ResizeObserver((entries) => entries.forEach((e) => e.target._draw()));
const shortDate = (t) => new Date(t).toLocaleDateString("en-US", { month: "short", day: "numeric", timeZone: "UTC" });

// One y-axis, so every series shares a unit. series: [{label, color, value(row)}].
// kind "bar" draws the first series as one bar per day.
function trendChart({ rows, series, kind = "line", yFmt = fmt.n, yMax, label }) {
  const el = h("div", { class: "trend" });
  const vals = rows.flatMap((r) => series.map((s) => s.value(r))).filter(Number.isFinite);
  const max = Math.max(0, ...vals);
  const step = yMax ? yMax / 4 : max <= 4 ? 1 : (() => {
    const raw = max / 4, mag = 10 ** Math.floor(Math.log10(raw));
    return [1, 2, 2.5, 5, 10].map((m) => m * mag).find((x) => x >= raw);
  })();
  const top = yMax || Math.max(step, Math.ceil(max / step) * step);
  const day = (d) => Date.parse(d + "T00:00:00Z");
  const t0 = day(rows[0].date), days = Math.max(1, Math.round((day(rows[rows.length - 1].date) - t0) / 864e5));
  const last = rows[rows.length - 1];
  const summary = `${label}. ${fmt.date(last.date)}: ${series.map((s) => `${s.label} ${yFmt(s.value(last))}`).join(", ")}.`;

  el._draw = () => {
    const W = Math.max(260, el.clientWidth), H = 200;
    const direct = kind === "line" && series.length > 1 && W >= 560;
    const L = 40, T = 8, B = 24, R = direct ? 20 + 7 * Math.max(...series.map((s) => s.label.length)) : 12;
    const pw = W - L - R, ph = H - T - B;
    const slot = kind === "bar" ? pw / (days + 1) : 0;
    const x = (d) => L + slot / 2 + ((day(d) - t0) / 864e5) * (kind === "bar" ? slot : pw / days);
    const y = (v) => T + (1 - v / top) * ph;
    const yTicks = [];
    for (let v = 0; v <= top + 1e-9; v += step) yTicks.push(v);
    const every = [1, 2, 7, 14, 30, 61, 91, 183].find((n) => days / n <= Math.max(2, Math.floor(pw / 80))) || 183;
    const xTicks = [];
    for (let i = 0; i <= days; i += every) xTicks.push(t0 + i * 864e5);

    const marks = [];
    if (kind === "bar") {
      const s = series[0], w = Math.max(1, slot - 2);
      for (const r of rows) {
        const v = s.value(r);
        if (!(v > 0)) continue;
        const bx = x(r.date) - w / 2, y0 = y(0), y1 = y(v), rad = Math.min(4, w / 2, y0 - y1);
        marks.push(svg("path", { style: `fill:${s.color}`, d: `M${bx},${y0}V${y1 + rad}Q${bx},${y1} ${bx + rad},${y1}H${bx + w - rad}Q${bx + w},${y1} ${bx + w},${y1 + rad}V${y0}Z` }));
      }
    } else {
      const ends = [];
      for (const s of series) {
        let d = "", pen = "M", end = null;
        for (const r of rows) {
          const v = s.value(r);
          if (!Number.isFinite(v)) { pen = "M"; continue; }
          d += `${pen}${x(r.date).toFixed(1)},${y(v).toFixed(1)}`;
          pen = "L";
          end = [r.date, v];
        }
        if (!end) continue;
        marks.push(svg("path", { class: "halo", d }), svg("path", { class: "series", style: `stroke:${s.color}`, d }));
        marks.push(svg("circle", { class: "marker", cx: x(end[0]), cy: y(end[1]), r: 4, style: `fill:${s.color}` }));
        ends.push({ s, y: y(end[1]) });
      }
      // Direct labels at the line ends, nudged apart so close values don't overlap.
      if (direct) {
        ends.sort((a, b) => a.y - b.y);
        for (let i = 1; i < ends.length; i++) ends[i].y = Math.max(ends[i].y, ends[i - 1].y + 14);
        const over = ends.length ? ends[ends.length - 1].y - (H - B) : 0;
        for (const e of ends) {
          marks.push(svg("text", { x: W - R + 10, y: e.y - Math.max(0, over) + 4 }, e.s.label));
        }
      }
    }

    const cross = svg("line", { class: "cross", y1: T, y2: H - B, visibility: "hidden" });
    const dots = svg("g", { visibility: "hidden" });
    let at = -1;
    const point = (i, clientX, clientY) => {
      at = Math.max(0, Math.min(rows.length - 1, i));
      const r = rows[at], cx = x(r.date);
      cross.setAttribute("x1", cx); cross.setAttribute("x2", cx);
      cross.setAttribute("visibility", "visible"); dots.setAttribute("visibility", "visible");
      dots.replaceChildren(...(kind === "line" ? series.map((s) => {
        const v = s.value(r);
        return Number.isFinite(v) ? svg("circle", { class: "marker", cx, cy: y(v), r: 4, style: `fill:${s.color}` }) : null;
      }) : []));
      const box = g.getBoundingClientRect();
      showTip([fmt.date(r.date), ...series.map((s) => `${s.label}: ${yFmt(s.value(r))}`)].join("\n"),
        clientX ?? box.left + cx, clientY ?? box.top + T);
    };
    const clear = () => { cross.setAttribute("visibility", "hidden"); dots.setAttribute("visibility", "hidden"); hideTip(); };
    const nearest = (px) => rows.reduce((best, r, i) => (Math.abs(x(r.date) - px) < Math.abs(x(rows[best].date) - px) ? i : best), 0);
    const hit = svg("rect", { class: "hit", x: L, y: T, width: pw, height: ph, tabindex: "0", "aria-label": `${summary} Use the arrow keys to read each day.` });
    hit.addEventListener("mousemove", (e) => point(nearest(e.clientX - g.getBoundingClientRect().left), e.clientX, e.clientY));
    hit.addEventListener("mouseleave", clear);
    hit.addEventListener("blur", clear);
    hit.addEventListener("focus", () => point(at < 0 ? rows.length - 1 : at));
    hit.addEventListener("keydown", (e) => {
      const move = { ArrowLeft: -1, ArrowRight: 1, Home: -rows.length, End: rows.length }[e.key];
      if (move) { e.preventDefault(); point(at + move); }
    });

    const g = svg("svg", { width: "100%", height: H, viewBox: `0 0 ${W} ${H}`, role: "img", "aria-label": summary },
      yTicks.map((v) => [svg("line", { class: v ? "grid-line" : "axis", x1: L, x2: W - R, y1: y(v), y2: y(v) }),
        svg("text", { x: L - 8, y: y(v) + 4, "text-anchor": "end" }, yFmt(v))]),
      xTicks.map((t) => svg("text", { x: x(new Date(t).toISOString().slice(0, 10)), y: H - 6, "text-anchor": "middle" }, shortDate(t))),
      marks, cross, dots, hit);
    el.replaceChildren(...[g, series.length > 1 ? legend(series.map((s) => [s.label, s.color])) : null].filter(Boolean));
    el._drawn = true;
  };
  for (const c of trendCharts) if (c._drawn && !c.isConnected) { trendRO.unobserve(c); trendCharts.delete(c); }
  trendCharts.add(el);
  trendRO.observe(el);
  return el;
}

// Fills #trend with one panel per chart and #trend-tools with the range switch.
// panels: [{title, series, kind, yFmt, yMax, note}]; fields: the trend fields they read.
function renderTrend(fields, panels) {
  const rows = trendRows(fields);
  const range = h("div", { class: "seg-toggle small", role: "group", "aria-label": "Trend range" },
    RANGES.map(([d, label]) => h("button", { type: "button", "aria-pressed": RANGE === d ? "true" : "false",
      onclick: () => { RANGE = d; store.set("dex-range", String(d)); renderTrend(fields, panels); } }, label)));
  mount("trend-tools", rows.length ? copyButton("Copy as CSV", () => csv([["date", ...fields], ...rows.map((r) => [r.date, ...fields.map((f) => r[f])])])) : null, range);
  const since = DATA.trend && DATA.trend.length ? DATA.trend[0][0] : null;
  mount("trend", panels.map((p) => h("div", { class: "panel" }, h("h3", {}, p.title),
    rows.length >= 2 ? trendChart({ ...p, rows, label: p.title })
      : empty("Not enough history yet", `${since ? `Daily history starts ${fmt.date(since)}. ` : ""}A trend appears once two days are recorded in the selected range.`),
    p.note ? h("p", { class: "panel-note" }, p.note) : null)));
}

// One row per item; each row is a track of colored segments.
function barRows(items, max) {
  const top = max || Math.max(1, ...items.map((it) => it.segs.reduce((a, s) => a + s.value, 0)));
  return h("div", { class: "bars" }, items.map((it) => {
    const total = it.segs.reduce((a, s) => a + s.value, 0);
    const track = h("div", { class: "track", role: "img", "aria-label": `${it.label}: ${it.valueText ?? total}` },
      it.segs.filter((s) => s.value > 0).map((s) => tip(h("div", { class: "seg", style: `width:${(100 * s.value) / top}%;background:${s.color}` }), s.tip)));
    return h("div", { class: "bar-row" },
      h("span", { class: "lbl", title: it.label }, it.label), track, h("span", { class: "val" }, it.valueText ?? fmt.n(total)));
  }));
}
function legend(entries) {
  return h("div", { class: "legend" }, entries.map(([label, color]) => h("span", {}, h("i", { style: `background:${color}` }), label)));
}
function pill(label, tone) {
  return h("span", { class: "pill", style: `--tone:var(--${tone})` }, label);
}
function mini(p, tone) {
  return h("span", { class: "mini", style: `--tone:var(--${tone})`, "aria-hidden": "true" }, h("span", { style: `width:${Math.max(0, Math.min(100, p))}%` }));
}
function empty(title, body) {
  return h("div", { class: "empty" }, h("b", {}, title), body);
}

// Sortable table. columns: {key, label, num, value(row), cell(row)}.
// Each group shows its first 200 rows, then a button for more, so a big fleet doesn't
// put thousands of rows on the page at once.
const TABLE_ROWS = 200;
function sortTable({ columns, groups, sort, onRow, detail, id }) {
  let [key, dir] = sort;
  let open = null;
  const shown = groups.map(() => TABLE_ROWS);
  const wrap = h("div", { class: "tbl-wrap" });
  function draw() {
    const col = columns.find((c) => c.key === key);
    const cmp = (a, b) => {
      const va = col.value(a), vb = col.value(b);
      if (va == null && vb == null) return 0;
      if (va == null) return 1;
      if (vb == null) return -1;
      return (va < vb ? -1 : va > vb ? 1 : 0) * (dir === "asc" ? 1 : -1);
    };
    const head = h("tr", {}, columns.map((c) => h("th", { class: c.num ? "r" : null, scope: "col", "aria-sort": c.key === key ? (dir === "asc" ? "ascending" : "descending") : null },
      h("button", { type: "button", onclick: () => { if (key === c.key) dir = dir === "asc" ? "desc" : "asc"; else { key = c.key; dir = c.num ? "desc" : "asc"; } draw(); } }, c.label))));
    const body = h("tbody");
    groups.forEach((g, gi) => {
      if (g.label) body.append(h("tr", { class: "group" }, h("td", { colspan: columns.length }, g.pill || "", " ", g.label, g.hint ? h("span", { class: "hint" }, g.hint) : "")));
      const sorted = [...g.rows].sort(cmp);
      for (const row of sorted.slice(0, shown[gi])) {
        const rid = row._id;
        const tr = h("tr", { class: onRow || detail ? "click" : null }, columns.map((c) => h("td", { class: c.num ? "r" : null }, c.cell ? c.cell(row) : c.value(row) ?? "---")));
        if (detail) {
          tr.setAttribute("tabindex", "0");
          tr.setAttribute("aria-expanded", open === rid ? "true" : "false");
          const toggle = () => { open = open === rid ? null : rid; draw(); };
          tr.addEventListener("click", toggle);
          tr.addEventListener("keydown", (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); toggle(); } });
        }
        body.append(tr);
        if (detail && open === rid) body.append(h("tr", { class: "detail" }, h("td", { colspan: columns.length }, detail(row))));
      }
      const more = sorted.length - shown[gi];
      if (more > 0) body.append(h("tr", {}, h("td", { colspan: columns.length },
        h("button", { type: "button", class: "btn", onclick: () => { shown[gi] += 1000; draw(); } }, more > 1000 ? `Show 1,000 more of ${fmt.n(more)}` : `Show ${fmt.n(more)} more`))));
    });
    wrap.replaceChildren(h("table", { class: "tbl", id }, h("thead", {}, head), body));
  }
  draw();
  return wrap;
}

// Clipboard copy needs a click; fall back to a selectable box.
function copyButton(label, text) {
  const btn = h("button", { type: "button", class: "btn" }, label);
  btn.addEventListener("click", async () => {
    const t = text();
    try {
      await navigator.clipboard.writeText(t);
      btn.textContent = "Copied";
      setTimeout(() => (btn.textContent = label), 1600);
    } catch {
      const ta = h("textarea", { class: "mono", style: "width:100%;height:140px;margin-top:8px", readonly: true }, t);
      btn.after(ta);
      ta.select();
    }
  });
  return btn;
}
const csv = (rows) => rows.map((r) => r.map((v) => `"${String(v ?? "").replace(/"/g, '""')}"`).join(",")).join("\n");
