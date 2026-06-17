// Frontend logic for the AmneziaWG Manager. Backend methods are exposed by Wails
// at window.go.main.App.*; runtime events arrive via window.runtime.EventsOn.

const $ = (id) => document.getElementById(id);
const backend = () => window.go.main.App;

const IOS_APP_URL = "https://apps.apple.com/app/amneziawg/id6478942365";
const OTHER_APP_URL = "https://amnezia.org/downloads";

// openExternal opens a URL in the user's real browser, not the app webview.
function openExternal(url) {
  if (window.runtime && window.runtime.BrowserOpenURL) window.runtime.BrowserOpenURL(url);
}

let authMode = "password";
let lastResult = null;

// --- small UI helpers ------------------------------------------------------

function show(el) { el.classList.remove("hidden"); }
function hide(el) { el.classList.add("hidden"); }

function busy(on, text) {
  $("busy-text").textContent = text || "Подождите…";
  on ? show($("busy")) : hide($("busy"));
}

// confirmDialog shows an in-app modal and resolves true/false. Native confirm()
// is unreliable inside the Wails webview, so we never use it.
function confirmDialog(message, okLabel = "Удалить") {
  return new Promise((resolve) => {
    $("modal-text").textContent = message;
    $("modal-ok").textContent = okLabel;
    show($("modal"));
    const done = (val) => {
      hide($("modal"));
      $("modal-ok").onclick = null;
      $("modal-cancel").onclick = null;
      resolve(val);
    };
    $("modal-ok").onclick = () => done(true);
    $("modal-cancel").onclick = () => done(false);
  });
}

let toastTimer = null;
function toast(message, kind = "ok") {
  const t = $("toast");
  t.textContent = message;
  t.className = "toast " + kind;
  show(t);
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => hide(t), 4500);
}

function errMsg(err) {
  return typeof err === "string" ? err : (err && err.message) || String(err);
}

function showError(el, err) {
  el.textContent = typeof err === "string" ? err : (err && err.message) || String(err);
  show(el);
}

function appendLog(text) {
  const log = $("log");
  log.textContent += text;
  log.scrollTop = log.scrollHeight;
}

// --- connect ---------------------------------------------------------------

function selectAuth(mode) {
  authMode = mode;
  document.querySelectorAll(".tab").forEach((t) => t.classList.toggle("on", t.dataset.auth === mode));
  $("field-password").classList.toggle("hidden", mode !== "password");
  $("field-key").classList.toggle("hidden", mode !== "key");
}

function initAuthTabs() {
  document.querySelectorAll(".tab").forEach((tab) => {
    tab.addEventListener("click", () => selectAuth(tab.dataset.auth));
  });
}

// fillForm populates the connect form from a saved Prefs/Profile object.
function fillForm(p) {
  if (!p) return;
  $("host").value = p.host || "";
  $("user").value = p.user || "root";
  $("identity").value = p.identityPath || "";
  selectAuth(p.authMode || "password");
  $("remember").checked = !!p.remember;
  $("password").value = p.password || "";
}

// prefill restores the last-used connection (password from the OS secret store)
// and renders the saved-servers list.
async function prefill() {
  try {
    fillForm(await backend().LoadPrefs());
  } catch (_) {
    /* no saved prefs yet — ignore */
  }
  await loadProfiles();
}

// loadProfiles renders saved servers as one-click reconnect rows.
async function loadProfiles() {
  let list = [];
  try {
    list = (await backend().ListProfiles()) || [];
  } catch (_) {
    return;
  }
  const box = $("profiles");
  const ul = $("profiles-list");
  ul.innerHTML = "";
  if (list.length === 0) {
    hide(box);
    return;
  }
  list.forEach((p) => {
    const label = (p.user || "root") + "@" + p.host;
    const row = document.createElement("div");
    row.className = "profile";

    const pick = document.createElement("button");
    pick.className = "pick";
    pick.textContent = label;
    pick.addEventListener("click", () => {
      fillForm(p);
      connect();
    });

    const x = document.createElement("button");
    x.className = "x";
    x.textContent = "×";
    x.title = "Удалить из списка";
    x.addEventListener("click", async (e) => {
      e.stopPropagation();
      const ok = await confirmDialog(`Убрать сервер «${label}» из списка? Сам сервер не изменится.`, "Убрать");
      if (!ok) return;
      try {
        await backend().DeleteProfile(p.host, p.user);
      } catch (_) {
        /* ignore */
      }
      await loadProfiles();
    });

    row.append(pick, x);
    ul.appendChild(row);
  });
  show(box);
}

async function connect() {
  hide($("connect-err"));
  const host = $("host").value.trim();
  if (!host) { showError($("connect-err"), "Укажите IP-адрес сервера"); return; }

  const req = {
    host,
    user: $("user").value.trim() || "root",
    password: authMode === "password" ? $("password").value : "",
    identityPath: authMode === "key" ? $("identity").value.trim() : "",
    authMode,
    remember: $("remember").checked,
  };

  busy(true, "Подключаюсь к серверу…");
  try {
    await backend().Connect(req);
    $("conn-pill").textContent = req.user + "@" + host;
    show($("conn-pill"));
    hide($("view-connect"));
    show($("view-server"));
    await refreshStatus();
    loadProfiles();
  } catch (err) {
    showError($("connect-err"), err);
  } finally {
    busy(false);
  }
}

// --- server status: install vs manage --------------------------------------

async function refreshStatus() {
  busy(true, "Проверяю сервер…");
  try {
    const status = await backend().ServerStatus();
    hide($("result"));
    if (status.installed) {
      hide($("block-install"));
      show($("block-manage"));
      await refreshClients();
      await refreshPanel();
      await refreshHealth();
      startTraffic();
    } else {
      show($("block-install"));
      hide($("block-manage"));
      stopTraffic();
    }
  } catch (err) {
    toast("Не удалось проверить сервер: " + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

// --- install ---------------------------------------------------------------

async function install() {
  const req = {
    port: $("port").value.trim(),
    client: $("first-client").value.trim() || "phone",
  };
  $("log").textContent = "";
  $("log-title").textContent = "Установка";
  show($("log-panel"));
  $("btn-install").disabled = true;
  busy(true, "Устанавливаю AmneziaWG… это займёт пару минут");
  try {
    const res = await backend().Install(req);
    await refreshStatus();
    showResult(res);
    toast("AmneziaWG установлен", "ok");
  } catch (err) {
    toast("Установка не удалась: " + errMsg(err), "err");
  } finally {
    busy(false);
    $("btn-install").disabled = false;
  }
}

// --- manage clients --------------------------------------------------------

async function refreshClients() {
  const names = (await backend().ListClients()) || [];
  const body = $("clients-body");
  body.innerHTML = "";
  if (names.length === 0) {
    const tr = document.createElement("tr");
    tr.innerHTML = '<td colspan="2" class="empty">Пока нет клиентов. Добавьте первого выше.</td>';
    body.appendChild(tr);
    return;
  }
  names.forEach((name) => {
    const tr = document.createElement("tr");
    const nameTd = document.createElement("td");
    nameTd.className = "name";
    nameTd.textContent = name;
    const actTd = document.createElement("td");
    actTd.className = "r";
    actTd.innerHTML = '<div class="row-actions"></div>';
    const actions = actTd.querySelector(".row-actions");

    const conf = document.createElement("button");
    conf.className = "link";
    conf.textContent = "конфиг / QR";
    conf.addEventListener("click", () => showClientConfig(name));
    actions.appendChild(conf);

    const del = document.createElement("button");
    del.className = "link del";
    del.textContent = "удалить";
    del.addEventListener("click", () => removeClient(name));
    actions.appendChild(del);

    tr.append(nameTd, actTd);
    body.appendChild(tr);
  });
}

async function addClient(e) {
  e.preventDefault();
  const name = $("new-client").value.trim();
  if (!name) return;
  busy(true, "Создаю клиента…");
  try {
    const res = await backend().AddClient(name);
    $("new-client").value = "";
    await refreshClients();
    showResult(res);
  } catch (err) {
    toast("Не удалось создать клиента: " + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function showClientConfig(name) {
  busy(true, "Загружаю конфиг…");
  try {
    const res = await backend().ClientConfig(name);
    showResult(res);
  } catch (err) {
    toast("Не удалось получить конфиг: " + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function removeClient(name) {
  const ok = await confirmDialog(`Удалить клиента «${name}»? Его профиль перестанет работать.`);
  if (!ok) return;
  busy(true, "Удаляю клиента…");
  try {
    await backend().RemoveClient(name);
    await refreshClients();
    toast(`Клиент «${name}» удалён`, "ok");
  } catch (err) {
    toast("Не удалось удалить клиента: " + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function uninstall() {
  const ok = await confirmDialog(
    "Это ПОЛНОСТЬЮ удалит AmneziaWG, веб-панель, всех клиентов и конфиги с сервера. Продолжить?",
    "Удалить всё"
  );
  if (!ok) return;
  $("log").textContent = "";
  $("log-title").textContent = "Удаление";
  show($("log-panel"));
  busy(true, "Удаляю AmneziaWG…");
  try {
    await backend().Uninstall();
    await refreshStatus();
    toast("AmneziaWG полностью удалён", "ok");
  } catch (err) {
    toast("Не удалось удалить: " + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

// --- server health & live traffic ------------------------------------------

async function refreshHealth() {
  try {
    const h = await backend().ServerHealth();
    $("health-dot").className = "health-dot " + (h.running ? "up" : "down");
    $("health-state").textContent = h.running ? "VPN работает" : "VPN остановлен";
    $("health-clients").textContent = h.clients + " клиент(ов)";
    $("health-uptime").textContent = h.uptime || "—";
    $("health-version").textContent = h.version || "—";
  } catch (_) {
    $("health-dot").className = "health-dot";
    $("health-state").textContent = "статус недоступен";
  }
}

let trafficTimer = null;
let trafficBusy = false;

async function refreshTraffic() {
  if (trafficBusy) return;
  trafficBusy = true;
  try {
    const r = await backend().Traffic();
    $("traffic-summary").textContent = `(онлайн ${r.online} из ${r.total})`;
    const body = $("traffic-body");
    body.innerHTML = "";
    if (!r.peers || r.peers.length === 0) {
      body.innerHTML = '<tr><td colspan="5" class="traffic-empty">Пока нет данных</td></tr>';
      return;
    }
    r.peers.forEach((p) => {
      const tr = document.createElement("tr");
      const dotTd = document.createElement("td");
      dotTd.innerHTML = `<span class="tdot${p.online ? " on" : ""}"></span>`;
      const name = document.createElement("td");
      name.className = "name";
      name.textContent = p.name;
      const rx = document.createElement("td");
      rx.className = "r mono";
      rx.textContent = p.rx;
      const tx = document.createElement("td");
      tx.className = "r mono";
      tx.textContent = p.tx;
      const hs = document.createElement("td");
      hs.className = "r dim";
      hs.textContent = p.handshake;
      tr.append(dotTd, name, rx, tx, hs);
      body.appendChild(tr);
    });
  } catch (_) {
    /* transient — keep last values */
  } finally {
    trafficBusy = false;
  }
}

function startTraffic() {
  refreshTraffic();
  if (!trafficTimer) trafficTimer = setInterval(refreshTraffic, 5000);
}

function stopTraffic() {
  if (trafficTimer) {
    clearInterval(trafficTimer);
    trafficTimer = null;
  }
}

// --- web panel (advanced limits) -------------------------------------------

async function refreshPanel() {
  try {
    const p = await backend().PanelStatus();
    $("panel-url").textContent = p.url;
    $("panel-absent").classList.toggle("hidden", p.installed);
    $("panel-present").classList.toggle("hidden", !p.installed);
  } catch (err) {
    toast("Не удалось проверить панель: " + errMsg(err), "err");
  }
}

async function installPanel() {
  const pass = $("panel-pass").value;
  if (pass.length < 8) {
    toast("Пароль панели — минимум 8 символов", "err");
    return;
  }
  $("log").textContent = "";
  $("log-title").textContent = "Установка веб-панели";
  show($("log-panel"));
  busy(true, "Устанавливаю веб-панель…");
  try {
    await backend().InstallPanel(pass);
    $("panel-pass").value = "";
    await refreshPanel();
    toast("Веб-панель установлена", "ok");
  } catch (err) {
    toast("Не удалось установить панель: " + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function removePanel() {
  const ok = await confirmDialog("Удалить веб-панель с сервера? AmneziaWG и клиенты останутся.", "Удалить панель");
  if (!ok) return;
  busy(true, "Удаляю панель…");
  try {
    await backend().RemovePanel();
    await refreshPanel();
    toast("Веб-панель удалена", "ok");
  } catch (err) {
    toast("Не удалось удалить панель: " + errMsg(err), "err");
  } finally {
    busy(false);
  }
}

async function openPanel() {
  try {
    await backend().OpenPanel();
  } catch (err) {
    toast("Не удалось открыть браузер: " + errMsg(err), "err");
  }
}

// --- client result (QR + conf) ---------------------------------------------

function showResult(res) {
  lastResult = res;
  $("result-name").textContent = res.name;
  $("result-qr").src = res.qr || "";
  $("result-conf").textContent = res.conf || "";
  show($("result"));
  $("result").scrollIntoView({ behavior: "smooth", block: "start" });
}

function downloadConf() {
  if (!lastResult) return;
  const blob = new Blob([lastResult.conf], { type: "text/plain" });
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = lastResult.name + ".conf";
  a.click();
  URL.revokeObjectURL(a.href);
}

// --- wire up ---------------------------------------------------------------

window.addEventListener("DOMContentLoaded", () => {
  initAuthTabs();
  prefill();
  $("btn-connect").addEventListener("click", connect);
  $("password").addEventListener("keydown", (e) => { if (e.key === "Enter") connect(); });
  $("btn-install").addEventListener("click", install);
  $("add-form").addEventListener("submit", addClient);
  $("btn-uninstall").addEventListener("click", uninstall);
  $("btn-install-panel").addEventListener("click", installPanel);
  $("btn-open-panel").addEventListener("click", openPanel);
  $("btn-remove-panel").addEventListener("click", removePanel);
  $("result-close").addEventListener("click", () => hide($("result")));
  $("result-download").addEventListener("click", downloadConf);
  $("link-ios").addEventListener("click", (e) => { e.preventDefault(); openExternal(IOS_APP_URL); });
  $("link-other").addEventListener("click", (e) => { e.preventDefault(); openExternal(OTHER_APP_URL); });

  if (window.runtime) {
    window.runtime.EventsOn("install:log", appendLog);
    window.runtime.EventsOn("client:log", appendLog);
    window.runtime.EventsOn("panel:log", appendLog);
  }
});
