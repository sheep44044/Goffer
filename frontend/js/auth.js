(function (window) {
  "use strict";

  const TOKEN_KEY = "goffer.token";
  const USER_KEY = "goffer.username";
  let unauthorizedHandler = null;

  function getToken() {
    return localStorage.getItem(TOKEN_KEY) || "";
  }

  function getUsername() {
    return localStorage.getItem(USER_KEY) || "";
  }

  function setSession(token, username) {
    localStorage.setItem(TOKEN_KEY, token || "");
    localStorage.setItem(USER_KEY, username || "");
  }

  function clearSession() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
  }

  function isLoggedIn() {
    return Boolean(getToken());
  }

  function onUnauthorized(handler) {
    unauthorizedHandler = handler;
  }

  function handleUnauthorized() {
    clearSession();
    if (window.Goffer) window.Goffer.showToast("登录已过期，请重新登录", "error");
    if (unauthorizedHandler) unauthorizedHandler();
  }

  async function login(username, password) {
    const payload = await window.Goffer.request("/api/user/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    setSession(payload.data, username);
    return payload.data;
  }

  async function register(username, password) {
    return window.Goffer.request("/api/user/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
  }

  window.GofferAuth = {
    getToken,
    getUsername,
    setSession,
    clearSession,
    isLoggedIn,
    onUnauthorized,
    handleUnauthorized,
    login,
    register,
  };
})(window);
