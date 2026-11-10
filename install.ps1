# ktdev installer for Windows (PowerShell).
# Usage — paste this one line in PowerShell:
#   irm https://raw.githubusercontent.com/masudibnbelat/ktdev-cli/main/install.ps1 | iex
#
# What it does:
#   1. Finds the latest GitHub Release of masudibnbelat/ktdev-cli.
#   2. Downloads ktdev-windows-amd64.exe.
#   3. Installs it to $env:LOCALAPPDATA\ktdev\bin\ktdev.exe.
#   4. Adds that folder to the user PATH (new terminals pick it up).
$ErrorActionPreference = "Stop"
$Repo = "masudibnbelat/ktdev-cli"
$InstallDir = Join-Path $env:LOCALAPPDATA "ktdev\bin"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

Write-Host "== ktdev: finding latest release of $Repo"
$Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$Tag = $Release.tag_name
if (-not $Tag) { throw "Could not find the latest release. Has a release (tag vX.Y.Z) been published yet?" }
Write-Host "== latest: $Tag — downloading ktdev-windows-amd64.exe"
$Url = "https://github.com/$Repo/releases/download/$Tag/ktdev-windows-amd64.exe"
Invoke-WebRequest -Uri $Url -OutFile (Join-Path $InstallDir "ktdev.exe")

# Add to the user PATH (persists; new terminals get it).
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
}
$env:Path = "$env:Path;$InstallDir"

Write-Host "== DONE: ktdev $Tag installed to $InstallDir\ktdev.exe"
& ktdev --version
Write-Host "Close and reopen the terminal, then in any Android project:"
Write-Host "  cd CalNyx"
Write-Host "  ktdev run"
Write-Host '  ktdev release v1.5.0 "notes"'
Write-Host "Note: Android SDK, adb, Java and gh still need to be installed once"
Write-Host "on Windows (Android Studio installs SDK + adb; `gh auth login` once)."
