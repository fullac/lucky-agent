param(
  [string]$SourceRoot = (Split-Path -Parent $PSCommandPath),
  [string]$InstallDir = (Join-Path $env:LOCALAPPDATA "LuckyAgent")
)

$ErrorActionPreference = "Stop"

$Binary = Join-Path $SourceRoot "lh.exe"
if (-not (Test-Path $Binary)) { throw "LuckyAgent payload is missing $Binary" }

$Parent = Split-Path -Parent $InstallDir
New-Item -ItemType Directory -Force -Path $Parent | Out-Null
$Stage = Join-Path $Parent (".LuckyAgent-stage-" + [guid]::NewGuid().ToString())
New-Item -ItemType Directory -Force -Path $Stage | Out-Null
Copy-Item -Force $Binary (Join-Path $Stage "lh.exe")
$Installer = Join-Path $SourceRoot "Install-Portable.ps1"
if (Test-Path $Installer) {
  Copy-Item -Force $Installer (Join-Path $Stage "Install-Portable.ps1")
}

if (Test-Path $InstallDir) { Remove-Item -Recurse -Force $InstallDir }
Move-Item -Force $Stage $InstallDir

$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
$parts = @($currentPath -split ';' | Where-Object { $_ })
if (-not ($parts | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') })) {
  $parts += $InstallDir
}
[Environment]::SetEnvironmentVariable("Path", ($parts -join ';'), "User")

& (Join-Path $InstallDir "lh.exe") init | Out-Null
Write-Host "LuckyAgent installed to $InstallDir"
Write-Host "Command: lh"
Write-Host "Open a new terminal so the updated user PATH is available."
