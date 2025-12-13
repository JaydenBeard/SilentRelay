/**
 * Call State Store
 *
 * Manages WebRTC call state globally.
 */

import { create } from 'zustand';
import type { CallState } from '../services/webrtc';

interface CallStore {
  // State
  currentCall: CallState | null;
  localStream: MediaStream | null;
  remoteStream: MediaStream | null;
  isMuted: boolean;
  isVideoOff: boolean;

  // Actions
  setCurrentCall: (call: CallState | null) => void;
  setLocalStream: (stream: MediaStream | null) => void;
  setRemoteStream: (stream: MediaStream | null) => void;
  setMuted: (muted: boolean) => void;
  setVideoOff: (videoOff: boolean) => void;
  reset: () => void;
}

const initialState = {
  currentCall: null,
  localStream: null,
  remoteStream: null,
  isMuted: false,
  isVideoOff: false,
};

export const useCallStore = create<CallStore>((set) => ({
  ...initialState,

  setCurrentCall: (call) => set({ currentCall: call }),
  setLocalStream: (stream) => set({ localStream: stream }),
  setRemoteStream: (stream) => set({ remoteStream: stream }),
  setMuted: (muted) => set({ isMuted: muted }),
  setVideoOff: (videoOff) => set({ isVideoOff: videoOff }),
  reset: () => set(initialState),
}));
