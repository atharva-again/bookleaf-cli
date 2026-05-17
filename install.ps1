#!/usr/bin/env pwsh
# BookLeaf CLI installer for Windows
# Usage: irm https://bookleaf-assignment-atharva.vercel.app/cli/install.ps1 | iex

param(
  [string]$Version = "",
  [switch]$NoPathUpdate = $false,
  [switch]$Help = $false
)

$ErrorActionPreference = "Stop"

if ($Help) {
  Write-Output @"
BookLeaf CLI Installer for Windows

Install the BookLeaf CLI to manage the BookLeaf support portal.

Usage: irm https://bookleaf-assignment-atharva.vercel.app/cli/install.ps1 | iex

Parameters:
    -Version <ver>     Install a specific version (e.g., v0.1.5)
    -NoPathUpdate      Don't add bookleaf to PATH
    -Help              Display this help message

Examples:
    irm https://bookleaf-assignment-atharva.vercel.app/cli/install.ps1 | iex
    irm .../install.ps1 | iex -Version v0.1.5
"@
  return
}

$Repo = "atharva-again/bookleaf-cli"

$BookleafRoot = if ($env:BOOKLEAF_INSTALL) { $env:BOOKLEAF_INSTALL } else { "$HOME\.bookleaf" }
$BinDir = mkdir -Force "$BookleafRoot\bin"
$Exe = "$BinDir\bookleaf.exe"

# ---- Detect architecture ----
$Arch = "x86_64"
try {
  $cpu = (Get-CimInstance Win32_ComputerSystem).SystemType
  if ($cpu -match "ARM64" -or $cpu -match "aarch64") {
    $Arch = "arm64"
  }
} catch {
  # fallback
}

$Filename = "bookleaf_Windows_${Arch}.zip"

# ---- Version resolution ----
if ([string]::IsNullOrEmpty($Version)) {
  Write-Output "Fetching latest version..."
  try {
    $tag = (Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest").tag_name
  } catch {
    try {
      $resp = Invoke-WebRequest -Uri "https://github.com/$Repo/releases/latest" -MaximumRedirection 0 -ErrorAction SilentlyContinue
      $tag = [System.IO.Path]::GetFileName($resp.Headers.Location)
    } catch {
      Write-Output "Could not determine latest version. Specify: iex ...\install.ps1 -Version v0.1.5"
      return 1
    }
  }
  $Version = "$tag"
}

$VersionStripped = $Version -replace '^v', ''
$DownloadUrl = "https://github.com/$Repo/releases/download/v${VersionStripped}/$Filename"

Write-Output "Downloading bookleaf v${VersionStripped}..."

$ZipPath = "$BinDir\$Filename"
try {
  Invoke-RestMethod -Uri $DownloadUrl -OutFile $ZipPath
} catch {
  Write-Output "Download failed: $DownloadUrl"
  return 1
}

# ---- Checksum verification ----
try {
  $CheckUrl = "https://github.com/$Repo/releases/download/v${VersionStripped}/checksums.txt"
  $Checksums = Invoke-RestMethod -Uri $CheckUrl
  $ExpectedLine = $Checksums -split "`n" | Where-Object { $_ -match [regex]::Escape($Filename) }
  if ($ExpectedLine) {
    $ExpectedHash = ($ExpectedLine -split '\s+')[0]
    $ActualHash = (Get-FileHash $ZipPath -Algorithm SHA256).Hash
    if ($ExpectedHash -ne $ActualHash) {
      Write-Output "Checksum mismatch"
      Remove-Item $ZipPath -Force
      return 1
    }
    Write-Output "Checksum verified."
  }
} catch {
  Write-Output "Warning: checksum verification skipped"
}

# ---- Extract ----
Write-Output "Extracting..."
if (Get-Command tar.exe -ErrorAction SilentlyContinue) {
  tar.exe xf "$ZipPath" -C "$BinDir"
} else {
  Expand-Archive -Path "$ZipPath" -DestinationPath "$BinDir" -Force
}

# Move from wrapped directory
$WrappedExe = Get-ChildItem -Recurse "$BinDir\bookleaf.exe" | Select-Object -First 1
if ($WrappedExe -and ($WrappedExe.Directory.FullName -ne $BinDir)) {
  Move-Item $WrappedExe.FullName "$Exe" -Force
}

$LoadedVersion = if (Test-Path $Exe) {
  $v = & $Exe --version 2>$null
  if ($LASTEXITCODE -eq 0) { " ($v)" } else { "" }
} else { "" }

Remove-Item $ZipPath -Force -ErrorAction SilentlyContinue

Write-Output "bookleaf v${VersionStripped} installed successfully to $Exe$LoadedVersion"

# ---- PATH setup ----
if (-not $NoPathUpdate) {
  $UserPath = [System.Environment]::GetEnvironmentVariable('PATH', 'User')
  if ($UserPath -notlike "*$BinDir*") {
    [System.Environment]::SetEnvironmentVariable('PATH', "$UserPath;$BinDir", 'User')
    $env:PATH += ";$BinDir"
    Write-Output "Added $BinDir to your PATH (restart your terminal to apply)"
  }
}

Write-Output ""
Write-Output "To get started, run:"
Write-Output "  bookleaf --help"
