/**
 * Vitest setup file - provides Web Crypto API polyfill for Node.js
 */

import { webcrypto } from 'crypto';

// Polyfill Web Crypto API for Node.js
if (typeof globalThis.crypto === 'undefined') {
    Object.defineProperty(globalThis, 'crypto', {
        value: webcrypto,
        writable: true,
        configurable: true,
    });
}

// Mock TextEncoder/TextDecoder if needed
if (typeof globalThis.TextEncoder === 'undefined') {
    const { TextEncoder, TextDecoder } = await import('util');
    Object.defineProperty(globalThis, 'TextEncoder', { value: TextEncoder });
    Object.defineProperty(globalThis, 'TextDecoder', { value: TextDecoder });
}
