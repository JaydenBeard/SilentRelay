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

// Fallback ICE servers (STUN only)
const FALLBACK_ICE_SERVERS: RTCConfiguration = {
  iceServers: [
    { urls: 'stun:stun.l.google.com:19302' },
    { urls: 'stun:stun1.l.google.com:19302' },
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
    return cachedIceConfig;
  }

  try {
    // Import dynamically to avoid circular dependencies
    const { useAuthStore } = await import('@/core/store/authStore');
    const token = useAuthStore.getState().token;

    if (!token) {
      console.warn('[WebRTC] No auth token, using fallback ICE servers');
      return FALLBACK_ICE_SERVERS;
    }

    const response = await fetch('/api/v1/rtc/turn-credentials', {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      console.warn('[WebRTC] Failed to fetch TURN credentials, using fallback');
      return FALLBACK_ICE_SERVERS;
    }

    const data = await response.json();
    cachedIceConfig = { iceServers: data.iceServers };
    cacheExpiry = Date.now() + (data.ttl * 1000); // TTL is in seconds

    console.log('[WebRTC] Got TURN credentials, expires in', data.ttl, 'seconds');
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
      // Get local media
      await this.getLocalMedia(callType);

      // Create peer connection and offer
      await this.createPeerConnection();

      // Add local tracks to connection
      if (this.localStream && this.peerConnection) {
        this.localStream.getTracks().forEach(track => {
          this.peerConnection!.addTrack(track, this.localStream!);
        });
      }

      // Create and send offer
      const offer = await this.peerConnection!.createOffer({
        offerToReceiveAudio: true,
        offerToReceiveVideo: callType === 'video',
      });
      await this.peerConnection!.setLocalDescription(offer);

      // Send offer through signaling
      this.handlers.onSignal?.({
        type: 'offer',
        callId,
        peerId,
        data: offer,
      });
    } catch (error) {
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
    if (this.currentCall) {
      // Already in a call, send busy signal
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

    this.updateCallState(this.currentCall);

    // Store offer for when call is answered
    await this.createPeerConnection();
    await this.peerConnection!.setRemoteDescription(offer);

    // Process queued ICE candidates
    this.processIceCandidateQueue();
  }

  /**
   * Answer an incoming call
   */
  async answerCall(): Promise<void> {
    if (!this.currentCall || this.currentCall.direction !== 'incoming') {
      throw new Error('No incoming call to answer');
    }

    if (!this.peerConnection) {
      throw new Error('Peer connection not initialized');
    }

    try {
      // Get local media
      await this.getLocalMedia(this.currentCall.type);

      // Add local tracks
      if (this.localStream) {
        this.localStream.getTracks().forEach(track => {
          this.peerConnection!.addTrack(track, this.localStream!);
        });
      }

      // Create and send answer
      const answer = await this.peerConnection.createAnswer();
      await this.peerConnection.setLocalDescription(answer);

      this.currentCall.status = 'connecting';
      this.updateCallState(this.currentCall);

      this.handlers.onSignal?.({
        type: 'answer',
        callId: this.currentCall.callId,
        peerId: this.currentCall.peerId,
        data: answer,
      });
    } catch (error) {
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

    this.handlers.onSignal?.({
      type: 'hangup',
      callId: this.currentCall.callId,
      peerId: this.currentCall.peerId,
    });

    this.endCall('declined');
  }

  /**
   * Handle answer from remote peer
   */
  async handleAnswer(answer: RTCSessionDescriptionInit): Promise<void> {
    if (!this.peerConnection || !this.currentCall) {
      return;
    }

    await this.peerConnection.setRemoteDescription(answer);
    this.currentCall.status = 'connecting';
    this.updateCallState(this.currentCall);

    // Process queued ICE candidates
    this.processIceCandidateQueue();
  }

  /**
   * Handle ICE candidate from remote peer
   */
  async handleIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    if (!this.peerConnection) {
      // Queue candidate if connection not ready
      this.iceCandidateQueue.push(candidate);
      return;
    }

    if (!this.peerConnection.remoteDescription) {
      this.iceCandidateQueue.push(candidate);
      return;
    }

    try {
      await this.peerConnection.addIceCandidate(candidate);
    } catch (error) {
      console.error('Failed to add ICE candidate:', error);
    }
  }

  /**
   * Handle remote hangup
   */
  handleRemoteHangup(reason?: string): void {
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
    if (this.currentCall && this.currentCall.status !== 'ended') {
      // Send hangup signal
      this.handlers.onSignal?.({
        type: 'hangup',
        callId: this.currentCall.callId,
        peerId: this.currentCall.peerId,
      });
    }

    // Stop all tracks
    this.localStream?.getTracks().forEach(track => track.stop());
    this.remoteStream?.getTracks().forEach(track => track.stop());

    // Close peer connection
    this.peerConnection?.close();

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
  }

  /**
   * Toggle audio mute
   */
  toggleMute(): boolean {
    if (!this.localStream) return false;

    const audioTrack = this.localStream.getAudioTracks()[0];
    if (audioTrack) {
      audioTrack.enabled = !audioTrack.enabled;
      return !audioTrack.enabled; // Return true if muted
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
      return !videoTrack.enabled; // Return true if video off
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

    // Get current facing mode
    const settings = videoTrack.getSettings();
    const newFacingMode = settings.facingMode === 'user' ? 'environment' : 'user';

    // Stop current track
    videoTrack.stop();

    // Get new video track
    const newStream = await navigator.mediaDevices.getUserMedia({
      video: { facingMode: newFacingMode },
    });

    const newVideoTrack = newStream.getVideoTracks()[0];

    // Replace track in local stream
    this.localStream.removeTrack(videoTrack);
    this.localStream.addTrack(newVideoTrack);

    // Replace track in peer connection
    const sender = this.peerConnection?.getSenders().find(s => s.track?.kind === 'video');
    if (sender) {
      await sender.replaceTrack(newVideoTrack);
    }
  }

  // Private methods

  private async getLocalMedia(callType: 'audio' | 'video'): Promise<void> {
    const constraints: MediaStreamConstraints = {
      audio: true,
      video: callType === 'video' ? { facingMode: 'user' } : false,
    };

    this.localStream = await navigator.mediaDevices.getUserMedia(constraints);
  }

  private async createPeerConnection(): Promise<void> {
    const iceConfig = await fetchIceServers();
    this.peerConnection = new RTCPeerConnection(iceConfig);

    // Handle ICE candidates
    this.peerConnection.onicecandidate = (event) => {
      if (event.candidate && this.currentCall) {
        this.handlers.onSignal?.({
          type: 'candidate',
          callId: this.currentCall.callId,
          peerId: this.currentCall.peerId,
          data: event.candidate.toJSON(),
        });
      }
    };

    // Handle connection state changes
    this.peerConnection.onconnectionstatechange = () => {
      if (!this.currentCall || !this.peerConnection) return;

      switch (this.peerConnection.connectionState) {
        case 'connected':
          this.currentCall.status = 'connected';
          this.currentCall.startTime = Date.now();
          this.updateCallState(this.currentCall);
          break;
        case 'disconnected':
        case 'failed':
          this.endCall('failed');
          break;
        case 'closed':
          this.endCall('completed');
          break;
      }
    };

    // Handle remote stream
    this.peerConnection.ontrack = (event) => {
      if (!this.remoteStream) {
        this.remoteStream = new MediaStream();
      }
      this.remoteStream.addTrack(event.track);
      this.handlers.onRemoteStream?.(this.remoteStream);
    };
  }

  private processIceCandidateQueue(): void {
    if (!this.peerConnection?.remoteDescription) return;

    while (this.iceCandidateQueue.length > 0) {
      const candidate = this.iceCandidateQueue.shift();
      if (candidate) {
        this.peerConnection.addIceCandidate(candidate).catch(console.error);
      }
    }
  }

  private updateCallState(state: CallState): void {
    this.handlers.onCallStateChange?.(state);
  }
}

// Singleton instance
export const webrtcService = new WebRTCService();
