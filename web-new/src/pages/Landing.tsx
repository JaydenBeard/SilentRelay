import { Link } from 'react-router-dom';
import { Button } from '@/components/ui/button';
import {
  Shield,
  Lock,
  Zap,
  Eye,
  EyeOff,
  Server,
  Key,
  RefreshCw,
  CheckCircle2,
  Check,
  XCircle,
  ArrowRight,
  Code2,
  FileCode,
  Github,
  ChevronDown,
  User,
} from 'lucide-react';
import { useState, useEffect } from 'react';
import { useScrollAnimations } from '@/hooks/useScrollAnimation';

function ScrollArrow({ targetId }: { targetId: string }) {
  const scrollToSection = () => {
    const element = document.getElementById(targetId);
    if (element) {
      element.scrollIntoView({ behavior: 'smooth' });
    }
  };

  return (
    <button
      onClick={scrollToSection}
      className="absolute bottom-8 left-1/2 -translate-x-1/2 text-foreground-muted hover:text-primary transition-colors scroll-arrow"
      aria-label="Scroll to next section"
    >
      <ChevronDown className="h-8 w-8" />
    </button>
  );
}

export default function LandingPage() {
  // Initialize scroll animations for all .animate-on-scroll elements
  useScrollAnimations();

  return (
    <div className="min-h-screen bg-background snap-y snap-proximity overflow-y-scroll h-screen w-full max-w-[100vw] overflow-x-hidden">
      {/* Navigation */}
      <nav className="sticky top-0 z-50 bg-background/80 backdrop-blur-lg border-b border-border">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Shield className="h-8 w-8 text-primary logo-glitch" />
            <span className="text-xl font-semibold logo-text-glitch">SilentRelay</span>
          </div>
          <div className="hidden md:flex items-center gap-8">
            <a href="#security" className="text-sm text-foreground-secondary hover:text-foreground transition-colors">
              Security
            </a>
            <a href="#how-it-works" className="text-sm text-foreground-secondary hover:text-foreground transition-colors">
              How It Works
            </a>
            <a href="#compare" className="text-sm text-foreground-secondary hover:text-foreground transition-colors">
              Compare
            </a>
            <a href="#faq" className="text-sm text-foreground-secondary hover:text-foreground transition-colors">
              FAQ
            </a>
          </div>
          <Link to="/auth">
            <Button variant="outline" size="sm">
              Sign In
            </Button>
          </Link>
        </div>
      </nav>

      {/* Hero Section */}
      <header className="relative overflow-hidden min-h-screen snap-start flex items-center px-6 py-24 md:py-32 lg:py-40">
        <div className="absolute inset-0 bg-gradient-to-br from-primary/10 via-background to-accent/5" />
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,_var(--tw-gradient-stops))] from-primary/5 via-transparent to-transparent" />

        <div className="relative z-10 max-w-7xl mx-auto w-full">
          <div className="grid lg:grid-cols-2 gap-12 lg:gap-16 items-center">
            {/* Left column - Hero content */}
            <div className="max-w-xl hero-fade-left">
              <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary/10 text-primary text-sm font-medium mb-6 hero-fade-up">
                <Lock className="h-4 w-4" />
                End-to-End Encrypted by Default
              </div>
              <h1 className="text-4xl md:text-5xl lg:text-6xl font-bold tracking-tight leading-tight hero-fade-up-delay-1">
                Your conversations.
                <br />
                <span className="gradient-text">Truly private.</span>
              </h1>
              <p className="mt-6 text-lg md:text-xl text-foreground-secondary leading-relaxed hero-fade-up-delay-2">
                SilentRelay implements the Signal Protocol for military-grade encryption.
                We literally cannot read your messages - even if compelled by law.
              </p>
              <div className="mt-10 flex flex-col sm:flex-row gap-4 hero-fade-up-delay-3">
                <Link to="/auth">
                  <Button size="lg" className="w-full sm:w-auto text-base">
                    Start Messaging Securely
                    <ArrowRight className="ml-2 h-5 w-5" />
                  </Button>
                </Link>
                <a href="#security">
                  <Button variant="outline" size="lg" className="w-full sm:w-auto text-base">
                    See How It Works
                  </Button>
                </a>
              </div>

              {/* Trust indicators */}
              <div className="mt-12 flex flex-wrap gap-6 text-sm text-foreground-muted hero-fade-up-delay-4">
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>Open Source</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>No Ads, Ever</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>No Tracking</span>
                </div>
                <div className="flex items-center gap-2">
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>Zero Knowledge</span>
                </div>
              </div>
            </div>

            {/* Right column - Phone mockup */}
            <div className="hidden lg:flex justify-center lg:justify-end hero-fade-right">
              <PhoneMockup />
            </div>
          </div>
        </div>
        <ScrollArrow targetId="problem" />
      </header>

      {/* Problem Section */}
      <section id="problem" className="relative py-20 px-6 bg-background-secondary snap-start min-h-screen flex items-center">
        <div className="max-w-7xl mx-auto">
          <div className="max-w-3xl mx-auto text-center animate-on-scroll fade-up">
            <h2 className="text-3xl md:text-4xl font-bold mb-6">
              Most "secure" messaging apps aren't actually secure
            </h2>
            <p className="text-lg text-foreground-secondary leading-relaxed">
              Big tech companies claim to protect your privacy, but they still collect metadata,
              store your messages on their servers, and can access your conversations when convenient.
              Your data is their product.
            </p>
          </div>

          <div className="mt-16 grid md:grid-cols-3 gap-8">
            <div className="animate-on-scroll fade-up delay-100">
              <ProblemCard
                icon={<Eye className="h-6 w-6" />}
                title="Metadata Surveillance"
                description="They may not read your messages, but they know who you talk to, when, and how often. That's often enough."
              />
            </div>
            <div className="animate-on-scroll fade-up delay-200">
              <ProblemCard
                icon={<Server className="h-6 w-6" />}
                title="Server-Side Storage"
                description="Your messages sit on corporate servers, accessible to employees, hackers, and law enforcement."
              />
            </div>
            <div className="animate-on-scroll fade-up delay-300">
              <ProblemCard
                icon={<Key className="h-6 w-6" />}
                title="Key Management"
                description="When companies control your encryption keys, they can decrypt your messages whenever they choose."
              />
            </div>
          </div>
        </div>
        <ScrollArrow targetId="how-it-works" />
      </section>

      {/* How It Works */}
      <section id="how-it-works" className="relative py-24 px-6 bg-background-secondary snap-start min-h-screen flex items-center">
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-16 animate-on-scroll fade-up">
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              How SilentRelay protects your privacy
            </h2>
            <p className="text-lg text-foreground-secondary max-w-2xl mx-auto">
              Security shouldn't be complicated. Here's what happens behind the scenes.
            </p>
          </div>

          <div className="grid md:grid-cols-4 gap-8">
            <div className="animate-on-scroll fade-up delay-100">
              <StepCard
                number="1"
                title="Key Generation"
                description="When you sign up, your device generates unique cryptographic keys. These keys never leave your device."
              />
            </div>
            <div className="animate-on-scroll fade-up delay-200">
              <StepCard
                number="2"
                title="Key Exchange"
                description="Before your first message, devices perform a secure handshake to establish a shared secret without transmitting it."
              />
            </div>
            <div className="animate-on-scroll fade-up delay-300">
              <StepCard
                number="3"
                title="Message Encryption"
                description="Each message is encrypted on your device using a unique key. Only the recipient's device can decrypt it."
              />
            </div>
            <div className="animate-on-scroll fade-up delay-400">
              <StepCard
                number="4"
                title="Secure Relay"
                description="Our servers blindly relay encrypted data. We never see, store, or have access to your actual messages."
              />
            </div>
          </div>
        </div>
        <ScrollArrow targetId="security" />
      </section>

      {/* Security Deep Dive */}
      <section id="security" className="relative py-24 px-6 snap-start min-h-screen flex items-center">
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-16 animate-on-scroll fade-up">
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              Security that's mathematically provable
            </h2>
            <p className="text-lg text-foreground-secondary max-w-2xl mx-auto">
              SilentRelay uses the Signal Protocol—the same encryption used by journalists,
              activists, and security researchers worldwide.
            </p>
          </div>

          <div className="grid lg:grid-cols-2 gap-12 items-stretch">
            {/* Left column - Technical details */}
            <div className="flex flex-col space-y-8">
              <SecurityFeature
                icon={<Lock className="h-6 w-6" />}
                title="X3DH Key Agreement"
                description="Extended Triple Diffie-Hellman establishes a shared secret between two parties without ever transmitting the secret itself. Even if our servers are compromised, your messages remain encrypted."
                technical="Combines identity keys, signed pre-keys, and one-time pre-keys for asynchronous key exchange."
              />
              <SecurityFeature
                icon={<RefreshCw className="h-6 w-6" />}
                title="Double Ratchet Algorithm"
                description="Every single message uses a unique encryption key. If any key is compromised, it cannot decrypt past or future messages."
                technical="Combines a root chain, sending chain, and receiving chain with HKDF key derivation."
              />
              <SecurityFeature
                icon={<Zap className="h-6 w-6" />}
                title="Perfect Forward Secrecy"
                description="Encryption keys are deleted immediately after use. There's no master key that can decrypt your message history."
                technical="Session keys are derived using ephemeral key pairs that are discarded after each message."
              />
              <SecurityFeature
                icon={<EyeOff className="h-6 w-6" />}
                title="Zero-Knowledge Server"
                description="Our servers cannot see who you're talking to, what you're saying, or even how many messages you've sent."
                technical="All encryption happens client-side. Server only sees encrypted blobs with no metadata."
              />
            </div>

            {/* Right column - Encryption Flow Diagram */}
            <div className="bg-background-secondary/50 rounded-2xl border border-border p-8">
              <h3 className="text-lg font-semibold text-center mb-8">End-to-End Encryption Flow</h3>

              <div className="max-w-md mx-auto space-y-4">
                {/* Sender */}
                <div className="flex flex-col items-center animate-on-scroll fade-up relative z-10" style={{ animationDelay: '0.1s' }}>
                  <div className="w-14 h-14 rounded-full bg-primary/20 border-2 border-primary flex items-center justify-center mb-2 pulse-zoom">
                    <User className="h-7 w-7 text-primary" />
                  </div>
                  <div className="text-sm font-medium">You</div>
                  <div className="text-xs text-foreground-muted mt-1">Send message:</div>
                  <div className="mt-2 px-4 py-2 rounded-lg bg-primary/10 text-foreground text-sm max-w-xs">
                    "Hey! Want to grab coffee tomorrow?"
                  </div>
                </div>

                {/* Arrow 1 + Encryption */}
                <div className="flex flex-col items-center gap-2 animate-on-scroll fade-up relative z-10" style={{ animationDelay: '0.2s' }}>
                  <div className="w-0.5 h-6 bg-primary/20 flow-line flow-1" />
                  <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-primary/10">
                    <Lock className="h-4 w-4 text-primary" />
                    <span className="text-xs text-foreground-muted">Encrypted with AES-256-GCM</span>
                  </div>
                  <div className="w-0.5 h-6 bg-primary/20 flow-line flow-2" />
                </div>

                {/* Server + Encrypted Data */}
                <div className="flex flex-col items-center animate-on-scroll fade-up relative z-10" style={{ animationDelay: '0.3s' }}>
                  <div className="w-14 h-14 rounded-full bg-red-500/20 border-2 border-red-500 flex items-center justify-center mb-2 pulse-zoom">
                    <Shield className="h-7 w-7 text-red-500" />
                  </div>
                  <div className="text-sm font-medium">Our Server</div>
                  <div className="text-xs text-foreground-muted mt-1">Only sees encrypted data:</div>
                  <div className="mt-2 px-3 py-2 rounded-lg bg-red-500/10 text-foreground-muted font-mono text-xs max-w-xs break-all">
                    7f4e8b2a9c1d3f5e6a8b0c2d4e6f8a0b...
                  </div>
                  <div className="text-xs text-red-400 mt-2">✗ Cannot decrypt</div>
                </div>

                {/* Arrow 2 + Decryption */}
                <div className="flex flex-col items-center gap-2 animate-on-scroll fade-up relative z-10" style={{ animationDelay: '0.4s' }}>
                  <div className="w-0.5 h-6 bg-primary/20 flow-line flow-3" />
                  <div className="flex items-center gap-2 px-3 py-1.5 rounded-full bg-primary/10">
                    <Key className="h-4 w-4 text-primary" />
                    <span className="text-xs text-foreground-muted">Decrypted on device</span>
                  </div>
                  <div className="w-0.5 h-6 bg-primary/20 flow-line flow-4" />
                </div>

                {/* Recipient */}
                <div className="flex flex-col items-center animate-on-scroll fade-up relative z-10" style={{ animationDelay: '0.5s' }}>
                  <div className="w-14 h-14 rounded-full bg-primary/20 border-2 border-primary flex items-center justify-center mb-2 pulse-zoom">
                    <User className="h-7 w-7 text-primary" />
                  </div>
                  <div className="text-sm font-medium">Recipient</div>
                  <div className="text-xs text-foreground-muted mt-1">Receives message:</div>
                  <div className="mt-2 px-4 py-2 rounded-lg bg-primary/10 text-foreground text-sm max-w-xs">
                    "Hey! Want to grab coffee tomorrow?"
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <ScrollArrow targetId="compare" />
      </section>

      {/* Comparison Section */}
      <section id="compare" className="relative py-24 px-4 sm:px-6 snap-start min-h-screen flex items-center w-full max-w-[100vw]">
        <div className="w-full max-w-7xl mx-auto overflow-hidden">
          <div className="text-center mb-16 animate-on-scroll fade-up px-2">
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              How we compare
            </h2>
            <p className="text-base sm:text-lg text-foreground-secondary max-w-2xl mx-auto">
              Not all encrypted messaging is created equal. See how SilentRelay stacks up.
            </p>
          </div>

          <div className="overflow-x-auto pb-4 animate-on-scroll fade-up delay-200">
            <table className="w-full min-w-[600px]">
              <thead>
                <tr className="border-b border-border">
                  <th className="text-left py-4 px-4 font-medium">Feature</th>
                  <th className="text-center py-4 px-4">
                    <div className="flex items-center justify-center gap-2">
                      <Shield className="h-5 w-5 text-primary" />
                      <span className="font-semibold">SilentRelay</span>
                    </div>
                  </th>
                  <th className="text-center py-4 px-4 text-foreground-secondary font-normal">Signal</th>
                  <th className="text-center py-4 px-4 text-foreground-secondary font-normal">WhatsApp</th>
                  <th className="text-center py-4 px-4 text-foreground-secondary font-normal">Telegram</th>
                </tr>
              </thead>
              <tbody className="text-sm">
                <ComparisonRow
                  feature="End-to-End Encryption"
                  silentRelay={true}
                  signal={true}
                  whatsapp={true}
                  telegram="Partial"
                />
                <ComparisonRow
                  feature="Open Source (Client)"
                  silentRelay={true}
                  signal={true}
                  whatsapp={false}
                  telegram={true}
                />
                <ComparisonRow
                  feature="Open Source (Server)"
                  silentRelay={true}
                  signal={true}
                  whatsapp={false}
                  telegram={false}
                />
                <ComparisonRow
                  feature="No Metadata Collection"
                  silentRelay={true}
                  signal={true}
                  whatsapp={false}
                  telegram={false}
                />
                <ComparisonRow
                  feature="Perfect Forward Secrecy"
                  silentRelay={true}
                  signal={true}
                  whatsapp={true}
                  telegram="Secret Chats Only"
                />
                <ComparisonRow
                  feature="Self-Hostable"
                  silentRelay={true}
                  signal={false}
                  whatsapp={false}
                  telegram={false}
                />
                <ComparisonRow
                  feature="No Phone Number Required"
                  silentRelay="Coming Soon"
                  signal={false}
                  whatsapp={false}
                  telegram={false}
                />
                <ComparisonRow
                  feature="Decentralized"
                  silentRelay="Coming Soon"
                  signal={false}
                  whatsapp={false}
                  telegram={false}
                />
              </tbody>
            </table>
          </div>
        </div>
        <ScrollArrow targetId="docs" />
      </section>

      {/* Documentation Section */}
      <section id="docs" className="relative py-24 px-6 snap-start min-h-screen flex items-center">
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-16">
            <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary/10 text-primary text-sm font-medium mb-6">
              <FileCode className="h-4 w-4" />
              Comprehensive Documentation
            </div>
            <h2 className="text-3xl md:text-4xl font-bold mb-4">
              Everything you need to know
            </h2>
            <p className="text-lg text-foreground-secondary max-w-2xl mx-auto">
              From security architecture to deployment guides, our documentation covers every aspect of SilentRelay.
            </p>
          </div>

          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
            <DocCard
              title="Getting Started"
              description="Quick setup guide for development and deployment"
              links={[
                { name: "README", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/README.md" },
                { name: "Quick Start", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/QUICKSTART.md" },
                { name: "Environment Setup", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/ENVIRONMENT_SETUP.md" },
              ]}
            />
            <DocCard
              title="Security Architecture"
              description="Deep dive into our encryption and security measures"
              links={[
                { name: "Security Overview", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/SECURITY.md" },
                { name: "Threat Model", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/THREAT_MODEL.md" },
                { name: "Crypto Implementation", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/CRYPTO_IMPLEMENTATION.md" },
                { name: "Key Rotation", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/KEY_ROTATION_IMPLEMENTATION.md" },
              ]}
            />
            <DocCard
              title="API Documentation"
              description="Complete API reference for developers"
              links={[
                { name: "API Index", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/API_DOCUMENTATION_INDEX.md" },
                { name: "Authentication", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/API_AUTHENTICATION.md" },
                { name: "Messages API", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/API_MESSAGES.md" },
                { name: "WebSocket API", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/API_WEBSOCKET.md" },
                { name: "Device Management", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/API_DEVICES.md" },
              ]}
            />
            <DocCard
              title="Deployment & Operations"
              description="Production deployment and maintenance guides"
              links={[
                { name: "Deployment Guide", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/DEPLOY.md" },
                { name: "Monitoring Setup", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/MONITORING_SETUP_GUIDE.md" },
                { name: "Backup Strategy", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/BACKUP_STRATEGY_GUIDE.md" },
                { name: "Maintenance", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/MAINTENANCE_PROCEDURES.md" },
              ]}
            />
            <DocCard
              title="Security Operations"
              description="Incident response, auditing, and compliance"
              links={[
                { name: "Incident Response", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/INCIDENT_RESPONSE_PLAYBOOK.md" },
                { name: "Security Testing", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/SECURITY_TESTING_PROCEDURES.md" },
                { name: "Audit Logging", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/AUDIT_LOGGING.md" },
                { name: "Bug Bounty", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/BUG_BOUNTY.md" },
              ]}
            />
            <DocCard
              title="Advanced Topics"
              description="Post-quantum crypto, supply chain security, and more"
              links={[
                { name: "Post-Quantum Migration", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/POST_QUANTUM_MIGRATION_PLAN.md" },
                { name: "Supply Chain Security", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/SUPPLY_CHAIN_SECURITY.md" },
                { name: "Honeypot System", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/HONEYPOT_SYSTEM_DOCUMENTATION.md" },
                { name: "Intrusion Detection", url: "https://github.com/JaydenBeard/SilentRelay/blob/main/docs/INTRUSION_DETECTION_SYSTEM.md" },
              ]}
            />
          </div>
        </div>
        <ScrollArrow targetId="opensource" />
      </section>

      {/* Open Source Section */}
      <section id="opensource" className="relative py-24 px-6 bg-background-secondary snap-start min-h-screen flex items-center">
        <div className="max-w-7xl mx-auto">
          <div className="grid lg:grid-cols-2 gap-12 items-center">
            <div>
              <div className="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary/10 text-primary text-sm font-medium mb-6">
                <Code2 className="h-4 w-4" />
                100% Open Source
              </div>
              <h2 className="text-3xl md:text-4xl font-bold mb-6">
                Don't trust us. Verify.
              </h2>
              <p className="text-lg text-foreground-secondary leading-relaxed mb-6">
                Our entire codebase - client and server - is publicly available. Security researchers,
                cryptographers, and anyone can audit our implementation. We believe security through
                transparency beats security through obscurity every time.
              </p>
              <div className="flex flex-col sm:flex-row gap-4">
                <a href="https://github.com/JaydenBeard/SilentRelay" target="_blank" rel="noopener noreferrer">
                  <Button variant="outline" className="inline-flex items-center gap-2">
                    <Github className="h-4 w-4" />
                    View Source Code
                  </Button>
                </a>
                <a href="https://github.com/JaydenBeard/SilentRelay/blob/main/docs/SECURITY_DOCUMENTATION_INDEX.md" target="_blank" rel="noopener noreferrer">
                  <Button variant="ghost" className="inline-flex items-center gap-2">
                    <FileCode className="h-4 w-4" />
                    Security Documentation
                  </Button>
                </a>
              </div>
            </div>

            {/* Encryption Demo with attention-grabbing elements */}
            <div className="relative">
              {/* "Try it" badge at corner of demo */}
              <div className="absolute -top-3 -right-3 z-10">
                <span className="text-sm font-medium bg-primary px-3 py-1.5 rounded-full shadow-lg shadow-primary/30 flex items-center gap-1.5 text-primary-foreground">
                  ✨ Try it yourself
                </span>
              </div>

              {/* Glow effect behind demo */}
              <div className="absolute inset-0 bg-gradient-to-r from-primary/20 via-cyan-500/20 to-primary/20 blur-xl opacity-50 rounded-3xl" />

              {/* The demo itself */}
              <div className="relative">
                <InteractiveEncryptionDemo />
              </div>
            </div>
          </div>
        </div>
        <ScrollArrow targetId="faq" />
      </section>

      {/* FAQ Section */}
      <FAQSection />

      {/* Final CTA */}
      <section id="cta" className="relative py-24 px-6 bg-gradient-to-br from-primary/10 via-background to-accent/5 snap-start min-h-screen flex items-center">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-3xl md:text-4xl font-bold mb-6">
            Ready for truly private messaging?
          </h2>
          <p className="text-lg text-foreground-secondary mb-8">
            Join thousands who've switched to SilentRelay for conversations that stay private.
            No ads. No tracking. No compromises.
          </p>
          <Link to="/auth">
            <Button size="lg" className="text-base">
              Create Your Free Account
              <ArrowRight className="ml-2 h-5 w-5" />
            </Button>
          </Link>
          <p className="mt-6 text-sm text-foreground-muted">
            No credit card required. Start messaging in under 60 seconds.
          </p>
        </div>
        <button
          onClick={() => {
            const mainContainer = document.querySelector('.min-h-screen.overflow-y-scroll');
            if (mainContainer) {
              mainContainer.scrollTo({
                top: mainContainer.scrollHeight,
                behavior: 'smooth'
              });
            }
          }}
          className="absolute bottom-12 left-1/2 -translate-x-1/2 flex flex-col items-center gap-1 scroll-arrow text-foreground-muted hover:text-primary transition-colors"
          aria-label="Scroll to footer"
        >
          <span className="text-xs">Footer</span>
          <ChevronDown className="h-8 w-8" />
        </button>
      </section>

      {/* Footer */}
      <footer id="footer" className="py-12 px-6 border-t border-border">
        <div className="max-w-7xl mx-auto">
          <div className="grid md:grid-cols-4 gap-8 mb-8">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <Shield className="h-6 w-6 text-primary" />
                <span className="font-semibold">SilentRelay</span>
              </div>
              <p className="text-sm text-foreground-muted">
                Private messaging built on the Signal Protocol. Open source and auditable.
              </p>
            </div>
            <div>
              <h4 className="font-medium mb-4">Product</h4>
              <ul className="space-y-2 text-sm text-foreground-secondary">
                <li><a href="#security" className="hover:text-foreground transition-colors">Security</a></li>
                <li><a href="#how-it-works" className="hover:text-foreground transition-colors">How It Works</a></li>
                <li><a href="#compare" className="hover:text-foreground transition-colors">Compare</a></li>
                <li><a href="#faq" className="hover:text-foreground transition-colors">FAQ</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-medium mb-4">Developers</h4>
              <ul className="space-y-2 text-sm text-foreground-secondary">
                <li><a href="https://github.com/JaydenBeard/SilentRelay" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">GitHub Repository</a></li>
                <li><a href="https://github.com/JaydenBeard/SilentRelay/blob/main/README.md" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">Getting Started</a></li>
                <li><a href="https://github.com/JaydenBeard/SilentRelay/blob/main/docs/API_DOCUMENTATION_INDEX.md" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">API Documentation</a></li>
                <li><a href="https://github.com/JaydenBeard/SilentRelay/blob/main/DEPLOY.md" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">Self-Hosting Guide</a></li>
                <li><a href="https://github.com/JaydenBeard/SilentRelay/blob/main/docs/SECURITY.md" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">Security Architecture</a></li>
                <li><a href="https://github.com/JaydenBeard/SilentRelay/blob/main/docs/THREAT_MODEL.md" target="_blank" rel="noopener noreferrer" className="hover:text-foreground transition-colors">Threat Model</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-medium mb-4">Legal</h4>
              <ul className="space-y-2 text-sm text-foreground-secondary">
                <li><Link to="/privacy" className="hover:text-foreground transition-colors">Privacy Policy</Link></li>
                <li><Link to="/terms" className="hover:text-foreground transition-colors">Terms of Service</Link></li>
                <li><Link to="/security" className="hover:text-foreground transition-colors">Security Policy</Link></li>
              </ul>
            </div>
          </div>
          <div className="pt-8 border-t border-border flex flex-col md:flex-row items-center justify-between gap-4 text-sm text-foreground-muted">
            <p>&copy; {new Date().getFullYear()} SilentRelay. All rights reserved.</p>
            <p>Built with privacy as the foundation, not an afterthought.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}

function ProblemCard({
  icon,
  title,
  description,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
}) {
  return (
    <div className="h-full p-6 rounded-xl bg-background border border-border flex flex-col group hover:-translate-y-1 hover:shadow-lg hover:shadow-red-500/5 hover:border-red-500/30 transition-all duration-300 cursor-default">
      <div className="w-12 h-12 rounded-lg bg-red-500/10 flex items-center justify-center text-red-400 mb-4 flex-shrink-0 group-hover:animate-pulse group-hover:bg-red-500/20 transition-colors duration-300">
        {icon}
      </div>
      <h3 className="text-lg font-semibold mb-2">{title}</h3>
      <p className="text-foreground-secondary text-sm leading-relaxed flex-1">{description}</p>
    </div>
  );
}

function SecurityFeature({
  icon,
  title,
  description,
  technical,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  technical: string;
}) {
  return (
    <div className="p-6 rounded-xl bg-background-secondary border border-border">
      <div className="flex items-start gap-4">
        <div className="w-12 h-12 rounded-lg bg-primary/10 flex-shrink-0 flex items-center justify-center text-primary">
          {icon}
        </div>
        <div>
          <h3 className="text-lg font-semibold mb-2">{title}</h3>
          <p className="text-foreground-secondary text-sm leading-relaxed mb-3">{description}</p>
          <p className="text-xs text-foreground-muted font-mono bg-background rounded px-2 py-1 inline-block">
            {technical}
          </p>
        </div>
      </div>
    </div>
  );
}

function StepCard({
  number,
  title,
  description,
}: {
  number: string;
  title: string;
  description: string;
}) {
  return (
    <div className="relative">
      <div className="text-6xl font-bold text-primary/10 absolute -top-2 -left-2">{number}</div>
      <div className="relative pt-8 pl-6">
        <h3 className="text-lg font-semibold mb-2">{title}</h3>
        <p className="text-foreground-secondary text-sm leading-relaxed">{description}</p>
      </div>
    </div>
  );
}

function ComparisonRow({
  feature,
  silentRelay,
  signal,
  whatsapp,
  telegram,
}: {
  feature: string;
  silentRelay: boolean | string;
  signal: boolean | string;
  whatsapp: boolean | string;
  telegram: boolean | string;
}) {
  const renderValue = (value: boolean | string) => {
    if (value === true) {
      return <CheckCircle2 className="h-5 w-5 text-green-500 mx-auto" />;
    }
    if (value === false) {
      return <XCircle className="h-5 w-5 text-red-400 mx-auto" />;
    }
    return <span className="text-yellow-500 text-xs">{value}</span>;
  };

  return (
    <tr className="border-b border-border/50">
      <td className="py-4 px-4 text-foreground-secondary">{feature}</td>
      <td className="py-4 px-4 text-center bg-primary/5">{renderValue(silentRelay)}</td>
      <td className="py-4 px-4 text-center">{renderValue(signal)}</td>
      <td className="py-4 px-4 text-center">{renderValue(whatsapp)}</td>
      <td className="py-4 px-4 text-center">{renderValue(telegram)}</td>
    </tr>
  );
}

function DocCard({
  title,
  description,
  links,
}: {
  title: string;
  description: string;
  links: Array<{ name: string; url: string }>;
}) {
  return (
    <div className="bg-background-secondary rounded-xl border border-border p-6 hover:border-primary/50 transition-colors">
      <h3 className="text-lg font-semibold mb-2">{title}</h3>
      <p className="text-foreground-secondary text-sm mb-4">{description}</p>
      <ul className="space-y-2">
        {links.map((link, index) => (
          <li key={index}>
            <a
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
              className="text-primary hover:text-primary/80 text-sm transition-colors flex items-center gap-1"
            >
              <FileCode className="h-3 w-3" />
              {link.name}
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}

function FAQSection() {
  const [openIndex, setOpenIndex] = useState<number | null>(null);

  const faqs = [
    {
      question: "Can SilentRelay read my messages?",
      answer: "No. Encryption happens entirely on your device before any data reaches our servers. We don't have the keys to decrypt your messages, and mathematically cannot access them. This isn't a policy—it's how the system is built."
    },
    {
      question: "What happens if SilentRelay is hacked?",
      answer: "Attackers would only get encrypted data they can't read. Without your device's private keys, the encrypted messages are indistinguishable from random noise. Even we can't decrypt them, so neither can attackers."
    },
    {
      question: "What if the government demands my messages?",
      answer: "We can only provide what we have: encrypted blobs that no one can decrypt. We don't have access to your messages, contact lists, or conversation metadata. We're designed this way specifically so we have nothing to hand over."
    },
    {
      question: "Why should I trust you over Signal?",
      answer: "Signal is excellent—we use their protocol! But Signal is centralized with no self-hosting option. SilentRelay is fully open source (client AND server) and can be self-hosted. For organizations that need complete control, that matters."
    },
    {
      question: "Is SilentRelay free?",
      answer: "Yes, SilentRelay is free for personal use. We're funded by optional donations and enterprise licensing for organizations that want self-hosted deployments with support."
    },
    {
      question: "How do I verify my messages are actually encrypted?",
      answer: "Check our source code—it's all open source. You can also verify encryption using our safety number feature, which lets you confirm you're talking to the right person without any middleman interference."
    }
  ];

  return (
    <section id="faq" className="relative py-24 px-6 snap-start min-h-screen flex items-center">
      <div className="max-w-3xl mx-auto w-full">
        <div className="text-center mb-16">
          <h2 className="text-3xl md:text-4xl font-bold mb-4">
            Frequently Asked Questions
          </h2>
        </div>

        <div className="space-y-4">
          {faqs.map((faq, index) => (
            <div key={index} className="border border-border rounded-lg overflow-hidden">
              <button
                onClick={() => setOpenIndex(openIndex === index ? null : index)}
                className="w-full flex items-center justify-between p-4 text-left hover:bg-background-secondary transition-colors"
              >
                <span className="font-medium">{faq.question}</span>
                <ChevronDown
                  className={`h-5 w-5 text-foreground-muted transition-transform duration-300 ${openIndex === index ? 'rotate-180' : ''
                    }`}
                />
              </button>
              <div
                className={`grid transition-all duration-300 ease-in-out ${openIndex === index ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'
                  }`}
              >
                <div className="overflow-hidden">
                  <div className="px-4 pb-4 text-foreground-secondary text-sm leading-relaxed">
                    {faq.answer}
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
      <ScrollArrow targetId="cta" />
    </section>
  );
}

/**
 * Phone mockup showing the SilentRelay chat interface with animated messages
 */
function PhoneMockup() {
  // Animation state - which messages are visible
  const [visibleMessages, setVisibleMessages] = useState(0);
  const [showTyping, setShowTyping] = useState(false);
  const [firstMessageRead, setFirstMessageRead] = useState(false);
  const [animationPhase, setAnimationPhase] = useState<'empty' | 'chat' | 'lock' | 'locked'>('empty');
  const [animationKey, setAnimationKey] = useState(0);

  useEffect(() => {
    // Stagger message appearances
    const timers: NodeJS.Timeout[] = [];

    // Start with empty/blank state
    setVisibleMessages(0);
    setShowTyping(false);
    setFirstMessageRead(false);
    setAnimationPhase('empty');

    // Transition to chat phase and show first message (after 300ms)
    timers.push(setTimeout(() => {
      setAnimationPhase('chat');
    }, 300));

    // Message 1: Incoming (after 600ms)
    timers.push(setTimeout(() => setVisibleMessages(1), 600));

    // Message 2: Outgoing (after 1600ms)
    timers.push(setTimeout(() => setVisibleMessages(2), 1600));

    // Message 2 gets read (after 2600ms)
    timers.push(setTimeout(() => setFirstMessageRead(true), 2600));

    // Show typing indicator (after 3100ms)
    timers.push(setTimeout(() => setShowTyping(true), 3100));

    // Message 3: Incoming (after 4100ms)
    timers.push(setTimeout(() => {
      setShowTyping(false);
      setVisibleMessages(3);
    }, 4100));

    // Message 4: Outgoing (after 5600ms)
    timers.push(setTimeout(() => setVisibleMessages(4), 5600));

    // Show typing indicator briefly (after 7100ms)
    timers.push(setTimeout(() => setShowTyping(true), 7100));

    // Fade to lock (after 8100ms) - skip scrolling, just fade
    timers.push(setTimeout(() => {
      setShowTyping(false);
      setAnimationPhase('lock');
    }, 8100));

    // Lock closes (after 8600ms)
    timers.push(setTimeout(() => setAnimationPhase('locked'), 8600));

    // Fade lock out to empty (after 11000ms)
    timers.push(setTimeout(() => setAnimationPhase('empty'), 11000));

    // Restart animation (after 12000ms) - brief pause on empty before loop
    timers.push(setTimeout(() => {
      setAnimationKey(k => k + 1);
    }, 12000));

    return () => timers.forEach(clearTimeout);
  }, [animationKey]);

  return (
    <div className="relative">
      {/* Glow effect behind phone */}
      <div className="absolute inset-0 blur-3xl opacity-30 bg-gradient-to-br from-primary via-primary/50 to-accent scale-110" />

      {/* Phone frame */}
      <div className="relative w-[280px] sm:w-[320px] bg-background-secondary rounded-[2.5rem] p-2 shadow-2xl border border-border/50">
        {/* Inner screen */}
        <div className="bg-background rounded-[2rem] overflow-hidden">
          {/* Status bar */}
          <div className="px-6 pt-3 pb-2 flex items-center justify-between text-xs text-foreground-muted">
            <span>9:41</span>
            <div className="flex items-center gap-1">
              <div className="w-4 h-2 border border-foreground-muted rounded-sm">
                <div className="w-3/4 h-full bg-green-500 rounded-sm" />
              </div>
            </div>
          </div>

          {/* App header */}
          <div className="px-4 py-3 border-b border-border flex items-center gap-3">
            <div className="flex items-center gap-2">
              <Shield className="h-5 w-5 text-primary" />
              <span className="font-semibold text-sm">SilentRelay</span>
            </div>
            <div className="ml-auto flex items-center gap-1">
              <div className="w-2 h-2 rounded-full bg-green-500" />
              <span className="text-xs text-foreground-muted">Secure</span>
            </div>
          </div>

          {/* Chat content */}
          <div className="h-[360px] sm:h-[420px] relative overflow-hidden bg-gradient-to-b from-background to-background-secondary/50">
            {/* Chat messages layer */}
            <div
              className={`absolute inset-0 p-3 space-y-3 transition-all duration-500 ${animationPhase !== 'chat'
                ? 'opacity-0'
                : ''
                }`}
            >
              {/* Incoming message 1 */}
              <div
                className={`flex gap-2 transition-all duration-500 ${visibleMessages >= 1 ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4'
                  }`}
              >
                <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white text-xs font-medium flex-shrink-0">
                  A
                </div>
                <div className="max-w-[75%]">
                  <div className="bg-background-tertiary rounded-2xl rounded-tl-sm px-3 py-2 text-sm">
                    Hey! How's the new encryption working?
                  </div>
                  <div className="flex items-center gap-1 mt-1 px-1">
                    <Lock className="h-3 w-3 text-primary" />
                    <span className="text-[10px] text-foreground-muted">End-to-end encrypted</span>
                  </div>
                </div>
              </div>

              {/* Outgoing message 1 */}
              <div
                className={`flex gap-2 justify-end transition-all duration-500 ${visibleMessages >= 2 ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4'
                  }`}
              >
                <div className="max-w-[75%]">
                  <div className="bg-primary text-primary-foreground rounded-2xl rounded-tr-sm px-3 py-2 text-sm">
                    It's amazing! 🔒 Zero-knowledge architecture means even they can't read this.
                  </div>
                  <div className="flex items-center gap-1 mt-1 px-1 justify-end">
                    <span
                      className={`text-[10px] transition-colors duration-300 ${firstMessageRead ? 'text-primary' : 'text-foreground-muted'
                        }`}
                    >
                      {firstMessageRead ? 'Read' : 'Delivered'}
                    </span>
                    <div
                      className={`w-3.5 h-3.5 rounded-full flex items-center justify-center transition-colors duration-300 ${firstMessageRead ? 'bg-primary' : 'bg-transparent border border-foreground-muted'
                        }`}
                    >
                      <Check
                        className={`h-2 w-2 transition-colors duration-300 ${firstMessageRead ? 'text-primary-foreground' : 'text-foreground-muted'
                          }`}
                        strokeWidth={3}
                      />
                    </div>
                  </div>
                </div>
              </div>

              {/* Incoming message 2 */}
              <div
                className={`flex gap-2 transition-all duration-500 ${visibleMessages >= 3 ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4'
                  }`}
              >
                <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white text-xs font-medium flex-shrink-0">
                  A
                </div>
                <div className="max-w-[75%]">
                  <div className="bg-background-tertiary rounded-2xl rounded-tl-sm px-3 py-2 text-sm">
                    Perfect forward secrecy too? 🤯
                  </div>
                </div>
              </div>

              {/* Outgoing message 2 */}
              <div
                className={`flex gap-2 justify-end transition-all duration-500 ${visibleMessages >= 4 ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4'
                  }`}
              >
                <div className="max-w-[75%]">
                  <div className="bg-primary text-primary-foreground rounded-2xl rounded-tr-sm px-3 py-2 text-sm">
                    Yep! Every message uses a unique key. If one is compromised, past and future messages stay safe.
                  </div>
                  <div className="flex items-center gap-1 mt-1 px-1 justify-end">
                    <span className="text-[10px] text-foreground-muted">Delivered</span>
                    <CheckCircle2 className="h-3 w-3 text-foreground-muted" />
                  </div>
                </div>
              </div>

              {/* Typing indicator */}
              <div
                className={`flex gap-2 transition-all duration-300 ${showTyping ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-4'
                  }`}
              >
                <div className="w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center text-white text-xs font-medium flex-shrink-0">
                  A
                </div>
                <div className="bg-background-tertiary rounded-2xl rounded-tl-sm px-4 py-3">
                  <div className="flex gap-1">
                    <div className="w-2 h-2 rounded-full bg-foreground-muted animate-bounce" style={{ animationDelay: '0ms' }} />
                    <div className="w-2 h-2 rounded-full bg-foreground-muted animate-bounce" style={{ animationDelay: '150ms' }} />
                    <div className="w-2 h-2 rounded-full bg-foreground-muted animate-bounce" style={{ animationDelay: '300ms' }} />
                  </div>
                </div>
              </div>
            </div>

            {/* Lock animation overlay */}
            <div
              className={`absolute inset-0 flex flex-col items-center justify-center transition-all duration-500 ${animationPhase === 'lock' || animationPhase === 'locked'
                ? 'opacity-100'
                : 'opacity-0 pointer-events-none'
                }`}
            >
              {/* Lock icon container */}
              <div className={`relative transition-transform duration-500 ${animationPhase === 'locked' ? 'scale-110' : 'scale-100'}`}>
                {/* Glow effect */}
                <div className={`absolute inset-0 blur-2xl transition-opacity duration-500 ${animationPhase === 'locked' ? 'opacity-60' : 'opacity-0'
                  } bg-primary rounded-full scale-150`} />

                {/* Lock icon */}
                <div className={`relative w-20 h-20 rounded-2xl flex items-center justify-center transition-all duration-300 ${animationPhase === 'locked'
                  ? 'bg-primary shadow-lg shadow-primary/50'
                  : 'bg-background-tertiary border-2 border-primary/50'
                  }`}>
                  <Lock className={`h-10 w-10 transition-all duration-300 ${animationPhase === 'locked'
                    ? 'text-primary-foreground'
                    : 'text-primary'
                    }`} />
                </div>
              </div>

              {/* Text */}
              <p className={`mt-4 text-sm font-medium transition-all duration-500 ${animationPhase === 'locked' ? 'opacity-100 translate-y-0' : 'opacity-0 translate-y-2'
                }`}>
                Your messages are <span className="text-primary">secure</span>
              </p>
            </div>
          </div>

          {/* Input bar */}
          <div className="px-3 py-3 border-t border-border bg-background flex items-center gap-2">
            <div className="flex-1 bg-background-secondary rounded-full px-4 py-2 text-sm text-foreground-muted">
              Message...
            </div>
            <div className="w-8 h-8 rounded-full bg-primary flex items-center justify-center">
              <ArrowRight className="h-4 w-4 text-primary-foreground" />
            </div>
          </div>

          {/* Home indicator */}
          <div className="flex justify-center py-2">
            <div className="w-32 h-1 bg-foreground/20 rounded-full" />
          </div>
        </div>
      </div>

      {/* Security info below phone */}
      <div className="flex items-center justify-center gap-4 mt-6 text-xs text-foreground-muted">
        <div className="flex items-center gap-1.5">
          <Shield className="h-3.5 w-3.5 text-primary" />
          <span>Signal Protocol</span>
        </div>
        <div className="w-1 h-1 rounded-full bg-foreground-muted/50" />
        <div className="flex items-center gap-1.5">
          <Lock className="h-3.5 w-3.5 text-primary" />
          <span>256-bit AES</span>
        </div>
      </div>
    </div>
  );
}

/**
 * Interactive encryption demo - users type a message and see it encrypted
 */
function InteractiveEncryptionDemo() {
  const [inputText, setInputText] = useState('Try typing a message here...');
  const [encryptedText, setEncryptedText] = useState('');
  const [isEncrypting, setIsEncrypting] = useState(false);

  // Simulate encryption with a visual effect
  const encryptMessage = (text: string) => {
    if (!text) {
      setEncryptedText('');
      return;
    }

    setIsEncrypting(true);

    // Generate fake encrypted output (looks like real ciphertext)
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';
    let encrypted = '';
    for (let i = 0; i < text.length * 2; i++) {
      encrypted += chars.charAt(Math.floor(Math.random() * chars.length));
    }

    // Add some structure to make it look like base64
    const formatted = encrypted.match(/.{1,64}/g)?.join('\n') || encrypted;

    setTimeout(() => {
      setEncryptedText(formatted);
      setIsEncrypting(false);
    }, 150);
  };

  useEffect(() => {
    encryptMessage(inputText);
  }, [inputText]);

  return (
    <div className="bg-background rounded-xl border border-border overflow-hidden">
      {/* Window chrome */}
      <div className="flex items-center gap-2 px-4 py-3 border-b border-border bg-background-secondary/50">
        <div className="w-3 h-3 rounded-full bg-red-500" />
        <div className="w-3 h-3 rounded-full bg-yellow-500" />
        <div className="w-3 h-3 rounded-full bg-green-500" />
        <span className="ml-2 text-xs text-foreground-muted">Live Encryption Demo</span>
      </div>

      <div className="p-6 space-y-4">
        {/* Input section */}
        <div>
          <label className="block text-xs text-foreground-muted mb-2 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-blue-500" />
            Your Message (plaintext)
          </label>
          <input
            type="text"
            value={inputText}
            onChange={(e) => setInputText(e.target.value)}
            className="w-full bg-background-secondary border border-border rounded-lg px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary transition-all"
            placeholder="Type something to encrypt..."
          />
        </div>

        {/* Encryption animation */}
        <div className="flex items-center justify-center py-2">
          <div className={`flex items-center gap-2 px-3 py-1.5 rounded-full text-xs ${isEncrypting ? 'bg-primary/20 text-primary' : 'bg-background-secondary text-foreground-muted'} transition-colors`}>
            <Lock className={`h-3 w-3 ${isEncrypting ? 'animate-pulse' : ''}`} />
            <span>{isEncrypting ? 'Encrypting with AES-256-GCM...' : 'End-to-end encrypted'}</span>
          </div>
        </div>

        {/* Output section */}
        <div>
          <label className="block text-xs text-foreground-muted mb-2 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-green-500" />
            Encrypted Output (ciphertext)
          </label>
          <div className="bg-background-secondary border border-border rounded-lg px-4 py-3 font-mono text-xs text-foreground-muted min-h-[80px] overflow-x-auto">
            {encryptedText ? (
              <pre className="whitespace-pre-wrap break-all">{encryptedText}</pre>
            ) : (
              <span className="text-foreground-muted/50">Encrypted output will appear here...</span>
            )}
          </div>
        </div>

        <p className="text-xs text-foreground-muted text-center">
          This is a visual demonstration. Real encryption uses the Signal Protocol.
        </p>
      </div>
    </div>
  );
}
