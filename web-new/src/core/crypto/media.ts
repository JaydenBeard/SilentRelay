/**
 * Media Encryption Module
 *
 * Provides AES-256-GCM encryption for file attachments.
 * Files are encrypted client-side before upload, ensuring
 * the server only sees encrypted blobs.
 */

const ALGORITHM = 'AES-GCM';
const KEY_LENGTH = 256;
const IV_LENGTH = 12;
const TAG_LENGTH = 128;

export interface EncryptedMedia {
  ciphertext: Uint8Array;
  key: Uint8Array;
  iv: Uint8Array;
}

/**
 * Encrypt media data with AES-256-GCM
 * Returns the encrypted data along with the key and IV needed for decryption
 */
export async function encryptMedia(data: ArrayBuffer): Promise<EncryptedMedia> {
  // Generate random key and IV
  const key = crypto.getRandomValues(new Uint8Array(KEY_LENGTH / 8));
  const iv = crypto.getRandomValues(new Uint8Array(IV_LENGTH));

  // Import key for encryption
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    key,
    { name: ALGORITHM },
    false,
    ['encrypt']
  );

  // Encrypt
  const ciphertext = await crypto.subtle.encrypt(
    { name: ALGORITHM, iv, tagLength: TAG_LENGTH },
    cryptoKey,
    data
  );

  return {
    ciphertext: new Uint8Array(ciphertext),
    key,
    iv,
  };
}

/**
 * Decrypt media data with AES-256-GCM
 */
export async function decryptMedia(
  ciphertext: Uint8Array,
  key: Uint8Array,
  iv: Uint8Array
): Promise<ArrayBuffer> {
  // Import key for decryption
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    key,
    { name: ALGORITHM },
    false,
    ['decrypt']
  );

  // Decrypt
  const plaintext = await crypto.subtle.decrypt(
    { name: ALGORITHM, iv, tagLength: TAG_LENGTH },
    cryptoKey,
    ciphertext
  );

  return plaintext;
}

/**
 * Generate a thumbnail for image files
 * Returns a base64-encoded JPEG thumbnail
 */
export async function generateThumbnail(
  file: File,
  maxSize: number = 200
): Promise<string | null> {
  if (!file.type.startsWith('image/')) {
    return null;
  }

  return new Promise((resolve) => {
    const img = new Image();
    img.onload = () => {
      // Calculate thumbnail dimensions
      let width = img.width;
      let height = img.height;

      if (width > height) {
        if (width > maxSize) {
          height = Math.round((height * maxSize) / width);
          width = maxSize;
        }
      } else {
        if (height > maxSize) {
          width = Math.round((width * maxSize) / height);
          height = maxSize;
        }
      }

      // Draw to canvas
      const canvas = document.createElement('canvas');
      canvas.width = width;
      canvas.height = height;

      const ctx = canvas.getContext('2d');
      if (!ctx) {
        resolve(null);
        return;
      }

      ctx.drawImage(img, 0, 0, width, height);

      // Convert to base64 JPEG
      resolve(canvas.toDataURL('image/jpeg', 0.7));
    };

    img.onerror = () => resolve(null);
    img.src = URL.createObjectURL(file);
  });
}

/**
 * Validate file for upload
 */
export function validateFile(file: File): { valid: boolean; error?: string } {
  const MAX_SIZE = 100 * 1024 * 1024; // 100MB

  const ALLOWED_TYPES = [
    // Images
    'image/jpeg',
    'image/png',
    'image/gif',
    'image/webp',
    // Videos
    'video/mp4',
    'video/webm',
    'video/quicktime',
    // Audio
    'audio/mpeg',
    'audio/wav',
    'audio/ogg',
    'audio/webm',
    // Documents
    'application/pdf',
    'application/msword',
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    'application/vnd.ms-excel',
    'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    'text/plain',
  ];

  if (file.size > MAX_SIZE) {
    return { valid: false, error: 'File size exceeds 100MB limit' };
  }

  if (!ALLOWED_TYPES.includes(file.type)) {
    return { valid: false, error: 'File type not supported' };
  }

  return { valid: true };
}

/**
 * Get media type category from MIME type
 */
export function getMediaType(mimeType: string): 'image' | 'video' | 'audio' | 'document' {
  if (mimeType.startsWith('image/')) return 'image';
  if (mimeType.startsWith('video/')) return 'video';
  if (mimeType.startsWith('audio/')) return 'audio';
  return 'document';
}
