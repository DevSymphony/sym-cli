# Symphony Windows Installer
# Run with: powershell -ExecutionPolicy Bypass -File install-windows.ps1

$ErrorActionPreference = "Stop"

# Set console encoding to UTF-8 for proper Korean display
$OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$PSDefaultParameterValues['*:Encoding'] = 'utf8'

# Set console code page to UTF-8 (65001)
chcp 65001 > $null

Write-Host "🎵 Symphony Windows Installer" -ForegroundColor Cyan
Write-Host ""

# Check if running as Administrator
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
$isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

# Determine installation directory
if ($isAdmin) {
    $installDir = "C:\Program Files\Symphony"
    Write-Host "✓ Running with administrator privileges" -ForegroundColor Green
} else {
    $installDir = "$env:LOCALAPPDATA\Symphony"
    Write-Host "⚠ Not running as administrator - installing to user directory" -ForegroundColor Yellow
}

Write-Host "📂 Installation directory: $installDir" -ForegroundColor Cyan
Write-Host ""

# Ask for confirmation
$confirmation = Read-Host "Continue with installation? (Y/n)"
if ($confirmation -eq 'n' -or $confirmation -eq 'N') {
    Write-Host "❌ Installation cancelled" -ForegroundColor Red
    exit 1
}

# Create installation directory
Write-Host "📁 Creating installation directory..."
New-Item -ItemType Directory -Force -Path $installDir | Out-Null

# Copy binary
$binaryName = "symphony-windows-amd64.exe"
$targetBinary = "$installDir\symphony.exe"

if (Test-Path $binaryName) {
    Write-Host "📋 Copying Symphony binary..."
    Copy-Item -Path $binaryName -Destination $targetBinary -Force
} else {
    Write-Host "❌ Error: $binaryName not found in current directory" -ForegroundColor Red
    Write-Host "   Please run this script from the dist/ directory" -ForegroundColor Yellow
    exit 1
}

# Add to PATH
Write-Host ""
Write-Host "🔧 Configuring PATH environment variable..." -ForegroundColor Cyan

$addToPath = Read-Host "Add Symphony to PATH? This allows you to run 'symphony' from anywhere (Y/n)"
if ($addToPath -ne 'n' -and $addToPath -ne 'N') {
    try {
        if ($isAdmin) {
            # System PATH (requires admin)
            $envTarget = [System.EnvironmentVariableTarget]::Machine
            $pathScope = "system"
        } else {
            # User PATH
            $envTarget = [System.EnvironmentVariableTarget]::User
            $pathScope = "user"
        }

        $currentPath = [Environment]::GetEnvironmentVariable("Path", $envTarget)

        if ($currentPath -notlike "*$installDir*") {
            $newPath = "$currentPath;$installDir"
            [Environment]::SetEnvironmentVariable("Path", $newPath, $envTarget)
            Write-Host "✓ Added to $pathScope PATH" -ForegroundColor Green

            # Update current session PATH
            $env:Path = "$env:Path;$installDir"

            Write-Host ""
            Write-Host "⚠ Note: You may need to restart your terminal for PATH changes to take effect" -ForegroundColor Yellow
        } else {
            Write-Host "✓ Already in PATH" -ForegroundColor Green
        }
    } catch {
        Write-Host "❌ Failed to update PATH: $_" -ForegroundColor Red
        Write-Host "   You can manually add '$installDir' to your PATH" -ForegroundColor Yellow
    }
} else {
    Write-Host "⊘ Skipped PATH configuration" -ForegroundColor Yellow
    Write-Host "  To use Symphony, either:" -ForegroundColor Yellow
    Write-Host "    1. Run: $targetBinary" -ForegroundColor Yellow
    Write-Host "    2. Manually add '$installDir' to your PATH" -ForegroundColor Yellow
}

# Verify installation
Write-Host ""
Write-Host "🔍 Verifying installation..." -ForegroundColor Cyan

if (Test-Path $targetBinary) {
    Write-Host "✓ Binary installed successfully" -ForegroundColor Green

    # Try to run symphony version check
    try {
        $version = & $targetBinary whoami --help 2>&1
        Write-Host "✓ Symphony is executable" -ForegroundColor Green
    } catch {
        Write-Host "⚠ Warning: Could not execute symphony" -ForegroundColor Yellow
    }
} else {
    Write-Host "❌ Installation failed: binary not found" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✅ Symphony installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📖 Next steps:" -ForegroundColor Cyan
Write-Host "  1. Configure: symphony config" -ForegroundColor White
Write-Host "  2. Login:     symphony login" -ForegroundColor White
Write-Host "  3. Init repo: symphony init" -ForegroundColor White
Write-Host "  4. Dashboard: symphony dashboard" -ForegroundColor White
Write-Host ""
Write-Host "📚 Documentation: See README.md" -ForegroundColor Cyan
Write-Host ""
