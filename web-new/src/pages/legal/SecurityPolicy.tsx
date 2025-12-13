import { Link } from 'react-router-dom';
import { Shield, ArrowLeft, Lock, Key, RefreshCw, Server, Eye, FileCode, AlertTriangle } from 'lucide-react';
import { useEffect } from 'react';

export default function SecurityPolicy() {
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
          <h1 className="text-3xl font-bold">Security Policy</h1>
        </div>

        <p className="text-foreground-secondary mb-8">
          Last updated: {new Date().toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' })}
        </p>

        <div className="prose prose-invert max-w-none space-y-8">
          <section>
            <h2 className="text-xl font-semibold mb-4">Our Security Philosophy</h2>
            <p className="text-foreground-secondary leading-relaxed">
              Security isn't a feature we added—it's the foundation everything is built on. SilentRelay implements
              state-of-the-art cryptographic protocols to ensure your communications remain private. We believe in
              security through transparency: our code is open source, our protocols are well-documented, and our
              claims are verifiable.
            </p>
          </section>

          {/* Security Features Grid */}
          <section className="grid md:grid-cols-2 gap-6 my-12">
            <SecurityCard
              icon={<Lock className="h-6 w-6" />}
              title="End-to-End Encryption"
              description="All messages are encrypted on your device using the Signal Protocol before being transmitted. Only the intended recipient can decrypt them."
            />
            <SecurityCard
              icon={<Key className="h-6 w-6" />}
              title="X3DH Key Exchange"
              description="Extended Triple Diffie-Hellman enables secure key agreement even when recipients are offline, without compromising forward secrecy."
            />
            <SecurityCard
              icon={<RefreshCw className="h-6 w-6" />}
              title="Double Ratchet"
              description="Each message uses a unique encryption key. Keys are immediately discarded after use, providing perfect forward secrecy."
            />
            <SecurityCard
              icon={<Server className="h-6 w-6" />}
              title="Zero-Knowledge Architecture"
              description="Our servers only relay encrypted data. We have no technical capability to access message contents or metadata."
            />
            <SecurityCard
              icon={<Eye className="h-6 w-6" />}
              title="No Metadata Collection"
              description="We don't store who you talk to, when you talk, or how often. Conversation metadata never touches our servers."
            />
            <SecurityCard
              icon={<FileCode className="h-6 w-6" />}
              title="Open Source"
              description="Our entire codebase is publicly available for security audits. Verify our security claims yourself."
            />
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Cryptographic Specifications</h2>
            <div className="bg-background-secondary rounded-lg p-6 border border-border">
              <table className="w-full text-sm">
                <tbody>
                  <tr className="border-b border-border/50">
                    <td className="py-3 text-foreground-secondary">Key Agreement</td>
                    <td className="py-3 font-mono">X3DH (Extended Triple Diffie-Hellman)</td>
                  </tr>
                  <tr className="border-b border-border/50">
                    <td className="py-3 text-foreground-secondary">Message Encryption</td>
                    <td className="py-3 font-mono">Double Ratchet with AES-256-GCM</td>
                  </tr>
                  <tr className="border-b border-border/50">
                    <td className="py-3 text-foreground-secondary">Identity Keys</td>
                    <td className="py-3 font-mono">Curve25519 / Ed25519</td>
                  </tr>
                  <tr className="border-b border-border/50">
                    <td className="py-3 text-foreground-secondary">Key Derivation</td>
                    <td className="py-3 font-mono">HKDF-SHA256</td>
                  </tr>
                  <tr className="border-b border-border/50">
                    <td className="py-3 text-foreground-secondary">Message Authentication</td>
                    <td className="py-3 font-mono">HMAC-SHA256</td>
                  </tr>
                  <tr className="border-b border-border/50">
                    <td className="py-3 text-foreground-secondary">File Encryption</td>
                    <td className="py-3 font-mono">AES-256-GCM with random IV</td>
                  </tr>
                  <tr>
                    <td className="py-3 text-foreground-secondary">Transport Security</td>
                    <td className="py-3 font-mono">TLS 1.3</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">How Encryption Works</h2>

            <h3 className="text-lg font-medium mt-6 mb-3">Key Generation</h3>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              When you register, your device generates:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">Identity Key Pair</strong> — A long-term Curve25519 key pair that identifies your device</li>
              <li><strong className="text-foreground">Signed Pre-Key</strong> — A medium-term key signed by your identity key, rotated periodically</li>
              <li><strong className="text-foreground">One-Time Pre-Keys</strong> — 100 ephemeral keys used once and discarded</li>
            </ul>
            <p className="text-foreground-secondary leading-relaxed mt-4">
              Only public keys are uploaded to our servers. Private keys never leave your device.
            </p>

            <h3 className="text-lg font-medium mt-6 mb-3">Session Establishment (X3DH)</h3>
            <p className="text-foreground-secondary leading-relaxed">
              Before sending your first message to someone, your device performs an X3DH handshake using the recipient's
              public keys. This establishes a shared secret without ever transmitting it. The handshake uses multiple
              Diffie-Hellman operations to provide forward secrecy and deniability.
            </p>

            <h3 className="text-lg font-medium mt-6 mb-3">Message Encryption (Double Ratchet)</h3>
            <p className="text-foreground-secondary leading-relaxed">
              Each message is encrypted with a unique key derived from a chain of ratcheting keys. After encryption,
              the key is discarded. This means that even if an attacker compromises a single message key, they cannot
              decrypt any other messages—past or future.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Safety Numbers</h2>
            <p className="text-foreground-secondary leading-relaxed">
              SilentRelay provides safety numbers that allow you to verify you're communicating with the intended person.
              Safety numbers are derived from both parties' identity keys and can be compared in person or over a trusted
              channel. If a safety number changes, it means the recipient's keys have changed (new device, reinstall, or
              potential attack).
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Infrastructure Security</h2>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">Transport encryption</strong> — All connections use TLS 1.3</li>
              <li><strong className="text-foreground">Server hardening</strong> — Production servers follow industry best practices for security configuration</li>
              <li><strong className="text-foreground">Access controls</strong> — Strict access controls limit who can access production systems</li>
              <li><strong className="text-foreground">Audit logging</strong> — Security-relevant events are logged (not message content)</li>
              <li><strong className="text-foreground">Regular updates</strong> — Systems are regularly updated to patch security vulnerabilities</li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Vulnerability Disclosure</h2>
            <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-6 mb-4">
              <div className="flex items-start gap-3">
                <AlertTriangle className="h-5 w-5 text-yellow-500 flex-shrink-0 mt-0.5" />
                <div>
                  <h3 className="font-medium text-yellow-500 mb-2">Report Security Vulnerabilities</h3>
                  <p className="text-foreground-secondary text-sm leading-relaxed">
                    If you discover a security vulnerability in SilentRelay, please report it responsibly.
                    Do not disclose the vulnerability publicly until we've had a chance to address it.
                  </p>
                </div>
              </div>
            </div>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              To report a security vulnerability:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li>Email <a href="mailto:security@silentrelay.com" className="text-primary hover:underline">security@silentrelay.com</a> with details</li>
              <li>Include steps to reproduce the vulnerability</li>
              <li>Allow reasonable time for us to address the issue before public disclosure</li>
              <li>We commit to acknowledging reports within 48 hours</li>
            </ul>
            <p className="text-foreground-secondary leading-relaxed mt-4">
              We appreciate security researchers who help us keep SilentRelay secure and will acknowledge
              contributions in our security advisories (unless you prefer to remain anonymous).
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Security Best Practices for Users</h2>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">Verify safety numbers</strong> — Confirm safety numbers with contacts for sensitive conversations</li>
              <li><strong className="text-foreground">Keep your device secure</strong> — Use device encryption and a strong passcode</li>
              <li><strong className="text-foreground">Keep the app updated</strong> — Updates often include security improvements</li>
              <li><strong className="text-foreground">Be cautious of phishing</strong> — We will never ask for your password or keys</li>
              <li><strong className="text-foreground">Report suspicious activity</strong> — Contact us if you notice unusual behavior</li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Limitations</h2>
            <p className="text-foreground-secondary leading-relaxed mb-4">
              While we implement strong security measures, no system is perfect. Please understand:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary">
              <li><strong className="text-foreground">Device security</strong> — If your device is compromised, an attacker may access decrypted messages</li>
              <li><strong className="text-foreground">Screenshots</strong> — Recipients can screenshot or photograph messages</li>
              <li><strong className="text-foreground">Recipient trust</strong> — Encryption protects messages in transit, not from recipients sharing content</li>
              <li><strong className="text-foreground">Metadata on your device</strong> — Message timestamps and contact information are stored locally</li>
            </ul>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Open Source Verification</h2>
            <p className="text-foreground-secondary leading-relaxed">
              Our commitment to security extends to transparency. Our source code is available for review:
            </p>
            <ul className="list-disc pl-6 space-y-2 text-foreground-secondary mt-4">
              <li><strong className="text-foreground">Client code</strong> — React/TypeScript frontend with Signal Protocol implementation</li>
              <li><strong className="text-foreground">Server code</strong> — Go backend that relays encrypted messages</li>
              <li><strong className="text-foreground">Cryptographic implementation</strong> — Built on well-audited libraries and standard protocols</li>
            </ul>
            <p className="text-foreground-secondary leading-relaxed mt-4">
              We encourage security researchers to audit our code and report any findings.
            </p>
          </section>

          <section>
            <h2 className="text-xl font-semibold mb-4">Contact</h2>
            <p className="text-foreground-secondary leading-relaxed">
              For security-related inquiries, contact{' '}
              <a href="mailto:security@silentrelay.com" className="text-primary hover:underline">
                security@silentrelay.com
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
            <Link to="/terms" className="hover:text-foreground transition-colors">Terms</Link>
          </div>
        </div>
      </footer>
    </div>
  );
}

function SecurityCard({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="p-6 rounded-xl bg-background-secondary border border-border">
      <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center text-primary mb-4">
        {icon}
      </div>
      <h3 className="font-semibold mb-2">{title}</h3>
      <p className="text-foreground-secondary text-sm leading-relaxed">{description}</p>
    </div>
  );
}
