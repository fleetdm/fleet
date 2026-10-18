const DATA = window.__DATA__;

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
  for (const kid of kids.flat()) {
    if (kid == null || kid === false) continue;
    el.append(kid instanceof Node ? kid : document.createTextNode(String(kid)));
  }
  return el;
}
function svg(tag, attrs, ...kids) {
  const el = document.createElementNS("http://www.w3.org/2000/svg", tag);
  for (const [k, v] of Object.entries(attrs || {})) if (v != null) el.setAttribute(k, v);
  for (const kid of kids.flat()) if (kid != null) el.append(kid instanceof Node ? kid : document.createTextNode(String(kid)));
  return el;
}
function mount(id, ...kids) {
  const el = document.getElementById(id);
  el.replaceChildren(...kids.flat().filter(Boolean));
  return el;
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
  mount("top", toggle,
    h("div", { class: "meta" },
      h("span", {}, `Updated ${fmt.date(DATA.generated_at)}, ${DATA.generated_at.slice(11, 16)} UTC · `, h("a", { href: DATA.fleet_url, target: "_blank", rel: "noopener" }, host)),
      sel, themeToggle()));
}
const inPlatform = (p) => !PLATFORM || p === PLATFORM;

// Light, dark, or the viewer's system setting. "System" leaves the theme to
// prefers-color-scheme and the viewer's own setting.
const THEMES = [["light", "Light"], ["dark", "Dark"], ["", "System"]];
function applyTheme(t) {
  if (t) document.documentElement.setAttribute("data-theme", t);
  else document.documentElement.removeAttribute("data-theme");
}
function themeToggle() {
  let theme = store.get("dex-theme") || "";
  if (theme) applyTheme(theme);
  const el = h("div", { class: "seg-toggle small", role: "group", "aria-label": "Theme" });
  const draw = () => el.replaceChildren(...THEMES.map(([k, label]) => h("button", { type: "button", "aria-pressed": theme === k ? "true" : "false",
    onclick: () => { theme = k; store.set("dex-theme", k); applyTheme(k); draw(); } }, label)));
  draw();
  return el;
}
const inFleet = (f) => !FLEET || f === FLEET;

// Tooltip shared by every chart.
const tipEl = h("div", { class: "tip", role: "tooltip", hidden: true });
document.body.append(tipEl);
function tip(el, text) {
  const show = (x, y) => {
    tipEl.textContent = typeof text === "function" ? text() : text;
    tipEl.hidden = false;
    const r = tipEl.getBoundingClientRect();
    tipEl.style.left = `${Math.min(Math.max(8, x + 12), innerWidth - r.width - 8)}px`;
    tipEl.style.top = `${Math.max(8, y - r.height - 12)}px`;
  };
  el.addEventListener("mousemove", (e) => show(e.clientX, e.clientY));
  el.addEventListener("mouseleave", () => (tipEl.hidden = true));
  el.addEventListener("focus", () => { const r = el.getBoundingClientRect(); show(r.left + r.width / 2, r.top); });
  el.addEventListener("blur", () => (tipEl.hidden = true));
  if (!el.hasAttribute("tabindex")) el.setAttribute("tabindex", "0");
  return el;
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
function sortTable({ columns, groups, sort, onRow, detail, id }) {
  let [key, dir] = sort;
  let open = null;
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
    for (const g of groups) {
      if (g.label) body.append(h("tr", { class: "group" }, h("td", { colspan: columns.length }, g.pill || "", " ", g.label, g.hint ? h("span", { class: "hint" }, g.hint) : "")));
      for (const row of [...g.rows].sort(cmp)) {
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
    }
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
