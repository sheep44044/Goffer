(function (window) {
  "use strict";

  const { qs, qsa, showAlert, clearAlert, showToast, setLoading, createEl } = window.Goffer;

  const state = {
    jdTags: [],
    questionTags: [],
  };

  function init() {
    bindAuth();
    bindTabs();
    bindTags();
    bindForms();
    bindUploads();
    window.GofferAuth.onUnauthorized(() => renderAuthState(false));
    renderAuthState(window.GofferAuth.isLoggedIn());
  }

  function bindAuth() {
    qs("#loginForm").addEventListener("submit", onLogin);
    qs("#logoutBtn").addEventListener("click", logout);
  }

  async function onLogin(event) {
    event.preventDefault();
    const username = qs("#loginUsername").value.trim();
    const password = qs("#loginPassword").value.trim();
    const button = qs("#loginBtn");
    clearAlert("#loginAlert");
    if (!username || !password) {
      showAlert("#loginAlert", "请输入用户名和密码", "error");
      return;
    }
    setLoading(button, true, "登录中...");
    try {
      await window.GofferAuth.login(username, password);
      renderAuthState(true);
      showToast("登录成功", "success");
    } catch (err) {
      showAlert("#loginAlert", err.message, "error");
    } finally {
      setLoading(button, false);
    }
  }

  function logout() {
    window.GofferAuth.clearSession();
    renderAuthState(false);
  }

  function renderAuthState(isLoggedIn) {
    qs("#loginSection").hidden = isLoggedIn;
    qs("#adminShell").hidden = !isLoggedIn;
    qs("#logoutBtn").hidden = !isLoggedIn;
    qs("#userBadge").hidden = !isLoggedIn;
    qs("#userBadge").textContent = isLoggedIn ? window.GofferAuth.getUsername() || "管理员" : "";
  }

  function bindTabs() {
    qsa("[data-tab]").forEach((button) => {
      button.addEventListener("click", () => {
        const tab = button.dataset.tab;
        qsa("[data-tab]").forEach((item) => item.classList.toggle("active", item === button));
        qsa("[data-panel]").forEach((panel) => {
          panel.hidden = panel.dataset.panel !== tab;
        });
      });
    });
  }

  function bindTags() {
    qs("#jdTagInput").addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        addTag("jd");
      }
    });
    qs("#questionTagInput").addEventListener("keydown", (event) => {
      if (event.key === "Enter") {
        event.preventDefault();
        addTag("question");
      }
    });
    qs("#addJdTagBtn").addEventListener("click", () => addTag("jd"));
    qs("#addQuestionTagBtn").addEventListener("click", () => addTag("question"));
  }

  function addTag(type) {
    const input = type === "jd" ? qs("#jdTagInput") : qs("#questionTagInput");
    const list = type === "jd" ? state.jdTags : state.questionTags;
    const value = input.value.trim();
    if (!value || list.includes(value)) return;
    list.push(value);
    input.value = "";
    renderTags(type);
  }

  function removeTag(type, index) {
    const list = type === "jd" ? state.jdTags : state.questionTags;
    list.splice(index, 1);
    renderTags(type);
  }

  function renderTags(type) {
    const list = type === "jd" ? state.jdTags : state.questionTags;
    const box = type === "jd" ? qs("#jdTags") : qs("#questionTags");
    box.replaceChildren();
    list.forEach((tag, index) => {
      const item = createEl("span", "tag", tag);
      const button = createEl("button", "", "x");
      button.type = "button";
      button.setAttribute("aria-label", "删除标签 " + tag);
      button.addEventListener("click", () => removeTag(type, index));
      item.appendChild(button);
      box.appendChild(item);
    });
  }

  function bindForms() {
    qs("#jdForm").addEventListener("submit", submitJD);
    qs("#questionForm").addEventListener("submit", submitQuestion);
  }

  async function submitJD(event) {
    event.preventDefault();
    const button = qs("#jdSubmitBtn");
    clearAlert("#jdAlert");
    const body = {
      company: qs("#jdCompany").value.trim(),
      title: qs("#jdTitle").value.trim(),
      responsibilities: qs("#jdResponsibilities").value.trim(),
      requirements: qs("#jdRequirements").value.trim(),
      tags: state.jdTags,
    };
    if (!body.company || !body.title || !body.responsibilities || !body.requirements) {
      showAlert("#jdAlert", "请填写公司、岗位、职责和要求", "error");
      return;
    }
    setLoading(button, true, "提交中...");
    try {
      const payload = await window.Goffer.request("/api/knowledge/jd/ingest", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = payload.data || {};
      showAlert("#jdAlert", "JD 录入成功" + (data.jd_id ? "，ID: " + data.jd_id : ""), "success");
      qs("#jdForm").reset();
      state.jdTags = [];
      renderTags("jd");
    } catch (err) {
      showAlert("#jdAlert", err.message, "error");
    } finally {
      setLoading(button, false);
    }
  }

  async function submitQuestion(event) {
    event.preventDefault();
    const button = qs("#questionSubmitBtn");
    clearAlert("#questionAlert");
    const difficulty = qs("#questionDifficulty").value;
    const body = {
      question_content: qs("#questionContent").value.trim(),
      standard_answer: qs("#questionAnswer").value.trim(),
      difficulty,
      tags: state.questionTags,
    };
    if (!body.question_content || !body.standard_answer) {
      showAlert("#questionAlert", "请填写题目内容和标准答案", "error");
      return;
    }
    setLoading(button, true, "提交中...");
    try {
      const payload = await window.Goffer.request("/api/knowledge/question/ingest", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      const data = payload.data || {};
      showAlert("#questionAlert", "题目录入成功" + (data.question_id ? "，ID: " + data.question_id : ""), "success");
      qs("#questionForm").reset();
      qs("#questionDifficulty").value = "中等";
      state.questionTags = [];
      renderTags("question");
    } catch (err) {
      showAlert("#questionAlert", err.message, "error");
    } finally {
      setLoading(button, false);
    }
  }

  function bindUploads() {
    window.Goffer.bindDropZone(qs("#jdCsvDrop"), qs("#jdCsvInput"), (file) => uploadCSV("jd", file));
    window.Goffer.bindDropZone(qs("#questionCsvDrop"), qs("#questionCsvInput"), (file) => uploadCSV("question", file));
  }

  async function uploadCSV(type, file) {
    const error = window.Goffer.validateFile(file, {
      extensions: [".csv"],
      types: ["text/csv", "application/vnd.ms-excel"],
      message: "请上传 CSV 文件",
    });
    const alertSelector = type === "jd" ? "#jdCsvAlert" : "#questionCsvAlert";
    const button = type === "jd" ? qs("#jdCsvState") : qs("#questionCsvState");
    const endpoint = type === "jd" ? "/api/knowledge/jd/upload" : "/api/knowledge/question/upload";
    if (error) {
      showAlert(alertSelector, error, "error");
      return;
    }
    clearAlert(alertSelector);
    button.hidden = false;
    button.textContent = "上传中...";
    try {
      const payload = await window.Goffer.uploadFile(endpoint, file);
      const data = payload.data || {};
      const parts = ["上传成功"];
      if (data.taskID || data.task_id) parts.push("TaskID: " + (data.taskID || data.task_id));
      if (data.fileURL || data.file_url) parts.push("FileURL: " + (data.fileURL || data.file_url));
      showAlert(alertSelector, parts.join("，"), "success");
    } catch (err) {
      showAlert(alertSelector, err.message, "error");
    } finally {
      button.hidden = true;
    }
  }

  window.GofferAdmin = { init };
  document.addEventListener("DOMContentLoaded", init);
})(window);
