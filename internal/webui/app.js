"use strict";

(() => {
  const $ = (selector, root = document) => root.querySelector(selector);
  const el = (tag, className, text) => {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined) node.textContent = String(text);
    return node;
  };
  let csrfToken = "";
  let projects = [];
  let noticeTimer;
  const pendingKeyStorage = "asteriade.pending_mutation_keys.v1";
  const pendingMutationKeys = new Map();

  function restorePendingKeys() {
    try {
      const records = JSON.parse(sessionStorage.getItem(pendingKeyStorage) || "[]");
      if (Array.isArray(records)) {
        for (const record of records) {
          if (record && typeof record.digest === "string" && typeof record.key === "string") pendingMutationKeys.set(record.digest, record.key);
        }
      }
    } catch { /* storage may be disabled; in-memory retry still works */ }
  }
  function persistPendingKeys() {
    try {
      sessionStorage.setItem(pendingKeyStorage, JSON.stringify(Array.from(pendingMutationKeys, ([digest, key]) => ({ digest, key }))));
    } catch { /* do not block a local request when storage is unavailable */ }
  }
  async function mutationIdentity(path, bodyText) {
    const bytes = new TextEncoder().encode(`${path}\n${bodyText}`);
    const digestBuffer = await crypto.subtle.digest("SHA-256", bytes);
    const digest = Array.from(new Uint8Array(digestBuffer), byte => byte.toString(16).padStart(2, "0")).join("");
    let key = pendingMutationKeys.get(digest);
    if (!key) {
      key = crypto.randomUUID();
      pendingMutationKeys.set(digest, key);
      persistPendingKeys();
    }
    return { digest, key };
  }

  function notify(message) {
    const node = $("#notice");
    node.textContent = message;
    node.hidden = false;
    clearTimeout(noticeTimer);
    noticeTimer = setTimeout(() => { node.hidden = true; }, 6000);
  }
  function showFatal(message) {
    const node = $("#fatal");
    node.textContent = message;
    node.hidden = false;
  }
  function apiError(status, body) {
    const detail = body && typeof body.error === "string" ? body.error : "サーバーから有効なエラー情報がありません";
    const code = body && typeof body.code === "string" ? ` (${body.code})` : "";
    const error = new Error(`HTTP ${status}${code}: ${detail}`);
    error.status = status;
    return error;
  }
  async function request(path, options = {}) {
    const headers = new Headers(options.headers || {});
    if (options.body !== undefined) headers.set("Content-Type", "application/json");
    if (options.method && options.method !== "GET") {
      if (!csrfToken) throw new Error("セッションがありません。ページを再読み込みしてください。");
      headers.set("X-CSRF-Token", csrfToken);
      if (options.idempotencyKey) headers.set("Idempotency-Key", options.idempotencyKey);
    }
    const response = await fetch(path, { ...options, headers, credentials: "same-origin" });
    let data = null;
    const raw = await response.text();
    if (raw) {
      try { data = JSON.parse(raw); }
      catch { if (response.ok) throw new Error(`HTTP ${response.status}: サーバー応答を読み取れません`); }
    }
    if (!response.ok) throw apiError(response.status, data);
    return data;
  }
  function statusPill(value) {
    const status = value || "unknown";
    let kind = "";
    if (["ready", "available", "connected", "granted", "active", "passed"].includes(status)) kind = "ready";
    else if (["blocked", "unavailable", "failed", "denied", "error", "not_ready"].includes(status)) kind = "blocked";
    else if (["pending", "paused", "awaiting_approval"].includes(status)) kind = "pending";
    return el("span", `pill ${kind}`, status);
  }
  function empty(message) { return el("p", "empty", message); }
  function labeled(label, value) {
    const frag = document.createDocumentFragment();
    frag.append(el("dt", "", label), el("dd", "", value || "—"));
    return frag;
  }
  function errorInto(container, error) {
    container.replaceChildren(empty(`読み込みに失敗しました: ${error.message}`));
  }

  async function loadDiagnostics() {
    const container = $("#diagnostics");
    container.replaceChildren(el("p", "muted", "診断を読み込んでいます…"));
    try {
      const data = await request("/api/v1/diagnostics");
      const cards = [];
      const runner = data.runner || {};
      const runnerCard = el("article", "diag-card");
      runnerCard.append(el("h3", "", `Runner · ${runner.os || "OS不明"}`));
      const runnerState = el("p");
      runnerState.append(statusPill(runner.status), document.createTextNode(` ${runner.reason || "理由の情報なし"}`));
      runnerCard.append(runnerState);
      runnerCard.append(el("p", "muted", `Fingerprint: ${runner.fingerprint || "未取得"}`));
      const caps = runner.capabilities && typeof runner.capabilities === "object" ? Object.entries(runner.capabilities) : [];
      runnerCard.append(el("p", "muted", caps.length ? `Capabilities: ${caps.map(([key, value]) => `${key}=${value ? "yes" : "no"}`).join(" · ")}` : "Capabilities: 未確認"));
      cards.push(runnerCard);
      for (const provider of Array.isArray(data.providers) ? data.providers : []) {
        const card = el("article", "diag-card");
        card.append(el("h3", "", provider.name || "Provider"));
        const state = el("p");
        state.append(statusPill(provider.status), document.createTextNode(` ${provider.reason || "理由の情報なし"}`));
        card.append(state);
        cards.push(card);
      }
      const stage = el("article", "diag-card");
      stage.append(el("h3", "", `Stage ${data.stage || "?"}`));
      stage.append(el("p", "", data.execution_enabled === true ? "execution_enabled=true" : "execution_enabled=false"));
      stage.append(el("p", "muted", "実行許可はこの画面で変更できません。"));
      cards.push(stage);
      container.replaceChildren(...cards);
    } catch (error) { errorInto(container, error); }
  }

  function renderProjectOptions() {
    const select = $("#pause-project");
    const current = select.value;
    select.replaceChildren(new Option("選択してください", ""));
    for (const project of projects) select.add(new Option(project.title || project.id, project.id));
    if (projects.some(project => project.id === current)) select.value = current;
  }
  async function loadProjects() {
    const container = $("#projects");
    container.replaceChildren(el("p", "muted", "読み込み中…"));
    try {
      const data = await request("/api/v1/projects");
      projects = Array.isArray(data.projects) ? data.projects : [];
      $("#project-count").textContent = `${projects.length} 件`;
      renderProjectOptions();
      if (!projects.length) { container.replaceChildren(empty("保存したプロジェクトはありません。")); return; }
      const cards = projects.map(project => {
        const card = el("article", "project-card");
        const head = el("div", "card-head");
        const title = el("h3", "", project.title || "名称なし");
        const meta = el("div", "project-meta");
        meta.append(statusPill(project.status || project.lifecycle_status), el("span", "muted", `Version ${project.version ?? "?"}`));
        head.append(title, meta);
        card.append(head, el("p", "project-goal", project.goal || "目標なし"));
        const actions = el("div", "actions");
        const eventsButton = el("button", "button secondary", "イベントを表示");
        eventsButton.type = "button";
        eventsButton.addEventListener("click", () => loadEvents(project, card));
        const cancelButton = el("button", "button danger", "キャンセル");
        cancelButton.type = "button";
        cancelButton.disabled = ["cancelled", "completed", "failed"].includes(project.lifecycle_status || project.status);
        cancelButton.addEventListener("click", () => mutateAndRefresh(`/api/v1/projects/${encodeURIComponent(project.id)}/cancel`, { expected_version: project.version }, "キャンセル要求を記録しました。最終状態は一覧とイベントで確認してください。", cancelButton));
        const interruptButton = el("button", "button secondary", "即時停止を要求");
        interruptButton.type = "button";
        interruptButton.disabled = ["cancelled", "completed", "failed"].includes(project.lifecycle_status || project.status);
        interruptButton.addEventListener("click", () => mutateAndRefresh(`/api/v1/projects/${encodeURIComponent(project.id)}/interrupt`, { expected_version: project.version }, "即時停止要求を記録しました。停止完了は別途イベントで確認してください。", interruptButton));
        const startButton = el("button", "button secondary", "開始（Stage 0 では無効）");
        startButton.type = "button";
        startButton.disabled = true;
        actions.append(eventsButton, interruptButton, cancelButton, startButton);
        card.append(actions);
        return card;
      });
      container.replaceChildren(...cards);
    } catch (error) { errorInto(container, error); }
  }
  async function loadEvents(project, card) {
    let output = $(".event-output", card);
    if (!output) { output = el("div", "event-output"); card.append(output); }
    output.replaceChildren(el("p", "muted", "イベントを読み込んでいます…"));
    try {
      const data = await request(`/api/v1/projects/${encodeURIComponent(project.id)}/events?after=0`);
      const events = Array.isArray(data.events) ? data.events : [];
      if (!events.length) { output.replaceChildren(empty("イベントはありません。")); return; }
      const list = el("ol", "event-list");
      for (const event of events) {
        const item = el("li");
        item.append(el("strong", "", `#${event.sequence} ${event.type || "event"}`));
        item.append(el("span", "muted", ` · ${event.occurred_at || "時刻不明"}`));
        if (event.payload) item.append(el("pre", "event-payload", event.payload));
        list.append(item);
      }
      output.replaceChildren(list);
    } catch (error) { errorInto(output, error); }
  }
  async function loadApprovals() {
    const container = $("#approvals");
    container.replaceChildren(el("p", "muted", "読み込み中…"));
    try {
      const data = await request("/api/v1/approvals");
      const approvals = Array.isArray(data.approvals) ? data.approvals : [];
      if (!approvals.length) { container.replaceChildren(empty("承認待ちの要求はありません。")); return; }
      const cards = approvals.map(approval => {
        const operation = approval.operation || {};
        const card = el("article", "approval-card");
        const head = el("div", "card-head");
        head.append(el("h3", "", `${operation.kind || "操作"} · ${approval.scope || "scope不明"}`), statusPill(approval.status));
        card.append(head);
        const details = el("dl", "approval-details");
        details.append(labeled("送信先", operation.destination), labeled("アカウント", operation.account), labeled("対象", operation.target), labeled("内容 hash", approval.operation_hash || operation.payload_hash), labeled("期限", approval.expires_at), labeled("見積費用", operation.estimated_cost_micro_jpy === undefined ? "—" : `${operation.estimated_cost_micro_jpy} micro-JPY`));
        card.append(details);
        if (approval.status === "pending") {
          const actions = el("div", "actions");
          for (const decision of ["grant", "deny"]) {
            const button = el("button", `button ${decision === "deny" ? "danger" : "primary"}`, decision === "grant" ? "承認" : "拒否");
            button.type = "button";
            button.addEventListener("click", () => mutateAndRefresh(`/api/v1/approvals/${encodeURIComponent(approval.id)}/decision`, { expected_version: approval.version, operation_hash: approval.operation_hash, decision }, decision === "grant" ? "承認を記録しました" : "拒否を記録しました", button));
            actions.append(button);
          }
          card.append(actions);
        }
        return card;
      });
      container.replaceChildren(...cards);
    } catch (error) { errorInto(container, error); }
  }
  async function loadHolds() {
    const container = $("#holds");
    container.replaceChildren(el("p", "muted", "読み込み中…"));
    try {
      const data = await request("/api/v1/holds");
      const holds = Array.isArray(data.holds) ? data.holds : [];
      const active = holds.filter(hold => !hold.released_at);
      if (!active.length) { container.replaceChildren(empty("有効な一時停止はありません。")); return; }
      container.replaceChildren(...active.map(hold => {
        const row = el("article", "hold-card");
        const description = el("div");
        description.append(el("strong", "", hold.scope === "global" ? "全体" : `プロジェクト ${hold.project_id || "不明"}`), el("p", "muted", hold.reason || "理由の記録なし"));
        const resume = el("button", "button secondary", "再開");
        resume.type = "button";
        resume.addEventListener("click", () => mutateAndRefresh(`/api/v1/holds/${encodeURIComponent(hold.id)}/resume`, { expected_version: hold.version }, "再開を記録しました", resume));
        row.append(description, resume);
        return row;
      }));
    } catch (error) { errorInto(container, error); }
  }
  async function mutateAndRefresh(path, body, successMessage, button) {
    const bodyText = JSON.stringify(body);
    const originalDisabled = button ? button.disabled : false;
    if (button) button.disabled = true;
    let identity;
    try {
      identity = await mutationIdentity(path, bodyText);
      await request(path, { method: "POST", body: bodyText, idempotencyKey: identity.key });
      pendingMutationKeys.delete(identity.digest);
      persistPendingKeys();
      notify(successMessage);
      await refreshAll();
    } catch (error) {
      if (identity && (error.status === undefined || error.status >= 500)) {
        notify(`結果未確認です。${error.message} 同じ内容で再試行するか、一覧・イベントを確認してください。再試行には同じ冪等キーを使います。`);
      } else {
        if (identity) { pendingMutationKeys.delete(identity.digest); persistPendingKeys(); }
        notify(`操作に失敗しました: ${error.message}`);
      }
    }
    finally { if (button) button.disabled = originalDisabled; }
  }
  async function refreshAll() {
    await Promise.all([loadDiagnostics(), loadProjects(), loadApprovals(), loadHolds()]);
  }
  async function initialize() {
    const fragment = new URLSearchParams(location.hash.slice(1));
    const bootstrapToken = fragment.get("bootstrap");
    if (bootstrapToken) {
      history.replaceState(null, "", `${location.pathname}${location.search}`);
      try {
        const response = await fetch("/api/v1/session", {
          method: "POST", credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ token: bootstrapToken })
        });
        let data = null;
        try { data = await response.json(); } catch { /* reported below */ }
        if (!response.ok) throw apiError(response.status, data);
      } catch (error) { showFatal(`初期セッションを取得できませんでした: ${error.message}`); return; }
    } else if (location.hash) {
      history.replaceState(null, "", `${location.pathname}${location.search}`);
    }
    try {
      const session = await request("/api/v1/session");
      if (!session || typeof session.csrf_token !== "string" || !session.csrf_token) throw new Error("CSRF token が応答にありません");
      csrfToken = session.csrf_token;
    } catch (error) { showFatal(`セッションを確認できませんでした: ${error.message}`); return; }
    await refreshAll();
  }

  $("#project-form").addEventListener("submit", async event => {
    event.preventDefault();
    const form = event.currentTarget;
    const submit = $("button[type=submit]", form);
    submit.disabled = true;
    try {
      const body = { title: form.elements.title.value.trim(), goal: form.elements.goal.value.trim(), execution_mode: "plan_confirmation" };
      const bodyText = JSON.stringify(body);
      const identity = await mutationIdentity("/api/v1/projects", bodyText);
      try {
        await request("/api/v1/projects", { method: "POST", body: bodyText, idempotencyKey: identity.key });
        pendingMutationKeys.delete(identity.digest);
        persistPendingKeys();
      } catch (error) {
        if (error.status === undefined || error.status >= 500) {
          notify(`結果未確認です。${error.message} 同じ内容で再試行するか、一覧を確認してください。再試行には同じ冪等キーを使います。`);
        } else {
          pendingMutationKeys.delete(identity.digest);
          persistPendingKeys();
          notify(`保存に失敗しました: ${error.message}`);
        }
        return;
      }
      form.reset();
      notify("目標を保存しました。実行は開始していません。");
      await refreshAll();
    } catch (error) { notify(`保存に失敗しました: ${error.message}`); }
    finally { submit.disabled = false; }
  });
  $("#pause-form").addEventListener("submit", event => {
    event.preventDefault();
    const form = event.currentTarget;
    const scope = form.elements.scope.value;
    const projectId = form.elements.project_id.value;
    if (scope === "project" && !projectId) { notify("一時停止するプロジェクトを選択してください。"); return; }
    const submit = $("button[type=submit]", form);
    mutateAndRefresh("/api/v1/scheduler/pause", { scope, ...(scope === "project" ? { project_id: projectId } : {}) }, "一時停止を記録しました", submit);
  });
  $("#refresh-button").addEventListener("click", refreshAll);
  restorePendingKeys();
  initialize();
})();
