/**
 * Crypto Module Exports
 *
 * Provides Signal Protocol encryption for messages and
 * AES-256-GCM encryption for media files.
 */

export { SignalProtocol, signalProtocol } from './signal';
export type { KeyPair, SignedPreKey, PreKey, EncryptedMessage } from './signal';

export { encryptMedia, decryptMedia, generateThumbnail, validateFile, getMediaType } from './media';
export type { EncryptedMedia } from './media';
