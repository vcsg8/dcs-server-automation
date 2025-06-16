param(
    [string]$BinarySource = ".\\dcsdog.exe",
    [string]$InstallDir = "C:\\Program Files\\dcsdog",
    [string]$ServiceName = "dcsdog"
)

# Check for admin rights
if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Warning "You must run this script as an Administrator!"
    exit 1
}

# Create install directory if it doesn't exist
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

# Copy the binary
Copy-Item $BinarySource -Destination "$InstallDir\\dcsdog.exe" -Force

# Register the service if it doesn't exist
if (-not (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
    New-Service -Name $ServiceName -BinaryPathName "`"$InstallDir\\dcsdog.exe`"" -DisplayName "DCS Dog" -StartupType Automatic
}

# Start the service
Start-Service -Name $ServiceName

Write-Host "dcsdog installed and running as a service." 