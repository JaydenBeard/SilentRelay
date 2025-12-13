/**
 * useViewport Hook
 *
 * Handles Visual Viewport API for keyboard detection and safe viewport management.
 * Provides dynamic viewport height that accounts for mobile browser chrome and virtual keyboards.
 */

import { useState, useEffect, useCallback } from 'react';

interface ViewportState {
  /** Current visual viewport height in pixels */
  height: number;
  /** Current visual viewport width in pixels */
  width: number;
  /** Current scroll offset */
  offsetTop: number;
  /** Whether the virtual keyboard is open */
  isKeyboardOpen: boolean;
  /** Estimated keyboard height */
  keyboardHeight: number;
  /** Whether in landscape orientation */
  isLandscape: boolean;
  /** Safe area insets */
  safeAreaInsets: {
    top: number;
    bottom: number;
    left: number;
    right: number;
  };
}

export function useViewport() {
  const [viewport, setViewport] = useState<ViewportState>(() => ({
    height: typeof window !== 'undefined' ? window.innerHeight : 0,
    width: typeof window !== 'undefined' ? window.innerWidth : 0,
    offsetTop: 0,
    isKeyboardOpen: false,
    keyboardHeight: 0,
    isLandscape: typeof window !== 'undefined' ? window.innerWidth > window.innerHeight : false,
    safeAreaInsets: { top: 0, bottom: 0, left: 0, right: 0 },
  }));

  const updateViewport = useCallback(() => {
    const vv = window.visualViewport;
    const windowHeight = window.innerHeight;
    
    if (vv) {
      const keyboardHeight = windowHeight - vv.height;
      const isKeyboardOpen = keyboardHeight > 150; // Threshold to detect keyboard
      
      setViewport({
        height: vv.height,
        width: vv.width,
        offsetTop: vv.offsetTop,
        isKeyboardOpen,
        keyboardHeight: isKeyboardOpen ? keyboardHeight : 0,
        isLandscape: vv.width > vv.height,
        safeAreaInsets: getSafeAreaInsets(),
      });

      // Update CSS custom properties
      document.documentElement.style.setProperty('--visual-viewport-height', `${vv.height}px`);
      document.documentElement.style.setProperty('--keyboard-height', `${isKeyboardOpen ? keyboardHeight : 0}px`);
      document.documentElement.style.setProperty('--vh', `${vv.height * 0.01}px`);
    } else {
      setViewport((prev) => ({
        ...prev,
        height: windowHeight,
        width: window.innerWidth,
        isLandscape: window.innerWidth > window.innerHeight,
        safeAreaInsets: getSafeAreaInsets(),
      }));

      document.documentElement.style.setProperty('--visual-viewport-height', `${windowHeight}px`);
      document.documentElement.style.setProperty('--vh', `${windowHeight * 0.01}px`);
    }
  }, []);

  useEffect(() => {
    const vv = window.visualViewport;

    // Initial update
    updateViewport();

    // Listen to visual viewport changes
    if (vv) {
      vv.addEventListener('resize', updateViewport);
      vv.addEventListener('scroll', updateViewport);
    }

    // Fallback for browsers without visualViewport
    window.addEventListener('resize', updateViewport);
    window.addEventListener('orientationchange', updateViewport);

    return () => {
      if (vv) {
        vv.removeEventListener('resize', updateViewport);
        vv.removeEventListener('scroll', updateViewport);
      }
      window.removeEventListener('resize', updateViewport);
      window.removeEventListener('orientationchange', updateViewport);
    };
  }, [updateViewport]);

  return viewport;
}

/**
 * Get safe area insets from CSS environment variables
 */
function getSafeAreaInsets() {
  if (typeof window === 'undefined') {
    return { top: 0, bottom: 0, left: 0, right: 0 };
  }

  const computedStyle = getComputedStyle(document.documentElement);
  
  const parseInset = (prop: string): number => {
    const value = computedStyle.getPropertyValue(prop);
    return parseInt(value, 10) || 0;
  };

  return {
    top: parseInset('--safe-area-top') || parseInt(computedStyle.getPropertyValue('env(safe-area-inset-top)'), 10) || 0,
    bottom: parseInset('--safe-area-bottom') || parseInt(computedStyle.getPropertyValue('env(safe-area-inset-bottom)'), 10) || 0,
    left: parseInset('--safe-area-left') || parseInt(computedStyle.getPropertyValue('env(safe-area-inset-left)'), 10) || 0,
    right: parseInset('--safe-area-right') || parseInt(computedStyle.getPropertyValue('env(safe-area-inset-right)'), 10) || 0,
  };
}

/**
 * Check if the current device is likely a mobile device
 */
export function useIsMobile(breakpoint: number = 768) {
  const [isMobile, setIsMobile] = useState(
    typeof window !== 'undefined' ? window.innerWidth < breakpoint : false
  );

  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth < breakpoint);
    };

    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [breakpoint]);

  return isMobile;
}

/**
 * Detect if device supports touch
 */
export function useIsTouchDevice() {
  const [isTouch, setIsTouch] = useState(false);

  useEffect(() => {
    setIsTouch(
      'ontouchstart' in window ||
      navigator.maxTouchPoints > 0 ||
      // @ts-expect-error - Legacy check
      navigator.msMaxTouchPoints > 0
    );
  }, []);

  return isTouch;
}