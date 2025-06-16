param(
    [string]$ServiceName = "dcsdog",
    [string]$InstallDir = "C:\\Program Files\\dcsdog"
)

# Check for admin rights
if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Warning "You must run this script as an Administrator!"
    exit 1
}

# Stop and remove the service
if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
    Stop-Service -Name $ServiceName -Force
    sc.exe delete $ServiceName | Out-Null
}

# Remove the binary and directory
if (Test-Path $InstallDir) {
    Remove-Item -Path $InstallDir -Recurse -Force
}

Write-Host "dcsdog service and files removed." 