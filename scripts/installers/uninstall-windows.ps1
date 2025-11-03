# Symphony Windows Uninstaller
# Run with: powershell -ExecutionPolicy Bypass -File uninstall-windows.ps1

$ErrorActionPreference = "Stop"

# Set console encoding to UTF-8 for proper Korean display
$OutputEncoding = [System.Text.Encoding]::UTF8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$PSDefaultParameterValues['*:Encoding'] = 'utf8'

# Set console code page to UTF-8 (65001)
chcp 65001 > $null

Write-Host "🎵 Symphony Windows Uninstaller" -ForegroundColor Cyan
Write-Host ""

# Check possible installation locations
$possibleLocations = @(
    "C:\Program Files\Symphony",
    "$env:LOCALAPPDATA\Symphony"
)

$foundLocations = @()
foreach ($loc in $possibleLocations) {
    if (Test-Path "$loc\symphony.exe") {
        $foundLocations += $loc
    }
}

if ($foundLocations.Count -eq 0) {
    Write-Host "❌ Symphony installation not found" -ForegroundColor Red
    Write-Host ""
    Write-Host "Checked locations:" -ForegroundColor Yellow
    foreach ($loc in $possibleLocations) {
        Write-Host "  - $loc" -ForegroundColor Yellow
    }
    exit 1
}

Write-Host "📂 Found Symphony installations:" -ForegroundColor Cyan
foreach ($loc in $foundLocations) {
    Write-Host "  - $loc" -ForegroundColor White
}
Write-Host ""

# Ask for confirmation
$confirmation = Read-Host "Uninstall Symphony? (y/N)"
if ($confirmation -ne 'y' -and $confirmation -ne 'Y') {
    Write-Host "❌ Uninstallation cancelled" -ForegroundColor Red
    exit 1
}

# Check if running as Administrator
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
$isAdmin = $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)

# Remove installations
foreach ($installDir in $foundLocations) {
    Write-Host ""
    Write-Host "🗑️  Removing $installDir..." -ForegroundColor Cyan

    try {
        Remove-Item -Path $installDir -Recurse -Force
        Write-Host "✓ Removed" -ForegroundColor Green
    } catch {
        if ($installDir -like "C:\Program Files*" -and -not $isAdmin) {
            Write-Host "❌ Failed: Administrator privileges required" -ForegroundColor Red
            Write-Host "   Please run this script as Administrator" -ForegroundColor Yellow
        } else {
            Write-Host "❌ Failed: $_" -ForegroundColor Red
        }
    }
}

# Remove from PATH
Write-Host ""
Write-Host "🔧 Cleaning PATH environment variable..." -ForegroundColor Cyan

foreach ($installDir in $foundLocations) {
    # Try system PATH (requires admin)
    if ($isAdmin) {
        try {
            $envTarget = [System.EnvironmentVariableTarget]::Machine
            $currentPath = [Environment]::GetEnvironmentVariable("Path", $envTarget)

            if ($currentPath -like "*$installDir*") {
                $newPath = ($currentPath.Split(';') | Where-Object { $_ -ne $installDir }) -join ';'
                [Environment]::SetEnvironmentVariable("Path", $newPath, $envTarget)
                Write-Host "✓ Removed from system PATH" -ForegroundColor Green
            }
        } catch {
            Write-Host "⚠ Could not update system PATH: $_" -ForegroundColor Yellow
        }
    }

    # Try user PATH
    try {
        $envTarget = [System.EnvironmentVariableTarget]::User
        $currentPath = [Environment]::GetEnvironmentVariable("Path", $envTarget)

        if ($currentPath -like "*$installDir*") {
            $newPath = ($currentPath.Split(';') | Where-Object { $_ -ne $installDir }) -join ';'
            [Environment]::SetEnvironmentVariable("Path", $newPath, $envTarget)
            Write-Host "✓ Removed from user PATH" -ForegroundColor Green
        }
    } catch {
        Write-Host "⚠ Could not update user PATH: $_" -ForegroundColor Yellow
    }
}

# Optional: Remove config files
Write-Host ""
$removeConfig = Read-Host "Remove configuration files in $env:USERPROFILE\.config\symphony? (y/N)"
if ($removeConfig -eq 'y' -or $removeConfig -eq 'Y') {
    $configDir = "$env:USERPROFILE\.config\symphony"
    if (Test-Path $configDir) {
        Remove-Item -Path $configDir -Recurse -Force
        Write-Host "✓ Configuration files removed" -ForegroundColor Green
    } else {
        Write-Host "⊘ No configuration files found" -ForegroundColor Yellow
    }
}

Write-Host ""
Write-Host "✅ Symphony uninstalled successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "⚠ You may need to restart your terminal" -ForegroundColor Yellow
Write-Host ""
