# Script untuk build otomatis ke format .zip untuk x86, x86_64, dan arm64 (Windows & Linux)
param (
    [string]$Version = "v1.0.0"
)

$AppName = "bot_wa"
$DistDir = Join-Path $PSScriptRoot "..\dist"
$RootDir = Join-Path $PSScriptRoot ".."

# Buat direktori dist jika belum ada
if (Test-Path $DistDir) {
    Remove-Item -Recurse -Force $DistDir
}
New-Item -ItemType Directory -Path $DistDir -Force | Out-Null

$Targets = @(
    # Linux
    @{ OS = "linux"; ARCH = "amd64"; Label = "linux-amd64" },
    @{ OS = "linux"; ARCH = "arm64"; Label = "linux-arm64" },
    @{ OS = "linux"; ARCH = "386";   Label = "linux-x86_32" },
    # Windows
    @{ OS = "windows"; ARCH = "amd64"; Label = "windows-amd64" },
    @{ OS = "windows"; ARCH = "arm64"; Label = "windows-arm64" },
    @{ OS = "windows"; ARCH = "386";   Label = "windows-x86_32" }
)

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  Memulai Build Release: $Version" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

foreach ($target in $Targets) {
    $os = $target.OS
    $arch = $target.ARCH
    $label = $target.Label
    $outputName = if ($os -eq "windows") { "$AppName.exe" } else { $AppName }
    $pkgDir = Join-Path $DistDir "$AppName-$Version-$label"
    $zipFile = Join-Path $DistDir "$AppName-$Version-$label.zip"

    Write-Host "[+] Compiling $label (OS: $os, ARCH: $arch)..." -ForegroundColor Yellow

    New-Item -ItemType Directory -Path $pkgDir -Force | Out-Null

    $binaryPath = Join-Path $pkgDir $outputName

    # Set Go build environment
    $env:GOOS = $os
    $env:GOARCH = $arch
    $env:CGO_ENABLED = "0"

    # Build binary with symbol stripping (-s -w) for smaller size
    go build -ldflags="-s -w" -o $binaryPath (Join-Path $RootDir "main.go")

    if ($LASTEXITCODE -ne 0) {
        Write-Host "[-] Gagal compile $label!" -ForegroundColor Red
        continue
    }

    # Copy supporting files
    Copy-Item (Join-Path $RootDir ".env.example") -Destination (Join-Path $pkgDir ".env.example") -Force
    if (Test-Path (Join-Path $RootDir "README.md")) {
        Copy-Item (Join-Path $RootDir "README.md") -Destination (Join-Path $pkgDir "README.md") -Force
    }
    if (Test-Path (Join-Path $RootDir "DEPLOYMENT_UBUNTU.md")) {
        Copy-Item (Join-Path $RootDir "DEPLOYMENT_UBUNTU.md") -Destination (Join-Path $pkgDir "DEPLOYMENT_UBUNTU.md") -Force
    }

    # Compress to ZIP
    Write-Host "    -> Zipping to $(Split-Path $zipFile -Leaf)..." -ForegroundColor Green
    Compress-Archive -Path "$pkgDir\*" -DestinationPath $zipFile -Force

    # Clean up uncompressed folder
    Remove-Item -Recurse -Force $pkgDir
}

# Reset environment variable
$env:GOOS = ""
$env:GOARCH = ""
$env:CGO_ENABLED = ""

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  Build selesai! File tersedia di folder: dist" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Cyan
Get-ChildItem -Path $DistDir -Filter "*.zip" | Select-Object Name, Length
