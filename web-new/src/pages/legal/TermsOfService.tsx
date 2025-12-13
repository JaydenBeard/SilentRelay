import { Link } from 'react-router-dom';
import { Shield, ArrowLeft } from 'lucide-react';
import { useEffect } from 'react';

export default function TermsOfService() {
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
          <h1 className="text-3xl font-bold">Terms of Service</h1>
        </div>

        <p className="text-foreground-secondary mb-8">
          Last updated: {new Date().toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })}
        </p>

        <div className="prose prose-invert max-w-none space-y-8">
          <section>
            <h2 className="text-xl font-semibold mb-4">1. Agreement to Terms</h2>
            <p className="text-foreground-secondary leading-relaxed">
              By accessing or using SilentRelay ("the Service"), you agree to be bound by these Terms of Service ("Terms").
              If you disagree with any part of these terms, you may not access the Service.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">2. Description of Service</h2>
            <p className="text-foreground-secondary leading-relaxed">
              SilentRelay is an end-to-end encrypted messaging service. We provide a platform for private communication
              where messages are encrypted on your device before transmission. We act solely as a relay for encrypted data
              and cannot access the content of your communications.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">3. Account Registration</h2>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              To use SilentRelay, you must:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li>Provide a valid phone number for verification</li>
              <li>Be at least 13 years of age (or the minimum age in your jurisdiction)</li>
              <li>Not be prohibited from using the Service under applicable laws</li>
              <li>Not have been previously banned from the Service</li>
            </ul>
            <p className="text-foreground-secondary leading-relaxed mt-4">
              You are responsible for maintaining the security of your device and account credentials.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">4. Acceptable Use</h2>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              You agree not to use SilentRelay to:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li>Violate any applicable laws or regulations</li>
              <li>Harass, abuse, or threaten others</li>
              <li>Distribute malware, spam, or unwanted content</li>
              <li>Impersonate others or misrepresent your identity</li>
              <li>Interfere with or disrupt the Service or its infrastructure</li>
              <li>Attempt to gain unauthorized access to the Service or other users' accounts</li>
              <li>Distribute child sexual abuse material (CSAM) or exploit minors</li>
              <li>Engage in terrorism or incite violence</li>
            </ul>
            <p className="text-foreground-secondary leading-relaxed mt-4">
              While we cannot monitor encrypted message content, we may take action on accounts reported for abuse
              or that violate these terms through other means.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">5. Your Content and Data</h2>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              Due to end-to-end encryption:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">You own your data</strong> — Your messages and files belong to you. We cannot access them.</li>
              <li><strong className="text-foreground">You are responsible for backups</strong> — Since we don't store decrypted messages, you are responsible for backing up your data if desired.</li>
              <li><strong className="text-foreground">Loss of keys means loss of data</strong> — If you lose your device without backup, your message history cannot be recovered.</li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">6. Intellectual Property</h2>
            <p className="text-foreground-secondary leading-relaxed">
              The SilentRelay service, including its original content, features, and functionality, is owned by SilentRelay
              and is protected by international copyright, trademark, and other intellectual property laws.
              Our client and server code is open source under the applicable licenses specified in our repositories.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">7. Third-Party Services</h2>
            <p className="text-foreground-secondary leading-relaxed">
              The Service may contain links to third-party websites or services that are not owned or controlled by SilentRelay.
              We have no control over, and assume no responsibility for, the content, privacy policies, or practices of any
              third-party websites or services.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">8. Termination</h2>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              We may terminate or suspend your account immediately, without prior notice or liability, for any reason,
              including if you breach these Terms. Upon termination:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li>Your right to use the Service will immediately cease</li>
              <li>We will delete your account data within 30 days</li>
              <li>Local data on your device will remain unless you delete it</li>
            </ul>
            <p className="text-foreground-secondary leading-relaxed mt-4">
              You may delete your account at any time through the app settings.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">9. Disclaimer of Warranties</h2>
            <p className="text-foreground-secondary leading-relaxed">
              The Service is provided "AS IS" and "AS AVAILABLE" without warranties of any kind, either express or implied,
              including but not limited to implied warranties of merchantability, fitness for a particular purpose,
              non-infringement, or course of performance. We do not warrant that the Service will function uninterrupted,
              secure, or available at any particular time or location.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">10. Limitation of Liability</h2>
            <p className="text-foreground-secondary leading-relaxed">
              To the maximum extent permitted by law, in no event shall SilentRelay, its directors, employees, partners,
              agents, suppliers, or affiliates be liable for any indirect, incidental, special, consequential, or punitive
              damages, including without limitation, loss of profits, data, use, goodwill, or other intangible losses,
              resulting from your access to or use of (or inability to access or use) the Service.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">11. Indemnification</h2>
            <p className="text-foreground-secondary leading-relaxed">
              You agree to defend, indemnify, and hold harmless SilentRelay and its licensees, licensors, employees,
              contractors, agents, officers, and directors from and against any claims, damages, obligations, losses,
              liabilities, costs, or debt arising from your use of the Service or violation of these Terms.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">12. Governing Law</h2>
            <p className="text-foreground-secondary leading-relaxed">
              These Terms shall be governed by and construed in accordance with the laws of the jurisdiction in which
              SilentRelay operates, without regard to its conflict of law provisions.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">13. Changes to Terms</h2>
            <p className="text-foreground-secondary leading-relaxed">
              We reserve the right to modify or replace these Terms at any time. If a revision is material, we will
              provide at least 30 days' notice prior to any new terms taking effect. What constitutes a material change
              will be determined at our sole discretion. By continuing to access or use our Service after any revisions
              become effective, you agree to be bound by the revised terms.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">14. Contact Us</h2>
            <p className="text-foreground-secondary leading-relaxed">
              If you have any questions about these Terms, please contact us at{' '}
              <a href="mailto:legal@silentrelay.com" className="text-primary hover:underline">
                legal@silentrelay.com
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
            <Link to="/privacy" className="hover:text-foreground transition-colors">Privacy</Link>
            <Link to="/security" className="hover:text-foreground transition-colors">Security</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
