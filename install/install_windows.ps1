# AutoCap Windows Installation Script
# This script:
# 1. Checks if Go is installed, and if not, installs it silently via winget, choco, or MSI download.
# 2. Builds autocap.exe.
# 3. Installs autocap.exe to AppData/Local/autocap.
# 4. Registers a Task Scheduler task to run AutoCap at logon.

$ErrorActionPreference = "Stop"

# Check if Go is installed
$goInstalled = $null
try {
    $goInstalled = Get-Command go -ErrorAction SilentlyContinue
} catch {}

if (-not $goInstalled) {
    Write-Host "Go is not installed. Attempting to install Go..."
    
    # Try winget
    $wingetInstalled = Get-Command winget -ErrorAction SilentlyContinue
    if ($wingetInstalled) {
        Write-Host "Installing Go via winget..."
        Start-Process winget -ArgumentList "install --id GoLang.Go --silent --accept-package-agreements --accept-source-agreements" -Wait
    } else {
        # Try Chocolatey
        $chocoInstalled = Get-Command choco -ErrorAction SilentlyContinue
        if ($chocoInstalled) {
            Write-Host "Installing Go via Chocolatey..."
            Start-Process choco -ArgumentList "install golang -y" -Wait
        } else {
            # Download and run MSI installer silently
            Write-Host "Downloading Go installer from go.dev..."
            $msiPath = "$env:TEMP\go.msi"
            # Using TLS 1.2
            [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
            Invoke-WebRequest -Uri "https://go.dev/dl/go1.22.4.windows-amd64.msi" -OutFile $msiPath
            Write-Host "Running MSI installer silently..."
            Start-Process msiexec.exe -ArgumentList "/i $msiPath /quiet /qn /norestart" -Wait
            Remove-Item $msiPath -Force
        }
    }

    # Refresh path
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
}

# Verify Go installation
$goInstalled = $null
try {
    $goInstalled = Get-Command go -ErrorAction SilentlyContinue
} catch {}

if (-not $goInstalled) {
    Write-Warning "Go was installed but is not in the current session path. Please restart PowerShell and run this script again."
    exit 1
}

Write-Host "Go is installed: $(go version)"

# Build autocap
Write-Host "Building AutoCap..."
go build -o autocap.exe ./cmd/autocap

# Install directory setup
$installDir = "$env:USERPROFILE\AppData\Local\autocap"
if (-not (Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
}

# Copy binary
Write-Host "Installing binary to $installDir..."
Copy-Item autocap.exe -Destination "$installDir\autocap.exe" -Force

# Setup Task Scheduler Task
Write-Host "Setting up Scheduled Task..."
$Action = New-ScheduledTaskAction -Execute "$installDir\autocap.exe"
$Trigger = New-ScheduledTaskTrigger -AtLogon
$Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries
Register-ScheduledTask -TaskName "AutoCap" -Action $Action -Trigger $Trigger -Settings $Settings -Description "AutoCap Captive Portal Automator" -Force

Write-Host "Done! AutoCap is installed to $installDir"
Write-Host "A Scheduled Task 'AutoCap' has been registered to run at user logon."
