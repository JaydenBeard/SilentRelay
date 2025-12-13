#!/bin/bash
#
# SOPS Helper Script for .env file management
#
# Prerequisites:
#   - sops installed (winget install Mozilla.SOPS)
#   - age installed (winget install FiloSottile.age)
#   - Age private key in ~/.config/sops/age/keys.txt
#
# Usage:
#   ./scripts/sops_env.sh encrypt    # Encrypt .env -> .env.encrypted
#   ./scripts/sops_env.sh decrypt    # Decrypt .env.encrypted -> .env
#   ./scripts/sops_env.sh edit       # Edit encrypted file in-place
#   ./scripts/sops_env.sh view       # View decrypted contents
#

set -e

ENV_FILE=".env"
ENCRYPTED_FILE=".env.encrypted"

# Check if sops is installed
if ! command -v sops &> /dev/null; then
    echo "❌ Error: sops is not installed"
    echo "Install with: winget install Mozilla.SOPS"
    exit 1
fi

encrypt() {
    if [ ! -f "$ENV_FILE" ]; then
        echo "❌ Error: $ENV_FILE not found"
        exit 1
    fi

    echo "🔐 Encrypting $ENV_FILE with SOPS..."
    
    # SOPS encrypts dotenv files by treating them as key=value pairs
    sops --encrypt --input-type dotenv --output-type dotenv "$ENV_FILE" > "$ENCRYPTED_FILE"
    
    echo "✅ Encrypted to $ENCRYPTED_FILE"
    echo ""
    echo "You can now safely commit $ENCRYPTED_FILE to Git:"
    echo "  git add $ENCRYPTED_FILE .sops.yaml"
    echo "  git commit -m 'chore: update encrypted environment'"
    echo "  git push"
}

decrypt() {
    if [ ! -f "$ENCRYPTED_FILE" ]; then
        echo "❌ Error: $ENCRYPTED_FILE not found"
        echo "Make sure you've pulled the latest from Git."
        exit 1
    fi

    echo "🔓 Decrypting $ENCRYPTED_FILE with SOPS..."
    
    sops --decrypt --input-type dotenv --output-type dotenv "$ENCRYPTED_FILE" > "$ENV_FILE"
    
    echo "✅ Decrypted to $ENV_FILE"
    echo ""
    echo "Your environment is ready!"
}

edit() {
    if [ ! -f "$ENCRYPTED_FILE" ]; then
        echo "❌ Error: $ENCRYPTED_FILE not found"
        echo "Run 'encrypt' first to create the encrypted file."
        exit 1
    fi

    echo "📝 Opening $ENCRYPTED_FILE for editing..."
    echo "   (Changes will be automatically re-encrypted when you save)"
    
    sops --input-type dotenv --output-type dotenv "$ENCRYPTED_FILE"
}

view() {
    if [ ! -f "$ENCRYPTED_FILE" ]; then
        echo "❌ Error: $ENCRYPTED_FILE not found"
        exit 1
    fi

    echo "👀 Decrypted contents of $ENCRYPTED_FILE:"
    echo "=================================="
    sops --decrypt --input-type dotenv --output-type dotenv "$ENCRYPTED_FILE"
}

case "$1" in
    encrypt)
        encrypt
        ;;
    decrypt)
        decrypt
        ;;
    edit)
        edit
        ;;
    view)
        view
        ;;
    *)
        echo "SOPS Environment Manager"
        echo ""
        echo "Usage: $0 {encrypt|decrypt|edit|view}"
        echo ""
        echo "  encrypt  - Encrypt .env to .env.encrypted (safe to commit)"
        echo "  decrypt  - Decrypt .env.encrypted to .env (for server use)"
        echo "  edit     - Edit encrypted file in-place (auto re-encrypts)"
        echo "  view     - View decrypted contents without writing to disk"
        echo ""
        echo "Setup on a new server:"
        echo "  1. Copy your private key to ~/.config/sops/age/keys.txt"
        echo "  2. Run: ./scripts/sops_env.sh decrypt"
        exit 1
        ;;
esac
