# CS2 Admin Build Script
param(
    [string]$Version = "0.1.0",
    [switch]$Installer
)

$ErrorActionPreference = "Stop"
$BuildDir = Join-Path $PSScriptRoot ".." "dist"

Write-Host "Building CS2 Admin v$Version..."

# Clean
if (Test-Path $BuildDir) { Remove-Item $BuildDir -Recurse -Force }
New-Item -ItemType Directory -Path $BuildDir | Out-Null

# Build Windows
Write-Host "Building Windows x64..."
$env:GOOS = "windows"
$env:GOARCH = "amd64"
wails build -clean -o CS2Admin.exe -ldflags "-X main.appVersion=$Version"

# Copy to dist
Copy-Item (Join-Path $PSScriptRoot ".." "build" "bin" "CS2Admin.exe") $BuildDir

# Create portable zip
$zipPath = Join-Path $BuildDir "CS2Admin-v$Version-windows-portable.zip"
Compress-Archive -Path (Join-Path $BuildDir "CS2Admin.exe") -DestinationPath $zipPath

# Build NSIS installer if requested
if ($Installer) {
    Write-Host "Building NSIS installer..."
    $installerDir = Join-Path $PSScriptRoot ".." "build" "windows" "installer"
    Copy-Item (Join-Path $BuildDir "CS2Admin.exe") $installerDir -Force
    Push-Location $installerDir
    try {
        & makensis /DVERSION=$Version "project.nsi"
        Move-Item "CS2Admin-Setup.exe" $BuildDir -Force
    } finally {
        Pop-Location
    }
}

# Generate checksums
$files = Get-ChildItem $BuildDir -File
foreach ($f in $files) {
    $hash = Get-FileHash $f.FullName -Algorithm SHA256
    "$($hash.Hash)  $($f.Name)" | Out-File -Append (Join-Path $BuildDir "SHA256SUMS.txt")
}

Write-Host "Build complete! Output in $BuildDir"
