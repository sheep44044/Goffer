(function (window) {
  "use strict";

  class GofferWebRTC {
    constructor(options) {
      this.options = Object.assign(
        {
          wsUrl: window.Goffer.WS_URL,
          stunServers: window.Goffer.STUN_SERVERS,
          reconnectDelay: 3000,
          speakingThreshold: 30,
        },
        options || {}
      );
      this.reset();
    }

    reset() {
      this.pc = null;
      this.ws = null;
      this.stream = null;
      this.dataChannel = null;
      this.audioContext = null;
      this.analyser = null;
      this.vadFrame = 0;
      this.reconnectTimer = 0;
      this.wasSpeaking = false;
      this.closed = true;
      this.muted = false;
      this.roomId = "";
      this.userId = "";
      this.connecting = false;
    }

    async start(config) {
      if (this.connecting || this.pc || this.ws) return;
      if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia || !window.RTCPeerConnection) {
        this.emitState("unsupported", "当前浏览器不支持 WebRTC，可继续使用文本面试");
        return;
      }

      this.closed = false;
      this.connecting = true;
      this.roomId = config.roomId;
      this.userId = config.userId || "candidate";
      this.emitState("connecting", "语音连接中");

      try {
        this.stream = await navigator.mediaDevices.getUserMedia({
          audio: { echoCancellation: true, noiseSuppression: true, autoGainControl: true },
          video: false,
        });
        this.setupPeer();
        this.connectWebSocket();
        this.startVAD();
      } catch (err) {
        this.stop();
        this.emitState("permission-denied", "麦克风不可用，可继续使用文本面试");
        if (this.options.onError) this.options.onError(err);
      } finally {
        this.connecting = false;
      }
    }

    setupPeer() {
      this.pc = new RTCPeerConnection({
        iceServers: this.options.stunServers.map((url) => ({ urls: [url] })),
      });

      this.stream.getTracks().forEach((track) => {
        this.pc.addTrack(track, this.stream);
      });

      this.dataChannel = this.pc.createDataChannel("control", { ordered: true });
      this.dataChannel.onopen = () => this.emitDataChannel(true);
      this.dataChannel.onclose = () => this.emitDataChannel(false);
      this.dataChannel.onerror = () => this.emitDataChannel(false);

      this.pc.ontrack = (event) => {
        if (this.options.remoteAudio && event.streams && event.streams[0]) {
          this.options.remoteAudio.srcObject = event.streams[0];
        }
      };

      this.pc.onicecandidate = (event) => {
        if (!event.candidate) return;
        this.sendSignal({
          type: "ice-candidate",
          room_id: this.roomId,
          user_id: this.userId,
          candidate: event.candidate.candidate,
        });
      };

      this.pc.oniceconnectionstatechange = () => {
        const state = this.pc ? this.pc.iceConnectionState : "closed";
        if (state === "connected" || state === "completed") {
          this.clearReconnect();
          this.emitState("connected", "语音已连接");
        }
        if (state === "disconnected" || state === "failed") {
          this.emitState("reconnecting", "语音断开，正在重连");
          this.scheduleReconnect();
        }
        if (state === "closed") {
          this.emitState("disconnected", "语音未连接");
        }
      };
    }

    connectWebSocket() {
      if (this.closed || !this.pc) return;
      if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
        return;
      }

      this.ws = new WebSocket(this.options.wsUrl);
      this.ws.onopen = () => this.createOffer();
      this.ws.onmessage = (event) => this.handleSignal(event);
      this.ws.onerror = () => this.emitState("reconnecting", "语音信令异常，正在重连");
      this.ws.onclose = () => {
        if (!this.closed) {
          this.emitState("reconnecting", "语音信令断开，正在重连");
          this.scheduleReconnect();
        }
      };
    }

    async createOffer() {
      if (!this.pc || !this.ws || this.ws.readyState !== WebSocket.OPEN) return;
      try {
        const offer = await this.pc.createOffer();
        await this.pc.setLocalDescription(offer);
        this.sendSignal({
          type: "offer",
          room_id: this.roomId,
          user_id: this.userId,
          sdp: offer.sdp,
        });
      } catch (err) {
        if (this.options.onError) this.options.onError(err);
      }
    }

    async handleSignal(event) {
      let message;
      try {
        message = JSON.parse(event.data);
      } catch (err) {
        return;
      }
      if (message.type === "error") {
        if (this.options.onError) this.options.onError(new Error(message.error || "WebRTC 信令错误"));
        return;
      }
      if (!this.pc) return;
      try {
        if (message.type === "answer" && message.sdp) {
          await this.pc.setRemoteDescription({ type: "answer", sdp: message.sdp });
        }
        if (message.type === "ice-candidate" && message.candidate) {
          await this.pc.addIceCandidate({ candidate: message.candidate, sdpMid: "0" });
        }
      } catch (err) {
        if (this.options.onError) this.options.onError(err);
      }
    }

    sendSignal(payload) {
      if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
      this.ws.send(JSON.stringify(payload));
    }

    scheduleReconnect() {
      if (this.closed || this.reconnectTimer) return;
      this.reconnectTimer = window.setTimeout(() => {
        this.reconnectTimer = 0;
        this.reconnect();
      }, this.options.reconnectDelay);
    }

    async reconnect() {
      if (this.closed || !this.roomId) return;
      this.closeTransportOnly();
      this.emitState("reconnecting", "语音断开，正在重连");
      try {
        this.setupPeer();
        this.connectWebSocket();
        if (this.options.onReconnect) this.options.onReconnect();
      } catch (err) {
        if (this.options.onError) this.options.onError(err);
        this.scheduleReconnect();
      }
    }

    clearReconnect() {
      if (this.reconnectTimer) {
        window.clearTimeout(this.reconnectTimer);
        this.reconnectTimer = 0;
      }
    }

    closeTransportOnly() {
      if (this.ws) {
        this.ws.onclose = null;
        try {
          this.ws.close();
        } catch (err) {}
        this.ws = null;
      }
      if (this.pc) {
        try {
          this.pc.close();
        } catch (err) {}
        this.pc = null;
      }
      this.dataChannel = null;
      this.emitDataChannel(false);
    }

    startVAD() {
      if (!this.stream || !this.stream.getAudioTracks().length || !window.AudioContext) return;
      const track = this.stream.getAudioTracks()[0];
      this.audioContext = new AudioContext();
      const source = this.audioContext.createMediaStreamSource(new MediaStream([track]));
      this.analyser = this.audioContext.createAnalyser();
      this.analyser.fftSize = 256;
      source.connect(this.analyser);
      const data = new Uint8Array(this.analyser.frequencyBinCount);

      const tick = () => {
        if (this.closed || !this.analyser) return;
        this.analyser.getByteFrequencyData(data);
        const avg = data.reduce((sum, value) => sum + value, 0) / data.length;
        if (this.options.onMicLevel) this.options.onMicLevel(Math.min(100, avg * 2));
        const speaking = avg > this.options.speakingThreshold;
        if (speaking && !this.wasSpeaking) {
          this.wasSpeaking = true;
          this.sendCancel("voice");
        }
        if (!speaking) this.wasSpeaking = false;
        this.vadFrame = window.requestAnimationFrame(tick);
      };
      tick();
    }

    sendCancel(source) {
      if (!this.canInterrupt()) return false;
      this.dataChannel.send(JSON.stringify({ action: "cancel" }));
      if (this.options.onInterrupt) this.options.onInterrupt(source || "manual");
      return true;
    }

    canInterrupt() {
      return Boolean(this.dataChannel && this.dataChannel.readyState === "open");
    }

    setMuted(muted) {
      this.muted = Boolean(muted);
      if (this.stream) {
        this.stream.getAudioTracks().forEach((track) => {
          track.enabled = !this.muted;
        });
      }
      return this.muted;
    }

    toggleMuted() {
      return this.setMuted(!this.muted);
    }

    stop() {
      this.closed = true;
      this.clearReconnect();
      if (this.vadFrame) {
        window.cancelAnimationFrame(this.vadFrame);
        this.vadFrame = 0;
      }
      this.closeTransportOnly();
      if (this.stream) {
        this.stream.getTracks().forEach((track) => track.stop());
        this.stream = null;
      }
      if (this.audioContext) {
        this.audioContext.close().catch(() => {});
        this.audioContext = null;
      }
      this.analyser = null;
      this.wasSpeaking = false;
      this.muted = false;
      this.emitState("disconnected", "语音未连接");
    }

    emitState(state, label) {
      if (this.options.onStateChange) this.options.onStateChange(state, label);
    }

    emitDataChannel(ready) {
      if (this.options.onDataChannel) this.options.onDataChannel(ready);
    }
  }

  window.GofferWebRTC = GofferWebRTC;
})(window);
