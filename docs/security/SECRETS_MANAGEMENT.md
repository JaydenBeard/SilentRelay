# Secrets Management with SOPS

This document explains how to securely manage environment secrets using SOPS (Secrets OPerationS) with age encryption.

## Overview

We use Mozilla SOPS to encrypt our `.env` file before committing to Git. This allows:

- Secrets stored in Git (encrypted)
- Easy deployment to servers (just decrypt)
- Version control of secret changes
- No passwords to remember (key files)
- Secrets never visible on GitHub

## Prerequisites

### On Windows

```powershell
# Install SOPS and age
winget install Mozilla.SOPS FiloSottile.age
```

### On Linux/macOS

```bash
# macOS
brew install sops age

# Ubuntu/Debian
sudo apt install age
# Download SOPS from: https://github.com/mozilla/sops/releases
```

## File Locations

| File | Description | In Git? |
|------|-------------|---------|
| `.env` | Plaintext secrets (working file) | No |
| `.env.encrypted` | SOPS-encrypted secrets | Yes |
| `.sops.yaml` | SOPS configuration | Yes |
| `~/.config/sops/age/keys.txt` | Your private key | **Never!** |

## Initial Setup (One Time)

### 1. Generate Age Key Pair

```powershell
# Create key directory
mkdir -p ~/.config/sops/age

# Generate key pair
age-keygen -o ~/.config/sops/age/keys.txt
```

This creates:
- **Private key**: Stays on your machine(s)
- **Public key**: Goes in `.sops.yaml`

### 2. Configure SOPS

Update `.sops.yaml` with your public key:

```yaml
creation_rules:
  - path_regex: \.env.*$
    age: age1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## Daily Workflow

### Encrypting Secrets (After Updating .env)

```powershell
# Encrypt .env to .env.encrypted
.\scripts\sops_env.ps1 encrypt

# Commit the encrypted file
git add .env.encrypted
git commit -m "chore: update secrets"
git push
```

### Decrypting Secrets (On New Machine/Server)

```powershell
# Make sure you have the private key
# Location: ~/.config/sops/age/keys.txt

# Pull latest from Git
git pull

# Decrypt to .env
.\scripts\sops_env.ps1 decrypt
```

### Viewing Secrets (Without Writing to Disk)

```powershell
.\scripts\sops_env.ps1 view
```

## Server Deployment

### First-Time Server Setup

1. **Install prerequisites:**

   ```bash
   sudo apt update && sudo apt install -y age jq
   
   # Install SOPS
   wget https://github.com/getsops/sops/releases/download/v3.8.1/sops-v3.8.1.linux.amd64 -O /usr/local/bin/sops
   chmod +x /usr/local/bin/sops
   ```

2. **Copy your private key to the server:**

   ```bash
   # From your local machine
   scp ~/.config/sops/age/keys.txt user@server:~/.config/sops/age/keys.txt
   
   # Or manually create the file on the server
   mkdir -p ~/.config/sops/age
   nano ~/.config/sops/age/keys.txt
   # Paste your private key
   ```

3. **Clone the repository:**

   ```bash
   git clone git@github.com:YourOrg/your-repo.git /opt/app
   cd /opt/app
   ```

### Deploying with deploy.sh (Recommended)

The easiest way to deploy is using the included `deploy.sh` script:

```bash
# Full deploy: git pull + decrypt + docker build
./deploy.sh

# Deploy without git pull (use current code)
./deploy.sh --no-pull

# Only decrypt secrets (don't start Docker)
./deploy.sh --decrypt

# Check container status
./deploy.sh --status
```

### Manual Deployment

If you prefer manual steps:

```bash
# 1. Pull latest code
git pull origin main

# 2. Decrypt secrets
export SOPS_AGE_KEY_FILE=~/.config/sops/age/keys.txt
sops --decrypt --output-type json .env.encrypted | \
  jq -r 'to_entries | .[] | select(.key | startswith("sops") | not) | "\(.key)=\(.value)"' > .env

# 3. Build and start containers
docker compose up --build -d
```

### Updating Secrets on Server

```bash
# Option 1: Full redeploy
./deploy.sh

# Option 2: Just update secrets and restart
./deploy.sh --decrypt
docker compose restart
```

## Security Best Practices

### Private Key Protection

1. **Never commit** `keys.txt` to Git
2. **Backup** the key securely (password manager, encrypted USB, etc.)
3. **Limit access** - only deploy to servers that need it
4. **Rotate keys** if compromised (see Key Rotation below)

### Access Control

- Store the private key only on:
  - Your development machine
  - Production servers that need secrets
  - CI/CD systems (as a secret variable)

### Audit Trail

Since `.env.encrypted` is in Git, you get:
- Full history of secret changes
- Who made changes (via Git commits)
- When changes were made

## Key Rotation

If your private key is compromised:

### 1. Generate New Key Pair

```powershell
age-keygen -o ~/.config/sops/age/keys-new.txt
```

### 2. Re-encrypt with New Key

```powershell
# Decrypt with old key
$env:SOPS_AGE_KEY_FILE = "~/.config/sops/age/keys.txt"
.\scripts\sops_env.ps1 decrypt

# Update .sops.yaml with new public key
# Then encrypt with new key
$env:SOPS_AGE_KEY_FILE = "~/.config/sops/age/keys-new.txt"
.\scripts\sops_env.ps1 encrypt
```

### 3. Update All Servers

Deploy the new private key to all servers and re-decrypt.

### 4. Rotate All Secrets

Since the old key may be compromised, also rotate:
- Database passwords
- API keys
- JWT secrets
- etc.

Use `scripts/rotate_secrets.sh` to generate new values.

## Troubleshooting

### "Failed to get the data key"

**Cause:** Private key not found or incorrect.

**Fix:**
```powershell
# Check key file exists
Test-Path ~/.config/sops/age/keys.txt

# Set environment variable
$env:SOPS_AGE_KEY_FILE = "$env:USERPROFILE\.config\sops\age\keys.txt"
```

### "No matching creation rules found"

**Cause:** File doesn't match any pattern in `.sops.yaml`.

**Fix:** Update `.sops.yaml` to include your file pattern.

### "Invalid dotenv input line"

**Cause:** SOPS dotenv format doesn't support comments.

**Fix:** Use the `sops_env.ps1` script which converts to JSON format.

## CI/CD Integration

### GitHub Actions

```yaml
jobs:
  deploy:
    steps:
      - name: Decrypt secrets
        env:
          SOPS_AGE_KEY: ${{ secrets.SOPS_AGE_KEY }}
        run: |
          mkdir -p ~/.config/sops/age
          echo "$SOPS_AGE_KEY" > ~/.config/sops/age/keys.txt
          sops --decrypt .env.encrypted > .env
```

Store the entire content of `keys.txt` as a GitHub secret named `SOPS_AGE_KEY`.

## Files Reference

### .sops.yaml

```yaml
creation_rules:
  - path_regex: \.env.*$
    age: age106awgavqzmgca62zzjzjrnnq92flauujrncp60d77n62wmj209zqejre89
```

### Private Key Format

```
# created: 2025-12-08T15:42:30+10:00
# public key: age106awgavqzmgca62zzjzjrnnq92flauujrncp60d77n62wmj209zqejre89
AGE-SECRET-KEY-17TUK92KNFEJVHU8ZKYFV2SSMVSVET2WNDC84KNWL5UR7M7SZRR9QUJLULP89
```

## Resources

- [SOPS Documentation](https://github.com/mozilla/sops)
- [Age Encryption](https://github.com/FiloSottile/age)
- [SOPS with Age Tutorial](https://devops.datenkollektiv.de/using-sops-with-age-and-git-like-a-pro.html)
