/* URL Shortener admin UI: vanilla JS, no dependencies. */
(() => {
  "use strict";

  const $ = (id) => document.getElementById(id);

  const views = {
    login: $("login-view"),
    dashboard: $("dashboard-view"),
  };

  function show(name) {
    for (const [key, el] of Object.entries(views)) {
      el.hidden = key !== name;
    }
  }

  // fetch wrapper: any 401 bounces back to the login view
  async function api(path, options = {}) {
    const res = await fetch(path, {
      headers: { "Content-Type": "application/json" },
      ...options,
    });

    if (res.status === 401) {
      show("login");
      throw new Error("unauthorized");
    }

    return res;
  }

  let editing = null; // alias currently being edited

  // ---- messages -----------------------------------------------------------

  function showMsg(id, text) {
    const el = $(id);
    el.textContent = text;
    el.hidden = false;
  }

  function hideMsgs() {
    $("form-msg").hidden = true;
    $("form-ok").hidden = true;
  }

  // ---- link list ----------------------------------------------------------

  async function loadList() {
    const res = await api("/url");
    const body = await res.json();
    renderRows(body.items || []);
    $("empty-list").hidden = (body.items || []).length > 0;
  }

  function renderRows(items) {
    const tbody = $("links-body");
    tbody.textContent = "";

    for (const it of items) {
      const tr = document.createElement("tr");

      const alias = document.createElement("td");
      const a = document.createElement("a");
      a.href = "/" + encodeURIComponent(it.alias);
      a.textContent = it.alias;
      a.target = "_blank";
      a.rel = "noopener";
      alias.appendChild(a);

      const target = document.createElement("td");
      target.className = "target";
      target.textContent = it.url;
      target.title = it.url;

      const expires = document.createElement("td");
      if (it.expires_at) {
        const d = new Date(it.expires_at);
        expires.textContent = d.toLocaleString();
        if (d < new Date()) expires.classList.add("expired");
      } else {
        expires.textContent = "never";
      }

      const visits = document.createElement("td");
      visits.textContent = it.visits;

      const actions = document.createElement("td");
      actions.className = "actions";
      actions.append(
        btn("Stats", "secondary", () => showStats(it.alias)),
        btn("Edit", "secondary", () => startEdit(it)),
        btn("Delete", "danger", () => removeLink(it.alias)),
      );

      tr.append(alias, target, expires, visits, actions);
      tbody.appendChild(tr);
    }
  }

  function btn(label, cls, onClick) {
    const b = document.createElement("button");
    b.type = "button";
    b.textContent = label;
    b.className = cls;
    b.addEventListener("click", onClick);
    return b;
  }

  // ---- create / update ----------------------------------------------------

  $("link-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    hideMsgs();

    const url = $("f-url").value.trim();
    const ttlRaw = $("f-ttl").value;
    const ttl = ttlRaw === "" ? 0 : Number(ttlRaw);

    try {
      let res;
      if (editing) {
        res = await api("/url/" + encodeURIComponent(editing), {
          method: "PUT",
          body: JSON.stringify({ url, ttl_seconds: ttl }),
        });
      } else {
        res = await api("/url", {
          method: "POST",
          body: JSON.stringify({
            url,
            alias: $("f-alias").value.trim(),
            ttl_seconds: ttl,
          }),
        });
      }

      const body = await res.json();
      if (!res.ok) throw new Error(body.error || "request failed");

      const savedAlias = body.alias || editing;
      resetForm();
      await loadList();
      showMsg("form-ok", "Link saved: " + savedAlias);
    } catch (err) {
      if (err.message !== "unauthorized") showMsg("form-msg", err.message);
    }
  });

  function startEdit(it) {
    editing = it.alias;
    hideMsgs();
    resetStats();

    $("form-title").textContent = "Edit link: " + it.alias;
    $("f-alias").value = it.alias;
    $("f-alias").disabled = true;
    $("f-url").value = it.url;
    $("f-ttl").value = "";
    $("f-submit").textContent = "Save";
    $("f-cancel").hidden = false;
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function resetForm() {
    editing = null;
    $("link-form").reset();
    hideMsgs();
    resetStats();

    $("form-title").textContent = "Create a short link";
    $("f-alias").disabled = false;
    $("f-submit").textContent = "Create";
    $("f-cancel").hidden = true;
  }

  $("f-cancel").addEventListener("click", resetForm);

  async function removeLink(alias) {
    if (!confirm('Delete link "' + alias + '"?')) return;
    hideMsgs();

    try {
      const res = await api("/url/" + encodeURIComponent(alias), { method: "DELETE" });
      if (!res.ok) {
        const body = await res.json();
        throw new Error(body.error || "failed to delete");
      }

      await loadList();
    } catch (err) {
      if (err.message !== "unauthorized") showMsg("form-msg", err.message);
    }
  }

  // ---- stats --------------------------------------------------------------

  async function showStats(alias) {
    const res = await api("/url/" + encodeURIComponent(alias) + "/stats?days=30");
    const body = await res.json();

    $("stats-title").textContent = "Stats: " + alias;
    $("stats-summary").textContent =
      body.visits + " total visits" +
      (body.expires_at
        ? " · expires " + new Date(body.expires_at).toLocaleString()
        : " · never expires");

    $("stats-chart").innerHTML = chartSVG(body.visits_by_day || []);
    $("stats-panel").hidden = false;
    $("stats-panel").scrollIntoView({ behavior: "smooth", block: "nearest" });
  }

  function resetStats() {
    $("stats-panel").hidden = true;
  }

  $("stats-close").addEventListener("click", resetStats);

  function chartSVG(byDay) {
    const W = 720;
    const H = 200;
    const P = 24; // top/bottom padding, room for labels
    const max = Math.max(1, ...byDay.map((d) => d.count));
    const n = byDay.length || 1;
    const bw = (W - P) / n;

    let bars = "";
    byDay.forEach((d, i) => {
      const h = Math.round((d.count / max) * (H - P * 2));
      const x = P + i * bw;
      bars +=
        '<g><title>' + d.date + ": " + d.count + "</title>" +
        '<rect class="bar" x="' + (x + 1).toFixed(1) + '" y="' + (H - P - h).toFixed(1) +
        '" width="' + Math.max(bw - 2, 1).toFixed(1) + '" height="' + h + '" rx="2"/></g>';
    });

    let labels = "";
    const step = Math.ceil(n / 6);
    byDay.forEach((d, i) => {
      if (i % step === 0) {
        labels +=
          '<text class="lbl" text-anchor="middle" x="' + (P + i * bw + bw / 2).toFixed(1) +
          '" y="' + (H - 6) + '">' + d.date.slice(5) + "</text>";
      }
    });

    return '<svg class="chart" viewBox="0 0 ' + W + " " + H + '" role="img">' + bars + labels + "</svg>";
  }

  // ---- auth ---------------------------------------------------------------

  $("login-form").addEventListener("submit", async (e) => {
    e.preventDefault();
    $("login-error").hidden = true;

    const res = await fetch("/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        user: $("login-user").value,
        password: $("login-password").value,
      }),
    });

    if (res.ok) {
      await enter();
      return;
    }

    const body = await res.json().catch(() => ({}));
    showMsg("login-error", body.error || "invalid credentials");
  });

  $("logout-btn").addEventListener("click", async () => {
    await fetch("/logout", { method: "POST" });
    show("login");
  });

  async function enter() {
    await loadList(); // throws into the login view on 401
    show("dashboard");
  }

  enter(); // probe the session on page load
})();
