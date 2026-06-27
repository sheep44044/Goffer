(function (window) {
  "use strict";

  const { qs, qsa, show, hide, setText, createEl, showToast, showAlert, clearAlert, setLoading } = window.Goffer;

  const stageOrder = ["greeting", "tech_foundation", "tech_architecture", "evaluator"];
  const stageFallbackRounds = {
    greeting: 1,
    tech_foundation: 4,
    tech_architecture: 4,
    evaluator: 2,
  };

  const state = {
    resumeId: "",
    resumeFileName: "",
    resumeFileURL: "",
    sessionId: "",
    chatBusy: false,
    currentAiBubble: null,
    currentAiText: "",
    fsmState: "greeting",
    fsmRound: 0,
    interruptCount: 0,
    voiceMode: false,
    recognition: null,
    synthUtterance: null,
    webrtc: null,
  };

  function init() {
    bindAuth();
    bindResumeUpload();
    bindChatInput();
    bindButtons();
    window.GofferAuth.onUnauthorized(() => renderAuthState(false));
    renderAuthState(window.GofferAuth.isLoggedIn());
    renderSpeechSupport();
  }

  function bindAuth() {
    qsa("[data-auth-tab]").forEach((button) => {
      button.addEventListener("click", () => switchAuthTab(button.dataset.authTab));
    });
    qs("#loginForm").addEventListener("submit", onLogin);
    qs("#registerForm").addEventListener("submit", onRegister);
    qs("#logoutBtn").addEventListener("click", logout);
  }

  function switchAuthTab(tab) {
    qsa("[data-auth-tab]").forEach((button) => button.classList.toggle("active", button.dataset.authTab === tab));
    qs("#loginPane").hidden = tab !== "login";
    qs("#registerPane").hidden = tab !== "register";
    clearAlert("#authAlert");
  }

  async function onRegister(event) {
    event.preventDefault();
    const username = qs("#registerUsername").value.trim();
    const password = qs("#registerPassword").value.trim();
    const button = qs("#registerBtn");
    clearAlert("#authAlert");
    if (!username || !password) {
      showAlert("#authAlert", "请输入用户名和密码", "error");
      return;
    }
    setLoading(button, true, "注册中...");
    try {
      await window.GofferAuth.register(username, password);
      showAlert("#authAlert", "注册成功，请登录", "success");
      qs("#loginUsername").value = username;
      qs("#loginPassword").value = password;
      switchAuthTab("login");
    } catch (err) {
      showAlert("#authAlert", err.message, "error");
    } finally {
      setLoading(button, false);
    }
  }

  async function onLogin(event) {
    event.preventDefault();
    const username = qs("#loginUsername").value.trim();
    const password = qs("#loginPassword").value.trim();
    const button = qs("#loginBtn");
    clearAlert("#authAlert");
    if (!username || !password) {
      showAlert("#authAlert", "请输入用户名和密码", "error");
      return;
    }
    setLoading(button, true, "登录中...");
    try {
      await window.GofferAuth.login(username, password);
      renderAuthState(true);
      showToast("登录成功", "success");
    } catch (err) {
      showAlert("#authAlert", err.message, "error");
    } finally {
      setLoading(button, false);
    }
  }

  function logout() {
    endInterview();
    window.GofferAuth.clearSession();
    resetResume();
    renderAuthState(false);
  }

  function renderAuthState(isLoggedIn) {
    qs("#authSection").hidden = isLoggedIn;
    qs("#candidateShell").hidden = !isLoggedIn;
    qs("#logoutBtn").hidden = !isLoggedIn;
    setText("#userBadge", isLoggedIn ? window.GofferAuth.getUsername() || "已登录" : "");
    qs("#userBadge").hidden = !isLoggedIn;
    if (isLoggedIn) renderSetupState();
  }

  function resetResume() {
    state.resumeId = "";
    state.resumeFileName = "";
    state.resumeFileURL = "";
    qs("#resumeResult").hidden = true;
    setText("#resumeName", "");
    setText("#resumeID", "");
    setText("#resumeURL", "");
  }

  function bindResumeUpload() {
    window.Goffer.bindDropZone(qs("#resumeDrop"), qs("#resumeInput"), handleResumeFile);
  }

  async function handleResumeFile(file) {
    const error = window.Goffer.validateFile(file, {
      extensions: [".pdf", ".jpg", ".jpeg", ".png"],
      types: ["application/pdf", "image/jpeg", "image/png"],
      message: "仅支持 PDF / JPG / JPEG / PNG 简历",
    });
    if (error) {
      showAlert("#resumeAlert", error, "error");
      return;
    }
    clearAlert("#resumeAlert");
    const button = qs("#resumeUploadState");
    setText(button, "上传中...");
    button.hidden = false;
    try {
      const payload = await window.Goffer.uploadFile("/api/resume/upload", file);
      const data = payload.data || {};
      state.resumeId = data.resumeID || data.resume_id || data.resumeId || "";
      state.resumeFileName = file.name;
      state.resumeFileURL = data.fileURL || data.file_url || "";
      if (!state.resumeId) throw new Error("后端未返回 resumeID");
      setText("#resumeName", state.resumeFileName);
      setText("#resumeID", state.resumeId);
      setText("#resumeURL", state.resumeFileURL || "未返回");
      qs("#resumeResult").hidden = false;
      showAlert("#resumeAlert", "简历上传成功，可以开始面试", "success");
      renderSetupState();
    } catch (err) {
      showAlert("#resumeAlert", err.message, "error");
    } finally {
      button.hidden = true;
    }
  }

  function bindButtons() {
    qs("#startInterviewBtn").addEventListener("click", startInterview);
    qs("#endInterviewBtn").addEventListener("click", endInterview);
    qs("#muteBtn").addEventListener("click", toggleMute);
    qs("#interruptBtn").addEventListener("click", () => interrupt("manual"));
    qs("#voiceModeBtn").addEventListener("click", toggleVoiceMode);
  }

  function renderSetupState() {
    qs("#startInterviewBtn").disabled = !state.resumeId;
    qs("#setupEmpty").hidden = Boolean(state.resumeId);
  }

  function normalizeStartResponse(payload) {
    const data = payload && payload.data !== undefined ? payload.data : payload;
    let sessionId = "";
    let opening = "";
    if (typeof data === "string") {
      opening = data;
    } else if (data && typeof data === "object") {
      sessionId = data.session_id || data.sessionID || data.sessionId || "";
      opening = data.opening_remark || data.openingRemark || data.opening || "";
    }
    if (!sessionId && payload && typeof payload === "object") {
      sessionId = payload.session_id || payload.sessionID || payload.sessionId || "";
    }
    if (!opening && payload && typeof payload === "object") {
      opening = payload.opening_remark || payload.openingRemark || payload.opening || "";
    }
    return {
      sessionId: sessionId || "front-" + Date.now(),
      opening: opening || "你好，我是本次 AI 面试官。我们先从你的项目经历开始。",
    };
  }

  async function startInterview() {
    if (!state.resumeId) {
      showAlert("#startAlert", "请先上传简历", "error");
      return;
    }
    clearAlert("#startAlert");
    const button = qs("#startInterviewBtn");
    setLoading(button, true, "启动中...");
    try {
      const payload = await window.Goffer.request("/api/interview/start", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ resume_id: state.resumeId }),
      });
      const startData = normalizeStartResponse(payload);
      state.sessionId = startData.sessionId;
      state.fsmState = "greeting";
      state.fsmRound = 0;
      state.interruptCount = 0;
      qs("#setupSection").hidden = true;
      qs("#chatSection").hidden = false;
      qs("#chatBox").replaceChildren();
      renderProgress();
      addMessage("system", "会话已创建，Session: " + state.sessionId);
      addMessage("ai", startData.opening);
      restoreSession();
      startVoiceTransport();
    } catch (err) {
      showAlert("#startAlert", err.message, "error");
    } finally {
      setLoading(button, false);
    }
  }

  async function restoreSession() {
    if (!state.sessionId) return;
    try {
      const payload = await window.Goffer.request("/api/interview/resume", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ session_id: state.sessionId }),
      });
      const data = payload.data || {};
      if (data.fsm_state) state.fsmState = data.fsm_state;
      if (Number.isFinite(Number(data.round))) state.fsmRound = Number(data.round);
      renderProgress();
    } catch (err) {
      addMessage("system", "会话恢复接口暂不可用，已使用前端本地进度估算");
    }
  }

  function startVoiceTransport() {
    if (state.webrtc) state.webrtc.stop();
    state.webrtc = new window.GofferWebRTC({
      remoteAudio: qs("#remoteAudio"),
      onStateChange: updateVoiceState,
      onMicLevel: (level) => {
        qs("#micLevel").style.width = level + "%";
      },
      onDataChannel: (ready) => {
        qs("#interruptBtn").disabled = !ready;
      },
      onInterrupt: () => markInterrupted(),
      onReconnect: restoreSession,
      onError: (err) => addMessage("system", "语音链路提示：" + err.message),
    });
    state.webrtc.start({
      roomId: state.sessionId,
      userId: window.GofferAuth.getUsername() || "candidate",
    });
  }

  function updateVoiceState(status, label) {
    const dot = qs("#voiceDot");
    const text = qs("#voiceStateText");
    dot.className = "status-dot " + status;
    setText(text, label);
    qs("#voiceReconnect").hidden = status !== "reconnecting";
    qs("#muteBtn").disabled = !(status === "connected" || status === "connecting");
  }

  function toggleMute() {
    if (!state.webrtc) return;
    const muted = state.webrtc.toggleMuted();
    setText("#muteBtn", muted ? "取消静音" : "静音");
  }

  function interrupt(source) {
    if (!state.webrtc || !state.webrtc.canInterrupt()) {
      addMessage("system", "语音控制通道未连接，当前只能等待文本回复结束");
      return;
    }
    if (state.webrtc.sendCancel(source)) markInterrupted();
  }

  function markInterrupted() {
    state.interruptCount += 1;
    stopTTS();
    const lastAi = qsa(".chat-msg.ai").pop();
    if (lastAi) lastAi.classList.add("interrupted");
    addMessage("system", "已发送打断信号，可以继续作答");
  }

  function bindChatInput() {
    qs("#chatInput").addEventListener("keydown", (event) => {
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        sendMessage();
      }
    });
    qs("#sendBtn").addEventListener("click", () => sendMessage());
  }

  function addMessage(role, text) {
    const item = createEl("div", "chat-msg " + role);
    const bubble = createEl("div", "bubble", text);
    const time = createEl("div", "time", new Date().toLocaleTimeString());
    item.appendChild(bubble);
    item.appendChild(time);
    qs("#chatBox").appendChild(item);
    qs("#chatBox").scrollTop = qs("#chatBox").scrollHeight;
    return bubble;
  }

  async function sendMessage(optionalText) {
    if (state.chatBusy) return;
    const input = qs("#chatInput");
    const text = (optionalText != null ? optionalText : input.value).trim();
    if (!text) return;
    if (!state.sessionId) {
      showAlert("#chatAlert", "请先开始面试", "error");
      return;
    }
    clearAlert("#chatAlert");
    state.chatBusy = true;
    qs("#sendBtn").disabled = true;
    input.disabled = true;
    input.value = "";
    addMessage("user", text);
    state.currentAiText = "";
    state.currentAiBubble = addMessage("ai", "");
    try {
      await window.Goffer.streamSSE(
        "/api/interview/chat",
        { session_id: state.sessionId, content: text },
        {
          message(chunk) {
            state.currentAiText += chunk;
            state.currentAiBubble.textContent = state.currentAiText;
            qs("#chatBox").scrollTop = qs("#chatBox").scrollHeight;
          },
          done() {
            advanceProgressFallback();
            speakText(state.currentAiText);
          },
        }
      );
    } catch (err) {
      state.currentAiBubble.textContent = "AI 面试官暂时无法响应，请稍后重试。";
      showAlert("#chatAlert", err.message, "error");
    } finally {
      state.chatBusy = false;
      qs("#sendBtn").disabled = false;
      input.disabled = false;
      input.focus();
    }
  }

  function advanceProgressFallback() {
    // 前端 fallback：当前后端 chat SSE 没有返回 fsm_state/round，先按轮次估算阶段。
    state.fsmRound += 1;
    const maxRound = stageFallbackRounds[state.fsmState] || 2;
    if (state.fsmRound >= maxRound) {
      const index = stageOrder.indexOf(state.fsmState);
      if (index >= 0 && index < stageOrder.length - 1) {
        state.fsmState = stageOrder[index + 1];
        state.fsmRound = 0;
      }
    }
    renderProgress();
  }

  function renderProgress() {
    const activeIndex = Math.max(0, stageOrder.indexOf(state.fsmState));
    qsa(".progress-step").forEach((step, index) => {
      step.classList.toggle("done", index < activeIndex);
      step.classList.toggle("active", index === activeIndex);
    });
    setText("#roundText", "第 " + (state.fsmRound + 1) + " 轮");
  }

  function renderSpeechSupport() {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    if (!SpeechRecognition) {
      setText("#speechSupport", "当前浏览器不支持语音输入，可继续文本面试");
    }
    if (!window.speechSynthesis) {
      setText("#ttsSupport", "当前浏览器不支持朗读");
    }
  }

  function toggleVoiceMode() {
    state.voiceMode = !state.voiceMode;
    qs("#voiceModeBtn").classList.toggle("active", state.voiceMode);
    setText("#voiceModeBtn", state.voiceMode ? "语音输入开" : "语音输入");
    if (state.voiceMode) startSpeechRecognition();
    else stopSpeechRecognition();
  }

  function startSpeechRecognition() {
    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
    if (!SpeechRecognition) {
      addMessage("system", "当前浏览器不支持语音识别，请使用 Chrome 或 Edge，或继续文本作答");
      state.voiceMode = false;
      qs("#voiceModeBtn").classList.remove("active");
      return;
    }
    stopSpeechRecognition();
    const recognition = new SpeechRecognition();
    recognition.lang = "zh-CN";
    recognition.interimResults = false;
    recognition.continuous = true;
    recognition.onresult = (event) => {
      for (let i = event.resultIndex; i < event.results.length; i += 1) {
        if (event.results[i].isFinal) {
          const text = event.results[i][0].transcript.trim();
          if (text) sendMessage(text);
        }
      }
    };
    recognition.onerror = (event) => {
      if (event.error !== "no-speech" && event.error !== "aborted") {
        addMessage("system", "语音识别错误：" + event.error);
      }
    };
    recognition.onend = () => {
      if (state.voiceMode && state.recognition) {
        try {
          recognition.start();
        } catch (err) {}
      }
    };
    state.recognition = recognition;
    recognition.start();
    addMessage("system", "语音输入已开启");
  }

  function stopSpeechRecognition() {
    if (!state.recognition) return;
    const recognition = state.recognition;
    state.recognition = null;
    try {
      recognition.stop();
    } catch (err) {}
  }

  function speakText(text) {
    if (!state.voiceMode || !window.speechSynthesis || !text) return;
    stopTTS();
    const utterance = new SpeechSynthesisUtterance(text);
    utterance.lang = "zh-CN";
    utterance.rate = 1.05;
    state.synthUtterance = utterance;
    utterance.onend = utterance.onerror = () => {
      state.synthUtterance = null;
    };
    window.speechSynthesis.speak(utterance);
  }

  function stopTTS() {
    if (window.speechSynthesis) window.speechSynthesis.cancel();
    state.synthUtterance = null;
  }

  function endInterview() {
    stopSpeechRecognition();
    stopTTS();
    state.voiceMode = false;
    if (state.webrtc) {
      state.webrtc.stop();
      state.webrtc = null;
    }
    state.sessionId = "";
    state.chatBusy = false;
    state.currentAiBubble = null;
    state.currentAiText = "";
    state.fsmState = "greeting";
    state.fsmRound = 0;
    qs("#chatInput").value = "";
    qs("#chatInput").disabled = false;
    qs("#sendBtn").disabled = false;
    qs("#interruptBtn").disabled = true;
    qs("#muteBtn").disabled = true;
    setText("#muteBtn", "静音");
    setText("#voiceModeBtn", "语音输入");
    qs("#voiceModeBtn").classList.remove("active");
    qs("#chatBox").replaceChildren();
    qs("#chatSection").hidden = true;
    qs("#setupSection").hidden = false;
    updateVoiceState("disconnected", "语音未连接");
    renderProgress();
  }

  window.GofferCandidate = {
    init,
    sendMessage,
    endInterview,
  };

  document.addEventListener("DOMContentLoaded", init);
})(window);
