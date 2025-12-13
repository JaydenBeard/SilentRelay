/**
 * Protocol Adapter Layer - Bridges Matrix Olm (frontend) with Signal Protocol (backend)
 *
 * This adapter resolves the protocol implementation mismatch between:
 * - Frontend: Matrix Olm library (Curve25519, Double Ratchet, specific message formats)
 * - Backend: Signal Protocol implementation (X25519, HKDF, different message structures)
 *
 * Key Responsibilities:
 * 1. Convert between Olm and Signal key formats
 * 2. Adapt message structures for compatibility
 * 3. Handle protocol flow differences
 * 4. Maintain security properties across both implementations
 */

import { signalProtocol, type KeyPair, type SignedPreKey, type PreKey, type EncryptedMessage } from './signal';

// Backend API types (Signal Protocol format)
interface BackendKeyPair {
  privateKey: string; // Base64 encoded 32-byte X25519 private key
  publicKey: string;  // Base64 encoded 32-byte X25519 public key
}

interface BackendSignedPreKey {
  keyId: number;
  publicKey: string;  // Base64 encoded 32-byte X25519 public key
  privateKey: string; // Base64 encoded 32-byte X25519 private key
  signature: string;  // Base64 encoded signature
  timestamp: number;
}

interface BackendPreKey {
  keyId: number;
  publicKey: string;  // Base64 encoded 32-byte X25519 public key
  privateKey: string; // Base64 encoded 32-byte X25519 private key
}

interface BackendEncryptedMessage {
  ciphertext: string; // Base64 encoded ciphertext
  messageType: 'prekey' | 'whisper';
}

interface BackendPreKeyBundle {
  registrationId: number;
  deviceId: number;
  identityKey: string;      // Base64 encoded 32-byte X25519 public key
  signedPreKey: string;     // Base64 encoded 32-byte X25519 public key
  signedPreKeyId: number;
  signedPreKeySignature: string; // Base64 encoded signature
  preKeyId?: number;
  preKey?: string;          // Base64 encoded 32-byte X25519 public key
}

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
 * ProtocolAdapter class - Bridges Olm and Signal Protocol implementations
 */
export class ProtocolAdapter {
  private olmProtocol: typeof signalProtocol;

  constructor() {
    this.olmProtocol = signalProtocol;
  }

  /**
   * Convert Olm KeyPair to Backend KeyPair format
   */
  convertKeyPairToBackend(olmKeyPair: KeyPair): BackendKeyPair {
    return {
      privateKey: uint8ArrayToBase64(olmKeyPair.privateKey),
      publicKey: uint8ArrayToBase64(olmKeyPair.publicKey),
    };
  }

  /**
   * Convert Backend KeyPair to Olm KeyPair format
   */
  convertKeyPairFromBackend(backendKeyPair: BackendKeyPair): KeyPair {
    return {
      privateKey: base64ToUint8Array(backendKeyPair.privateKey),
      publicKey: base64ToUint8Array(backendKeyPair.publicKey),
    };
  }

  /**
   * Convert Olm SignedPreKey to Backend SignedPreKey format
   */
  convertSignedPreKeyToBackend(olmSignedPreKey: SignedPreKey): BackendSignedPreKey {
    return {
      keyId: olmSignedPreKey.keyId,
      publicKey: uint8ArrayToBase64(olmSignedPreKey.publicKey),
      privateKey: uint8ArrayToBase64(olmSignedPreKey.privateKey),
      signature: uint8ArrayToBase64(olmSignedPreKey.signature),
      timestamp: olmSignedPreKey.timestamp,
    };
  }

  /**
   * Convert Backend SignedPreKey to Olm SignedPreKey format
   */
  convertSignedPreKeyFromBackend(backendSignedPreKey: BackendSignedPreKey): SignedPreKey {
    return {
      keyId: backendSignedPreKey.keyId,
      publicKey: base64ToUint8Array(backendSignedPreKey.publicKey),
      privateKey: base64ToUint8Array(backendSignedPreKey.privateKey),
      signature: base64ToUint8Array(backendSignedPreKey.signature),
      timestamp: backendSignedPreKey.timestamp,
    };
  }

  /**
   * Convert Olm PreKey to Backend PreKey format
   */
  convertPreKeyToBackend(olmPreKey: PreKey): BackendPreKey {
    return {
      keyId: olmPreKey.keyId,
      publicKey: uint8ArrayToBase64(olmPreKey.publicKey),
      privateKey: uint8ArrayToBase64(olmPreKey.privateKey),
    };
  }

  /**
   * Convert Backend PreKey to Olm PreKey format
   */
  convertPreKeyFromBackend(backendPreKey: BackendPreKey): PreKey {
    return {
      keyId: backendPreKey.keyId,
      publicKey: base64ToUint8Array(backendPreKey.publicKey),
      privateKey: base64ToUint8Array(backendPreKey.privateKey),
    };
  }

  /**
   * Convert Olm EncryptedMessage to Backend EncryptedMessage format
   */
  convertEncryptedMessageToBackend(olmMessage: EncryptedMessage): BackendEncryptedMessage {
    return {
      ciphertext: uint8ArrayToBase64(olmMessage.ciphertext),
      messageType: olmMessage.messageType,
    };
  }

  /**
   * Convert Backend EncryptedMessage to Olm EncryptedMessage format
   */
  convertEncryptedMessageFromBackend(backendMessage: BackendEncryptedMessage): EncryptedMessage {
    return {
      ciphertext: base64ToUint8Array(backendMessage.ciphertext),
      messageType: backendMessage.messageType,
    };
  }

  /**
   * Convert Backend PreKeyBundle to Olm-compatible format
   */
  convertPreKeyBundleFromBackend(backendBundle: BackendPreKeyBundle): {
    registrationId: number;
    identityKey: Uint8Array;
    signedPreKeyId: number;
    signedPreKey: Uint8Array;
    signedPreKeySignature: Uint8Array;
    preKeyId?: number;
    preKey?: Uint8Array;
  } {
    const result: any = {
      registrationId: backendBundle.registrationId,
      identityKey: base64ToUint8Array(backendBundle.identityKey),
      signedPreKeyId: backendBundle.signedPreKeyId,
      signedPreKey: base64ToUint8Array(backendBundle.signedPreKey),
      signedPreKeySignature: base64ToUint8Array(backendBundle.signedPreKeySignature),
    };

    if (backendBundle.preKeyId !== undefined) {
      result.preKeyId = backendBundle.preKeyId;
    }
    if (backendBundle.preKey !== undefined) {
      result.preKey = base64ToUint8Array(backendBundle.preKey);
    }

    return result;
  }

  /**
   * Adapt Olm pre-key bundle to Backend format
   */
  async adaptPreKeyBundleToBackend(): Promise<BackendPreKeyBundle> {
    // Get Olm account identity keys
    const identityKey = await this.olmProtocol.getIdentityPublicKey();
    if (!identityKey) {
      throw new Error('Identity key not available');
    }

    // Get Olm account signed pre-key (using the first one-time key as signed pre-key)
    const oneTimeKeys = await this.olmProtocol.getOneTimeKeys();
    const signedPreKeyEntries = Object.entries(oneTimeKeys.curve25519);
    if (signedPreKeyEntries.length === 0) {
      throw new Error('No signed pre-keys available');
    }

    const [signedPreKeyIdStr, signedPreKeyBase64] = signedPreKeyEntries[0];
    const signedPreKeyId = parseInt(signedPreKeyIdStr);

    // Generate a signature for the signed pre-key (simplified for compatibility)
    // In Olm, we sign with the account's Ed25519 key
    const signature = await this.olmProtocol.sign(signedPreKeyBase64);

    // Get registration and device IDs
    const registrationId = await this.olmProtocol.getRegistrationId();
    const deviceId = await this.olmProtocol.getDeviceId();

    return {
      registrationId,
      deviceId,
      identityKey: uint8ArrayToBase64(identityKey),
      signedPreKey: signedPreKeyBase64,
      signedPreKeyId,
      signedPreKeySignature: signature,
      // Note: Olm doesn't have separate one-time pre-keys in the same way as Signal
      // The one-time keys are handled differently in Olm
    };
  }

  /**
   * Create session with backend-compatible bundle
   */
  async createSessionWithBackendBundle(
    recipientId: string,
    deviceId: number,
    backendBundle: BackendPreKeyBundle
  ): Promise<void> {
    const olmBundle = this.convertPreKeyBundleFromBackend(backendBundle);
    return this.olmProtocol.createSession(recipientId, deviceId, olmBundle);
  }

  /**
   * Encrypt message for backend
   */
  async encryptMessageForBackend(
    recipientId: string,
    deviceId: number,
    plaintext: string
  ): Promise<BackendEncryptedMessage> {
    const olmMessage = await this.olmProtocol.encryptMessage(recipientId, deviceId, plaintext);
    return this.convertEncryptedMessageToBackend(olmMessage);
  }

  /**
   * Decrypt message from backend
   */
  async decryptMessageFromBackend(
    senderId: string,
    deviceId: number,
    backendMessage: BackendEncryptedMessage
  ): Promise<string> {
    const olmMessage = this.convertEncryptedMessageFromBackend(backendMessage);
    return this.olmProtocol.decryptMessage(senderId, deviceId, olmMessage.ciphertext, olmMessage.messageType);
  }

  /**
   * Get pre-key bundle in backend format
   */
  async getPreKeyBundleForBackend(): Promise<BackendPreKeyBundle> {
    return this.adaptPreKeyBundleToBackend();
  }

  /**
   * Generate identity key pair in backend format
   */
  async generateIdentityKeyPairForBackend(): Promise<BackendKeyPair> {
    const olmKeyPair = await this.olmProtocol.generateIdentityKeyPair();
    return this.convertKeyPairToBackend(olmKeyPair);
  }

  /**
   * Generate signed pre-key in backend format
   */
  async generateSignedPreKeyForBackend(keyId: number): Promise<BackendSignedPreKey> {
    const olmSignedPreKey = await this.olmProtocol.generateSignedPreKey(keyId);
    return this.convertSignedPreKeyToBackend(olmSignedPreKey);
  }

  /**
   * Generate pre-keys in backend format
   */
  async generatePreKeysForBackend(startId: number, count: number): Promise<BackendPreKey[]> {
    const olmPreKeys = await this.olmProtocol.generatePreKeys(startId, count);
    return olmPreKeys.map(preKey => this.convertPreKeyToBackend(preKey));
  }

  /**
   * Check if session exists
   */
  async hasSession(recipientId: string, deviceId: number): Promise<boolean> {
    return this.olmProtocol.hasSession(recipientId, deviceId);
  }

  /**
   * Delete session
   */
  async deleteSession(recipientId: string, deviceId: number): Promise<void> {
    return this.olmProtocol.deleteSession(recipientId, deviceId);
  }

  /**
   * Set up encryption with PIN
   */
  async setupEncryption(pin: string): Promise<void> {
    return this.olmProtocol.setupEncryption(pin);
  }

  /**
   * Unlock with PIN
   */
  async unlockWithPin(pin: string): Promise<void> {
    return this.olmProtocol.unlockWithPin(pin);
  }

  /**
   * Check if encryption is enabled
   */
  isEncryptionEnabled(): boolean {
    return this.olmProtocol.isEncryptionEnabled();
  }

  /**
   * Check if keys are unlocked
   */
  isUnlocked(): boolean {
    return this.olmProtocol.isUnlocked();
  }

  /**
   * Clear all stored data
   */
  async clear(): Promise<void> {
    return this.olmProtocol.clear();
  }
}

// Export singleton instance
export const protocolAdapter = new ProtocolAdapter();