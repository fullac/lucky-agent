param(
  [string]$Version = "dev",
  [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$StageDir = Join-Path $RepoRoot "dist\windows-installer"
$OutputPath = Join-Path $RepoRoot $OutputDir

Push-Location $RepoRoot
try {
  if (-not (Test-Path "$RepoRoot\dist\lh.exe")) {
    throw "Windows binary is missing: dist\lh.exe"
  }

  if (Test-Path $StageDir) {
    Remove-Item -Recurse -Force $StageDir
  }
  New-Item -ItemType Directory -Force -Path $StageDir | Out-Null

  Copy-Item -Force "$RepoRoot\dist\lh.exe" "$StageDir\lh.exe"
  Copy-Item -Force "$RepoRoot\packaging\windows\Install-Portable.ps1" "$StageDir\Install-Portable.ps1"

  $Compiler = Get-Command iscc -ErrorAction SilentlyContinue
  if (-not $Compiler) { throw "Inno Setup compiler (iscc) is required. Install it with: choco install innosetup" }
  & $Compiler.Source "/DSourceRoot=$StageDir" "/DMyAppVersion=$Version" "/O$OutputPath" "$RepoRoot\packaging\windows\LuckyAgent.iss"
  if ($LASTEXITCODE -ne 0) {
    throw "Inno Setup compilation failed with exit code $LASTEXITCODE"
  }
} finally {
  Pop-Location
}
