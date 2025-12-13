import { Link } from 'react-router-dom';
import { Shield, ArrowLeft } from 'lucide-react';
import { useEffect } from 'react';

export default function PrivacyPolicy() {
  // Scroll to top on mount
  useEffect(() => {
    window.scrollTo(0, 0);
  }, []);

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border">
        <div className="max-w-4xl mx-auto px-6 py-4">
          <Link to="/" className="inline-flex items-center gap-2 text-foreground-secondary hover:text-foreground transition-colors">
            <ArrowLeft className="h-4 w-4" />
            Back to Home
          </Link>
        </div>
      </header>

      {/* Content */}
      <main className="max-w-4xl mx-auto px-6 py-12">
        <div className="flex items-center gap-3 mb-8">
          <Shield className="h-8 w-8 text-primary" />
          <h1 className="text-3xl font-bold">Privacy Policy</h1>
        </div>

        <p className="text-foreground-secondary mb-8">
          Last updated: {new Date().toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })}
        </p>

        <div className="prose prose-invert max-w-none space-y-8">
          <section>
            <h2 className="text-xl font-semibold mb-4">Our Privacy Promise</h2>
            <p className="text-foreground-secondary leading-relaxed">
              SilentRelay is built on a simple principle: <strong className="text-foreground">your conversations are yours alone</strong>.
              Unlike most messaging services, we've architected our system so that we physically cannot access your messages,
              even if we wanted to. This isn't just a policy—it's mathematics.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">What We Cannot See</h2>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              Due to our end-to-end encryption architecture using the Signal Protocol:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">Message content</strong> — All messages are encrypted on your device before transmission. We only see encrypted data that is mathematically impossible for us to decrypt.</li>
              <li><strong className="text-foreground">Who you're messaging</strong> — Our zero-knowledge architecture means we don't maintain conversation metadata.</li>
              <li><strong className="text-foreground">Your contact list</strong> — Contact information is stored locally on your device, not on our servers.</li>
              <li><strong className="text-foreground">Message history</strong> — We don't store decrypted messages. Once delivered, encrypted messages are deleted from our relay servers.</li>
              <li><strong className="text-foreground">Your encryption keys</strong> — Private keys are generated on your device and never transmitted to us.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">What We Do Collect</h2>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              To provide the service, we necessarily collect minimal information:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">Phone number</strong> — Used for account registration and verification. We hash phone numbers for lookups and don't share them with third parties.</li>
              <li><strong className="text-foreground">Public encryption keys</strong> — Your public keys are stored on our servers so other users can initiate encrypted conversations with you. These cannot be used to decrypt messages.</li>
              <li><strong className="text-foreground">Basic account information</strong> — Account creation date and last activity timestamp for account maintenance.</li>
              <li><strong className="text-foreground">Encrypted message blobs</strong> — Temporarily stored for delivery to offline recipients, then deleted. We cannot read these.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">How We Protect Your Data</h2>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">End-to-end encryption</strong> — All messages use the Signal Protocol (X3DH key agreement + Double Ratchet algorithm).</li>
              <li><strong className="text-foreground">Perfect forward secrecy</strong> — Each message uses a unique encryption key. Compromising one key doesn't affect other messages.</li>
              <li><strong className="text-foreground">Local key storage</strong> — Your private keys never leave your device.</li>
              <li><strong className="text-foreground">Encrypted file transfers</strong> — Files are encrypted client-side with AES-256-GCM before upload.</li>
              <li><strong className="text-foreground">No cloud backups by default</strong> — We don't backup your messages to cloud services.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Third-Party Services</h2>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              We use minimal third-party services:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">SMS verification</strong> — We use a third-party SMS provider to send verification codes. They receive your phone number for this purpose only.</li>
              <li><strong className="text-foreground">Infrastructure providers</strong> — Our servers run on standard cloud infrastructure. These providers have no access to your encrypted data.</li>
            </ul>
            <p className="text-foreground-secondary leading-relaxed mt-4">
              We do not use analytics services, advertising networks, or any tracking technologies.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Law Enforcement Requests</h2>
            <p className="text-foreground-secondary leading-relaxed">
              If we receive a lawful request for user data, we can only provide what we have: encrypted data that we cannot decrypt,
              phone numbers, and public keys. We cannot provide message content, contact lists, or conversation history because
              we don't have access to this information. Our architecture is specifically designed this way.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Data Retention</h2>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">Messages</strong> — Encrypted messages are stored temporarily for delivery (typically under 30 days), then permanently deleted.</li>
              <li><strong className="text-foreground">Account data</strong> — Retained while your account is active. Deleted within 30 days of account deletion.</li>
              <li><strong className="text-foreground">Public keys</strong> — Retained while your account is active to enable others to message you.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Your Rights</h2>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              You have the right to:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">Delete your account</strong> — You can delete your account at any time. This removes all data we hold about you.</li>
              <li><strong className="text-foreground">Export your data</strong> — Since messages are stored locally on your device, you already have your data.</li>
              <li><strong className="text-foreground">Verify our claims</strong> — Our source code is open source. You can audit our encryption implementation yourself.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Children's Privacy</h2>
            <p className="text-foreground-secondary leading-relaxed">
              SilentRelay is not intended for users under 13 years of age. We do not knowingly collect personal information
              from children under 13.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Changes to This Policy</h2>
            <p className="text-foreground-secondary leading-relaxed">
              We may update this privacy policy from time to time. We will notify you of any material changes by posting
              the new policy on this page and updating the "Last updated" date. We encourage you to review this policy periodically.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Contact Us</h2>
            <p className="text-foreground-secondary leading-relaxed">
              If you have questions about this privacy policy or our privacy practices, please contact us at{' '}
              <a href="mailto:privacy@silentrelay.com" className="text-primary hover:underline">
                privacy@silentrelay.com
              </a>
            </p>
          </section>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-border mt-16">
        <div className="max-w-4xl mx-auto px-6 py-8 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Shield className="h-5 w-5 text-primary" />
            <span className="font-medium">SilentRelay</span>
          </div>
          <div className="flex gap-6 text-sm text-foreground-secondary">
            <Link to="/terms" className="hover:text-foreground transition-colors">Terms</Link>
            <Link to="/security" className="hover:text-foreground transition-colors">Security</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
