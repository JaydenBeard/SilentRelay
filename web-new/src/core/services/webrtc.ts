/**
 * WebRTC Service
 *
 * Handles peer-to-peer audio/video calls using WebRTC.
 * Signaling is done via WebSocket.
 */

export interface CallState {
  callId: string;
  peerId: string;
  peerName: string;
  peerAvatar?: string;
  type: 'audio' | 'video';
  direction: 'incoming' | 'outgoing';
  status: 'ringing' | 'connecting' | 'connected' | 'ended';
  startTime?: number;
  endTime?: number;
  endReason?: 'completed' | 'declined' | 'missed' | 'failed' | 'busy';
}

export interface CallSignal {
  type: 'offer' | 'answer' | 'candidate' | 'hangup';
  callId: string;
  peerId: string;
  data?: RTCSessionDescriptionInit | RTCIceCandidateInit;
}

type CallEventHandler = {
  onRemoteStream: (stream: MediaStream) => void;
  onCallStateChange: (state: CallState) => void;
  onSignal: (signal: CallSignal) => void;
};

// Fallback ICE servers (STUN only - will NOT work for users behind symmetric NAT)
const FALLBACK_ICE_SERVERS: RTCConfiguration = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' },
    { urls: 'stun:stun2.l.google.com:19302' },
    { urls: 'stun:stun3.l.google.com:19302' },
  ],
};

// Cached ICE config
let cachedIceConfig: RTCConfiguration | null = null;
let cacheExpiry: number = 0;

/**
 * Fetch ICE servers (STUN + TURN) from backend
 * The backend generates time-limited TURN credentials
 */
async function fetchIceServers(): Promise<RTCConfiguration> {
  // Return cached config if still valid (with 5 min buffer)
  if (cachedIceConfig && Date.now() < cacheExpiry - 300000) {
    console.log('[WebRTC] Using cached ICE config');
    return cachedIceConfig;
  }

  try {
    // Import dynamically to avoid circular dependencies
    const { useAuthStore } = await import('@/core/store/authStore');
    const token = useAuthStore.getState().token;

    if (!token) {
      console.warn('[WebRTC] No auth token, using fallback ICE servers (STUN only)');
      return FALLBACK_ICE_SERVERS;
    }

    const response = await fetch('/api/v1/rtc/turn-credentials', {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      console.warn('[WebRTC] Failed to fetch TURN credentials (status:', response.status, '), using fallback STUN only');
      return FALLBACK_ICE_SERVERS;
    }

    const data = await response.json();

    // Log what we got for debugging
    console.log('[WebRTC] Got ICE servers:', data.iceServers?.map((s: { urls: string }) => s.urls));

    // Verify we have TURN servers
    const hasTurn = data.iceServers?.some((s: { urls: string | string[] }) =>
      (typeof s.urls === 'string' ? s.urls : s.urls[0])?.startsWith('turn:')
    );

    if (!hasTurn) {
      console.warn('[WebRTC] ⚠️ No TURN servers returned! Calls may fail behind NAT');
    } else {
      console.log('[WebRTC] ✓ TURN server configured, expires in', data.ttl, 'seconds');
    }

    cachedIceConfig = { iceServers: data.iceServers };
    cacheExpiry = Date.now() + (data.ttl * 1000);

    return cachedIceConfig;
  } catch (error) {
    console.error('[WebRTC] Error fetching TURN credentials:', error);
    return FALLBACK_ICE_SERVERS;
  }
}

export class WebRTCService {
  private peerConnection: RTCPeerConnection | null = null;
  private localStream: MediaStream | null = null;
  private remoteStream: MediaStream | null = null;
  private currentCall: CallState | null = null;
  private handlers: Partial<CallEventHandler> = {};
  private iceCandidateQueue: RTCIceCandidateInit[] = [];
  private pendingOffer: RTCSessionDescriptionInit | null = null;

  /**
   * Set event handlers
   */
  setHandlers(handlers: Partial<CallEventHandler>) {
    this.handlers = { ...this.handlers, ...handlers };
  }

  /**
   * Get current call state
   */
  getCurrentCall(): CallState | null {
    return this.currentCall;
  }

  /**
   * Get local media stream
   */
  getLocalStream(): MediaStream | null {
    return this.localStream;
  }

  /**
   * Get remote media stream
   */
  getRemoteStream(): MediaStream | null {
    return this.remoteStream;
  }

  /**
   * Start an outgoing call
   */
  async startCall(
    peerId: string,
    peerName: string,
    peerAvatar: string | undefined,
    callType: 'audio' | 'video'
  ): Promise<void> {
    if (this.currentCall) {
      throw new Error('Already in a call');
    }

    console.log('[WebRTC] Starting', callType, 'call to', peerId);

    const callId = crypto.randomUUID();
    this.currentCall = {
      callId,
      peerId,
      peerName,
      peerAvatar,
      type: callType,
      direction: 'outgoing',
      status: 'ringing',
    };

    this.updateCallState(this.currentCall);

    try {
      // Get local media FIRST
      console.log('[WebRTC] Getting local media...');
      await this.getLocalMedia(callType);
      console.log('[WebRTC] Got local media:',
        'audio tracks:', this.localStream?.getAudioTracks().length,
        'video tracks:', this.localStream?.getVideoTracks().length
      );

      // Create peer connection
      console.log('[WebRTC] Creating peer connection...');
      await this.createPeerConnection();

      // Add local tracks to connection BEFORE creating offer
      if (this.localStream && this.peerConnection) {
        console.log('[WebRTC] Adding local tracks to peer connection...');
        this.localStream.getTracks().forEach(track => {
          console.log('[WebRTC] Adding track:', track.kind, track.id);
          this.peerConnection!.addTrack(track, this.localStream!);
        });
      }

      // Create and send offer
      console.log('[WebRTC] Creating offer...');
      const offer = await this.peerConnection!.createOffer({
        offerToReceiveAudio: true,
        offerToReceiveVideo: callType === 'video',
      });

      console.log('[WebRTC] Setting local description...');
      await this.peerConnection!.setLocalDescription(offer);

      // Send offer through signaling
      console.log('[WebRTC] Sending offer to peer...');
      this.handlers.onSignal?.({
        type: 'offer',
        callId,
        peerId,
        data: offer,
      });
    } catch (error) {
      console.error('[WebRTC] Failed to start call:', error);
      this.endCall('failed');
      throw error;
    }
  }

  /**
   * Handle incoming call offer
   */
  async handleIncomingCall(
    callId: string,
    peerId: string,
    peerName: string,
    peerAvatar: string | undefined,
    callType: 'audio' | 'video',
    offer: RTCSessionDescriptionInit
  ): Promise<void> {
    console.log('[WebRTC] Incoming', callType, 'call from', peerId);

    if (this.currentCall) {
      console.log('[WebRTC] Already in a call, sending busy signal');
      this.handlers.onSignal?.({
        type: 'hangup',
        callId,
        peerId,
        data: { sdp: 'busy' } as RTCSessionDescriptionInit,
      });
      return;
    }

    this.currentCall = {
      callId,
      peerId,
      peerName,
      peerAvatar,
      type: callType,
      direction: 'incoming',
      status: 'ringing',
    };

    // Store the offer for when the user answers
    this.pendingOffer = offer;

    this.updateCallState(this.currentCall);
  }

  /**
   * Answer an incoming call
   */
  async answerCall(): Promise<void> {
    if (!this.currentCall || this.currentCall.direction !== 'incoming') {
      throw new Error('No incoming call to answer');
    }

    if (!this.pendingOffer) {
      throw new Error('No pending offer to answer');
    }

    console.log('[WebRTC] Answering call...');

    try {
      // Get local media FIRST - before setting up WebRTC
      console.log('[WebRTC] Getting local media...');
      await this.getLocalMedia(this.currentCall.type);
      console.log('[WebRTC] Got local media:',
        'audio tracks:', this.localStream?.getAudioTracks().length,
        'video tracks:', this.localStream?.getVideoTracks().length
      );

      // Create peer connection
      console.log('[WebRTC] Creating peer connection...');
      await this.createPeerConnection();

      // Add local tracks BEFORE setting remote description
      // This is critical for bidirectional media!
      if (this.localStream && this.peerConnection) {
        console.log('[WebRTC] Adding local tracks to peer connection...');
        this.localStream.getTracks().forEach(track => {
          console.log('[WebRTC] Adding track:', track.kind, track.id);
          this.peerConnection!.addTrack(track, this.localStream!);
        });
      }

      // Now set the remote description (the offer)
      console.log('[WebRTC] Setting remote description (offer)...');
      await this.peerConnection!.setRemoteDescription(this.pendingOffer);
      this.pendingOffer = null;

      // Process any queued ICE candidates
      console.log('[WebRTC] Processing queued ICE candidates:', this.iceCandidateQueue.length);
      this.processIceCandidateQueue();

      // Create and send answer
      console.log('[WebRTC] Creating answer...');
      const answer = await this.peerConnection!.createAnswer();

      console.log('[WebRTC] Setting local description (answer)...');
      await this.peerConnection!.setLocalDescription(answer);

      this.currentCall.status = 'connecting';
      this.updateCallState(this.currentCall);

      console.log('[WebRTC] Sending answer to peer...');
      this.handlers.onSignal?.({
        type: 'answer',
        callId: this.currentCall.callId,
        peerId: this.currentCall.peerId,
        data: answer,
      });

      console.log('[WebRTC] Answer sent successfully');
    } catch (error) {
      console.error('[WebRTC] Failed to answer call:', error);
      this.endCall('failed');
      throw error;
    }
  }

  /**
   * Decline an incoming call
   */
  declineCall(): void {
    if (!this.currentCall || this.currentCall.direction !== 'incoming') {
      return;
    }

    console.log('[WebRTC] Declining call');
    this.handlers.onSignal?.({
      type: 'hangup',
      callId: this.currentCall.callId,
      peerId: this.currentCall.peerId,
    });

    this.pendingOffer = null;
    this.endCall('declined');
  }

  /**
   * Handle answer from remote peer
   */
  async handleAnswer(answer: RTCSessionDescriptionInit): Promise<void> {
    if (!this.peerConnection || !this.currentCall) {
      console.warn('[WebRTC] Received answer but no peer connection');
      return;
    }

    console.log('[WebRTC] Received answer, setting remote description...');
    try {
      await this.peerConnection.setRemoteDescription(answer);
      console.log('[WebRTC] Remote description set successfully');

      this.currentCall.status = 'connecting';
      this.updateCallState(this.currentCall);

      // Process queued ICE candidates now that we have remote description
      console.log('[WebRTC] Processing queued ICE candidates:', this.iceCandidateQueue.length);
      this.processIceCandidateQueue();
    } catch (error) {
      console.error('[WebRTC] Failed to set remote description:', error);
      this.endCall('failed');
    }
  }

  /**
   * Handle ICE candidate from remote peer
   */
  async handleIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    console.log('[WebRTC] Received ICE candidate:', candidate.candidate?.substring(0, 50) + '...');

    if (!this.peerConnection) {
      console.log('[WebRTC] Queuing ICE candidate (no peer connection yet)');
      this.iceCandidateQueue.push(candidate);
      return;
    }

    if (!this.peerConnection.remoteDescription) {
      console.log('[WebRTC] Queuing ICE candidate (no remote description yet)');
      this.iceCandidateQueue.push(candidate);
      return;
    }

    try {
      await this.peerConnection.addIceCandidate(candidate);
      console.log('[WebRTC] Added ICE candidate successfully');
    } catch (error) {
      console.error('[WebRTC] Failed to add ICE candidate:', error);
    }
  }

  /**
   * Handle remote hangup
   */
  handleRemoteHangup(reason?: string): void {
    console.log('[WebRTC] Remote hangup, reason:', reason);
    if (!this.currentCall) return;

    let endReason: CallState['endReason'] = 'completed';
    if (reason === 'busy') {
      endReason = 'busy';
    } else if (this.currentCall.status === 'ringing') {
      endReason = this.currentCall.direction === 'incoming' ? 'missed' : 'declined';
    }

    this.endCall(endReason);
  }

  /**
   * End the current call
   */
  endCall(reason: CallState['endReason'] = 'completed'): void {
    console.log('[WebRTC] Ending call, reason:', reason);

    if (this.currentCall && this.currentCall.status !== 'ended') {
      // Send hangup signal
      this.handlers.onSignal?.({
        type: 'hangup',
        callId: this.currentCall.callId,
        peerId: this.currentCall.peerId,
      });
    }

    // Stop all tracks
    this.localStream?.getTracks().forEach(track => {
      console.log('[WebRTC] Stopping local track:', track.kind);
      track.stop();
    });
    this.remoteStream?.getTracks().forEach(track => {
      console.log('[WebRTC] Stopping remote track:', track.kind);
      track.stop();
    });

    // Close peer connection
    if (this.peerConnection) {
      console.log('[WebRTC] Closing peer connection');
      this.peerConnection.close();
    }

    // Update call state
    if (this.currentCall) {
      this.currentCall.status = 'ended';
      this.currentCall.endTime = Date.now();
      this.currentCall.endReason = reason;
      this.updateCallState(this.currentCall);
    }

    // Clean up
    this.localStream = null;
    this.remoteStream = null;
    this.peerConnection = null;
    this.currentCall = null;
    this.iceCandidateQueue = [];
    this.pendingOffer = null;
  }

  /**
   * Toggle audio mute
   */
  toggleMute(): boolean {
    if (!this.localStream) return false;

    const audioTrack = this.localStream.getAudioTracks()[0];
    if (audioTrack) {
      audioTrack.enabled = !audioTrack.enabled;
      console.log('[WebRTC] Audio muted:', !audioTrack.enabled);
      return !audioTrack.enabled;
    }
    return false;
  }

  /**
   * Toggle video
   */
  toggleVideo(): boolean {
    if (!this.localStream) return false;

    const videoTrack = this.localStream.getVideoTracks()[0];
    if (videoTrack) {
      videoTrack.enabled = !videoTrack.enabled;
      console.log('[WebRTC] Video off:', !videoTrack.enabled);
      return !videoTrack.enabled;
    }
    return false;
  }

  /**
   * Switch camera (for mobile)
   */
  async switchCamera(): Promise<void> {
    if (!this.localStream || this.currentCall?.type !== 'video') return;

    const videoTrack = this.localStream.getVideoTracks()[0];
    if (!videoTrack) return;

    const settings = videoTrack.getSettings();
    const newFacingMode = settings.facingMode === 'user' ? 'environment' : 'user';

    console.log('[WebRTC] Switching camera to:', newFacingMode);

    videoTrack.stop();

    const newStream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: newFacingMode },
    });

    const newVideoTrack = newStream.getVideoTracks()[0];

    this.localStream.removeTrack(videoTrack);
    this.localStream.addTrack(newVideoTrack);

    const sender = this.peerConnection?.getSenders().find(s => s.track?.kind === 'video');
    if (sender) {
      await sender.replaceTrack(newVideoTrack);
    }
  }

  // Private methods

  private async getLocalMedia(callType: 'audio' | 'video'): Promise<void> {
    const constraints: MediaStreamConstraints = {
      audio: {
        echoCancellation: true,
        noiseSuppression: true,
        autoGainControl: true,
      },
      video: callType === 'video' ? {
        facingMode: 'user',
        width: { ideal: 1280 },
        height: { ideal: 720 },
      } : false,
    };

    console.log('[WebRTC] Requesting media with constraints:', JSON.stringify(constraints));

    try {
      this.localStream = await navigator.mediaDevices.getUserMedia(constraints);

      // Log track details
      this.localStream.getTracks().forEach(track => {
        console.log('[WebRTC] Got track:', track.kind,
          'enabled:', track.enabled,
          'muted:', track.muted,
          'readyState:', track.readyState
        );
      });
    } catch (error) {
      console.error('[WebRTC] Failed to get user media:', error);
      throw error;
    }
  }

  private async createPeerConnection(): Promise<void> {
    const iceConfig = await fetchIceServers();

    console.log('[WebRTC] Creating RTCPeerConnection with config:',
      iceConfig.iceServers?.map(s => typeof s.urls === 'string' ? s.urls : s.urls?.[0])
    );

    this.peerConnection = new RTCPeerConnection(iceConfig);

    // Handle ICE candidates
    this.peerConnection.onicecandidate = (event) => {
      if (event.candidate && this.currentCall) {
        // Log candidate type for debugging
        const candidateStr = event.candidate.candidate;
        const type = candidateStr.includes('typ host') ? 'host' :
          candidateStr.includes('typ srflx') ? 'srflx (STUN)' :
            candidateStr.includes('typ relay') ? 'relay (TURN)' : 'unknown';
        console.log('[WebRTC] ICE candidate:', type);

        this.handlers.onSignal?.({
          type: 'candidate',
          callId: this.currentCall.callId,
          peerId: this.currentCall.peerId,
          data: event.candidate.toJSON(),
        });
      } else if (event.candidate === null) {
        console.log('[WebRTC] ICE gathering complete');
      }
    };

    // Handle ICE connection state changes
    this.peerConnection.oniceconnectionstatechange = () => {
      console.log('[WebRTC] ICE connection state:', this.peerConnection?.iceConnectionState);

      if (this.peerConnection?.iceConnectionState === 'failed') {
        console.error('[WebRTC] ICE connection failed! This usually means:');
        console.error('[WebRTC] - TURN server not configured or unreachable');
        console.error('[WebRTC] - Firewall blocking UDP/TCP');
        console.error('[WebRTC] - NAT traversal failed');
      }
    };

    // Handle ICE gathering state changes
    this.peerConnection.onicegatheringstatechange = () => {
      console.log('[WebRTC] ICE gathering state:', this.peerConnection?.iceGatheringState);
    };

    // Handle connection state changes
    this.peerConnection.onconnectionstatechange = () => {
      console.log('[WebRTC] Connection state:', this.peerConnection?.connectionState);

      if (!this.currentCall || !this.peerConnection) return;

      switch (this.peerConnection.connectionState) {
        case 'connected':
          console.log('[WebRTC] ✓ Call connected successfully!');
          this.currentCall.status = 'connected';
          this.currentCall.startTime = Date.now();
          this.updateCallState(this.currentCall);
          break;
        case 'disconnected':
          console.warn('[WebRTC] Connection disconnected');
          // Don't immediately fail - could be temporary
          break;
        case 'failed':
          console.error('[WebRTC] Connection failed');
          this.endCall('failed');
          break;
        case 'closed':
          console.log('[WebRTC] Connection closed');
          this.endCall('completed');
          break;
      }
    };

    // Handle remote tracks
    this.peerConnection.ontrack = (event) => {
      console.log('[WebRTC] Received remote track:',
        event.track.kind,
        'enabled:', event.track.enabled,
        'muted:', event.track.muted,
        'readyState:', event.track.readyState
      );

      if (!this.remoteStream) {
        this.remoteStream = new MediaStream();
      }

      // Add track to remote stream
      this.remoteStream.addTrack(event.track);

      // Log all tracks in remote stream
      console.log('[WebRTC] Remote stream now has:',
        'audio tracks:', this.remoteStream.getAudioTracks().length,
        'video tracks:', this.remoteStream.getVideoTracks().length
      );

      // Notify handler
      this.handlers.onRemoteStream?.(this.remoteStream);

      // Handle track ending
      event.track.onended = () => {
        console.log('[WebRTC] Remote track ended:', event.track.kind);
      };

      // Handle track muted/unmuted
      event.track.onmute = () => {
        console.log('[WebRTC] Remote track muted:', event.track.kind);
      };

      event.track.onunmute = () => {
        console.log('[WebRTC] Remote track unmuted:', event.track.kind);
      };
    };

    // Handle negotiation needed (for renegotiation)
    this.peerConnection.onnegotiationneeded = () => {
      console.log('[WebRTC] Negotiation needed');
    };

    // Handle signaling state changes
    this.peerConnection.onsignalingstatechange = () => {
      console.log('[WebRTC] Signaling state:', this.peerConnection?.signalingState);
    };
  }

  private processIceCandidateQueue(): void {
    if (!this.peerConnection?.remoteDescription) {
      console.log('[WebRTC] Cannot process ICE queue - no remote description');
      return;
    }

    const queueLength = this.iceCandidateQueue.length;
    if (queueLength > 0) {
      console.log('[WebRTC] Processing', queueLength, 'queued ICE candidates');
    }

    while (this.iceCandidateQueue.length > 0) {
      const candidate = this.iceCandidateQueue.shift();
      if (candidate) {
        this.peerConnection.addIceCandidate(candidate)
          .then(() => console.log('[WebRTC] Added queued ICE candidate'))
          .catch(err => console.error('[WebRTC] Failed to add queued candidate:', err));
      }
    }
  }

  private updateCallState(state: CallState): void {
    this.handlers.onCallStateChange?.(state);
  }
}

// Singleton instance
export const webrtcService = new WebRTCService();
