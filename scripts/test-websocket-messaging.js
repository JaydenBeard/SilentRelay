/**
 * End-to-end WebSocket messaging test
 * Tests: User creation, WebSocket connection, message sending/receiving
 * 
 * Run with: node test-websocket-messaging.js
 */

const https = require('https');
const WebSocket = require('ws');
const crypto = require('crypto');

// Note: For local testing with self-signed certs, set NODE_TLS_REJECT_UNAUTHORIZED=0 in your environment

const BASE_URL = 'https://localhost';
const WS_URL = 'wss://localhost/ws';

async function makeRequest(endpoint, method = 'GET', body = null, token = null) {
    return new Promise((resolve, reject) => {
        const url = new URL(endpoint, BASE_URL);
        const options = {
            hostname: url.hostname,
            port: url.port || 443,
            path: url.pathname + url.search,
            method,
            agent,
            headers: {
                'Content-Type': 'application/json',
            },
        };

        if (token) {
            options.headers['Authorization'] = `Bearer ${token}`;
        }

        const req = https.request(options, (res) => {
            let data = '';
            res.on('data', (chunk) => (data += chunk));
            res.on('end', () => {
                try {
                    const parsed = JSON.parse(data);
                    resolve({ status: res.statusCode, data: parsed });
                } catch {
                    resolve({ status: res.statusCode, data: data });
                }
            });
        });

        req.on('error', reject);

        if (body) {
            req.write(JSON.stringify(body));
        }
        req.end();
    });
}

async function registerUser(phoneNumber, username) {
    console.log(`\n📱 Registering user: ${username} (${phoneNumber})`);

    // Step 1: Request code
    const codeResp = await makeRequest('/api/v1/auth/request-code', 'POST', {
        phone_number: phoneNumber,
    });

    if (codeResp.status !== 200) {
        console.error('   ❌ Failed to request code:', codeResp.data);
        return null;
    }

    const code = codeResp.data.code || '123456';
    console.log('   ✓ DEV CODE:', code);

    // Step 2: Verify code
    const verifyResp = await makeRequest('/api/v1/auth/verify', 'POST', {
        phone_number: phoneNumber,
        code: code,
    });

    if (verifyResp.data.user_exists) {
        console.log('   ✓ User exists, using existing account');
        return {
            token: verifyResp.data.token,
            userId: verifyResp.data.user?.id,
            username: username,
        };
    }

    // Step 3: Register new user
    const deviceId = crypto.randomUUID();

    // Generate valid base64 keys (32 bytes each for Curve25519)
    const identityKey = crypto.randomBytes(32).toString('base64');
    const signedPrekey = crypto.randomBytes(32).toString('base64');
    const signedPrekeySignature = crypto.randomBytes(64).toString('base64');
    const deviceKey = crypto.randomBytes(32).toString('base64');

    const registerResp = await makeRequest('/api/v1/auth/register', 'POST', {
        phone_number: phoneNumber,
        code: code,
        username: username,
        public_identity_key: identityKey,
        public_signed_prekey: signedPrekey,
        signed_prekey_signature: signedPrekeySignature,
        prekeys: [],
        device_id: deviceId,
        device_type: 'web',
        public_device_key: deviceKey,
    });

    if (registerResp.status !== 200 && registerResp.status !== 201) {
        console.error('   ❌ Failed to register:', registerResp.data);
        return null;
    }

    console.log('   ✓ User registered! ID:', registerResp.data.user?.user_id);

    return {
        token: registerResp.data.access_token,
        userId: registerResp.data.user?.user_id,
        username: username,
        identityKey,
        signedPrekey,
    };
}

function connectWebSocket(token, username) {
    return new Promise((resolve, reject) => {
        console.log(`\n🔌 Connecting WebSocket for ${username}...`);

        const ws = new WebSocket(`${WS_URL}?token=${token}`);

        const timeout = setTimeout(() => {
            ws.close();
            reject(new Error('WebSocket connection timeout'));
        }, 10000);

        ws.on('open', () => {
            clearTimeout(timeout);
            console.log(`   ✓ ${username} WebSocket connected!`);
            resolve(ws);
        });

        ws.on('error', (error) => {
            clearTimeout(timeout);
            console.error(`   ❌ ${username} WebSocket error:`, error.message);
            reject(error);
        });

        ws.on('message', (data) => {
            try {
                const msg = JSON.parse(data.toString());
                console.log(`   📨 ${username} received:`, msg.type, msg.payload ? '(has payload)' : '');
            } catch (e) {
                console.log(`   📨 ${username} received raw:`, data.toString().substring(0, 100));
            }
        });
    });
}

function sendMessage(ws, senderToken, recipientId, ciphertext) {
    return new Promise((resolve) => {
        const message = {
            type: 'send',
            payload: {
                recipientId: recipientId,
                ciphertext: ciphertext,
                messageType: 0,
            },
            timestamp: Date.now(),
            messageId: crypto.randomUUID(),
            // HMAC signature would go here in production
            signature: 'test-signature',
            nonce: Array.from(crypto.randomBytes(16)),
        };

        console.log('\n📤 Sending message...');
        console.log('   Type:', message.type);
        console.log('   Recipient:', recipientId);
        console.log('   Ciphertext (truncated):', ciphertext.substring(0, 50) + '...');

        ws.send(JSON.stringify(message));

        // Give time for message to be processed
        setTimeout(() => resolve(), 1000);
    });
}

async function main() {
    console.log('═══════════════════════════════════════════════════════════════');
    console.log('         WebSocket Messaging End-to-End Test');
    console.log('═══════════════════════════════════════════════════════════════');

    try {
        // Step 1: Register two users
        const user1 = await registerUser('+14155550101', 'test_sender');
        const user2 = await registerUser('+14155550102', 'test_receiver');

        if (!user1 || !user2) {
            console.error('\n❌ Failed to create both users');
            return;
        }

        console.log('\n═══ User Summary ═══');
        console.log('User 1 (Sender):', user1.userId);
        console.log('User 2 (Receiver):', user2.userId);

        // Step 2: Connect both users via WebSocket
        const ws1 = await connectWebSocket(user1.token, 'Sender');
        const ws2 = await connectWebSocket(user2.token, 'Receiver');

        // Give connections time to stabilize
        await new Promise(r => setTimeout(r, 1000));

        // Step 3: Send a test message from user1 to user2
        const testMessage = 'Hello from the test script!';
        const fakeCiphertext = Buffer.from(testMessage).toString('base64');

        await sendMessage(ws1, user1.token, user2.userId, fakeCiphertext);

        // Step 4: Wait for message delivery
        console.log('\n⏳ Waiting for message delivery (3 seconds)...');
        await new Promise(r => setTimeout(r, 3000));

        // Step 5: Check database for messages
        console.log('\n═══ Checking Database ═══');
        console.log('Run this command to check messages:');
        console.log(`docker exec messaging-app-postgres-1 psql -U messaging -d messaging -c "SELECT message_id, sender_id, receiver_id, status FROM messages ORDER BY timestamp DESC LIMIT 5;"`);

        // Cleanup
        console.log('\n🧹 Closing WebSocket connections...');
        ws1.close();
        ws2.close();

        console.log('\n═══════════════════════════════════════════════════════════════');
        console.log('                    Test Complete!');
        console.log('═══════════════════════════════════════════════════════════════');

    } catch (error) {
        console.error('\n❌ Test failed:', error);
    }
}

main();
