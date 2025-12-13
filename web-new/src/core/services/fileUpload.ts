/**
 * File Upload Service
 *
 * Handles client-side encryption of files before upload,
 * and decryption after download.
 */

import { encryptMedia, decryptMedia, generateThumbnail, validateFile } from '../crypto/media';
import { useAuthStore } from '../store/authStore';
import type { FileMetadata } from '../types';

export interface UploadProgress {
  loaded: number;
  total: number;
  percentage: number;
}

export interface UploadResult {
  mediaId: string;
  metadata: FileMetadata;
}

/**
 * Upload an encrypted file
 */
export async function uploadFile(
  file: File,
  onProgress?: (progress: UploadProgress) => void
): Promise<UploadResult> {
  // Validate file
  const validation = validateFile(file);
  if (!validation.valid) {
    throw new Error(validation.error);
  }

  // Read file as ArrayBuffer
  const fileData = await file.arrayBuffer();

  // Encrypt the file
  const encrypted = await encryptMedia(fileData);

  // Generate thumbnail for images
  const thumbnail = await generateThumbnail(file);

  // Get upload URL from server
  const token = useAuthStore.getState().token;
  const uploadUrlResponse = await fetch('/api/v1/media/upload-url', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      file_name: file.name,
      content_type: file.type,
      file_size: encrypted.ciphertext.length,
    }),
  });

  if (!uploadUrlResponse.ok) {
    throw new Error('Failed to get upload URL');
  }

  const { uploadUrl: rawUploadUrl, fileId: mediaId } = await uploadUrlResponse.json();

  // Convert absolute URL to relative for Vite proxy compatibility
  // Backend may return https://localhost/api/... but we need /api/... for proxy to work
  let uploadUrl = rawUploadUrl;
  if (typeof uploadUrl === 'string') {
    // Remove https://localhost or http://localhost prefix
    uploadUrl = uploadUrl.replace(/^https?:\/\/localhost(:\d+)?/, '');
  }

  // Upload encrypted file with progress tracking
  await uploadWithProgress(uploadUrl, encrypted.ciphertext, file.type, onProgress);

  // Return metadata needed for the message
  const metadata: FileMetadata = {
    fileName: file.name,
    fileSize: file.size,
    mimeType: file.type,
    mediaId,
    thumbnail: thumbnail || undefined,
    encryptionKey: Array.from(encrypted.key),
    iv: Array.from(encrypted.iv),
  };

  return { mediaId, metadata };
}

/**
 * Download and decrypt a file
 */
export async function downloadFile(metadata: FileMetadata): Promise<Blob> {
  const token = useAuthStore.getState().token;

  // Validate metadata has required encryption fields
  if (!metadata.encryptionKey || !metadata.iv || !metadata.mediaId) {
    console.error('Invalid file metadata - missing encryption info:', {
      hasKey: !!metadata.encryptionKey,
      hasIv: !!metadata.iv,
      hasMediaId: !!metadata.mediaId,
    });
    throw new Error('Invalid file metadata - missing encryption info');
  }

  // Get download URL
  const downloadUrlResponse = await fetch(`/api/v1/media/download-url/${metadata.mediaId}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!downloadUrlResponse.ok) {
    const errorText = await downloadUrlResponse.text();
    console.error('Failed to get download URL:', downloadUrlResponse.status, errorText);
    throw new Error('Failed to get download URL');
  }

  const { url: rawDownloadUrl } = await downloadUrlResponse.json();

  // Convert absolute URL to relative for Vite proxy compatibility
  // Backend may return https://localhost/api/... but we need /api/... for proxy
  let downloadUrl = rawDownloadUrl;
  if (typeof downloadUrl === 'string') {
    downloadUrl = downloadUrl.replace(/^https?:\/\/localhost(:\d+)?/, '');
  }

  // Download encrypted file
  const response = await fetch(downloadUrl, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });
  if (!response.ok) {
    const errorText = await response.text();
    console.error('Failed to download file:', response.status, errorText);
    throw new Error('Failed to download file');
  }

  const encryptedData = await response.arrayBuffer();

  // Validate encryption parameters
  const keyArray = Array.isArray(metadata.encryptionKey)
    ? metadata.encryptionKey
    : Object.values(metadata.encryptionKey as unknown as Record<string, number>);
  const ivArray = Array.isArray(metadata.iv)
    ? metadata.iv
    : Object.values(metadata.iv as unknown as Record<string, number>);

  if (keyArray.length !== 32) {
    console.error('Invalid encryption key length:', keyArray.length, 'expected 32');
    throw new Error(`Invalid encryption key length: ${keyArray.length}`);
  }
  if (ivArray.length !== 12) {
    console.error('Invalid IV length:', ivArray.length, 'expected 12');
    throw new Error(`Invalid IV length: ${ivArray.length}`);
  }

  // Decrypt the file
  try {
    const decrypted = await decryptMedia(
      new Uint8Array(encryptedData),
      new Uint8Array(keyArray),
      new Uint8Array(ivArray)
    );
    return new Blob([decrypted], { type: metadata.mimeType });
  } catch (error) {
    console.error('Decryption failed:', error);
    console.error('Encrypted data size:', encryptedData.byteLength);
    console.error('Key (first 4 bytes):', keyArray.slice(0, 4));
    console.error('IV:', ivArray);
    throw error;
  }
}

/**
 * Upload with XMLHttpRequest for progress tracking
 */
function uploadWithProgress(
  url: string,
  data: Uint8Array,
  contentType: string,
  onProgress?: (progress: UploadProgress) => void
): Promise<void> {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    const token = useAuthStore.getState().token;

    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable && onProgress) {
        onProgress({
          loaded: event.loaded,
          total: event.total,
          percentage: Math.round((event.loaded / event.total) * 100),
        });
      }
    });

    xhr.addEventListener('load', () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
      } else {
        reject(new Error(`Upload failed with status ${xhr.status}`));
      }
    });

    xhr.addEventListener('error', () => {
      reject(new Error('Upload failed'));
    });

    xhr.open('PUT', url);
    xhr.setRequestHeader('Content-Type', contentType);
    xhr.setRequestHeader('Authorization', `Bearer ${token}`);
    // Convert Uint8Array to Blob for XHR compatibility
    xhr.send(new Blob([data.buffer as ArrayBuffer]));
  });
}

/**
 * Create a file message content object
 * Format: [FILE:{json}] to match MessageBubble's isFileMessage check
 */
export function createFileMessageContent(metadata: FileMetadata): string {
  return `[FILE:${JSON.stringify(metadata)}]`;
}

/**
 * Parse a file message content
 */
export function parseFileMessageContent(content: string): FileMetadata & { type: string; mediaType: string } | null {
  try {
    const parsed = JSON.parse(content);
    if (parsed.type === 'file') {
      return parsed;
    }
    return null;
  } catch {
    return null;
  }
}
