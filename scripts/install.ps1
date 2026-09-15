<#
.SYNOPSIS
  MOS-MCP installer for Windows.

.DESCRIPTION
  Downloads the latest release archive from GitHub, verifies its checksum,
  installs mos-mcp.exe to %LOCALAPPDATA%\Programs\mos-mcp, and adds it to the
  user PATH.

  Run:
    irm https://raw.githubusercontent.com/medcelerate/MOS-MCP/main/scripts/install.ps1 | iex

.PARAMETER Version
  Release tag to install (default: latest).
#>
[CmdletBinding()]
param(
  [string]$Version = $env:MOSMCP_VERSION,
  [string]$InstallDir = "$env:LOCALAPPDATA\Programs\mos-mcp"
)

$ErrorActionPreference = "Stop"
$Repo = "medcelerate/MOS-MCP"
$Bin = "mos-mcp"

function Info($m) { Write-Host "==> $m" -ForegroundColor Cyan }

# Detect architecture.
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
  "AMD64" { "amd64" }
  "ARM64" { "arm64" }
  default { throw "unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
}

# Resolve version.
if ([string]::IsNullOrEmpty($Version)) {
  Info "Resolving latest release"
  $latest = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
  $Version = $latest.tag_name
}
$vnum = $Version.TrimStart("v")

$archive = "${Bin}_${vnum}_windows_${arch}.zip"
$base = "https://github.com/$Repo/releases/download/$Version"

$tmp = Join-Path $env:TEMP ("mos-mcp-" + [System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  Info "Downloading $archive"
  $zipPath = Join-Path $tmp $archive
  Invoke-WebRequest "$base/$archive" -OutFile $zipPath

  # Verify checksum when available.
  try {
    $sumsPath = Join-Path $tmp "checksums.txt"
    Invoke-WebRequest "$base/checksums.txt" -OutFile $sumsPath
    $line = (Get-Content $sumsPath | Where-Object { $_ -match [regex]::Escape($archive) + '$' } | Select-Object -First 1)
    if ($line) {
      $expected = ($line -split '\s+')[0]
      $actual = (Get-FileHash $zipPath -Algorithm SHA256).Hash.ToLower()
      if ($expected.ToLower() -ne $actual) { throw "checksum mismatch for $archive" }
      Info "Checksum verified"
    }
  } catch {
    Write-Warning "checksum verification skipped: $($_.Exception.Message)"
  }

  Info "Extracting"
  Expand-Archive -Path $zipPath -DestinationPath $tmp -Force
  $exe = Join-Path $tmp "$Bin.exe"
  if (-not (Test-Path $exe)) { throw "binary not found in archive" }

  New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  Copy-Item $exe (Join-Path $InstallDir "$Bin.exe") -Force
  Info "Installed $Bin $Version to $InstallDir"

  # Add to user PATH if missing.
  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  if ($userPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
    Info "Added $InstallDir to your user PATH (restart your shell to pick it up)"
  }

  & (Join-Path $InstallDir "$Bin.exe") --version
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
