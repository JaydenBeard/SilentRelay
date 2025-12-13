import { describe, it, expect, vi } from 'vitest';
import { sanitizeMessageContent, isSafeThumbnailUrl, sanitizeFileName, sanitizeUserContent } from './sanitization';
import { logger } from './logger';

// Mock logger
vi.mock('./logger', () => ({
  logger: {
    warn: vi.fn(),
    error: vi.fn(),
    info: vi.fn()
  }
}));

describe('Sanitization Utilities - Security Tests', () => {
  describe('sanitizeMessageContent', () => {
    it('should sanitize basic XSS attempts', () => {
      const malicious = '<script>alert("XSS")</script>';
      const result = sanitizeMessageContent(malicious);
      expect(result).not.toContain('<script>');
      expect(result).not.toContain('alert');
    });

    it('should block script injection patterns', () => {
      const malicious = 'Hello <img src="x" onerror="alert(1)">';
      const result = sanitizeMessageContent(malicious);
      expect(result).not.toContain('onerror');
      expect(result).not.toContain('alert');
    });

    it('should handle javascript: URLs', () => {
      const malicious = '<a href="javascript:alert(1)">Click</a>';
      const result = sanitizeMessageContent(malicious);
      expect(result).not.toContain('javascript:');
    });

    it('should allow safe formatting tags', () => {
      const safe = 'Hello <b>world</b> and <i>friends</i>';
      const result = sanitizeMessageContent(safe);
      expect(result).toContain('<b>world</b>');
      expect(result).toContain('<i>friends</i>');
    });

    it('should handle non-string input gracefully', () => {
      const result = sanitizeMessageContent(null as any);
      expect(result).toBe('');
    });

    it('should detect and block malicious patterns', () => {
      const malicious = 'Hello <script>alert(1)</script> world';
      const result = sanitizeMessageContent(malicious);
      expect(result).not.toContain('<script>');
    });
  });

  describe('isSafeThumbnailUrl', () => {
    it('should allow data image URLs', () => {
      const safeUrl = 'data:image/png;base64,abc123';
      expect(isSafeThumbnailUrl(safeUrl)).toBe(true);
    });

    it('should block javascript: URLs', () => {
      const maliciousUrl = 'javascript:alert(1)';
      expect(isSafeThumbnailUrl(maliciousUrl)).toBe(false);
    });

    it('should validate HTTPS URLs with domain restrictions', () => {
      const safeUrl = 'https://cdn.silentrelay.com/image.jpg';
      const maliciousUrl = 'https://evil.com/malicious.js';

      expect(isSafeThumbnailUrl(safeUrl)).toBe(true);
      expect(isSafeThumbnailUrl(maliciousUrl)).toBe(false);
    });

    it('should block URLs with malicious extensions', () => {
      const maliciousUrl = 'https://cdn.silentrelay.com/malicious.js';
      expect(isSafeThumbnailUrl(maliciousUrl)).toBe(false);
    });

    it('should handle non-string input gracefully', () => {
      expect(isSafeThumbnailUrl(null as any)).toBe(false);
    });
  });

  describe('sanitizeFileName', () => {
    it('should remove path traversal characters', () => {
      const malicious = '../../malicious.js';
      const result = sanitizeFileName(malicious);
      expect(result).not.toContain('..');
      expect(result).not.toContain('/');
      expect(result).not.toContain('\\');
    });

    it('should handle safe file names', () => {
      const safe = 'document.pdf';
      const result = sanitizeFileName(safe);
      expect(result).toBe('document.pdf');
    });

    it('should sanitize file names with special characters', () => {
      const malicious = 'malicious<>.js';
      const result = sanitizeFileName(malicious);
      expect(result).not.toContain('<');
      expect(result).not.toContain('>');
    });

    it('should handle non-string input gracefully', () => {
      const result = sanitizeFileName(null as any);
      expect(result).toBe('');
    });
  });

  describe('sanitizeUserContent', () => {
    it('should sanitize user-generated content strictly', () => {
      const malicious = '<script>alert(1)</script><b>Hello</b>';
      const result = sanitizeUserContent(malicious);
      expect(result).not.toContain('<script>');
      expect(result).toContain('<b>Hello</b>');
    });

    it('should handle XSS in user content', () => {
      const malicious = 'Hello <img src="x" onerror="alert(1)">';
      const result = sanitizeUserContent(malicious);
      expect(result).not.toContain('onerror');
    });

    it('should handle non-string input gracefully', () => {
      const result = sanitizeUserContent(null as any);
      expect(result).toBe('');
    });
  });

  describe('Security Integration Tests', () => {
    it('should handle complex XSS payloads', () => {
      const complexPayload = `
        <script>alert(1)</script>
        <img src="x" onerror="alert(2)">
        <svg onload="alert(3)">
        <a href="javascript:alert(4)">Click</a>
      `;
      const result = sanitizeMessageContent(complexPayload);
      expect(result).not.toContain('<script>');
      expect(result).not.toContain('onerror');
      expect(result).not.toContain('onload');
      expect(result).not.toContain('javascript:');
    });

    it('should maintain security with edge cases', () => {
      const edgeCases = [
        { input: '', expected: '' },
        { input: '   ', expected: '   ' },
        { input: '<b>Hello</b>', expected: '<b>Hello</b>' },
        { input: 'Hello & goodbye', expected: 'Hello & goodbye' }
      ];

      edgeCases.forEach(({ input, expected }) => {
        const result = sanitizeMessageContent(input);
        expect(result).toBe(expected);
      });
    });
  });
});