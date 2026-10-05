/* aion-op UI — Phase 0: observe-only, кнопки визуально есть, но всегда disabled. */
"use strict";

const TABS = [
  { id: "overview", title: "Overview", filter: null },
  { id: "auth",     title: "Auth",      filter: { group: "auth" } },
  { id: "cache",    title: "Cache",     filter: { ids: ["cache"] } },
  { id: "world",    title: "NPC+World", filter: { ids: ["npc", "main"], world: true } },
  { id: "logsrv",   title: "LogServer", filter: { ids: ["logsrv"] } },
  { id: "sql",      title: "SQL",       filter: { group: "sql" } },
  { id: "svc",      title: "Прочее",    filter: { group: "svc" } },
  { id: "logs",     title: "Логи",      filter: { placeholder: "Phase 0.5 — тейлеры + парсер: 2812, Too slow, DeadLock/Super-Lag, AboutToPlayerTimer, Shutdown By NpcSocket" } },
  { id: "settings", title: "Настройки", filter: { settings: true } },
];

let DATA = null;
let TAB = "overview";
let clockTimer = null;

const $ = (q) => document.querySelector(q);

function esc(s) {
  return String(s).replace(/[&<>"]/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;" }[c]));
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
  $("#foot-when").textContent = `срез ${d.when} · обновление ${d.refresh_sec}s · running ${d.summary.running} / warn ${d.summary.warn} / stopped ${d.summary.stopped}`;

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

  if (t.filter && t.filter.placeholder) {
    c.innerHTML = `<div class="placeholder">${esc(t.filter.placeholder)}</div>`;
    return;
  }
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
        5. Windows: управление только через /IT-задачи (schtasks /run), taskkill из SSH не убивает юзер-сессию.
      </div></div></div></div>`;
  } catch (e) {
    c.innerHTML = `<div class="placeholder">Ошибка: ${esc(e)}</div>`;
  }
}

tick();
clockTimer = setInterval(tick, 5000);
