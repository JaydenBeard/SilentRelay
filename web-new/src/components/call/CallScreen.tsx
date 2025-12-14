/**
 * Call Screen Components
 *
 * Incoming call notification and active call UI.
 */

import { useEffect, useRef, useState } from 'react';
import { useCallStore } from '@/core/store/callStore';
import { useChatStore } from '@/core/store/chatStore';
import { webrtcService } from '@/core/services/webrtc';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { cn } from '@/lib/utils';
import type { Message, CallMetadata } from '@/core/types';
import {
  Phone,
  PhoneOff,
  Mic,
  MicOff,
  Video,
  VideoOff,
  FlipHorizontal,
  X,
} from 'lucide-react';

/**
 * Incoming Call Notification
 */
export function IncomingCall() {
  const { currentCall } = useCallStore();

  if (!currentCall || currentCall.direction !== 'incoming' || currentCall.status !== 'ringing') {
    return null;
  }

  const handleAnswer = async () => {
    try {
      await webrtcService.answerCall();
    } catch (error) {
      console.error('Failed to answer call:', error);
    }
  };

  const handleDecline = () => {
    webrtcService.declineCall();
  };

  return (
    <div className="fixed inset-0 z-50 bg-background/95 backdrop-blur-sm flex flex-col items-center justify-center">
      {/* Caller Info */}
      <div className="text-center mb-8">
        <Avatar className="h-32 w-32 mx-auto mb-4 ring-4 ring-primary/20 animate-pulse">
          <AvatarImage src={currentCall.peerAvatar} />
          <AvatarFallback className="text-4xl">
            {currentCall.peerName.charAt(0)}
          </AvatarFallback>
        </Avatar>
        <h2 className="text-2xl font-bold">{currentCall.peerName}</h2>
        <p className="text-foreground-muted mt-2">
          Incoming {currentCall.type} call...
        </p>
      </div>

      {/* Call Actions */}
      <div className="flex items-center gap-8">
        <button
          onClick={handleDecline}
          className="p-5 rounded-full bg-destructive text-white hover:bg-destructive/90 transition-colors"
        >
          <PhoneOff className="h-8 w-8" />
        </button>
        <button
          onClick={handleAnswer}
          className="p-5 rounded-full bg-success text-white hover:bg-success/90 transition-colors animate-bounce"
        >
          <Phone className="h-8 w-8" />
        </button>
      </div>
    </div>
  );
}

/**
 * Active Call Screen
 */
export function ActiveCall() {
  const { currentCall, localStream, remoteStream, isMuted, isVideoOff } = useCallStore();
  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteVideoRef = useRef<HTMLVideoElement>(null);
  const remoteAudioRef = useRef<HTMLAudioElement>(null);
  const [callDuration, setCallDuration] = useState(0);

  // Attach streams to video elements
  useEffect(() => {
    if (localVideoRef.current && localStream) {
      localVideoRef.current.srcObject = localStream;
    }
  }, [localStream]);

  // Attach remote stream to both video and audio elements
  useEffect(() => {
    if (remoteStream) {
      if (remoteVideoRef.current) {
        remoteVideoRef.current.srcObject = remoteStream;
      }
      if (remoteAudioRef.current) {
        remoteAudioRef.current.srcObject = remoteStream;
      }
    }
  }, [remoteStream]);

  // Call duration timer
  useEffect(() => {
    if (currentCall?.status !== 'connected' || !currentCall.startTime) {
      return;
    }

    const interval = setInterval(() => {
      setCallDuration(Math.floor((Date.now() - currentCall.startTime!) / 1000));
    }, 1000);

    return () => clearInterval(interval);
  }, [currentCall?.status, currentCall?.startTime]);

  if (!currentCall || currentCall.status === 'ended') {
    return null;
  }

  // Don't show for ringing incoming calls (handled by IncomingCall)
  if (currentCall.direction === 'incoming' && currentCall.status === 'ringing') {
    return null;
  }

  const formatDuration = (seconds: number): string => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const handleToggleMute = () => {
    const muted = webrtcService.toggleMute();
    useCallStore.getState().setMuted(muted);
  };

  const handleToggleVideo = () => {
    const videoOff = webrtcService.toggleVideo();
    useCallStore.getState().setVideoOff(videoOff);
  };

  const handleSwitchCamera = async () => {
    try {
      await webrtcService.switchCamera();
    } catch (error) {
      console.error('Failed to switch camera:', error);
    }
  };

  const handleEndCall = () => {
    webrtcService.endCall();
  };

  const isVideoCall = currentCall.type === 'video';
  const isConnecting = currentCall.status === 'connecting' || currentCall.status === 'ringing';

  return (
    <div className="fixed inset-0 z-50 bg-background flex flex-col">
      {/* Hidden audio element for remote audio playback */}
      <audio ref={remoteAudioRef} autoPlay playsInline />

      {/* Video Area */}
      {isVideoCall ? (
        <div className="flex-1 relative bg-black">
          {/* Remote Video (Full Screen) */}
          <video
            ref={remoteVideoRef}
            autoPlay
            playsInline
            className={cn(
              'w-full h-full object-cover',
              isConnecting && 'hidden'
            )}
          />

          {/* Connecting State */}
          {isConnecting && (
            <div className="absolute inset-0 flex flex-col items-center justify-center">
              <Avatar className="h-32 w-32 mb-4">
                <AvatarImage src={currentCall.peerAvatar} />
                <AvatarFallback className="text-4xl">
                  {currentCall.peerName.charAt(0)}
                </AvatarFallback>
              </Avatar>
              <h2 className="text-2xl font-bold text-white">{currentCall.peerName}</h2>
              <p className="text-white/70 mt-2">
                {currentCall.status === 'ringing' ? 'Calling...' : 'Connecting...'}
              </p>
            </div>
          )}

          {/* Local Video (Picture-in-Picture) */}
          <div className="absolute top-4 right-4 w-32 h-48 rounded-xl overflow-hidden shadow-lg">
            {isVideoOff ? (
              <div className="w-full h-full bg-background-tertiary flex items-center justify-center">
                <VideoOff className="h-8 w-8 text-foreground-muted" />
              </div>
            ) : (
              <video
                ref={localVideoRef}
                autoPlay
                playsInline
                muted
                className="w-full h-full object-cover"
              />
            )}
          </div>
        </div>
      ) : (
        // Audio Call UI
        <div className="flex-1 flex flex-col items-center justify-center bg-gradient-to-b from-background to-background-secondary">
          <Avatar className="h-40 w-40 mb-6">
            <AvatarImage src={currentCall.peerAvatar} />
            <AvatarFallback className="text-5xl">
              {currentCall.peerName.charAt(0)}
            </AvatarFallback>
          </Avatar>
          <h2 className="text-3xl font-bold">{currentCall.peerName}</h2>
          <p className="text-foreground-muted mt-2">
            {isConnecting
              ? currentCall.status === 'ringing'
                ? 'Calling...'
                : 'Connecting...'
              : formatDuration(callDuration)}
          </p>
        </div>
      )}

      {/* Call Controls */}
      <div className="p-6 bg-background-secondary border-t border-border">
        <div className="flex items-center justify-center gap-4">
          {/* Mute */}
          <CallButton
            icon={isMuted ? <MicOff className="h-6 w-6" /> : <Mic className="h-6 w-6" />}
            label={isMuted ? 'Unmute' : 'Mute'}
            active={isMuted}
            onClick={handleToggleMute}
          />

          {/* Video Toggle (Video calls only) */}
          {isVideoCall && (
            <CallButton
              icon={isVideoOff ? <VideoOff className="h-6 w-6" /> : <Video className="h-6 w-6" />}
              label={isVideoOff ? 'Camera On' : 'Camera Off'}
              active={isVideoOff}
              onClick={handleToggleVideo}
            />
          )}

          {/* End Call */}
          <button
            onClick={handleEndCall}
            className="p-4 rounded-full bg-destructive text-white hover:bg-destructive/90 transition-colors"
          >
            <PhoneOff className="h-6 w-6" />
          </button>

          {/* Switch Camera (Video calls only) */}
          {isVideoCall && (
            <CallButton
              icon={<FlipHorizontal className="h-6 w-6" />}
              label="Switch"
              onClick={handleSwitchCamera}
            />
          )}
        </div>

        {/* Duration */}
        {currentCall.status === 'connected' && (
          <p className="text-center text-foreground-muted mt-4">
            {formatDuration(callDuration)}
          </p>
        )}
      </div>
    </div>
  );
}

/**
 * Call Button Component
 */
function CallButton({
  icon,
  label,
  active,
  onClick,
}: {
  icon: React.ReactNode;
  label: string;
  active?: boolean;
  onClick: () => void;
}) {
  return (
    <div className="flex flex-col items-center gap-1">
      <button
        onClick={onClick}
        className={cn(
          'p-4 rounded-full transition-colors',
          active
            ? 'bg-foreground text-background'
            : 'bg-background-tertiary hover:bg-background-tertiary/80'
        )}
      >
        {icon}
      </button>
      <span className="text-xs text-foreground-muted">{label}</span>
    </div>
  );
}

/**
 * Call Ended Notification
 */
export function CallEnded() {
  const { currentCall } = useCallStore();
  const [visible, setVisible] = useState(false);
  const [messageSaved, setMessageSaved] = useState(false);

  // Save call history to chat when call ends
  useEffect(() => {
    if (currentCall?.status === 'ended' && !messageSaved) {
      // Calculate duration in seconds
      const durationSeconds = currentCall.startTime && currentCall.endTime
        ? Math.floor((currentCall.endTime - currentCall.startTime) / 1000)
        : undefined;

      // Create call message
      const callMessage: Message = {
        id: `call-${currentCall.callId}`,
        conversationId: currentCall.peerId,
        senderId: currentCall.direction === 'outgoing' ? 'self' : currentCall.peerId,
        content: '', // Content handled by callMetadata
        timestamp: currentCall.endTime || Date.now(),
        status: 'sent',
        type: 'call',
        callMetadata: {
          callType: currentCall.type,
          duration: durationSeconds,
          endReason: currentCall.endReason || 'completed',
          direction: currentCall.direction,
        } as CallMetadata,
      };

      // Add message to chat
      useChatStore.getState().addMessage(callMessage);
      setMessageSaved(true);
    }
  }, [currentCall, messageSaved]);

  useEffect(() => {
    if (currentCall?.status === 'ended') {
      setVisible(true);
      const timeout = setTimeout(() => {
        setVisible(false);
        useCallStore.getState().reset();
        setMessageSaved(false);
      }, 3000);
      return () => clearTimeout(timeout);
    }
  }, [currentCall?.status]);

  if (!visible || !currentCall) {
    return null;
  }

  const getEndMessage = (): string => {
    switch (currentCall.endReason) {
      case 'declined':
        return currentCall.direction === 'outgoing' ? 'Call declined' : 'You declined';
      case 'missed':
        return 'Missed call';
      case 'failed':
        return 'Call failed';
      case 'busy':
        return 'User is busy';
      default:
        return 'Call ended';
    }
  };

  const getDuration = (): string | null => {
    if (!currentCall.startTime || !currentCall.endTime) return null;
    const seconds = Math.floor((currentCall.endTime - currentCall.startTime) / 1000);
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const duration = getDuration();

  return (
    <div className="fixed top-4 left-1/2 -translate-x-1/2 z-50 bg-background-secondary border border-border rounded-xl shadow-lg p-4 flex items-center gap-4">
      <Avatar className="h-12 w-12">
        <AvatarImage src={currentCall.peerAvatar} />
        <AvatarFallback>{currentCall.peerName.charAt(0)}</AvatarFallback>
      </Avatar>
      <div>
        <p className="font-medium">{currentCall.peerName}</p>
        <p className="text-sm text-foreground-muted">
          {getEndMessage()}
          {duration && ` · ${duration}`}
        </p>
      </div>
      <button
        onClick={() => {
          setVisible(false);
          useCallStore.getState().reset();
        }}
        className="p-2 hover:bg-background-tertiary rounded-lg transition-colors"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}
