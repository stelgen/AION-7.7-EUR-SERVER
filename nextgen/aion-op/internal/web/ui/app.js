/* aion-op UI — Phase 0.5: observe-only + глаза (логи/метрики/алерты). Кнопки disabled до Phase 1. */
"use strict";

const TABS = [
  { id: "overview", title: "Overview", filter: null },
  { id: "auth",     title: "Auth",      filter: { group: "auth" } },
  { id: "cache",    title: "Cache",     filter: { ids: ["cache"] } },
  { id: "world",    title: "NPC+World", filter: { ids: ["npc", "main"], world: true } },
  { id: "logsrv",   title: "LogServer", filter: { ids: ["logsrv"] } },
  { id: "sql",      title: "SQL",       filter: { group: "sql" } },
  { id: "svc",      title: "Прочее",    filter: { group: "svc" } },
  { id: "logs",     title: "Логи",      filter: { logs: true } },
  { id: "metrics",  title: "Метрики",   filter: { metrics: true } },
  { id: "alerts",   title: "Алерты",    filter: { alerts: true } },
  { id: "settings", title: "Настройки", filter: { settings: true } },
];

let DATA = null;
let TAB = "overview";
let clockTimer = null;

const $ = (q) => document.querySelector(q);

function esc(s) {
  return String(s).replace(/[&<>\"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
}

async function tick() {
  try {
    const r = await fetch("/api/status", { cache: "no-store" });
    DATA = await r.json();
    render();
  } catch (e) {
    const b = $("#banner");
    b.className = "banner red";
    b.textContent = "API недоступен: " + e;
    b.classList.remove("hidden");
  }
}

function setHead(d) {
  const mb = $("#mode-badge");
  mb.textContent = d.mode === "operate" ? "OPERATE" : "OBSERVE";
  mb.className = "badge " + (d.mode === "operate" ? "badge-operate" : "badge-observe");
  $("#probe-src").textContent = `проба: ${d.source} · при Live (ssh) — read-only tasklist/netstat/quser`;
  $("#clock").textContent = d.when;
  const al = (d.alerts || []).length;
  $("#foot-when").textContent =
    `срез ${d.when} · обновление ${d.refresh_sec}s · running ${d.summary.running} / warn ${d.summary.warn} / stopped ${d.summary.stopped}` +
    (al ? ` · алерты ${al}` : "");

  const cc = $("#console-chip");
  cc.textContent = d.console_session ? "консоль-сессия VM: есть (/IT ок)" : "консоль-сессия VM: НЕТ (/IT мертвы)";
  cc.className = "chip " + (d.console_session ? "ok" : "bad");

  const b = $("#banner");
  if (d.probe_err) {
    b.className = "banner red";
    b.textContent = "⚠ Проба VM не удалась — все состояния UNKNOWN: " + d.probe_err;
    b.classList.remove("hidden");
  } else if (d.world.pair_broken) {
    b.className = "banner red";
    b.textContent = "⚠ ПАРА РАЗОМКНУТА: " + d.world.pair_note;
    b.classList.remove("hidden");
  } else if (d.world.loading_window) {
    b.className = "banner amber";
    b.textContent = `⏳ Окно загрузки/спавна NPC: ${d.world.conns}/${d.world.expected} conns — рестарты пары заблокированы (10–15 мин).`;
    b.classList.remove("hidden");
  } else {
    b.classList.add("hidden");
  }
}

function svcRow(s) {
  const ports = (s.ports_ok || []).map((p) => `<span class="chip ok mono">${p}</span>`)
    .concat((s.ports_bad || []).map((p) => `<span class="chip bad mono">${p}✗</span>`)).join("");
  const sub = [s.exe, s.task ? `task: ${s.task}` : null].filter(Boolean).map(esc).join(" · ");
  const mem = s.mem_mb ? `<span class="p">${s.mem_mb} МБ</span>` : "";
  const pids = (s.pids || []).map((p) => `<span class="p">PID ${esc(p)}</span>`).join("");

  let actions;
  if (s.locked) {
    actions = `<span class="lock" title="Мёртвый сервис 7.7-кита — замок (PLAN.md §4)">🔒 замок</span>`;
  } else if (s.observe_only) {
    actions = `<span class="lock">наблюдение</span>`;
  } else {
    const tip = "Phase 1: кнопки оживут с агентом на VM (по «го»)";
    actions = `<div class="actions">
      <button class="btn" disabled title="Start — ${tip}">▶</button>
      <button class="btn" disabled title="Stop — ${tip}">■</button>
      <button class="btn" disabled title="Restart — ${tip}">⟳</button>
    </div>`;
  }
  return `<div class="svc">
    <div class="name">${esc(s.display)}<span class="sub">${sub}</span></div>
    <div class="state st-${esc(s.state)}">${esc(s.state)}</div>
    <div class="detail">${mem}${pids}<div>${ports || "&nbsp;"}</div><div>${esc(s.detail || "")}</div></div>
    ${actions}
  </div>`;
}

function worldPanel(w) {
  const pct = Math.min(100, Math.round((w.conns / Math.max(1, w.expected)) * 100));
  const cls = w.pair_broken ? "broken" : (w.collected ? "" : "part");
  const label = w.pair_broken ? "ПАРА РАЗОМКНУТА" : (w.collected ? "мир собран" : (w.loading_window ? "загрузка/спавн" : "мир не собран"));
  return `<div class="world-panel">
    <h2>ПАРА NPC+MAIN (единая единица управления)</h2>
    <div class="world-line">
      <b class="mono">${w.conns}/${w.expected}</b>
      <div class="bar ${cls}"><i style="width:${pct}%"></i></div>
      <span class="dim">${label} · ESTABLISHED на :2002</span>
    </div>
    ${w.pair_note ? `<div class="world-note">⚠ ${esc(w.pair_note)}</div>` : ""}
  </div>`;
}

// --- Phase 0.5: логи/метрики/алерты ---

function sysChips(ms) {
  if (!ms || !ms.sys) return "";
  const fc = ms.sys.free_commit_mb, fp = ms.sys.free_phys_mb;
  if (!fc && !fp) return "";
  const cls = (v, low) => v && v < low ? "bad" : "ok";
  return `<div class="syschips">
    <span class="chip ${cls(fc, 8192)}" title="Свободный коммит (лимит ~44 ГБ; ночь 03.10: тихая смерть при исчерпании)">FreeCommit: ${fc} МБ</span>
    <span class="chip ${cls(fp, 2048)}" title="Свободная физическая память">FreePhys: ${fp} МБ</span>
  </div>`;
}

function evtRow(e) {
  const t = e.ts ? new Date(e.ts * 1000).toLocaleTimeString() : "";
  return `<div class="evt sev${e.sev}">
    <span class="mono evt-time">${t}</span>
    <span class="chip evt-svc">${esc(e.svc)}</span>
    <span class="mono evt-kind">${esc(e.kind)}</span>
    <span class="evt-text">${esc(e.text)}</span>
  </div>`;
}

function renderLogs() {
  const evs = DATA.events || [];
  const counts = {};
  for (const e of evs) counts[e.kind] = (counts[e.kind] || 0) + 1;
  const legend = Object.entries(counts).map(([k, n]) => `<span class="chip">${esc(k)} ×${n}</span>`).join(" ");
  return `<div class="group"><h2>Лог-события (последние ${evs.length}, парсер Phase 0.5)</h2>
    <div class="legend">${legend || '<span class="dim">пока тихо</span>'}</div>
    <div class="card evlist">${evs.map(evtRow).join("") || '<div class="placeholder">событий нет</div>'}</div></div>`;
}

function alertRow(a) {
  const names = { 1: "LOW", 2: "MED", 3: "CRIT" };
  const t = a.opened_ts ? new Date(a.opened_ts * 1000).toLocaleTimeString() : "";
  return `<div class="alert sev${a.sev}">
    <span class="badge badge-al sev${a.sev}">${names[a.sev] || "ALERT"}</span>
    <span class="mono">${esc(a.idem)}</span>
    <span class="alert-text">${esc(a.text)}</span>
    <span class="dim mono">с ${t}</span>
  </div>`;
}

function renderAlerts() {
  const al = DATA.alerts || [];
  return `<div class="group"><h2>Активные алерты (${al.length})</h2>
    <div class="card">${al.map(alertRow).join("") || '<div class="placeholder">Тихо — активных алертов нет ✅</div>'}</div></div>
    <div class="dim" style="margin-top:8px">Правила: смерть сервиса, разрыв пары, FreeCommit&lt;8 ГБ, утечка хендлов (&gt;20k/мин ×3 тика), рейт 2812&gt;50/5мин, критичный лог (Super-Lag/Intentional/NpcSocket) — окно 15 мин.</div>`;
}

function spark(points, key) {
  if (!points || points.length < 2) return '<span class="dim">мало точек (копится история)</span>';
  const vals = points.map((p) => key === "handles" ? p.handles : p.mem_mb);
  const min = Math.min(...vals), max = Math.max(...vals);
  const range = (max - min) || 1;
  const w = 260, h = 44;
  const stepX = w / (points.length - 1);
  const pts = vals.map((v, i) =>
    `${(i * stepX).toFixed(1)},${(h - ((v - min) / range) * (h - 4) - 2).toFixed(1)}`).join(" ");
  const rising = vals[vals.length - 1] - vals[0] > 0;
  return `<svg class="spark" width="${w}" height="${h}" viewBox="0 0 ${w} ${h}"><polyline fill="none" stroke="${rising ? "var(--red)" : "var(--green)"}" stroke-width="1.5" points="${pts}"/></svg>`;
}

function mRow(p) {
  const dpmCls = Math.abs(p.handles_dpm) > 20000 ? "bad" : (Math.abs(p.handles_dpm) > 5000 ? "warn" : "");
  return `<div class="svc mrow">
    <div class="name">${esc(p.name)}<span class="sub">PID ${esc(p.pid)}</span></div>
    <div class="detail"><span class="p">${p.mem_mb} МБ</span></div>
    <div class="detail"><span class="p">${(p.handles || 0).toLocaleString("ru")}</span></div>
    <div class="detail ${dpmCls}">${p.handles_dpm > 0 ? "+" : ""}${p.handles_dpm}/мин</div>
  </div>`;
}

async function renderMetrics() {
  const ms = DATA.metrics || {};
  const procs = (ms.procs || []).slice().sort((a, b) => (b.handles || 0) - (a.handles || 0));
  let html = sysChips(ms);
  html += `<div class="group"><h2>Процессы (RAM / хендлы / динамика хендлов)</h2>
    <div class="card">
      <div class="svc mrow mhead"><div class="name">процесс</div><div class="detail">RAM</div><div class="detail">хендлы</div><div class="detail">Δ/мин</div></div>
      ${procs.map(mRow).join("") || '<div class="placeholder">метрик ещё нет (первый срез через poll_sec)</div>'}
    </div></div>`;
  for (const n of ["Server64.exe", "NPCSvr64.exe"]) {
    html += `<div class="group"><h2>${n}: хендлы (120 мин)</h2><div class="card sparkbox" id="spark-${n}">
      <div class="placeholder">тяну /api/metrics…</div></div></div>`;
  }
  $("#content").innerHTML = html;
  for (const n of ["Server64.exe", "NPCSvr64.exe"]) {
    try {
      const r = await fetch(`/api/metrics?name=${encodeURIComponent(n)}&minutes=120`, { cache: "no-store" });
      const d = await r.json();
      const box = document.getElementById(`spark-${n}`);
      if (box) box.innerHTML = spark(d.points, "handles");
    } catch (e) { /* тихо */ }
  }
}

function render() {
  if (!DATA) return;
  setHead(DATA);

  const nav = $("#tabs");
  nav.innerHTML = TABS.map((t) =>
    `<button class="tab ${t.id === TAB ? "active" : ""}" data-tab="${t.id}">${t.title}</button>`).join("");
  nav.querySelectorAll(".tab").forEach((el) =>
    el.addEventListener("click", () => { TAB = el.dataset.tab; render(); }));

  const t = TABS.find((x) => x.id === TAB);
  const c = $("#content");

  if (t.filter && t.filter.logs) { c.innerHTML = renderLogs(); return; }
  if (t.filter && t.filter.alerts) { c.innerHTML = renderAlerts(); return; }
  if (t.filter && t.filter.metrics) { renderMetrics(); return; }
  if (t.filter && t.filter.settings) { renderSettings(); return; }

  let groups = DATA.groups;
  if (t.filter) {
    groups = groups
      .map((g) => ({
        ...g,
        services: g.services.filter((s) =>
          t.filter.group ? s.group === t.filter.group : (t.filter.ids || []).includes(s.id)),
      }))
      .filter((g) => g.services.length > 0);
  }

  let html = "";
  if (t.filter && t.filter.world) html += worldPanel(DATA.world);
  if (t.id === "overview") {
    html += worldPanel(DATA.world);
    const al = DATA.alerts || [];
    if (al.length) {
      html += `<div class="group"><h2>🚨 Активные алерты (${al.length})</h2>
        <div class="card">${al.map(alertRow).join("")}</div></div>`;
    } else {
      html += sysChips(DATA.metrics);
    }
  }
  for (const g of groups) {
    html += `<div class="group"><h2>${esc(g.icon)} ${esc(g.title)}</h2>
      <div class="card">${g.services.map(svcRow).join("")}</div></div>`;
  }
  c.innerHTML = html || `<div class="placeholder">Пусто</div>`;
}

async function renderSettings() {
  const c = $("#content");
  c.innerHTML = `<div class="placeholder">тяну конфиг…</div>`;
  try {
    const r = await fetch("/api/config", { cache: "no-store" });
    const cfg = await r.json();
    c.innerHTML = `
      <div class="group"><h2>Топология (config.yaml, read-only)</h2>
      <pre class="settings">${esc(JSON.stringify(cfg, null, 2))}</pre></div>
      <div class="group"><h2>Правила оператора (TRACK-A-PLAN.md)</h2>
      <div class="card"><div class="svc"><div class="detail">
        1. Кнопка = переход состояния, идемпотентна.<br>
        2. Старт по order, стоп в обратном; пара NPC+MAIN — единая единица.<br>
        3. Health = процесс + порт + лог-маркер; окно загрузки блокирует рестарты.<br>
        4. Опасное — с typed-confirm; всё в audit-journal.<br>
        5. Windows: управление только через /IT-задачи (schtasks /run), taskkill из SSH не убивает юзер-сессию.<br>
        6. Phase 0.5: глаза (логи/метрики/алерты/SQLite/pprof) — всё read-only; ретеншн 30 дней.
      </div></div></div></div>`;
  } catch (e) {
    c.innerHTML = `<div class="placeholder">Ошибка: ${esc(e)}</div>`;
  }
}

tick();
clockTimer = setInterval(tick, 5000);
