param(
    [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root $OutputDir
if (Test-Path $dist) {
    Remove-Item $dist -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $dist | Out-Null

$env:GOCACHE = Join-Path $root ".gocache"
& "C:\Program Files\Go\bin\go.exe" build -o (Join-Path $dist "humanspeak.exe") .\cmd\humanspeak

Copy-Item (Join-Path $root "README.md") $dist -Force
Copy-Item (Join-Path $root "HumanSpeak_Beginner_to_Advanced_Guide.pdf") $dist -Force
Copy-Item (Join-Path $root "install.ps1") $dist -Force

$bundleRoot = Join-Path $dist "HumanSpeak"
New-Item -ItemType Directory -Force -Path $bundleRoot | Out-Null
Copy-Item (Join-Path $dist "humanspeak.exe") $bundleRoot -Force
Copy-Item (Join-Path $dist "README.md") $bundleRoot -Force
Copy-Item (Join-Path $dist "HumanSpeak_Beginner_to_Advanced_Guide.pdf") $bundleRoot -Force
Copy-Item (Join-Path $dist "install.ps1") $bundleRoot -Force

$zipPath = Join-Path $dist "humanspeak-windows-x64.zip"
if (Test-Path $zipPath) {
    Remove-Item $zipPath -Force
}
Compress-Archive -Path (Join-Path $bundleRoot "*") -DestinationPath $zipPath

Write-Host "Release bundle created at $zipPath"
