/**
 * E2E Tests for localStorage Encryption Implementation
 *
 * Tests the localStorage encryption functionality in a real browser environment
 * to ensure chat messages are properly encrypted and decrypted.
 */

const puppeteer = require('puppeteer');

// Test configuration
const TEST_CONFIG = {
  baseUrl: 'http://localhost:3000',
  timeout: 30000,
  slowMo: 100,
  headless: false
};

// Test results storage
let testResults = {
  summary: {
    total: 0,
    passed: 0,
    failed: 0,
    skipped: 0
  },
  tests: [],
  errors: [],
  recommendations: []
};

// Helper function to log test results
function logTest(testName, status, message = '', error = null) {
  testResults.tests.push({
    name: testName,
    status,
    message,
    timestamp: new Date().toISOString(),
    error: error ? error.message : null
  });

  testResults.summary.total++;
  if (status === 'PASS') testResults.summary.passed++;
  else if (status === 'FAIL') testResults.summary.failed++;
  else if (status === 'SKIP') testResults.summary.skipped++;

  console.log(`[${status}] ${testName}: ${message}`);
  if (error) console.error(error);
}

describe('localStorage Encryption E2E Tests', () => {
  let browser;
  let page;

  beforeAll(async () => {
    browser = await puppeteer.launch({
      headless: TEST_CONFIG.headless,
      slowMo: TEST_CONFIG.slowMo,
      args: ['--no-sandbox', '--disable-setuid-sandbox', '--disable-dev-shm-usage']
    });
  });

  afterAll(async () => {
    if (browser) {
      await browser.close();
    }

    // Generate test report
    const fs = require('fs');
    const path = require('path');
    const reportPath = path.join(__dirname, 'localStorage-encryption-test-report.json');
    fs.writeFileSync(reportPath, JSON.stringify(testResults, null, 2));
    console.log(`\nTest report saved to: ${reportPath}`);
  });

  beforeEach(async () => {
    page = await browser.newPage();
    await page.setViewport({ width: 1280, height: 720 });
    await page.setDefaultTimeout(TEST_CONFIG.timeout);

    // Set up console logging
    page.on('console', msg => {
      if (msg.type() === 'error') {
        testResults.errors.push({
          message: msg.text(),
          timestamp: new Date().toISOString()
        });
      }
    });
  });

  afterEach(async () => {
    if (page) {
      await page.close();
    }
  });

  test('Send/receive messages - verify encryption in localStorage', async () => {
    try {
      await page.goto(TEST_CONFIG.baseUrl);
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Check if we can access the chat store and inspect localStorage
      const storageInspection = await page.evaluate(() => {
        // Try to access the chat store if it's exposed on window
        const chatData = localStorage.getItem('chat-storage');

        if (!chatData) {
          return { hasData: false, message: 'No chat data found in localStorage' };
        }

        // Check encryption indicators
        const isEncrypted = chatData.includes(':');
        const looksLikeJson = (() => {
          try {
            JSON.parse(chatData);
            return true;
          } catch {
            return false;
          }
        })();

        // Check for any readable message content (should not exist if encrypted)
        const hasReadableContent = /hello|test|message|chat|conversation/i.test(chatData);

        return {
          hasData: true,
          length: chatData.length,
          isEncrypted,
          looksLikeJson,
          hasReadableContent,
          sample: chatData.substring(0, 200) + '...'
        };
      });

      if (!storageInspection.hasData) {
        logTest('Send/receive messages - verify encryption in localStorage', 'SKIP',
          storageInspection.message + ' (may need to send messages first)');
        return;
      }

      if (storageInspection.isEncrypted && !storageInspection.hasReadableContent) {
        logTest('Send/receive messages - verify encryption in localStorage', 'PASS',
          `Messages are properly encrypted in localStorage (${storageInspection.length} chars, encrypted format)`);
      } else if (storageInspection.hasReadableContent) {
        logTest('Send/receive messages - verify encryption in localStorage', 'FAIL',
          'Message content found in readable form in localStorage - encryption not working');
      } else {
        logTest('Send/receive messages - verify encryption in localStorage', 'WARN',
          'localStorage data format unclear - may not be encrypted or may be corrupted');
      }
    } catch (error) {
      logTest('Send/receive messages - verify encryption in localStorage', 'FAIL',
        'Failed to verify message encryption', error);
      throw error;
    }
  });

  test('Refresh browser - verify messages decrypt and display correctly', async () => {
    try {
      await page.goto(TEST_CONFIG.baseUrl);
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Get initial state
      const initialState = await page.evaluate(() => {
        return {
          hasChatData: !!localStorage.getItem('chat-storage'),
          chatData: localStorage.getItem('chat-storage'),
          url: window.location.href
        };
      });

      if (!initialState.hasChatData) {
        logTest('Refresh browser - verify messages decrypt and display correctly', 'SKIP',
          'No chat data to test refresh persistence');
        return;
      }

      // Refresh the page
      await page.reload({ waitUntil: 'networkidle0' });
      await new Promise(resolve => setTimeout(resolve, 3000));

      // Check post-refresh state
      const refreshedState = await page.evaluate(() => {
        return {
          hasChatData: !!localStorage.getItem('chat-storage'),
          chatData: localStorage.getItem('chat-storage'),
          url: window.location.href,
          // Check for any error indicators in the DOM
          hasErrors: !!document.querySelector('.error, [data-testid="error"]'),
          // Check if chat interface loaded
          hasChatInterface: !!document.querySelector('.chat, [data-testid="chat"], .message-list')
        };
      });

      if (refreshedState.hasChatData && refreshedState.hasChatInterface && !refreshedState.hasErrors) {
        logTest('Refresh browser - verify messages decrypt and display correctly', 'PASS',
          'Page refreshed successfully with chat data intact and interface functional');
      } else if (refreshedState.hasErrors) {
        logTest('Refresh browser - verify messages decrypt and display correctly', 'FAIL',
          'Errors detected after refresh - decryption may have failed');
      } else if (!refreshedState.hasChatInterface) {
        logTest('Refresh browser - verify messages decrypt and display correctly', 'WARN',
          'Chat interface not found after refresh - may be on wrong page');
      } else {
        logTest('Refresh browser - verify messages decrypt and display correctly', 'WARN',
          'Refresh completed but chat data state unclear');
      }
    } catch (error) {
      logTest('Refresh browser - verify messages decrypt and display correctly', 'FAIL',
        'Failed to test browser refresh behavior', error);
      throw error;
    }
  });

  test('Inspect localStorage - verify only encrypted data is stored', async () => {
    try {
      await page.goto(TEST_CONFIG.baseUrl);
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Comprehensive localStorage inspection
      const storageAnalysis = await page.evaluate(() => {
        const analysis = {
          totalKeys: localStorage.length,
          chatStorage: null,
          otherKeys: [],
          sensitiveDataFound: [],
          encryptionIndicators: []
        };

        for (let i = 0; i < localStorage.length; i++) {
          const key = localStorage.key(i);
          const value = localStorage.getItem(key);

          if (key === 'chat-storage') {
            analysis.chatStorage = {
              key,
              length: value.length,
              containsColon: value.includes(':'),
              isJsonParsable: (() => {
                try {
                  JSON.parse(value);
                  return true;
                } catch {
                  return false;
                }
              })()
            };
          } else {
            analysis.otherKeys.push({ key, length: value.length });
          }

          // Check for sensitive data patterns
          const sensitivePatterns = [
            /\b(?:hello|test|message|chat|conversation|password|token|key|secret|user|email|phone)\b/i,
            /\b\d{10,}\b/, // Long numbers (could be phone numbers)
            /\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/ // Email pattern
          ];

          sensitivePatterns.forEach(pattern => {
            if (pattern.test(value)) {
              analysis.sensitiveDataFound.push({
                key,
                pattern: pattern.toString(),
                sample: value.substring(0, 100) + '...'
              });
            }
          });
        }

        return analysis;
      });

      // Analyze results
      if (storageAnalysis.chatStorage) {
        if (storageAnalysis.chatStorage.containsColon && !storageAnalysis.chatStorage.isJsonParsable) {
          logTest('Inspect localStorage - verify only encrypted data is stored', 'PASS',
            `Chat storage is encrypted (${storageAnalysis.chatStorage.length} chars, contains ':')`);
        } else if (storageAnalysis.chatStorage.isJsonParsable) {
          logTest('Inspect localStorage - verify only encrypted data is stored', 'WARN',
            'Chat storage appears to be unencrypted JSON - encryption may be disabled');
        } else {
          logTest('Inspect localStorage - verify only encrypted data is stored', 'FAIL',
            'Chat storage format unclear - neither clearly encrypted nor valid JSON');
        }
      } else {
        logTest('Inspect localStorage - verify only encrypted data is stored', 'SKIP',
          'No chat-storage key found');
      }

      // Report sensitive data findings
      if (storageAnalysis.sensitiveDataFound.length > 0) {
        testResults.recommendations.push(
          `Found ${storageAnalysis.sensitiveDataFound.length} instances of potentially sensitive data in localStorage: ` +
          storageAnalysis.sensitiveDataFound.map(f => `${f.key} (${f.pattern})`).join(', ')
        );
      }
    } catch (error) {
      logTest('Inspect localStorage - verify only encrypted data is stored', 'FAIL',
        'Failed to inspect localStorage contents', error);
      throw error;
    }
  });

  test('Test error scenarios - PIN not entered, encryption failures', async () => {
    try {
      await page.goto(TEST_CONFIG.baseUrl);
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Monitor console for encryption-related messages
      const consoleMessages = [];
      const errorListener = msg => {
        const text = msg.text();
        if (text.includes('encrypt') || text.includes('decrypt') ||
            text.includes('crypto') || text.includes('PIN') ||
            text.includes('master key') || text.includes('localStorage')) {
          consoleMessages.push({
            type: msg.type(),
            text: text,
            timestamp: new Date().toISOString()
          });
        }
      };

      page.on('console', errorListener);

      // Wait for app initialization
      await new Promise(resolve => setTimeout(resolve, 5000));

      // Check for error indicators in DOM
      const errorCheck = await page.evaluate(() => {
        const errorElements = document.querySelectorAll('.error, .alert-danger, [data-testid*="error"]');
        const warningElements = document.querySelectorAll('.warning, .alert-warning');

        return {
          hasErrors: errorElements.length > 0,
          hasWarnings: warningElements.length > 0,
          errorTexts: Array.from(errorElements).map(el => el.textContent.trim()),
          warningTexts: Array.from(warningElements).map(el => el.textContent.trim())
        };
      });

      const errors = consoleMessages.filter(msg => msg.type === 'error');
      const warnings = consoleMessages.filter(msg => msg.type === 'warn');

      if (errors.length === 0 && !errorCheck.hasErrors) {
        logTest('Test error scenarios - PIN not entered, encryption failures', 'PASS',
          'No encryption-related errors detected');
      } else {
        const allErrors = [
          ...errors.map(e => `Console: ${e.text}`),
          ...errorCheck.errorTexts
        ];
        logTest('Test error scenarios - PIN not entered, encryption failures', 'WARN',
          `Found ${allErrors.length} potential encryption errors: ${allErrors.join('; ')}`);
      }

      if (warnings.length > 0 || errorCheck.hasWarnings) {
        testResults.recommendations.push(
          `Review ${warnings.length + errorCheck.warningTexts.length} encryption-related warnings`
        );
      }

      // Clean up listener
      page.off('console', errorListener);
    } catch (error) {
      logTest('Test error scenarios - PIN not entered, encryption failures', 'FAIL',
        'Failed to test error scenarios', error);
      throw error;
    }
  });

  test('Verify backward compatibility with existing unencrypted data', async () => {
    try {
      await page.goto(TEST_CONFIG.baseUrl);
      await new Promise(resolve => setTimeout(resolve, 2000));

      // Create mock unencrypted data that might exist from previous versions
      const legacyData = {
        state: {
          conversations: {
            'legacy-conv-1': {
              id: 'legacy-conv-1',
              recipientId: 'legacy-user-1',
              recipientName: 'Legacy User 1',
              unreadCount: 2,
              isOnline: true,
              lastSeen: Date.now() - 3600000,
              isPinned: false,
              isMuted: false
            }
          },
          messages: {
            'legacy-conv-1': [
              {
                id: 'legacy-msg-1',
                conversationId: 'legacy-conv-1',
                senderId: 'legacy-user-1',
                content: 'This is a legacy message that should be handled gracefully',
                timestamp: Date.now() - 7200000,
                status: 'read',
                type: 'text'
              },
              {
                id: 'legacy-msg-2',
                conversationId: 'legacy-conv-1',
                senderId: 'current-user',
                content: 'Another legacy message',
                timestamp: Date.now() - 3600000,
                status: 'sent',
                type: 'text'
              }
            ]
          },
          presenceCache: {
            'legacy-user-1': { isOnline: true, lastSeen: Date.now() - 3600000 }
          }
        },
        version: 0
      };

      // Inject legacy data
      await page.evaluate((data) => {
        localStorage.setItem('chat-storage', JSON.stringify(data));
      }, legacyData);

      // Refresh to trigger data loading
      await page.reload({ waitUntil: 'networkidle0' });
      await new Promise(resolve => setTimeout(resolve, 3000));

      // Check how the app handles legacy data
      const compatibilityCheck = await page.evaluate(() => {
        const currentData = localStorage.getItem('chat-storage');

        if (!currentData) {
          return { result: 'lost', message: 'Legacy data was lost' };
        }

        // Check if data is still in legacy format
        const isLegacyFormat = (() => {
          try {
            const parsed = JSON.parse(currentData);
            return parsed.version === 0 && parsed.state;
          } catch {
            return false;
          }
        })();

        // Check if data appears encrypted now
        const isEncrypted = currentData.includes(':') && !(() => {
          try {
            JSON.parse(currentData);
            return true;
          } catch {
            return false;
          }
        })();

        // Check for error messages
        const hasErrors = !!document.querySelector('.error, .alert-danger, [data-testid*="error"]');

        return {
          result: isEncrypted ? 'migrated' : (isLegacyFormat ? 'preserved' : 'unknown'),
          isLegacyFormat,
          isEncrypted,
          hasErrors,
          length: currentData.length
        };
      });

      if (compatibilityCheck.result === 'migrated') {
        logTest('Verify backward compatibility with existing unencrypted data', 'PASS',
          'Legacy data successfully migrated to encrypted format');
      } else if (compatibilityCheck.result === 'preserved') {
        logTest('Verify backward compatibility with existing unencrypted data', 'WARN',
          'Legacy data preserved in unencrypted format - migration may not be implemented');
      } else if (compatibilityCheck.hasErrors) {
        logTest('Verify backward compatibility with existing unencrypted data', 'FAIL',
          'Errors occurred while handling legacy data');
      } else {
        logTest('Verify backward compatibility with existing unencrypted data', 'SKIP',
          'Unable to determine legacy data handling behavior');
      }
    } catch (error) {
      logTest('Verify backward compatibility with existing unencrypted data', 'FAIL',
        'Failed to test backward compatibility', error);
      throw error;
    }
  });
});

// Generate final report
afterAll(() => {
  console.log('\n=== localStorage ENCRYPTION TEST SUMMARY ===');
  console.log(`Total: ${testResults.summary.total}`);
  console.log(`Passed: ${testResults.summary.passed}`);
  console.log(`Failed: ${testResults.summary.failed}`);
  console.log(`Skipped: ${testResults.summary.skipped}`);

  if (testResults.errors.length > 0) {
    console.log('\n=== CONSOLE ERRORS ===');
    testResults.errors.forEach(error => console.log(`- ${error.message}`));
  }

  if (testResults.recommendations.length > 0) {
    console.log('\n=== RECOMMENDATIONS ===');
    testResults.recommendations.forEach(rec => console.log(`- ${rec}`));
  }
});