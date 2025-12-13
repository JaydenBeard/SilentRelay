  1 | # User Security Guide
  2 | 
  3 | ## Welcome to SilentRelay
  4 | 
  5 | Thank you for choosing SilentRelay for your private communications. This guide explains the security features that protect your messages and how you can use them effectively.
  6 | 
  7 | ## Your Security Protections
  8 | 
  9 | ### End-to-End Encryption
 10 | 
 11 | **What it means:** Only you and the person you're communicating with can read your messages. Not even our servers can access the content.
 12 | 
 13 | **How it works:**
 14 | - Each message is encrypted on your device before sending
 15 | - Messages are decrypted only on the recipient's device
 16 | - Encryption keys are never stored on our servers
 17 | - Uses industry-standard Signal Protocol (compatible implementation)
 18 | 
 19 | **Visualization:**
 20 | ```
 21 | Your Device → [Encrypt] → SilentRelay Server → [Decrypt] → Recipient Device
 22 | ```
 23 | 
 24 | ### Device Verification
 25 | 
 26 | **What it means:** You can verify that you're communicating with the intended person's device.
 27 | 
 28 | **How to use it:**
 29 | 1. Go to a contact's profile
 30 | 2. Tap "Verify Security Code"
 31 | 3. Compare the security code in person or via another secure channel
 32 | 4. If codes match, the connection is secure
 33 | 
 34 | **Why it matters:** Prevents man-in-the-middle attacks where someone could intercept your messages.
 35 | 
 36 | ### Disappearing Messages
 37 | 
 38 | **What it means:** Messages automatically delete after a set time.
 39 | 
 40 | **How to use it:**
 41 | 1. Start a chat
 42 | 2. Tap the timer icon
 43 | 3. Choose a duration (from 5 seconds to 1 week)
 44 | 4. All messages in that chat will automatically disappear after the chosen time
 45 | 
 46 | **Important notes:**
 47 | - Disappearing messages are deleted from both devices
 48 | - Screenshots can still be taken (we notify when screenshots are detected)
 49 | - Messages may still exist in device backups
 50 | 
 51 | ### PIN Protection
 52 | 
 53 | **What it means:** Add an extra layer of security to your account.
 54 | 
 55 | **How to set it up:**
 56 | 1. Go to Settings → Security
 57 | 2. Tap "Set Up PIN"
 58 | 3. Choose a 6-digit PIN (avoid obvious combinations like 123456)
 59 | 4. Confirm your PIN
 60 | 
 61 | **How it protects you:**
 62 | - Required to access encrypted message history
 63 | - Prevents unauthorized access to your account
 64 | - Adds security for device backups
 65 | 
 66 | ### Recovery Key
 67 | 
 68 | **What it means:** A way to restore your account if you lose access to your device.
 69 | 
 70 | **How to set it up:**
 71 | 1. Go to Settings → Security → Recovery Key
 72 | 2. Tap "Generate Recovery Key"
 73 | 3. Write down the 24-word recovery phrase
 74 | 4. Store it securely (we recommend a physical safe or secure password manager)
 75 | 
 76 | **Important security tips:**
 77 | - Never share your recovery key with anyone
 78 | - Never store it digitally where it could be hacked
 79 | - If you lose it, you may permanently lose access to your account
 80 | 
 81 | ## Security Best Practices
 82 | 
 83 | ### Account Security
 84 | 
 85 | **Do:**
 86 | - Set up a strong PIN
 87 | - Enable biometric authentication if available
 88 | - Set up a recovery key and store it securely
 89 | - Verify security codes with important contacts
 90 | - Use disappearing messages for sensitive conversations
 91 | 
 92 | **Avoid:**
 93 | - Using obvious PINs like 123456 or your birthday
 94 | - Sharing your PIN or recovery key with anyone
 95 | - Storing your recovery key digitally
 96 | - Using SilentRelay on public or shared devices
 97 | - Jailbreaking or rooting your device
 98 | 
 99 | ### Device Security
100 | 
101 | **Do:**
102 | - Keep your device's operating system updated
103 | - Use a strong device passcode
104 | - Enable full-disk encryption on your device
105 | - Install security updates promptly
106 | - Use reputable antivirus software
107 | 
108 | **Avoid:**
109 | - Using public Wi-Fi for sensitive conversations
110 | - Installing apps from untrusted sources
111 | - Leaving your device unlocked and unattended
112 | - Using outdated operating systems
113 | 
114 | ### Communication Security
115 | 
116 | **Do:**
117 | - Verify security codes with important contacts
118 | - Use disappearing messages for sensitive information
119 | - Be cautious about what personal information you share
120 | - Report suspicious messages or contacts
121 | - Use voice/video calls for additional verification
122 | 
123 | **Avoid:**
124 | - Sharing sensitive information in group chats
125 | - Clicking on suspicious links
126 | - Sharing your location with untrusted contacts
127 | - Discussing highly sensitive topics without additional verification
128 | 
129 | ## What to Do If You Suspect a Security Issue
130 | 
131 | ### If Your Device is Lost or Stolen
132 | 
133 | 1. **Immediately** use another device to:
134 |    - Change your account PIN
135 |    - Enable account lock
136 |    - Revoke the lost device from your account
137 | 
138 | 2. **If you have a recovery key:**
139 |    - Use it to restore your account on a new device
140 |    - This will disable the lost device
141 | 
142 | 3. **If you don't have a recovery key:**
143 |    - Contact our support team immediately
144 |    - Be prepared to verify your identity
145 | 
146 | ### If You Receive a Suspicious Message
147 | 
148 | 1. **Do NOT** click any links or download any files
149 | 2. **Do NOT** reply to the message
150 | 3. **Do** report the message to us:
151 |    - Tap and hold the message
152 |    - Select "Report"
153 |    - Choose "Suspicious Content"
154 | 
155 | 4. **Do** verify with the contact through another channel if possible
156 | 
157 | ### If You Suspect Your Account is Compromised
158 | 
159 | 1. **Immediately** change your PIN
160 | 2. **Enable** account lock
161 | 3. **Review** your recent messages and contacts
162 | 4. **Report** the incident to our security team
163 | 5. **Consider** generating a new recovery key
164 | 
165 | ## Advanced Security Features
166 | 
167 | ### Key Rotation
168 | 
169 | **What it is:** Your encryption keys are automatically rotated to enhance security.
170 | 
171 | **How it works:**
172 | - One-time pre-keys rotate after each use
173 | - Signed pre-keys rotate weekly
174 | - This provides "forward secrecy" - if a key is compromised, it only affects a limited number of messages
175 | 
176 | **What you'll notice:**
177 | - Occasional brief reconnection when keys rotate
178 | - No action required on your part
179 | - Continuous protection without interruption
180 | 
181 | ### Sealed Sender
182 | 
183 | **What it is:** Additional protection that hides message metadata.
184 | 
185 | **How it works:**
186 | - Even the message headers are encrypted
187 | - Our servers don't know who is communicating with whom
188 | - Provides protection against traffic analysis
189 | 
190 | **What you'll notice:**
191 | - All your messages are protected this way automatically
192 | - No special setup required
193 | - Works seamlessly in the background
194 | 
195 | ### Certificate Pinning
196 | 
197 | **What it is:** Protection against man-in-the-middle attacks on your connection.
198 | 
199 | **How it works:**
200 | - Your app remembers the correct security certificates
201 | - If a fake certificate is presented, the connection is blocked
202 | - Prevents attackers from intercepting your traffic
203 | 
204 | **What you'll notice:**
205 | - If there's a certificate issue, you'll see a security warning
206 | - This is rare and usually indicates a real security problem
207 | - Contact support if you see certificate warnings
208 | 
209 | ## Understanding Security Indicators
210 | 
211 | ### In Your Chats
212 | 
213 | | Icon | Meaning | What to Do |
214 | |------|---------|------------|
215 | | Lock | End-to-end encrypted | Normal - your messages are secure |
216 | | Warning | Security issue detected | Tap for details and follow recommendations |
217 | | Timer | Disappearing messages enabled | Shows the timer duration |
218 | | Sync | Key rotation in progress | Brief reconnection - no action needed |
219 | 
220 | ### In Contact Profiles
221 | 
222 | | Icon | Meaning | What to Do |
223 | |------|---------|------------|
224 | | Checkmark | Security code verified | This contact's device is verified |
225 | | Question | Security code not verified | Consider verifying if this is an important contact |
226 | | Device | Multiple devices | This contact has multiple verified devices |
227 | | Warning | Security warning | Review the security details carefully |
228 | 
229 | ## Security Settings Explained
230 | 
231 | ### PIN Settings
232 | 
233 | | Setting | Recommendation | Why It Matters |
234 | |--------|---------------|----------------|
235 | | PIN Length | 6 digits minimum | Longer PINs are more secure |
236 | | PIN Timeout | 1-5 minutes | Balances security and convenience |
237 | | Biometric Auth | Enable if available | Adds convenience without reducing security |
238 | 
239 | ### Privacy Settings
240 | 
241 | | Setting | Recommendation | Why It Matters |
242 | |--------|---------------|----------------|
243 | | Read Receipts | Your choice | Shows when you've read messages |
244 | | Typing Indicators | Your choice | Shows when you're typing |
245 | | Last Seen | Your choice | Shows when you were last active |
246 | | Profile Visibility | Contacts only | Limits who can see your profile |
247 | 
248 | ### Security Settings
249 | 
250 | | Setting | Recommendation | Why It Matters |
251 | |--------|---------------|----------------|
252 | | Screen Lock | Enable | Prevents unauthorized access |
253 | | App Lock Timeout | 1-5 minutes | Balances security and convenience |
254 | | Backup Encryption | Enable | Protects your message history |
255 | | Security Notifications | Enable | Alerts you to important security events |
256 | 
257 | ## Device-Specific Security Tips
258 | 
259 | ### iOS Devices
260 | 
261 | - Enable Face ID or Touch ID for SilentRelay
262 | - Use iCloud Keychain to securely store your recovery key
263 | - Enable "Find My iPhone" to help recover lost devices
264 | - Keep iOS updated to the latest version
265 | - Enable "Stolen Device Protection" in iOS settings
266 | 
267 | ### Android Devices
268 | 
269 | - Use Google's built-in security features
270 | - Enable "Find My Device" for device recovery
271 | - Consider using a secure lock screen (PIN, pattern, or biometric)
272 | - Keep Android updated with security patches
273 | - Use Google Play Protect for app scanning
274 | 
275 | ### Desktop Devices
276 | 
277 | - Use full-disk encryption (FileVault for Mac, BitLocker for Windows)
278 | - Keep your operating system updated
279 | - Use reputable antivirus software
280 | - Be cautious about browser extensions
281 | - Log out when not using SilentRelay
282 | 
283 | ## Network Security Tips
284 | 
285 | ### Wi-Fi Security
286 | 
287 | - Avoid public Wi-Fi for sensitive conversations
288 | - Use a VPN if you must use public Wi-Fi
289 | - Forget public networks after using them
290 | - Use WPA3 encryption for your home Wi-Fi
291 | 
292 | ### Mobile Data
293 | 
294 | - Mobile data is generally more secure than public Wi-Fi
295 | - Consider using mobile data for sensitive communications
296 | - Be aware of potential SIM swap attacks
297 | 
298 | ## Support and Reporting
299 | 
300 | ### How to Contact Security Support
301 | 
302 | **For security issues:** `security@silentrelay.com.au`
303 | **For general support:** `support@silentrelay.com.au`
304 | **Emergency security issues:** `security-emergency@silentrelay.com.au` (24/7)
305 | 
306 | ### What to Report
307 | 
308 | - Suspicious messages or contacts
309 | - Unexpected security warnings
310 | - Potential account compromises
311 | - Bugs or vulnerabilities you discover
312 | - Any unusual app behavior
313 | 
314 | ### Our Commitments to You
315 | 
316 | - **Transparency:** We'll be honest about security issues
317 | - **Responsiveness:** We'll address security reports promptly
318 | - **Privacy:** We'll protect your personal information
319 | - **Continuous Improvement:** We're always working to make our security better
320 | 
321 | ## Keeping Your App Secure
322 | 
323 | ### Automatic Updates
324 | 
325 | - SilentRelay updates automatically
326 | - Updates include security improvements and bug fixes
327 | - No action required - updates happen seamlessly
328 | 
329 | ### What's in Updates
330 | 
331 | - Security vulnerability fixes
332 | - Performance improvements
333 | - New security features
334 | - Bug fixes
335 | - Compatibility enhancements
336 | 
337 | ## Additional Resources
338 | 
339 | ### Security Education
340 | 
341 | - [Electronic Frontier Foundation - Surveillance Self-Defense](https://ssd.eff.org/)
342 | - [Center for Internet Security - Security Tips](https://www.cisecurity.org/insights/white-papers)
343 | - [National Cyber Security Alliance](https://staysafeonline.org/)
344 | 
345 | ### Our Security Documentation
346 | 
347 | - [Security Architecture Overview](SECURITY.md)
348 | - [Threat Model](THREAT_MODEL.md)
349 | - [Security Certification Plan](SECURITY_CERTIFICATION_PLAN.md)
350 | 
351 | ## Your Security Checklist
352 | 
353 | ### Initial Setup
354 | 
355 | - [ ] Set up a strong PIN
356 | - [ ] Generate and securely store your recovery key
357 | - [ ] Enable biometric authentication if available
358 | - [ ] Verify security codes with important contacts
359 | - [ ] Review and adjust privacy settings
360 | 
361 | ### Regular Maintenance
362 | 
363 | - [ ] Review your security settings monthly
364 | - [ ] Verify important contacts periodically
365 | - [ ] Check for unusual devices or sessions
366 | - [ ] Update your recovery key storage if needed
367 | - [ ] Review privacy settings occasionally
368 | 
369 | ### If You Suspect an Issue
370 | 
371 | - [ ] Change your PIN immediately
372 | - [ ] Review recent activity
373 | - [ ] Report suspicious messages
374 | - [ ] Contact support if needed
375 | - [ ] Consider generating a new recovery key
376 | 
377 | ## Security Myths vs Facts
378 | 
379 | ### Myth: "End-to-end encryption means I'm completely anonymous"
380 | **Fact:** End-to-end encryption protects message content, but metadata (who you communicate with, when, how often) may still be visible. Use additional privacy features for more protection.
381 | 
382 | ### Myth: "SilentRelay can read my messages"
383 | **Fact:** With end-to-end encryption, we cannot read your message content. The encryption keys are only on your devices.
384 | 
385 | ### Myth: "If I delete a message, it's gone forever"
386 | **Fact:** Deleted messages are removed from our servers, but recipients may have copies. For sensitive info, use disappearing messages.
387 | 
388 | ### Myth: "Security features make the app slower"
389 | **Fact:** Our encryption is optimized for performance. You may notice brief reconnections during key rotation, but overall performance impact is minimal.
390 | 
391 | ## Our Security Commitments
392 | 
393 | ### What We Promise
394 | 
395 | - **No Backdoors:** We don't build ways to access your messages
396 | - **Transparency:** We'll be open about our security practices
397 | - **Continuous Improvement:** We're always working to make security better
398 | - **Responsible Disclosure:** We'll fix vulnerabilities promptly and responsibly
399 | - **User Control:** You control your security settings
400 | 
401 | ### What We Don't Do
402 | 
403 | - We don't sell your data
404 | - We don't read your messages
405 | - We don't share your contacts
406 | - We don't track your location
407 | - We don't keep logs of your message content
408 | 
409 | ## Security Glossary
410 | 
411 | **End-to-End Encryption (E2EE):** Messages are encrypted on the sender's device and only decrypted on the recipient's device.
412 | 
413 | **Forward Secrecy:** If an encryption key is compromised, it doesn't affect past communications.
414 | 
415 | **Man-in-the-Middle Attack:** When an attacker intercepts communication between two parties.
416 | 
417 | **Public Key Cryptography:** Uses a pair of keys - one public (can be shared) and one private (must be kept secret).
418 | 
419 | **Security Code:** A unique code that helps verify you're communicating with the intended device.
420 | 
421 | **Recovery Key:** A secret phrase that allows you to restore your account if you lose access to your device.
422 | 
423 | ## Contact Information
424 | 
425 | **Security Team:** `security@silentrelay.com.au`
426 | **Support Team:** `support@silentrelay.com.au`
427 | **Documentation Issues:** `docs@silentrelay.com.au`
428 | 
429 | **Last Updated:** 2025-12-07
430 | **Next Review:** 2026-01-15
431 | **Documentation Maintainer:** User Education Team
