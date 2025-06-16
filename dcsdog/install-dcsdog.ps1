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

# Get the directory of the script
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Service details
$DisplayName = "DCS Dog"
$ExePath = Join-Path $ScriptDir "dcsdog.exe"

# Stop and remove existing service if it exists
if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    Stop-Service -Name $ServiceName
    Remove-Service -Name $ServiceName
}

# Create the new service
New-Service -Name $ServiceName -BinaryPathName $ExePath -DisplayName $DisplayName -StartupType Automatic

# Start the service
Start-Service -Name $ServiceName

Write-Host "dcsdog installed and running as a service." 