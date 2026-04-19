param(
    [string]$Version = "v1.1.0",
    [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root $OutputDir
if (Test-Path $dist) {
    Remove-Item $dist -Recurse -Force
}
New-Item -ItemType Directory -Force -Path $dist | Out-Null

$commonFiles = @(
    "README.md",
    "HumanSpeak_Beginner_to_Advanced_Guide.pdf",
    "install.ps1",
    "install.sh"
)

function New-ArchiveBundle {
    param(
        [string]$Goos,
        [string]$Goarch,
        [string]$BinaryName,
        [string]$AssetName,
        [ValidateSet("zip", "tar.gz")]
        [string]$ArchiveType
    )

    $stageRoot = Join-Path $dist "$Goos-$Goarch"
    if (Test-Path $stageRoot) {
        Remove-Item $stageRoot -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $stageRoot | Out-Null

    $env:CGO_ENABLED = "0"
    $env:GOOS = $Goos
    $env:GOARCH = $Goarch
    & "C:\Program Files\Go\bin\go.exe" build -o (Join-Path $stageRoot $BinaryName) .\cmd\humanspeak

    foreach ($file in $commonFiles) {
        Copy-Item (Join-Path $root $file) $stageRoot -Force
    }

    $archivePath = Join-Path $dist $AssetName
    if (Test-Path $archivePath) {
        Remove-Item $archivePath -Force
    }

    if ($ArchiveType -eq "zip") {
        Compress-Archive -Path (Join-Path $stageRoot "*") -DestinationPath $archivePath
    } else {
        tar -czf $archivePath -C $stageRoot .
    }

    Remove-Item $stageRoot -Recurse -Force
}

New-ArchiveBundle -Goos "windows" -Goarch "amd64" -BinaryName "humanspeak.exe" -AssetName "humanspeak-windows-x64.zip" -ArchiveType "zip"
New-ArchiveBundle -Goos "linux" -Goarch "amd64" -BinaryName "humanspeak" -AssetName "humanspeak-linux-x64.tar.gz" -ArchiveType "tar.gz"
New-ArchiveBundle -Goos "linux" -Goarch "arm64" -BinaryName "humanspeak" -AssetName "humanspeak-linux-arm64.tar.gz" -ArchiveType "tar.gz"
New-ArchiveBundle -Goos "darwin" -Goarch "amd64" -BinaryName "humanspeak" -AssetName "humanspeak-macos-x64.tar.gz" -ArchiveType "tar.gz"
New-ArchiveBundle -Goos "darwin" -Goarch "arm64" -BinaryName "humanspeak" -AssetName "humanspeak-macos-arm64.tar.gz" -ArchiveType "tar.gz"

Write-Host "Release bundles created in $dist for $Version"
