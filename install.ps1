param(
    [string]$Repository = "YOUR_GITHUB_USER/HumanSpeak",
    [string]$InstallDir = (Join-Path $env:LOCALAPPDATA "HumanSpeak")
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

$zipPath = Join-Path $env:TEMP "humanspeak-latest.zip"
$url = "https://github.com/$Repository/releases/latest/download/humanspeak-windows-x64.zip"

Invoke-WebRequest -Uri $url -OutFile $zipPath
Expand-Archive -Path $zipPath -DestinationPath $InstallDir -Force
Remove-Item $zipPath -Force

$exePath = Join-Path $InstallDir "humanspeak.exe"
if (-not (Test-Path $exePath)) {
    throw "Installation failed: humanspeak.exe was not found after extraction."
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ([string]::IsNullOrWhiteSpace($userPath)) {
    $newPath = $InstallDir
} elseif ($userPath -notlike "*$InstallDir*") {
    $newPath = "$userPath;$InstallDir"
} else {
    $newPath = $userPath
}

[Environment]::SetEnvironmentVariable("Path", $newPath, "User")
$env:Path = "$InstallDir;$env:Path"

Write-Host "HumanSpeak installed to $InstallDir"
Write-Host "Restart your terminal, then run: humanspeak.exe version"
