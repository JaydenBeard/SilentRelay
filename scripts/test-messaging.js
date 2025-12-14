/**
 * Test script to verify messaging flow
 * Run with: node test-messaging.js
 */

const https = require('https');
const crypto = require('crypto');


// Note: For local testing with self-signed certs, set NODE_TLS_REJECT_UNAUTHORIZED=0 in your environment

const agent = new https.Agent({});

const BASE_URL = 'https://localhost';

function generateUUID() {
    return crypto.randomUUID();
}

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

async function registerUser(phoneNumber) {
    console.log(`\nRegistering user with phone: ${phoneNumber}`);

    // Step 1: Request code
    const codeResp = await makeRequest('/api/v1/auth/request-code', 'POST', {
        phone_number: phoneNumber,
    });

    if (codeResp.status !== 200) {
        console.error('   FAILED: Could not request code:', codeResp.data);
        return null;
    }

    const code = codeResp.data.code || '123456';
    console.log('   DEV CODE:', code);

    // Step 2: Verify code
    const verifyResp = await makeRequest('/api/v1/auth/verify', 'POST', {
        phone_number: phoneNumber,
        code: code,
    });

    if (verifyResp.data.user_exists) {
        console.log('   User already exists, logging in...');
        return {
            token: verifyResp.data.token,
            userId: verifyResp.data.user?.id,
        };
    }

    // Step 3: Register new user
    const deviceId = generateUUID();
    const registerResp = await makeRequest('/api/v1/auth/register', 'POST', {
        phone_number: phoneNumber,
        code: code,
        public_identity_key: 'VGVzdElkZW50aXR5S2V5QmFzZTY0U3RyaW5nPT0=',  // Valid base64 (32 bytes)
        public_signed_prekey: 'VGVzdFNpZ25lZFByZWtleUJhc2U2NFN0cmluZz0=',  // Valid base64 (32 bytes)
        signed_prekey_signature: 'VGVzdFNpZ25hdHVyZUJhc2U2NFN0cmluZ0xvbmdlcj0=',  // Valid base64 (64 bytes)
        prekeys: [],
        device_id: deviceId,
        device_type: 'web',
        public_device_key: 'VGVzdERldmljZUtleT09',  // Valid base64
    });

    console.log('   Register status:', registerResp.status);

    if (registerResp.status !== 200 && registerResp.status !== 201) {
        console.error('   FAILED:', registerResp.data);
        return null;
    }

    console.log('   SUCCESS! User ID:', registerResp.data.user?.user_id);

    return {
        token: registerResp.data.access_token,
        userId: registerResp.data.user?.user_id,
    };
}

async function main() {
    console.log('=== Messaging App Backend Test ===');

    // Register two users
    const user1 = await registerUser('+14155550001');
    const user2 = await registerUser('+14155550002');

    if (!user1 || !user2) {
        console.error('\nFailed to create both users');
        return;
    }

    console.log('\n--- User Summary ---');
    console.log('User 1:', user1);
    console.log('User 2:', user2);

    // Get user 2's keys from user 1's perspective
    if (user1.token && user2.userId) {
        console.log('\n--- Fetching User 2 Keys ---');
        const keysResp = await makeRequest(`/api/v1/users/${user2.userId}/keys`, 'GET', null, user1.token);
        console.log('Status:', keysResp.status);
        console.log('Keys:', JSON.stringify(keysResp.data, null, 2));
    }

    console.log('\n=== Test Complete ===');
    console.log('\nCheck database:');
    console.log('docker exec messaging-app-postgres-1 psql -U messaging -d messaging -c "SELECT user_id, phone_number, LEFT(public_identity_key, 30) as identity_key FROM users;"');
}

main().catch(console.error);
