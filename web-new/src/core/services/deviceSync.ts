/**
 * Device Sync Service
 *
 * Enables Signal-style device sync where new devices can request
 * encrypted session data from the primary device.
 *
 * Security: Server is just a relay - sync data is encrypted device-to-device.
 */

import { get, keys, set } from 'idb-keyval';

// Storage key prefixes (must match signal.ts)
const OLM_ACCOUNT = 'olm:account';
const OLM_SESSIONS_PREFIX = 'olm:session:';
const REGISTRATION_ID = 'olm:registration_id';
const DEVICE_ID = 'olm:device_id';
const ONE_TIME_KEYS_COUNTER = 'olm:one_time_keys_counter';
const IDENTITY_KEY_FINGERPRINT = 'olm:identity_fingerprint';

// Sync status
export type SyncStatus =
    | 'idle'
    | 'checking'
    | 'requesting'
    | 'waiting'
    | 'receiving'
    | 'success'
    | 'timeout'
    | 'no_primary'
    | 'failed';

export interface SyncState {
    status: SyncStatus;
    message: string;
    progress?: number;
}

export interface SessionBundle {
    // Account data (encrypted Olm account pickle)
    account: {
        encryptedPickle?: string;
        iv?: string;
        pickle?: string;
    };
    // Session data for each contact
    sessions: Array<{
        recipientId: string;
        deviceId: number;
        pickle: string;
        lastUsed: number;
    }>;
    // Metadata
    registrationId: number;
    deviceId: number;
    oneTimeKeysCounter: number;
    identityFingerprint: string;
    exportedAt: number;
}

// Listeners for sync state changes
type SyncStateListener = (state: SyncState) => void;
const listeners: Set<SyncStateListener> = new Set();

let currentState: SyncState = { status: 'idle', message: '' };

/**
 * Subscribe to sync state changes
 */
export function onSyncStateChange(listener: SyncStateListener): () => void {
    listeners.add(listener);
    return () => listeners.delete(listener);
}

/**
 * Get current sync state
 */
export function getSyncState(): SyncState {
    return currentState;
}

/**
 * Update sync state and notify listeners
 */
export function setSyncState(state: SyncState): void {
    currentState = state;
    listeners.forEach(listener => listener(state));
}

/**
 * Export all sessions from IndexedDB as a bundle
 * This is called on the PRIMARY device when responding to a sync request
 */
export async function exportSessionBundle(): Promise<SessionBundle> {
    // Get account data
    const account = await get<{
        pickle?: string;
        encryptedPickle?: string;
        iv?: string;
    }>(OLM_ACCOUNT);

    if (!account) {
        throw new Error('No account found to export');
    }

    // Get all session keys
    const allKeys = await keys();
    const sessionKeys = allKeys.filter(
        (k) => typeof k === 'string' && k.startsWith(OLM_SESSIONS_PREFIX)
    ) as string[];

    // Get all sessions
    const sessions: SessionBundle['sessions'] = [];
    for (const key of sessionKeys) {
        const sessionData = await get<{ pickle: string; lastUsed: number }>(key);
        if (sessionData) {
            // Parse recipient and device from key: "olm:session:recipientId:deviceId"
            const parts = key.replace(OLM_SESSIONS_PREFIX, '').split(':');
            const recipientId = parts[0];
            const deviceId = parseInt(parts[1], 10) || 1;

            sessions.push({
                recipientId,
                deviceId,
                pickle: sessionData.pickle,
                lastUsed: sessionData.lastUsed,
            });
        }
    }

    // Get metadata
    const registrationId = await get<number>(REGISTRATION_ID) || 0;
    const deviceId = await get<number>(DEVICE_ID) || 1;
    const oneTimeKeysCounter = await get<number>(ONE_TIME_KEYS_COUNTER) || 0;
    const identityFingerprint = await get<string>(IDENTITY_KEY_FINGERPRINT) || '';

    return {
        account,
        sessions,
        registrationId,
        deviceId: deviceId + 1, // New device gets next device ID
        oneTimeKeysCounter,
        identityFingerprint,
        exportedAt: Date.now(),
    };
}

/**
 * Import a session bundle into IndexedDB
 * This is called on the NEW device after receiving sync data
 */
export async function importSessionBundle(bundle: SessionBundle): Promise<void> {
    setSyncState({ status: 'receiving', message: 'Importing sessions...' });

    try {
        // Import account
        await set(OLM_ACCOUNT, bundle.account);

        // Import sessions
        for (const session of bundle.sessions) {
            const key = `${OLM_SESSIONS_PREFIX}${session.recipientId}:${session.deviceId}`;
            await set(key, {
                pickle: session.pickle,
                lastUsed: session.lastUsed,
            });
        }

        // Import metadata
        await set(REGISTRATION_ID, bundle.registrationId);
        await set(DEVICE_ID, bundle.deviceId);
        await set(ONE_TIME_KEYS_COUNTER, bundle.oneTimeKeysCounter);
        await set(IDENTITY_KEY_FINGERPRINT, bundle.identityFingerprint);

        setSyncState({ status: 'success', message: 'Sync complete!' });
    } catch (error) {
        setSyncState({
            status: 'failed',
            message: `Import failed: ${error instanceof Error ? error.message : 'Unknown error'}`
        });
        throw error;
    }
}

/**
 * Encrypt a session bundle for device-to-device transfer
 * Uses the PIN-derived master key for encryption
 */
export async function encryptBundleForSync(
    bundle: SessionBundle,
    masterKey: Uint8Array
): Promise<{ encrypted: string; iv: string }> {
    const bundleJson = JSON.stringify(bundle);

    const iv = crypto.getRandomValues(new Uint8Array(12));
    const cryptoKey = await crypto.subtle.importKey(
        'raw',
        masterKey.buffer as ArrayBuffer,
        { name: 'AES-GCM' },
        false,
        ['encrypt']
    );

    const plaintext = new TextEncoder().encode(bundleJson);
    const ciphertext = await crypto.subtle.encrypt(
        { name: 'AES-GCM', iv },
        cryptoKey,
        plaintext
    );

    // Convert to base64 for transport
    const encryptedBase64 = btoa(String.fromCharCode(...new Uint8Array(ciphertext)));
    const ivBase64 = btoa(String.fromCharCode(...iv));

    return { encrypted: encryptedBase64, iv: ivBase64 };
}

/**
 * Decrypt a session bundle received from another device
 */
export async function decryptBundleFromSync(
    encrypted: string,
    iv: string,
    masterKey: Uint8Array
): Promise<SessionBundle> {
    const cryptoKey = await crypto.subtle.importKey(
        'raw',
        masterKey.buffer as ArrayBuffer,
        { name: 'AES-GCM' },
        false,
        ['decrypt']
    );

    const ciphertextBytes = Uint8Array.from(atob(encrypted), c => c.charCodeAt(0));
    const ivBytes = Uint8Array.from(atob(iv), c => c.charCodeAt(0));

    const plaintext = await crypto.subtle.decrypt(
        { name: 'AES-GCM', iv: ivBytes },
        cryptoKey,
        ciphertextBytes
    );

    const bundleJson = new TextDecoder().decode(plaintext);
    return JSON.parse(bundleJson) as SessionBundle;
}

/**
 * Generate a fingerprint from an identity key for display
 * Format: 12 groups of 5 digits (like Signal)
 */
export async function generateFingerprintAsync(identityKey: Uint8Array): Promise<string> {
    // Ensure we have a proper ArrayBuffer for crypto.subtle.digest
    const buffer = identityKey.buffer.slice(
        identityKey.byteOffset,
        identityKey.byteOffset + identityKey.byteLength
    ) as ArrayBuffer;

    const hash = await crypto.subtle.digest('SHA-256', buffer);
    const hashArray = new Uint8Array(hash);

    // Convert to groups of 5 digits (12 groups = 60 chars)
    let result = '';
    for (let i = 0; i < 30; i += 5) {
        if (result) result += ' ';
        // Take 2.5 bytes (20 bits) and convert to 5-digit number
        const val = (hashArray[i] << 12) | (hashArray[i + 1] << 4) | (hashArray[i + 2] >> 4);
        result += val.toString().padStart(5, '0');
    }

    return result;
}

/**
 * Check if a stored fingerprint matches the current identity key
 * Used to detect when a contact's keys have changed
 */
export async function hasIdentityKeyChanged(
    contactId: string,
    currentIdentityKey: Uint8Array
): Promise<boolean> {
    const storedKey = `identity:${contactId}`;
    const storedFingerprint = await get<string>(storedKey);

    if (!storedFingerprint) {
        // First time seeing this contact - store their key
        const fingerprint = await generateFingerprintAsync(currentIdentityKey);
        await set(storedKey, fingerprint);
        return false;
    }

    const currentFingerprint = await generateFingerprintAsync(currentIdentityKey);
    return storedFingerprint !== currentFingerprint;
}

/**
 * Update stored identity key for a contact
 */
export async function updateStoredIdentityKey(
    contactId: string,
    identityKey: Uint8Array
): Promise<void> {
    const storedKey = `identity:${contactId}`;
    const fingerprint = await generateFingerprintAsync(identityKey);
    await set(storedKey, fingerprint);
}

/**
 * Request sync from primary device
 * Called when new device detects no local sessions
 */
export async function requestSync(
    sendMessage: (type: string, payload: object) => void,
    timeout: number = 30000
): Promise<SessionBundle | null> {
    setSyncState({ status: 'requesting', message: 'Requesting sync from primary device...' });

    // Send sync request via WebSocket
    sendMessage('sync_request', {
        timestamp: Date.now(),
    });

    setSyncState({ status: 'waiting', message: 'Waiting for response...' });

    // Wait for response (with timeout)
    return new Promise((resolve) => {
        const timeoutId = setTimeout(() => {
            setSyncState({ status: 'timeout', message: 'Primary device did not respond' });
            resolve(null);
        }, timeout);

        // The actual response handling will be done in useWebSocket
        // This promise is resolved by setSyncBundle() being called
        const unsubscribe = onSyncStateChange((state) => {
            if (state.status === 'success') {
                clearTimeout(timeoutId);
                unsubscribe();
                resolve(null); // Bundle was already imported
            } else if (state.status === 'failed') {
                clearTimeout(timeoutId);
                unsubscribe();
                resolve(null);
            }
        });
    });
}

/**
 * Check if this device can act as primary (has sessions)
 */
export async function canActAsPrimary(): Promise<boolean> {
    const account = await get(OLM_ACCOUNT);
    return account !== undefined && account !== null;
}

/**
 * Reset sync state to idle
 */
export function resetSyncState(): void {
    setSyncState({ status: 'idle', message: '' });
}
