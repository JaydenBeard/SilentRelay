/**
 * End-to-end WebSocket messaging test (simplified)
 * Run with: node test-ws-simple.js
 */

const https = require('https');
const WebSocket = require('ws');
const crypto = require('crypto');

// Note: For local testing with self-signed certs, set NODE_TLS_REJECT_UNAUTHORIZED=0 in your environment

const BASE_URL = 'https://localhost';
const WS_URL = 'wss://localhost/ws';

// Generate HMAC-SHA256 signature matching frontend/backend implementation
function generateHmacSignature(token, type, timestamp, messageId, payload) {
    // Create message string: type + timestamp + messageId + payload (as JSON)
    const messageStr = `${type}:${timestamp}:${messageId}:${JSON.stringify(payload)}`;

    // Use token as key (first 32 bytes, padded if shorter)
    let keyBuffer = Buffer.from(token, 'utf8');
    if (keyBuffer.length < 32) {
        const padding = Buffer.alloc(32 - keyBuffer.length);
        keyBuffer = Buffer.concat([keyBuffer, padding]);
    } else if (keyBuffer.length > 32) {
        keyBuffer = keyBuffer.slice(0, 32);
    }

    const hmac = crypto.createHmac('sha256', keyBuffer);
    hmac.update(messageStr);
    return hmac.digest('hex');
}

async function makeRequest(endpoint, method = 'GET', body = null, token = null) {
    return new Promise((resolve, reject) => {
        const url = new URL(endpoint, BASE_URL);
        const options = {
            hostname: url.hostname,
            port: url.port || 443,
            path: url.pathname + url.search,
            method,
            headers: { 'Content-Type': 'application/json' },
        };
        if (token) options.headers['Authorization'] = `Bearer ${token}`;

        const req = https.request(options, (res) => {
            let data = '';
            res.on('data', (chunk) => (data += chunk));
            res.on('end', () => {
                try {
                    resolve({ status: res.statusCode, data: JSON.parse(data) });
                } catch {
                    resolve({ status: res.statusCode, data: data });
                }
            });
        });
        req.on('error', reject);
        if (body) req.write(JSON.stringify(body));
        req.end();
    });
}

async function registerUser(phoneNumber) {
    const codeResp = await makeRequest('/api/v1/auth/request-code', 'POST', { phone_number: phoneNumber });
    if (codeResp.status !== 200) return null;

    const code = codeResp.data.code || '123456';
    const verifyResp = await makeRequest('/api/v1/auth/verify', 'POST', { phone_number: phoneNumber, code });

    if (verifyResp.data.user_exists) {
        return { token: verifyResp.data.token, userId: verifyResp.data.user?.id };
    }

    const registerResp = await makeRequest('/api/v1/auth/register', 'POST', {
        phone_number: phoneNumber,
        code: code,
        public_identity_key: crypto.randomBytes(32).toString('base64'),
        public_signed_prekey: crypto.randomBytes(32).toString('base64'),
        signed_prekey_signature: crypto.randomBytes(64).toString('base64'),
        prekeys: [],
        device_id: crypto.randomUUID(),
        device_type: 'web',
        public_device_key: crypto.randomBytes(32).toString('base64'),
    });

    if (registerResp.status !== 200 && registerResp.status !== 201) return null;
    return { token: registerResp.data.access_token, userId: registerResp.data.user?.user_id };
}

async function main() {
    console.log('Starting WebSocket messaging test...');

    // Register users
    console.log('Registering user 1...');
    const user1 = await registerUser('+14155550201');
    console.log('User 1:', user1 ? user1.userId : 'FAILED');

    console.log('Registering user 2...');
    const user2 = await registerUser('+14155550202');
    console.log('User 2:', user2 ? user2.userId : 'FAILED');

    if (!user1 || !user2) {
        console.log('Failed to create users');
        return;
    }

    // Connect WebSockets
    console.log('Connecting WebSocket for user 1...');
    const ws1 = new WebSocket(`${WS_URL}?token=${user1.token}`);

    const connected1 = await new Promise((resolve) => {
        ws1.on('open', () => { console.log('User 1 WS connected'); resolve(true); });
        ws1.on('error', (e) => { console.log('User 1 WS error:', e.message); resolve(false); });
        setTimeout(() => resolve(false), 5000);
    });

    console.log('Connecting WebSocket for user 2...');
    const ws2 = new WebSocket(`${WS_URL}?token=${user2.token}`);

    let receivedMessage = null;
    ws2.on('message', (data) => {
        console.log('User 2 received message:', data.toString().substring(0, 200));
        receivedMessage = data.toString();
    });

    const connected2 = await new Promise((resolve) => {
        ws2.on('open', () => { console.log('User 2 WS connected'); resolve(true); });
        ws2.on('error', (e) => { console.log('User 2 WS error:', e.message); resolve(false); });
        setTimeout(() => resolve(false), 5000);
    });

    if (!connected1 || !connected2) {
        console.log('WebSocket connection failed');
        return;
    }

    // Wait for connections to stabilize
    await new Promise(r => setTimeout(r, 1000));

    // Send message from user 1 to user 2
    console.log('Sending message from user 1 to user 2...');

    const type = 'send';
    const payload = {
        receiver_id: user2.userId,  // snake_case to match Go backend
        ciphertext: Buffer.from('Hello from test!').toString('base64'),
        message_type: 'whisper',  // string, not number
    };
    const timestamp = new Date().toISOString();  // ISO 8601 format for Go
    const messageId = crypto.randomUUID();

    // Generate proper HMAC signature using user1's token
    const signature = generateHmacSignature(user1.token, type, timestamp, messageId, payload);

    const message = {
        type,
        payload,
        timestamp,
        messageId,
        signature,
        nonce: crypto.randomBytes(16).toString('hex'),  // Hex string for Go
    };

    console.log('Signature generated:', signature.substring(0, 32) + '...');
    ws1.send(JSON.stringify(message));
    console.log('Message sent!');

    // Wait for delivery
    await new Promise(r => setTimeout(r, 3000));

    console.log('Received by user 2:', receivedMessage ? 'YES' : 'NO');

    // Check server logs
    console.log('\nCheck server logs with:');
    console.log('docker logs messaging-app-chat-server-1-1 --tail 20');

    // Cleanup
    ws1.close();
    ws2.close();
    console.log('Test complete!');
}

main().catch(console.error);
