/**
 * Local Storage Encryption Utilities
 *
 * Provides AES-256-GCM encryption/decryption for chat data stored in localStorage.
 * Uses the master key from the Signal Protocol for consistency with the PIN-based system.
 */

import { signalProtocol } from './signal';

// Utility functions for base64 conversion
function base64ToUint8Array(base64: string): Uint8Array {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

function uint8ArrayToBase64(bytes: Uint8Array): string {
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}

/**
 * Encrypt data with AES-256-GCM using the master key
 */
async function encryptWithMasterKey(
  masterKey: Uint8Array,
  data: string
): Promise<{ ciphertext: string; iv: string }> {
  const iv = crypto.getRandomValues(new Uint8Array(12));

  // Create a fresh ArrayBuffer copy for cross-environment importKey compatibility
  const keyBuffer = new ArrayBuffer(masterKey.length);
  const keyView = new Uint8Array(keyBuffer);
  keyView.set(masterKey);

  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    keyBuffer,
    { name: 'AES-GCM' },
    false,
    ['encrypt']
  );

  const plaintext = new TextEncoder().encode(data);
  const ciphertext = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv },
    cryptoKey,
    plaintext
  );

  return {
    ciphertext: uint8ArrayToBase64(new Uint8Array(ciphertext)),
    iv: uint8ArrayToBase64(iv),
  };
}

/**
 * Decrypt data with AES-256-GCM using the master key
 */
async function decryptWithMasterKey(
  masterKey: Uint8Array,
  ciphertext: string,
  iv: string
): Promise<string> {
  // Create a fresh ArrayBuffer copy for cross-environment importKey compatibility
  const keyBuffer = new ArrayBuffer(masterKey.length);
  const keyView = new Uint8Array(keyBuffer);
  keyView.set(masterKey);

  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    keyBuffer,
    { name: 'AES-GCM' },
    false,
    ['decrypt']
  );

  const ciphertextBytes = base64ToUint8Array(ciphertext);
  const ivBytes = base64ToUint8Array(iv);

  // Create fresh ArrayBuffer copies to satisfy TypeScript BufferSource requirements
  const ivBuffer = ivBytes.buffer.slice(ivBytes.byteOffset, ivBytes.byteOffset + ivBytes.byteLength) as ArrayBuffer;
  const ciphertextBuffer = ciphertextBytes.buffer.slice(ciphertextBytes.byteOffset, ciphertextBytes.byteOffset + ciphertextBytes.byteLength) as ArrayBuffer;

  const plaintext = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: ivBuffer },
    cryptoKey,
    ciphertextBuffer
  );

  return new TextDecoder().decode(plaintext);
}

/**
 * Encrypt chat data for localStorage
 *
 * @param data - The chat data to encrypt (will be JSON serialized)
 * @returns Base64 encoded encrypted data in format "iv:ciphertext"
 * @throws Error if encryption is not enabled or master key is not available
 */
export async function encryptChatData(data: any): Promise<string> {
  try {
    // Ensure Signal Protocol is initialized
    await signalProtocol.initialize();

    // Check if encryption is enabled
    if (!signalProtocol.isEncryptionEnabled()) {
      throw new Error('Encryption is not enabled - set up PIN encryption first');
    }

    // Check if master key is available (user has unlocked with PIN)
    if (!signalProtocol.isUnlocked()) {
      throw new Error('Master key not available - unlock with PIN first');
    }

    // Get master key (accessing private property - consider adding public getter)
    const masterKey = (signalProtocol as any).masterKey as Uint8Array;
    if (!masterKey) {
      throw new Error('Master key not available');
    }

    // Serialize data to JSON
    const jsonData = JSON.stringify(data);

    // Encrypt the data
    const { ciphertext, iv } = await encryptWithMasterKey(masterKey, jsonData);

    // Return in format "iv:ciphertext" for easy storage
    return `${iv}:${ciphertext}`;
  } catch (error) {
    if (error instanceof Error) {
      throw new Error(`Failed to encrypt chat data: ${error.message}`);
    }
    throw new Error('Failed to encrypt chat data: Unknown error');
  }
}

/**
 * Decrypt chat data from localStorage
 *
 * @param encryptedData - The encrypted data in format "iv:ciphertext"
 * @returns The decrypted chat data (JSON parsed)
 * @throws Error if decryption fails or encryption is not enabled
 */
export async function decryptChatData(encryptedData: string): Promise<any> {
  try {
    // Ensure Signal Protocol is initialized
    await signalProtocol.initialize();

    // Check if encryption is enabled
    if (!signalProtocol.isEncryptionEnabled()) {
      throw new Error('Encryption is not enabled - set up PIN encryption first');
    }

    // Check if master key is available (user has unlocked with PIN)
    if (!signalProtocol.isUnlocked()) {
      throw new Error('Master key not available - unlock with PIN first');
    }

    // Get master key (accessing private property - consider adding public getter)
    const masterKey = (signalProtocol as any).masterKey as Uint8Array;
    if (!masterKey) {
      throw new Error('Master key not available');
    }

    // Parse the encrypted data format "iv:ciphertext"
    const parts = encryptedData.split(':');
    if (parts.length !== 2) {
      throw new Error('Invalid encrypted data format');
    }

    const [iv, ciphertext] = parts;

    // Decrypt the data
    const jsonData = await decryptWithMasterKey(masterKey, ciphertext, iv);

    // Parse JSON
    return JSON.parse(jsonData);
  } catch (error) {
    if (error instanceof Error) {
      throw new Error(`Failed to decrypt chat data: ${error.message}`);
    }
    throw new Error('Failed to decrypt chat data: Unknown error');
  }
}