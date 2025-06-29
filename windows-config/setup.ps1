$repo = "https://github.com/vcsg8/dcs-server-automation.git"
$localPath = "C:\Scripts\dcs-server-automation\windows-config"
$scriptPath = "$localPath\setup.ps1"
$manifestPath = "$localPath\choco-manifest.txt"
$taskName = "Update Software"

if (-not (Test-Path "C:\Scripts")) {
    New-Item -Path "C:\Scripts" -ItemType Directory | Out-Null
}

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    Set-ExecutionPolicy Bypass -Scope Process -Force
    [System.Net.ServicePointManager]::SecurityProtocol = 3072
    iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))
    choco install -y git
}

if (Test-Path $localPath) {
    Set-Location $localPath
    git pull
} else {
    git clone $repo $localPath
}

$packages = Get-Content $manifestPath | Where-Object { $_ -and ($_ -notmatch "^#") }

foreach ($pkg in $packages) {
    choco install -y $pkg
}

choco upgrade all -y

$action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-ExecutionPolicy Bypass -File `"$scriptPath`""
$trigger = New-ScheduledTaskTrigger -Daily -At 3:00AM
$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -RunLevel Highest
$task = New-ScheduledTask -Action $action -Trigger $trigger -Principal $principal
Register-ScheduledTask -TaskName $taskName -InputObject $task -Force

## dcsdog
$dcsDogInstallScriptPath = "$localPath\dcsdog\install-dcsdog.ps1"
if (Test-Path $dcsDogInstallScriptPath) {
    powershell -ExecutionPolicy Bypass -File $dcsDogInstallScriptPath
}