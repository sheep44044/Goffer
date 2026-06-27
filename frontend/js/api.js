(function (window) {
  "use strict";

  const API_BASE = "http://localhost:8080";
  const WS_URL = "ws://localhost:8890/ws";
  const STUN_SERVERS = ["stun:stun.l.google.com:19302"];
  const MAX_UPLOAD_SIZE = 10 * 1024 * 1024;

  function qs(selector, root) {
    return (root || document).querySelector(selector);
  }

  function qsa(selector, root) {
    return Array.from((root || document).querySelectorAll(selector));
  }

  function show(el) {
    const node = typeof el === "string" ? qs(el) : el;
    if (node) node.hidden = false;
  }

  function hide(el) {
    const node = typeof el === "string" ? qs(el) : el;
    if (node) node.hidden = true;
  }

  function setText(el, value) {
    const node = typeof el === "string" ? qs(el) : el;
    if (node) node.textContent = value == null ? "" : String(value);
  }

  function createEl(tag, className, text) {
    const el = document.createElement(tag);
    if (className) el.className = className;
    if (text != null) el.textContent = text;
    return el;
  }

  function safeJson(response) {
    return response.text().then((text) => {
      if (!text) return null;
      try {
        return JSON.parse(text);
      } catch (err) {
        return { code: -1, message: text || "响应解析失败", data: null };
      }
    });
  }

  function isSuccess(payload) {
    return payload && Number(payload.code) === 0;
  }

  function getMessage(payload, fallback) {
    if (!payload) return fallback || "请求失败";
    return payload.message || payload.error || fallback || "请求失败";
  }

  function showToast(message, type) {
    let toast = qs("#toast");
    if (!toast) {
      toast = createEl("div", "toast");
      toast.id = "toast";
      document.body.appendChild(toast);
    }
    toast.className = "toast show " + (type || "info");
    toast.textContent = message;
    window.clearTimeout(showToast.timer);
    showToast.timer = window.setTimeout(() => {
      toast.classList.remove("show");
    }, 3200);
  }

  function showAlert(target, message, type) {
    const el = typeof target === "string" ? qs(target) : target;
    if (!el) return;
    el.className = "alert alert-" + (type || "info");
    el.textContent = message;
    el.hidden = false;
  }

  function clearAlert(target) {
    const el = typeof target === "string" ? qs(target) : target;
    if (!el) return;
    el.textContent = "";
    el.hidden = true;
  }

  function setLoading(button, loading, label) {
    if (!button) return;
    if (loading) {
      button.dataset.originalText = button.textContent;
      button.disabled = true;
      button.classList.add("is-loading");
      button.textContent = label || "处理中...";
    } else {
      button.disabled = false;
      button.classList.remove("is-loading");
      if (button.dataset.originalText) {
        button.textContent = button.dataset.originalText;
        delete button.dataset.originalText;
      }
    }
  }

  function authHeaders(headers) {
    const merged = Object.assign({}, headers || {});
    const token = window.GofferAuth && window.GofferAuth.getToken();
    if (token) merged.Authorization = "Bearer " + token;
    return merged;
  }

  async function request(path, options) {
    const opts = Object.assign({}, options || {});
    opts.headers = authHeaders(opts.headers);
    const response = await fetch(API_BASE + path, opts);
    if (response.status === 401) {
      if (window.GofferAuth) window.GofferAuth.handleUnauthorized();
      throw new Error("登录已过期，请重新登录");
    }
    const payload = await safeJson(response);
    if (!response.ok) {
      throw new Error(getMessage(payload, "网络请求失败"));
    }
    if (payload && payload.code != null && Number(payload.code) !== 0) {
      if (Number(payload.code) === 10004 && window.GofferAuth) {
        window.GofferAuth.handleUnauthorized();
      }
      throw new Error(getMessage(payload));
    }
    return payload;
  }

  async function uploadFile(path, file) {
    const form = new FormData();
    form.append("file", file);
    return request(path, { method: "POST", body: form });
  }

  function validateFile(file, config) {
    if (!file) return "请选择文件";
    const opts = config || {};
    if (file.size > (opts.maxSize || MAX_UPLOAD_SIZE)) return "文件不能超过 10MB";
    if (opts.extensions && opts.extensions.length) {
      const lower = file.name.toLowerCase();
      const ok = opts.extensions.some((ext) => lower.endsWith(ext));
      if (!ok) return opts.message || "文件格式不支持";
    }
    if (opts.types && opts.types.length && file.type) {
      const okType = opts.types.includes(file.type);
      if (!okType) return opts.message || "文件格式不支持";
    }
    return "";
  }

  function bindDropZone(dropZone, fileInput, onFile) {
    if (!dropZone || !fileInput) return;
    dropZone.addEventListener("click", () => fileInput.click());
    dropZone.addEventListener("keydown", (event) => {
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        fileInput.click();
      }
    });
    ["dragenter", "dragover"].forEach((name) => {
      dropZone.addEventListener(name, (event) => {
        event.preventDefault();
        dropZone.classList.add("dragover");
      });
    });
    ["dragleave", "drop"].forEach((name) => {
      dropZone.addEventListener(name, (event) => {
        event.preventDefault();
        dropZone.classList.remove("dragover");
      });
    });
    dropZone.addEventListener("drop", (event) => {
      const file = event.dataTransfer.files && event.dataTransfer.files[0];
      if (file) onFile(file);
    });
    fileInput.addEventListener("change", () => {
      const file = fileInput.files && fileInput.files[0];
      if (file) onFile(file);
      fileInput.value = "";
    });
  }

  function parseSSEChunk(buffer, onEvent) {
    buffer = buffer.replace(/\r\n/g, "\n");
    let cursor = 0;
    while (true) {
      const next = buffer.indexOf("\n\n", cursor);
      if (next === -1) break;
      const raw = buffer.slice(cursor, next);
      cursor = next + 2;
      const event = { event: "message", data: "" };
      raw.split(/\n/).forEach((line) => {
        const trimmed = line.replace(/\r$/, "");
        if (trimmed.startsWith("event:")) event.event = trimmed.slice(6).trim();
        if (trimmed.startsWith("data:")) {
          const value = trimmed.slice(5).replace(/^ /, "");
          event.data = event.data ? event.data + "\n" + value : value;
        }
      });
      if (event.data !== "") onEvent(event);
    }
    return buffer.slice(cursor);
  }

  async function streamSSE(path, body, handlers) {
    const response = await fetch(API_BASE + path, {
      method: "POST",
      headers: authHeaders({ "Content-Type": "application/json" }),
      body: JSON.stringify(body || {}),
    });
    if (response.status === 401) {
      if (window.GofferAuth) window.GofferAuth.handleUnauthorized();
      throw new Error("登录已过期，请重新登录");
    }
    if (!response.ok || !response.body) {
      const payload = await safeJson(response);
      throw new Error(getMessage(payload, "流式请求失败"));
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = "";
    while (true) {
      const result = await reader.read();
      if (result.done) break;
      buffer += decoder.decode(result.value, { stream: true });
      buffer = parseSSEChunk(buffer, (event) => {
        if (event.data === "[DONE]" || event.event === "done") {
          if (handlers && handlers.done) handlers.done(event);
          return;
        }
        if (handlers && handlers.message) handlers.message(event.data, event);
      });
    }
    if (buffer.trim()) {
      parseSSEChunk(buffer + "\n\n", (event) => {
        if (event.data === "[DONE]" || event.event === "done") {
          if (handlers && handlers.done) handlers.done(event);
        } else if (handlers && handlers.message) {
          handlers.message(event.data, event);
        }
      });
    }
  }

  window.Goffer = {
    API_BASE,
    WS_URL,
    STUN_SERVERS,
    MAX_UPLOAD_SIZE,
    qs,
    qsa,
    show,
    hide,
    setText,
    createEl,
    safeJson,
    isSuccess,
    getMessage,
    showToast,
    showAlert,
    clearAlert,
    setLoading,
    request,
    uploadFile,
    validateFile,
    bindDropZone,
    parseSSEChunk,
    streamSSE,
  };
})(window);
