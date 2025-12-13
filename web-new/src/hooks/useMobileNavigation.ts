/**
 * useMobileNavigation Hook
 *
 * Manages the dual-panel navigation state for mobile chat interface.
 * Handles transitions between conversation list and active chat view.
 */

import { useState, useCallback, useEffect, useRef } from 'react';
import { useIsMobile } from './useViewport';

export type MobileView = 'list' | 'chat';

export interface MobileNavigationState {
  /** Current view on mobile */
  currentView: MobileView;
  /** Whether showing the chat view */
  showChat: boolean;
  /** Whether showing the conversation list */
  showList: boolean;
  /** Animation state */
  animating: boolean;
  /** Animation direction */
  animationDirection: 'forward' | 'backward' | null;
}

interface UseMobileNavigationOptions {
  /** Initial view (default: 'list') */
  initialView?: MobileView;
  /** Callback when navigating to chat */
  onNavigateToChat?: (conversationId: string) => void;
  /** Callback when navigating to list */
  onNavigateToList?: () => void;
  /** Animation duration in ms (default: 250) */
  animationDuration?: number;
}

export function useMobileNavigation(options: UseMobileNavigationOptions = {}) {
  const {
    initialView = 'list',
    onNavigateToChat,
    onNavigateToList,
    animationDuration = 250,
  } = options;

  const isMobile = useIsMobile();
  const [currentView, setCurrentView] = useState<MobileView>(initialView);
  const [animating, setAnimating] = useState(false);
  const [animationDirection, setAnimationDirection] = useState<'forward' | 'backward' | null>(null);
  const animationTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  // Use refs for callbacks to prevent infinite render loops
  // This prevents the callbacks from being dependencies in useCallback/useEffect
  const onNavigateToChatRef = useRef(onNavigateToChat);
  const onNavigateToListRef = useRef(onNavigateToList);

  // Keep refs in sync with props
  useEffect(() => {
    onNavigateToChatRef.current = onNavigateToChat;
  }, [onNavigateToChat]);

  useEffect(() => {
    onNavigateToListRef.current = onNavigateToList;
  }, [onNavigateToList]);

  // Clear animation timeout on unmount
  useEffect(() => {
    return () => {
      if (animationTimeoutRef.current) {
        clearTimeout(animationTimeoutRef.current);
      }
    };
  }, []);

  const navigateToChat = useCallback((conversationId: string) => {
    if (!isMobile) {
      // On desktop, just trigger callback without view change
      onNavigateToChatRef.current?.(conversationId);
      return;
    }

    // Clear any existing animation
    if (animationTimeoutRef.current) {
      clearTimeout(animationTimeoutRef.current);
    }

    setAnimating(true);
    setAnimationDirection('forward');
    setCurrentView('chat');

    animationTimeoutRef.current = setTimeout(() => {
      setAnimating(false);
      setAnimationDirection(null);
      onNavigateToChatRef.current?.(conversationId);
    }, animationDuration);
  }, [isMobile, animationDuration]);

  const navigateToList = useCallback(() => {
    if (!isMobile) {
      onNavigateToListRef.current?.();
      return;
    }

    // Clear any existing animation
    if (animationTimeoutRef.current) {
      clearTimeout(animationTimeoutRef.current);
    }

    setAnimating(true);
    setAnimationDirection('backward');
    setCurrentView('list');

    animationTimeoutRef.current = setTimeout(() => {
      setAnimating(false);
      setAnimationDirection(null);
      onNavigateToListRef.current?.();
    }, animationDuration);
  }, [isMobile, animationDuration]);

  // Handle browser back button
  useEffect(() => {
    if (!isMobile) return;

    const handlePopState = (e: PopStateEvent) => {
      if (e.state?.view === 'list' && currentView === 'chat') {
        navigateToList();
      }
    };

    window.addEventListener('popstate', handlePopState);

    // Push initial state
    if (currentView === 'list') {
      window.history.replaceState({ view: 'list' }, '', window.location.href);
    }

    return () => {
      window.removeEventListener('popstate', handlePopState);
    };
  }, [isMobile, currentView, navigateToList]);

  // Push history state when navigating to chat
  useEffect(() => {
    if (!isMobile || currentView !== 'chat') return;

    window.history.pushState({ view: 'chat' }, '', window.location.href);
  }, [isMobile, currentView]);

  // Compute visibility states
  const showChat = !isMobile || currentView === 'chat';
  const showList = !isMobile || currentView === 'list';

  // Get animation classes for panels
  const getListPanelClasses = useCallback(() => {
    if (!isMobile) return '';

    if (animating && animationDirection === 'forward') {
      return 'slide-out-left';
    }
    if (animating && animationDirection === 'backward') {
      return 'slide-in-left';
    }

    return currentView === 'list' ? '' : 'hidden';
  }, [isMobile, animating, animationDirection, currentView]);

  const getChatPanelClasses = useCallback(() => {
    if (!isMobile) return '';

    if (animating && animationDirection === 'forward') {
      return 'slide-in-right';
    }
    if (animating && animationDirection === 'backward') {
      return 'slide-out-right';
    }

    return currentView === 'chat' ? '' : 'hidden';
  }, [isMobile, animating, animationDirection, currentView]);

  return {
    /** Current view state information */
    state: {
      currentView,
      showChat,
      showList,
      animating,
      animationDirection,
    } as MobileNavigationState,
    /** Navigate to chat view */
    navigateToChat,
    /** Navigate to list view */
    navigateToList,
    /** Whether device is mobile */
    isMobile,
    /** Get animation classes for list panel */
    getListPanelClasses,
    /** Get animation classes for chat panel */
    getChatPanelClasses,
  };
}

/**
 * Hook for handling back navigation on mobile
 */
export function useBackNavigation(onBack: () => void) {
  const isMobile = useIsMobile();

  useEffect(() => {
    if (!isMobile) return;

    const handleBackButton = (e: PopStateEvent) => {
      e.preventDefault();
      onBack();
    };

    window.addEventListener('popstate', handleBackButton);

    return () => {
      window.removeEventListener('popstate', handleBackButton);
    };
  }, [isMobile, onBack]);
}

/**
 * Hook for preventing scroll on body when modal/panel is open
 */
export function usePreventBodyScroll(prevent: boolean) {
  useEffect(() => {
    if (prevent) {
      const originalStyle = document.body.style.overflow;
      document.body.style.overflow = 'hidden';

      return () => {
        document.body.style.overflow = originalStyle;
      };
    }
  }, [prevent]);
}