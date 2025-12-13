/**
 * useSwipeGesture Hook
 *
 * Provides swipe gesture detection for touch interactions.
 * Supports swipe-to-reply, swipe-to-delete, and navigation gestures.
 */

import { useRef, useCallback, useState } from 'react';

export interface SwipeState {
  /** Current horizontal offset */
  offsetX: number;
  /** Current vertical offset */
  offsetY: number;
  /** Whether a swipe is currently active */
  isSwiping: boolean;
  /** Swipe direction once threshold is reached */
  direction: 'left' | 'right' | 'up' | 'down' | null;
  /** Swipe velocity (pixels per ms) */
  velocity: number;
}

export interface SwipeOptions {
  /** Minimum distance to trigger swipe (default: 50) */
  threshold?: number;
  /** Maximum distance for the swipe action */
  maxDistance?: number;
  /** Enable horizontal swipe */
  horizontal?: boolean;
  /** Enable vertical swipe */
  vertical?: boolean;
  /** Callback when swipe left is completed */
  onSwipeLeft?: () => void;
  /** Callback when swipe right is completed */
  onSwipeRight?: () => void;
  /** Callback when swipe up is completed */
  onSwipeUp?: () => void;
  /** Callback when swipe down is completed */
  onSwipeDown?: () => void;
  /** Callback during swipe with current offset */
  onSwipe?: (state: SwipeState) => void;
  /** Callback when swipe ends */
  onSwipeEnd?: (state: SwipeState) => void;
  /** Prevent default touch behavior */
  preventDefault?: boolean;
}

interface TouchPoint {
  x: number;
  y: number;
  time: number;
}

export function useSwipeGesture(options: SwipeOptions = {}) {
  const {
    threshold = 50,
    maxDistance = 150,
    horizontal = true,
    vertical = false,
    onSwipeLeft,
    onSwipeRight,
    onSwipeUp,
    onSwipeDown,
    onSwipe,
    onSwipeEnd,
    preventDefault = false,
  } = options;

  const [swipeState, setSwipeState] = useState<SwipeState>({
    offsetX: 0,
    offsetY: 0,
    isSwiping: false,
    direction: null,
    velocity: 0,
  });

  const startRef = useRef<TouchPoint | null>(null);
  const currentRef = useRef<TouchPoint | null>(null);
  const isSwipingRef = useRef(false);

  const handleTouchStart = useCallback((e: React.TouchEvent | TouchEvent) => {
    const touch = e.touches[0];
    startRef.current = {
      x: touch.clientX,
      y: touch.clientY,
      time: Date.now(),
    };
    currentRef.current = startRef.current;
    isSwipingRef.current = false;
  }, []);

  const handleTouchMove = useCallback((e: React.TouchEvent | TouchEvent) => {
    if (!startRef.current) return;

    const touch = e.touches[0];
    const deltaX = touch.clientX - startRef.current.x;
    const deltaY = touch.clientY - startRef.current.y;
    const absX = Math.abs(deltaX);
    const absY = Math.abs(deltaY);

    // Determine swipe direction
    let direction: SwipeState['direction'] = null;
    if (horizontal && absX > absY && absX > 10) {
      direction = deltaX > 0 ? 'right' : 'left';
      if (preventDefault) {
        e.preventDefault();
      }
    } else if (vertical && absY > absX && absY > 10) {
      direction = deltaY > 0 ? 'down' : 'up';
      if (preventDefault) {
        e.preventDefault();
      }
    }

    // Clamp offsets to max distance
    const clampedX = Math.max(-maxDistance, Math.min(maxDistance, deltaX));
    const clampedY = Math.max(-maxDistance, Math.min(maxDistance, deltaY));

    // Calculate velocity
    const now = Date.now();
    const timeDelta = now - (currentRef.current?.time || now);
    const distance = Math.sqrt(deltaX * deltaX + deltaY * deltaY);
    const velocity = timeDelta > 0 ? distance / timeDelta : 0;

    currentRef.current = {
      x: touch.clientX,
      y: touch.clientY,
      time: now,
    };

    const newState: SwipeState = {
      offsetX: horizontal ? clampedX : 0,
      offsetY: vertical ? clampedY : 0,
      isSwiping: direction !== null,
      direction,
      velocity,
    };

    isSwipingRef.current = direction !== null;
    setSwipeState(newState);
    onSwipe?.(newState);
  }, [horizontal, vertical, maxDistance, onSwipe, preventDefault]);

  const handleTouchEnd = useCallback(() => {
    if (!startRef.current || !currentRef.current) {
      setSwipeState({
        offsetX: 0,
        offsetY: 0,
        isSwiping: false,
        direction: null,
        velocity: 0,
      });
      return;
    }

    const deltaX = currentRef.current.x - startRef.current.x;
    const deltaY = currentRef.current.y - startRef.current.y;
    const absX = Math.abs(deltaX);
    const absY = Math.abs(deltaY);

    const finalState: SwipeState = {
      ...swipeState,
      isSwiping: false,
    };

    // Check if swipe threshold was reached
    if (horizontal && absX >= threshold && absX > absY) {
      if (deltaX > 0) {
        onSwipeRight?.();
      } else {
        onSwipeLeft?.();
      }
    } else if (vertical && absY >= threshold && absY > absX) {
      if (deltaY > 0) {
        onSwipeDown?.();
      } else {
        onSwipeUp?.();
      }
    }

    onSwipeEnd?.(finalState);

    // Reset state
    startRef.current = null;
    currentRef.current = null;
    isSwipingRef.current = false;
    setSwipeState({
      offsetX: 0,
      offsetY: 0,
      isSwiping: false,
      direction: null,
      velocity: 0,
    });
  }, [horizontal, vertical, threshold, swipeState, onSwipeLeft, onSwipeRight, onSwipeUp, onSwipeDown, onSwipeEnd]);

  const handlers = {
    onTouchStart: handleTouchStart,
    onTouchMove: handleTouchMove,
    onTouchEnd: handleTouchEnd,
    onTouchCancel: handleTouchEnd,
  };

  return {
    swipeState,
    handlers,
    reset: () => setSwipeState({
      offsetX: 0,
      offsetY: 0,
      isSwiping: false,
      direction: null,
      velocity: 0,
    }),
  };
}

/**
 * Hook for pull-to-refresh gesture
 */
export function usePullToRefresh(options: {
  onRefresh: () => Promise<void>;
  threshold?: number;
  maxPull?: number;
}) {
  const { onRefresh, threshold = 80, maxPull = 120 } = options;
  const [isPulling, setIsPulling] = useState(false);
  const [pullDistance, setPullDistance] = useState(0);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const startY = useRef(0);
  const scrollableRef = useRef<HTMLElement | null>(null);

  const handleTouchStart = useCallback((e: TouchEvent) => {
    // Only start pull if at top of scroll container
    if (scrollableRef.current && scrollableRef.current.scrollTop > 0) {
      return;
    }
    startY.current = e.touches[0].clientY;
    setIsPulling(true);
  }, []);

  const handleTouchMove = useCallback((e: TouchEvent) => {
    if (!isPulling || isRefreshing) return;

    const currentY = e.touches[0].clientY;
    const diff = currentY - startY.current;

    if (diff > 0) {
      // Apply resistance to pull
      const resistance = 0.5;
      const pull = Math.min(diff * resistance, maxPull);
      setPullDistance(pull);

      // Prevent scroll while pulling
      if (scrollableRef.current && scrollableRef.current.scrollTop === 0) {
        e.preventDefault();
      }
    }
  }, [isPulling, isRefreshing, maxPull]);

  const handleTouchEnd = useCallback(async () => {
    if (!isPulling) return;

    if (pullDistance >= threshold && !isRefreshing) {
      setIsRefreshing(true);
      setPullDistance(60); // Keep spinner visible

      try {
        await onRefresh();
      } finally {
        setIsRefreshing(false);
        setPullDistance(0);
      }
    } else {
      setPullDistance(0);
    }

    setIsPulling(false);
  }, [isPulling, pullDistance, threshold, isRefreshing, onRefresh]);

  const bindElement = useCallback((element: HTMLElement | null) => {
    scrollableRef.current = element;

    if (element) {
      element.addEventListener('touchstart', handleTouchStart, { passive: true });
      element.addEventListener('touchmove', handleTouchMove, { passive: false });
      element.addEventListener('touchend', handleTouchEnd);
    }

    return () => {
      if (element) {
        element.removeEventListener('touchstart', handleTouchStart);
        element.removeEventListener('touchmove', handleTouchMove);
        element.removeEventListener('touchend', handleTouchEnd);
      }
    };
  }, [handleTouchStart, handleTouchMove, handleTouchEnd]);

  return {
    pullDistance,
    isRefreshing,
    isPulling,
    bindElement,
    progress: Math.min(pullDistance / threshold, 1),
  };
}