/**
 * Signal Protocol Implementation using Matrix Olm Library
 *
 * This implementation uses @matrix-org/olm, a WebAssembly-based library that provides:
 * - Olm: 1:1 encrypted sessions using Double Ratchet algorithm (same as Signal)
 * - Megolm: Efficient group encryption for multi-device scenarios
 *
 * Why @matrix-org/olm?
 * ====================
 * 1. BROWSER COMPATIBLE: WASM-based, works in all modern browsers (Chrome, Firefox, Safari, Edge)
 * 2. BATTLE-TESTED: Powers Element, Beeper, and other major Matrix-based messengers
 * 3. DOUBLE RATCHET: Implements the same cryptographic protocol as Signal
 * 4. WELL AUDITED: NCC Group security audit conducted
 * 5. ACTIVELY MAINTAINED: By the Matrix.org Foundation
 *
 * Security Properties:
 * - Curve25519 for key agreement (more secure than P-256, same as Signal)
 * - Ed25519 for signatures
 * - Double Ratchet providing forward secrecy and post-compromise security
 * - AES-256-CBC + HMAC-SHA256 for symmetric encryption
 * - Key backup protection with PIN-derived master key
 *
 * @see https://gitlab.matrix.org/matrix-org/olm
 * @see https://matrix.org/docs/spec/olm/
 */

import Olm from '@matrix-org/olm';
import { get, set, del } from 'idb-keyval';




// Storage keys for IndexedDB
const OLM_ACCOUNT = 'olm:account';
const OLM_SESSIONS_PREFIX = 'olm:session:';
const REGISTRATION_ID = 'olm:registration_id';
const DEVICE_ID = 'olm:device_id';
const MASTER_KEY_SALT = 'olm:master_key_salt';
const ENCRYPTION_ENABLED = 'olm:encryption_enabled';
const ONE_TIME_KEYS_COUNTER = 'olm:one_time_keys_counter';

// Re-export types for backward compatibility
export interface KeyPair {
  publicKey: Uint8Array;
  privateKey: Uint8Array;
}

export interface SignedPreKey {
  keyId: number;
  publicKey: Uint8Array;
  privateKey: Uint8Array;
  signature: Uint8Array;
  timestamp: number;
}

export interface PreKey {
  keyId: number;
  publicKey: Uint8Array;
  privateKey: Uint8Array;
}

export interface EncryptedMessage {
  ciphertext: Uint8Array;
  messageType: 'prekey' | 'whisper';
}

// Olm-specific types
interface OlmIdentityKeys {
  curve25519: string;
  ed25519: string;
}

interface OlmOneTimeKeys {
  curve25519: Record<string, string>;
}

interface StoredSession {
  pickle: string;
  lastUsed: number;
}

// Utility functions
function base64ToUint8Array(base64: string): Uint8Array {
  // Handle unpadded base64 by adding padding if needed
  const padded = base64.replace(/-/g, '+').replace(/_/g, '/');
  const paddedWithEquals = padded + '='.repeat((4 - padded.length % 4) % 4);
  const binary = atob(paddedWithEquals);
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

// Olm requires UNPADDED base64 - strip trailing '=' characters
function uint8ArrayToUnpaddedBase64(bytes: Uint8Array): string {
  return uint8ArrayToBase64(bytes).replace(/=+$/, '');
}

function stringToUint8Array(str: string): Uint8Array {
  return new TextEncoder().encode(str);
}

function uint8ArrayToString(bytes: Uint8Array): string {
  return new TextDecoder().decode(bytes);
}

/**
 * PBKDF2 Security Constants
 * OWASP recommends minimum 600,000 iterations for PBKDF2-HMAC-SHA256
 * @see https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
 */
const PBKDF2_ITERATIONS = 600000;
const MIN_PBKDF2_ITERATIONS = 600000; // OWASP minimum recommendation

/**
 * Derive a master encryption key from a PIN using PBKDF2
 *
 * Security Notes:
 * - Uses PBKDF2-HMAC-SHA256 with 600,000 iterations (OWASP minimum recommendation)
 * - Derives 256-bit keys for AES-256-GCM encryption
 * - Includes performance monitoring for security auditing
 * - Validates iteration count meets minimum security requirements
 *
 * @param pin - User PIN/password for key derivation
 * @param salt - Random salt for PBKDF2 (should be 16+ bytes)
 * @param iterations - Optional iteration count (must be >= MIN_PBKDF2_ITERATIONS)
 * @returns Promise<Uint8Array> - 256-bit master encryption key
 * @throws Will throw if iterations are below minimum security requirements
 */
async function deriveMasterKey(
  pin: string,
  salt: Uint8Array,
  iterations: number = PBKDF2_ITERATIONS
): Promise<Uint8Array> {
  // Security validation: Ensure iterations meet OWASP minimum requirements
  if (iterations < MIN_PBKDF2_ITERATIONS) {
    throw new Error(
      `PBKDF2 iterations too low: ${iterations}. Minimum required: ${MIN_PBKDF2_ITERATIONS}`
    );
  }

  const encoder = new TextEncoder();
  const pinData = encoder.encode(pin);

  // Performance monitoring for security auditing
  console.time('PBKDF2 key derivation');

  const keyMaterial = await crypto.subtle.importKey('raw', pinData, 'PBKDF2', false, [
    'deriveBits',
  ]);

  const derivedBits = await crypto.subtle.deriveBits(
    {
      name: 'PBKDF2',
      salt: salt.buffer as ArrayBuffer,
      iterations: iterations,
      hash: 'SHA-256',
    },
    keyMaterial,
    256
  );

  // End performance monitoring
  console.timeEnd('PBKDF2 key derivation');

  return new Uint8Array(derivedBits);
}

/**
 * Encrypt data with AES-256-GCM using the master key
 */
async function encryptWithMasterKey(
  masterKey: Uint8Array,
  data: string
): Promise<{ ciphertext: string; iv: string }> {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    masterKey.buffer as ArrayBuffer,
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
  const cryptoKey = await crypto.subtle.importKey(
    'raw',
    masterKey.buffer as ArrayBuffer,
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
 * SignalProtocol class using Matrix Olm library
 *
 * Provides end-to-end encryption with:
 * - X3DH-equivalent key exchange using Curve25519
 * - Double Ratchet algorithm for forward secrecy
 * - PIN-based encryption for key storage
 */
export class SignalProtocol {
  private initialized = false;
  private olmInitialized = false;
  private account: Olm.Account | null = null;
  private registrationId = 0;
  private deviceId = 1;
  private masterKey: Uint8Array | null = null;
  private encryptionEnabled = false;
  private oneTimeKeysCounter = 0;

  // Session cache with LRU eviction to prevent memory exhaustion
  // Maximum 100 sessions in memory - oldest will be evicted when exceeded
  private static readonly MAX_SESSION_CACHE_SIZE = 100;
  private sessionCache: Map<string, Olm.Session> = new Map();
  private sessionAccessOrder: string[] = []; // Track access order for LRU

  /**
   * Initialize the Olm library and load stored account
   */
  async initialize(): Promise<void> {
    if (this.initialized) return;

    // Initialize Olm WASM module
    if (!this.olmInitialized) {
      await Olm.init();
      this.olmInitialized = true;
    }

    // Check if encryption is enabled
    const encryptionEnabled = await get<boolean>(ENCRYPTION_ENABLED);
    this.encryptionEnabled = encryptionEnabled || false;

    // Restore master key from sessionStorage if available (persists across page refresh)
    const storedMasterKey = sessionStorage.getItem('signal_master_key');
    if (storedMasterKey && this.encryptionEnabled) {
      try {
        this.masterKey = base64ToUint8Array(storedMasterKey);
      } catch {
        console.warn('Failed to restore master key from session storage');
        sessionStorage.removeItem('signal_master_key');
      }
    }

    // Load stored registration and device IDs
    const storedRegId = await get<number>(REGISTRATION_ID);
    if (storedRegId) {
      this.registrationId = storedRegId;
    }

    const storedDeviceId = await get<number>(DEVICE_ID);
    if (storedDeviceId) {
      this.deviceId = storedDeviceId;
    }

    const storedCounter = await get<number>(ONE_TIME_KEYS_COUNTER);
    if (storedCounter) {
      this.oneTimeKeysCounter = storedCounter;
    }

    // Load existing account if available
    const storedAccount = await get<{
      pickle?: string;
      encryptedPickle?: string;
      iv?: string;
    }>(OLM_ACCOUNT);

    if (storedAccount) {
      if (storedAccount.encryptedPickle && storedAccount.iv) {
        if (!this.masterKey) {
          // Account is encrypted but no master key - user needs to unlock first
          console.warn('Encrypted account found but no master key - unlock with PIN required');
          this.initialized = true;
          return;
        }
        // Decrypt the account pickle
        try {
          const pickle = await decryptWithMasterKey(
            this.masterKey,
            storedAccount.encryptedPickle,
            storedAccount.iv
          );
          this.account = new Olm.Account();
          this.account.unpickle('olm_pickle_key', pickle);
        } catch {
          throw new Error('Failed to decrypt account - invalid PIN or corrupted data');
        }
      } else if (storedAccount.pickle) {
        // Plaintext pickle (legacy or non-encrypted)
        this.account = new Olm.Account();
        this.account.unpickle('olm_pickle_key', storedAccount.pickle);
      }
    }

    this.initialized = true;
  }

  /**
   * Set up encryption with a user PIN/password
   */
  async setupEncryption(pin: string): Promise<void> {
    try {
      await this.ensureInitialized();

      if (pin.length < 6) {
        throw new Error('PIN must be at least 6 characters long');
      }

      // Generate a random salt for PBKDF2
      const salt = crypto.getRandomValues(new Uint8Array(16));

      // Derive master key from PIN
      this.masterKey = await deriveMasterKey(pin, salt);

      // Store the salt
      await set(MASTER_KEY_SALT, uint8ArrayToBase64(salt));

      // Store master key in sessionStorage for page refresh persistence
      sessionStorage.setItem('signal_master_key', uint8ArrayToBase64(this.masterKey));

      // Mark encryption as enabled
      this.encryptionEnabled = true;
      await set(ENCRYPTION_ENABLED, true);

      // If no account exists (new device), create one
      if (!this.account) {
        console.log('No Olm account found, creating new identity for this device');
        this.account = new Olm.Account();
        this.account.create();

        // Generate registration and device IDs
        this.registrationId = Math.floor(Math.random() * 16380) + 1;
        this.deviceId = 1;
        this.oneTimeKeysCounter = 0;

        await set(REGISTRATION_ID, this.registrationId);
        await set(DEVICE_ID, this.deviceId);
        await set(ONE_TIME_KEYS_COUNTER, this.oneTimeKeysCounter);
      }

      // Save the account (encrypted with master key)
      await this.saveAccount();
    } catch (error) {
      // Reset state on failure
      this.masterKey = null;
      sessionStorage.removeItem('signal_master_key');
      this.encryptionEnabled = false;
      throw new Error(
        `Failed to set up encryption: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
    }
  }

  /**
   * Unlock encrypted keys with PIN
   */
  async unlockWithPin(pin: string): Promise<void> {
    try {
      await this.ensureInitialized();

      if (!this.encryptionEnabled) {
        throw new Error('Encryption is not enabled');
      }

      if (!pin || pin.length < 6) {
        throw new Error('PIN must be at least 6 characters long');
      }

      const saltData = await get<string>(MASTER_KEY_SALT);
      if (!saltData) {
        throw new Error('Master key salt not found - encryption may not be properly set up');
      }

      const salt = base64ToUint8Array(saltData);
      this.masterKey = await deriveMasterKey(pin, salt);

      // Verify the key works by trying to decrypt the account
      const storedAccount = await get<{
        encryptedPickle?: string;
        iv?: string;
      }>(OLM_ACCOUNT);

      if (storedAccount?.encryptedPickle && storedAccount?.iv) {
        try {
          const pickle = await decryptWithMasterKey(
            this.masterKey,
            storedAccount.encryptedPickle,
            storedAccount.iv
          );
          this.account = new Olm.Account();
          this.account.unpickle('olm_pickle_key', pickle);
        } catch {
          this.masterKey = null;
          sessionStorage.removeItem('signal_master_key');
          throw new Error('Invalid PIN - unable to decrypt keys');
        }
      }

      // Store master key in sessionStorage for page refresh persistence
      sessionStorage.setItem('signal_master_key', uint8ArrayToBase64(this.masterKey));
    } catch (error) {
      this.masterKey = null;
      sessionStorage.removeItem('signal_master_key');
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Failed to unlock keys');
    }
  }

  /**
   * Check if encryption is set up
   */
  isEncryptionEnabled(): boolean {
    return this.encryptionEnabled;
  }

  /**
   * Check if keys are unlocked (master key available)
   */
  isUnlocked(): boolean {
    return this.masterKey !== null;
  }

  /**
   * Generate a new identity key pair (Olm Account)
   */
  async generateIdentityKeyPair(): Promise<KeyPair> {
    try {
      await this.ensureInitialized();

      // Create new Olm Account
      this.account = new Olm.Account();
      this.account.create();

      // Generate registration and device IDs
      this.registrationId = Math.floor(Math.random() * 16380) + 1;
      this.deviceId = 1;
      this.oneTimeKeysCounter = 0;

      // Get identity keys
      const identityKeys: OlmIdentityKeys = JSON.parse(this.account.identity_keys());

      // Save the account
      await this.saveAccount();
      await set(REGISTRATION_ID, this.registrationId);
      await set(DEVICE_ID, this.deviceId);
      await set(ONE_TIME_KEYS_COUNTER, this.oneTimeKeysCounter);

      // Return in the expected format
      // Note: Olm uses Curve25519 not P-256, but we maintain the same interface
      return {
        publicKey: base64ToUint8Array(identityKeys.curve25519),
        privateKey: stringToUint8Array(this.account.pickle('export_key')),
      };
    } catch (error) {
      this.account = null;
      throw new Error(
        `Failed to generate identity key pair: ${error instanceof Error ? error.message : 'Unknown error'}`
      );
    }
  }

  /**
   * Get one-time keys for pre-key bundle
   * In Olm, these are equivalent to Signal's one-time pre-keys
   */
  async getOneTimeKeys(): Promise<OlmOneTimeKeys> {
    if (!this.account) {
      throw new Error('Account not initialized');
    }
    return JSON.parse(this.account.one_time_keys());
  }

  /**
   * Generate a signed pre-key
   * Olm handles signed keys differently, but we maintain API compatibility
   */
  async generateSignedPreKey(keyId: number): Promise<SignedPreKey> {
    await this.ensureInitialized();

    if (!this.account) {
      throw new Error('Account not initialized');
    }

    // Generate a one-time key and mark it as the signed pre-key
    this.account.generate_one_time_keys(1);
    const oneTimeKeys: OlmOneTimeKeys = JSON.parse(this.account.one_time_keys());
    const keyEntries = Object.entries(oneTimeKeys.curve25519);

    if (keyEntries.length === 0) {
      throw new Error('Failed to generate signed pre-key');
    }

    const [, keyValue] = keyEntries[0];
    const keyData = base64ToUint8Array(keyValue);

    // Sign the key with the account's Ed25519 key
    const signature = this.account.sign(keyValue);

    // Mark the key as published
    this.account.mark_keys_as_published();

    // Save account state
    await this.saveAccount();

    const signedPreKey: SignedPreKey = {
      keyId,
      publicKey: keyData,
      privateKey: stringToUint8Array(this.account.pickle('export_key')),
      signature: base64ToUint8Array(signature),
      timestamp: Date.now(),
    };

    return signedPreKey;
  }

  /**
   * Generate one-time pre-keys
   */
  async generatePreKeys(_startId: number, count: number): Promise<PreKey[]> {
    await this.ensureInitialized();

    if (!this.account) {
      throw new Error('Account not initialized');
    }

    // Generate one-time keys
    this.account.generate_one_time_keys(count);
    const oneTimeKeys: OlmOneTimeKeys = JSON.parse(this.account.one_time_keys());

    // Mark keys as published
    this.account.mark_keys_as_published();

    // Update counter
    this.oneTimeKeysCounter += count;
    await set(ONE_TIME_KEYS_COUNTER, this.oneTimeKeysCounter);

    // Save account state
    await this.saveAccount();

    // Convert to PreKey format - use Olm's key IDs, not sequential IDs!
    // Olm key IDs are base64-encoded integers like "AAAAAA", "AAAAAQ", etc.
    const preKeys: PreKey[] = [];
    for (const [keyId, keyValue] of Object.entries(oneTimeKeys.curve25519)) {
      // Decode Olm's base64 key ID to get the numeric ID
      const keyIdBytes = base64ToUint8Array(keyId);
      // Olm uses big-endian 32-bit integers for key IDs
      let numericKeyId = 0;
      for (let i = 0; i < keyIdBytes.length; i++) {
        numericKeyId = (numericKeyId << 8) | keyIdBytes[i];
      }

      preKeys.push({
        keyId: numericKeyId,
        publicKey: base64ToUint8Array(keyValue),
        privateKey: stringToUint8Array(this.account.pickle('export_key')),
      });
    }

    return preKeys;
  }

  /**
   * Create a session with a recipient using their pre-key bundle
   * This is equivalent to Signal's X3DH key agreement
   */
  async createSession(
    recipientId: string,
    deviceId: number,
    bundle: {
      registrationId: number;
      identityKey: Uint8Array;
      signedPreKeyId: number;
      signedPreKey: Uint8Array;
      signedPreKeySignature: Uint8Array;
      preKeyId?: number;
      preKey?: Uint8Array;
    }
  ): Promise<void> {
    await this.ensureInitialized();

    if (!this.account) {
      throw new Error('Account not initialized');
    }

    // Create outbound session using the recipient's one-time key or signed pre-key
    const session = new Olm.Session();

    // IMPORTANT: Olm requires UNPADDED base64 (no trailing '=' characters)
    const theirIdentityKey = uint8ArrayToUnpaddedBase64(bundle.identityKey);
    const theirOneTimeKey = bundle.preKey
      ? uint8ArrayToUnpaddedBase64(bundle.preKey)
      : uint8ArrayToUnpaddedBase64(bundle.signedPreKey);

    // Create outbound session
    session.create_outbound(this.account, theirIdentityKey, theirOneTimeKey);

    // Store session
    await this.storeSession(recipientId, deviceId, session);
  }

  /**
   * Encrypt a message for a recipient
   */
  async encryptMessage(
    recipientId: string,
    deviceId: number,
    plaintext: string
  ): Promise<EncryptedMessage> {
    await this.ensureInitialized();

    const session = await this.loadSession(recipientId, deviceId);
    if (!session) {
      throw new Error('No session exists for this recipient');
    }

    // Encrypt the message
    const encrypted = session.encrypt(plaintext);

    // Save session state after encryption (ratchet has advanced)
    await this.storeSession(recipientId, deviceId, session);

    // Olm message types: 0 = prekey message, 1 = normal message
    return {
      ciphertext: stringToUint8Array(encrypted.body),
      messageType: encrypted.type === 0 ? 'prekey' : 'whisper',
    };
  }

  /**
   * Decrypt a received message
   */
  async decryptMessage(
    senderId: string,
    deviceId: number,
    ciphertext: Uint8Array,
    messageType: 'prekey' | 'whisper'
  ): Promise<string> {
    await this.ensureInitialized();

    if (!this.account) {
      throw new Error('Account not initialized');
    }

    const ciphertextStr = uint8ArrayToString(ciphertext);
    let session = await this.loadSession(senderId, deviceId);

    // If we have an existing session but message is prekey, the sender may have reset their keys
    // Try creating a new session from the prekey message
    if (session && messageType === 'prekey') {
      console.log('[Signal] Received prekey message from existing session - contact may have reset keys');
      // Try with new session first
      try {
        const newSession = new Olm.Session();
        newSession.create_inbound(this.account, ciphertextStr);

        // New session worked, use it
        this.account.remove_one_time_keys(newSession);
        await this.saveAccount();

        const plaintext = newSession.decrypt(0, ciphertextStr);

        // Replace old session with new one
        session.free();
        await this.storeSession(senderId, deviceId, newSession);

        console.log('[Signal] Successfully created new session from prekey message');
        return plaintext;
      } catch (prekeyError) {
        // New session failed, try with existing session
        console.log('[Signal] New session failed, trying existing session');
      }
    }

    if (!session && messageType === 'prekey') {
      // Create inbound session from pre-key message
      session = new Olm.Session();
      session.create_inbound(this.account, ciphertextStr);

      // Remove one-time key used for this session
      this.account.remove_one_time_keys(session);
      await this.saveAccount();
    }

    if (!session) {
      throw new Error('No session exists for this sender');
    }

    // Decrypt the message
    // Olm message types: 0 = prekey message, 1 = normal message
    const olmMsgType = messageType === 'prekey' ? 0 : 1;

    try {
      const plaintext = session.decrypt(olmMsgType, ciphertextStr);

      // Save session state after decryption (ratchet has advanced)
      await this.storeSession(senderId, deviceId, session);

      return plaintext;
    } catch (decryptError) {
      const errorMessage = decryptError instanceof Error ? decryptError.message : String(decryptError);

      // BAD_MESSAGE_MAC means session is corrupted or out of sync
      if (errorMessage.includes('BAD_MESSAGE_MAC') || errorMessage.includes('BAD_MESSAGE_KEY_ID')) {
        console.warn('[Signal] Session corrupted, deleting and retrying...');

        // Delete the corrupted session
        session.free();
        await this.deleteSession(senderId, deviceId);

        // If this is a prekey message, try to create new session
        if (messageType === 'prekey') {
          console.log('[Signal] Retrying with new session from prekey message');
          const newSession = new Olm.Session();
          newSession.create_inbound(this.account, ciphertextStr);

          this.account.remove_one_time_keys(newSession);
          await this.saveAccount();

          const plaintext = newSession.decrypt(0, ciphertextStr);
          await this.storeSession(senderId, deviceId, newSession);

          console.log('[Signal] Successfully recovered with new session');
          return plaintext;
        }

        // For whisper messages, we can't recover - need sender to resend as prekey
        throw new Error('Session corrupted and cannot be recovered. Contact needs to re-establish session.');
      }

      throw decryptError;
    }
  }

  /**
   * Get identity public key (Curve25519)
   */
  async getIdentityPublicKey(): Promise<Uint8Array | null> {
    await this.ensureInitialized();
    if (!this.account) return null;

    const identityKeys: OlmIdentityKeys = JSON.parse(this.account.identity_keys());
    return base64ToUint8Array(identityKeys.curve25519);
  }

  /**
   * Get Ed25519 signing public key
   */
  async getSigningPublicKey(): Promise<Uint8Array | null> {
    await this.ensureInitialized();
    if (!this.account) return null;

    const identityKeys: OlmIdentityKeys = JSON.parse(this.account.identity_keys());
    return base64ToUint8Array(identityKeys.ed25519);
  }

  /**
   * Get public keys in the format needed for server upload
   * Used when setting up encryption on a new device to notify contacts
   */
  async getPublicKeysForServer(): Promise<{
    publicIdentityKey: string;
    publicSignedPrekey: string;
    signedPrekeySignature: string;
  } | null> {
    await this.ensureInitialized();
    if (!this.account) return null;

    const identityKeys: OlmIdentityKeys = JSON.parse(this.account.identity_keys());

    // In Olm, we use the curve25519 key as both identity and signed prekey
    // The ed25519 key is used for signing
    // Create a signature of the curve25519 key using ed25519
    const curve25519Key = identityKeys.curve25519;
    const signature = this.account.sign(curve25519Key);

    return {
      publicIdentityKey: identityKeys.curve25519,
      publicSignedPrekey: identityKeys.curve25519, // Same key in Olm
      signedPrekeySignature: signature,
    };
  }

  /**
   * Get registration ID
   */
  async getRegistrationId(): Promise<number> {
    await this.ensureInitialized();
    return this.registrationId;
  }

  /**
   * Get device ID
   */
  async getDeviceId(): Promise<number> {
    await this.ensureInitialized();
    return this.deviceId;
  }

  /**
   * Check if a session exists
   */
  async hasSession(recipientId: string, deviceId: number): Promise<boolean> {
    const session = await this.loadSession(recipientId, deviceId);
    return session !== null;
  }

  /**
   * Delete a session
   */
  async deleteSession(recipientId: string, deviceId: number): Promise<void> {
    const sessionKey = `${OLM_SESSIONS_PREFIX}${recipientId}:${deviceId}`;

    // Remove from cache
    this.sessionCache.delete(`${recipientId}:${deviceId}`);

    // Remove from storage
    await del(sessionKey);
  }

  /**
   * Get public pre-key bundle for sharing with other users
   */
  async getPreKeyBundle(): Promise<{
    registrationId: number;
    deviceId: number;
    identityKey: string;
    signedPreKey: string;
    oneTimeKeys: string[];
  }> {
    await this.ensureInitialized();

    if (!this.account) {
      throw new Error('Account not initialized');
    }

    const identityKeys: OlmIdentityKeys = JSON.parse(this.account.identity_keys());
    const oneTimeKeys: OlmOneTimeKeys = JSON.parse(this.account.one_time_keys());

    return {
      registrationId: this.registrationId,
      deviceId: this.deviceId,
      identityKey: identityKeys.curve25519,
      signedPreKey: identityKeys.ed25519,
      oneTimeKeys: Object.values(oneTimeKeys.curve25519),
    };
  }

  /**
   * Sign data with Ed25519 key
   */
  async sign(data: string): Promise<string> {
    await this.ensureInitialized();

    if (!this.account) {
      throw new Error('Account not initialized');
    }

    return this.account.sign(data);
  }

  /**
   * Clear all stored data (for logout/reset)
   */
  async clear(): Promise<void> {
    // Free Olm objects
    if (this.account) {
      this.account.free();
      this.account = null;
    }

    for (const session of this.sessionCache.values()) {
      session.free();
    }
    this.sessionCache.clear();
    this.sessionAccessOrder = []; // Clear LRU tracking

    // Clear IndexedDB
    await del(OLM_ACCOUNT);
    await del(REGISTRATION_ID);
    await del(DEVICE_ID);
    await del(MASTER_KEY_SALT);
    await del(ENCRYPTION_ENABLED);
    await del(ONE_TIME_KEYS_COUNTER);

    // Reset state
    this.initialized = false;
    this.registrationId = 0;
    this.deviceId = 1;
    this.masterKey = null;
    sessionStorage.removeItem('signal_master_key');
    this.encryptionEnabled = false;
    this.oneTimeKeysCounter = 0;
  }

  // Private methods

  private async ensureInitialized(): Promise<void> {
    if (!this.initialized) {
      await this.initialize();
    }
  }

  private async saveAccount(): Promise<void> {
    if (!this.account) return;

    const pickle = this.account.pickle('olm_pickle_key');

    if (this.encryptionEnabled && this.masterKey) {
      // Encrypt the pickle
      const { ciphertext, iv } = await encryptWithMasterKey(this.masterKey, pickle);
      await set(OLM_ACCOUNT, {
        encryptedPickle: ciphertext,
        iv,
      });
    } else {
      // Store plaintext
      await set(OLM_ACCOUNT, { pickle });
    }
  }

  private async storeSession(
    recipientId: string,
    deviceId: number,
    session: Olm.Session
  ): Promise<void> {
    const sessionKey = `${OLM_SESSIONS_PREFIX}${recipientId}:${deviceId}`;
    const cacheKey = `${recipientId}:${deviceId}`;
    const pickle = session.pickle('olm_session_pickle');

    // LRU cache management: evict oldest sessions if cache is full
    if (this.sessionCache.size >= SignalProtocol.MAX_SESSION_CACHE_SIZE && !this.sessionCache.has(cacheKey)) {
      // Remove oldest entry (first in access order)
      if (this.sessionAccessOrder.length > 0) {
        const oldestKey = this.sessionAccessOrder.shift()!;
        const oldSession = this.sessionCache.get(oldestKey);
        if (oldSession) {
          oldSession.free(); // Free Olm memory
        }
        this.sessionCache.delete(oldestKey);
      }
    }

    // Update access order (move to end as most recently used)
    const accessIndex = this.sessionAccessOrder.indexOf(cacheKey);
    if (accessIndex > -1) {
      this.sessionAccessOrder.splice(accessIndex, 1);
    }
    this.sessionAccessOrder.push(cacheKey);

    // Store in cache
    this.sessionCache.set(cacheKey, session);

    // Store in IndexedDB
    const stored: StoredSession = {
      pickle,
      lastUsed: Date.now(),
    };

    if (this.encryptionEnabled && this.masterKey) {
      const { ciphertext, iv } = await encryptWithMasterKey(this.masterKey, pickle);
      await set(sessionKey, {
        encryptedPickle: ciphertext,
        iv,
        lastUsed: stored.lastUsed,
      });
    } else {
      await set(sessionKey, stored);
    }
  }

  private async loadSession(
    recipientId: string,
    deviceId: number
  ): Promise<Olm.Session | null> {
    const cacheKey = `${recipientId}:${deviceId}`;

    // Check cache first
    if (this.sessionCache.has(cacheKey)) {
      // Update access order (move to end as most recently used)
      const accessIndex = this.sessionAccessOrder.indexOf(cacheKey);
      if (accessIndex > -1) {
        this.sessionAccessOrder.splice(accessIndex, 1);
      }
      this.sessionAccessOrder.push(cacheKey);

      return this.sessionCache.get(cacheKey)!;
    }

    // Load from IndexedDB
    const sessionKey = `${OLM_SESSIONS_PREFIX}${recipientId}:${deviceId}`;
    const stored = await get<{
      pickle?: string;
      encryptedPickle?: string;
      iv?: string;
      lastUsed: number;
    }>(sessionKey);

    if (!stored) return null;

    let pickle: string;

    if (stored.encryptedPickle && stored.iv) {
      if (!this.masterKey) {
        throw new Error('Session is encrypted but no master key available');
      }
      pickle = await decryptWithMasterKey(this.masterKey, stored.encryptedPickle, stored.iv);
    } else if (stored.pickle) {
      pickle = stored.pickle;
    } else {
      return null;
    }

    const session = new Olm.Session();
    session.unpickle('olm_session_pickle', pickle);

    // Add to cache with LRU management
    // Check if we need to evict before adding
    if (this.sessionCache.size >= SignalProtocol.MAX_SESSION_CACHE_SIZE) {
      if (this.sessionAccessOrder.length > 0) {
        const oldestKey = this.sessionAccessOrder.shift()!;
        const oldSession = this.sessionCache.get(oldestKey);
        if (oldSession) {
          oldSession.free();
        }
        this.sessionCache.delete(oldestKey);
      }
    }

    this.sessionCache.set(cacheKey, session);
    this.sessionAccessOrder.push(cacheKey);

    return session;
  }
}

// Export singleton instance
export const signalProtocol = new SignalProtocol();
