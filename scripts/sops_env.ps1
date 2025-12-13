# SOPS Environment Manager for Windows
# =====================================
#
# Prerequisites:
#   - sops installed (winget install Mozilla.SOPS)
#   - age installed (winget install FiloSottile.age)
#   - Age private key in ~/.config/sops/age/keys.txt
#
# Usage:
#   .\scripts\sops_env.ps1 encrypt    # Encrypt .env -> .env.encrypted
#   .\scripts\sops_env.ps1 decrypt    # Decrypt .env.encrypted -> .env
#   .\scripts\sops_env.ps1 view       # View decrypted contents
#

param(
    [Parameter(Position=0)]
    [ValidateSet("encrypt", "decrypt", "view", "help")]
    [string]$Command = "help"
)

$ErrorActionPreference = "Stop"
$ENV_FILE = ".env"
$ENCRYPTED_FILE = ".env.encrypted"
$TEMP_JSON = ".env.temp.json"

# Set the key file location
$env:SOPS_AGE_KEY_FILE = "$env:USERPROFILE\.config\sops\age\keys.txt"

function Convert-EnvToJson {
    $envContent = Get-Content $ENV_FILE | Where-Object { $_ -match "^[A-Z_]" }
    $json = @{}
    foreach ($line in $envContent) {
        if ($line -match "^([^=]+)=(.*)$") {
            $json[$Matches[1]] = $Matches[2]
        }
    }
    $json | ConvertTo-Json -Depth 10 | Out-File $TEMP_JSON -Encoding UTF8
}

function Convert-JsonToEnv {
    param([string]$JsonContent)
    
    $data = $JsonContent | ConvertFrom-Json
    $lines = @()
    $data.PSObject.Properties | ForEach-Object {
        if ($_.Name -notmatch "^sops") {
            $lines += "$($_.Name)=$($_.Value)"
        }
    }
    $lines | Sort-Object | Out-File $ENV_FILE -Encoding UTF8
}

function Encrypt-Env {
    if (-not (Test-Path $ENV_FILE)) {
        Write-Host "❌ Error: $ENV_FILE not found" -ForegroundColor Red
        exit 1
    }

    Write-Host "🔐 Encrypting $ENV_FILE with SOPS..." -ForegroundColor Cyan
    
    # Convert .env to JSON (SOPS doesn't handle comments in dotenv)
    Convert-EnvToJson
    
    # Encrypt
    sops --encrypt $TEMP_JSON | Out-File $ENCRYPTED_FILE -Encoding UTF8
    
    # Cleanup temp file
    Remove-Item $TEMP_JSON -Force
    
    Write-Host "✅ Encrypted to $ENCRYPTED_FILE" -ForegroundColor Green
    Write-Host ""
    Write-Host "You can now safely commit to Git:" -ForegroundColor Yellow
    Write-Host "  git add $ENCRYPTED_FILE .sops.yaml"
    Write-Host "  git commit -m 'chore: update encrypted environment'"
    Write-Host "  git push"
}

function Decrypt-Env {
    if (-not (Test-Path $ENCRYPTED_FILE)) {
        Write-Host "❌ Error: $ENCRYPTED_FILE not found" -ForegroundColor Red
        Write-Host "Make sure you've pulled the latest from Git."
        exit 1
    }

    if (-not (Test-Path $env:SOPS_AGE_KEY_FILE)) {
        Write-Host "❌ Error: Age private key not found at $env:SOPS_AGE_KEY_FILE" -ForegroundColor Red
        Write-Host "Copy your private key file to this location."
        exit 1
    }

    Write-Host "🔓 Decrypting $ENCRYPTED_FILE with SOPS..." -ForegroundColor Cyan
    
    # Decrypt
    $decrypted = sops --decrypt $ENCRYPTED_FILE
    
    # Convert JSON back to .env format
    Convert-JsonToEnv -JsonContent ($decrypted -join "`n")
    
    Write-Host "✅ Decrypted to $ENV_FILE" -ForegroundColor Green
    Write-Host ""
    Write-Host "Your environment is ready!" -ForegroundColor Green
}

function View-Env {
    if (-not (Test-Path $ENCRYPTED_FILE)) {
        Write-Host "❌ Error: $ENCRYPTED_FILE not found" -ForegroundColor Red
        exit 1
    }

    Write-Host "👀 Decrypted contents of $ENCRYPTED_FILE:" -ForegroundColor Cyan
    Write-Host "==================================" -ForegroundColor Gray
    sops --decrypt $ENCRYPTED_FILE | ConvertFrom-Json | 
        ForEach-Object { $_.PSObject.Properties } | 
        Where-Object { $_.Name -notmatch "^sops" } |
        Sort-Object Name |
        ForEach-Object { Write-Host "$($_.Name)=$($_.Value)" }
}

function Show-Help {
    Write-Host "SOPS Environment Manager" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\scripts\sops_env.ps1 {encrypt|decrypt|view|help}" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "  encrypt  - Encrypt .env to .env.encrypted (safe to commit)"
    Write-Host "  decrypt  - Decrypt .env.encrypted to .env (for server use)"
    Write-Host "  view     - View decrypted contents without writing to disk"
    Write-Host "  help     - Show this help message"
    Write-Host ""
    Write-Host "Setup on a new server:" -ForegroundColor Yellow
    Write-Host "  1. Copy your private key to ~/.config/sops/age/keys.txt"
    Write-Host "  2. Run: .\scripts\sops_env.ps1 decrypt"
}

switch ($Command) {
    "encrypt" { Encrypt-Env }
    "decrypt" { Decrypt-Env }
    "view"    { View-Env }
    "help"    { Show-Help }
    default   { Show-Help }
}
