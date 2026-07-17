<# AI-GENERATED, FULLY APPROVED BY <me@idank.dev>. WE HAVE NO IDEA HOW TO WRITE PS1 SCRIPTS MANUALLY. #>

<#
.SYNOPSIS
    GoScouter installer for Windows.

.DESCRIPTION
    Downloads the gs binary from the GoScouter GitHub releases, verifies its
    sha256 checksum against the published checksums.txt, installs it, and adds
    the install directory to the user PATH.

.PARAMETER Version
    Release tag to install (e.g. v1.2.3). Defaults to the latest release.
    Also settable via the GS_VERSION environment variable.

.PARAMETER InstallDir
    Install directory. Defaults to %LOCALAPPDATA%\Programs\GoScouter.
    Also settable via the GS_INSTALL_DIR environment variable.

.PARAMETER NoVerify
    Skip the sha256 checksum check.

.PARAMETER NoPath
    Do not modify the user PATH.

.EXAMPLE
    irm https://raw.githubusercontent.com/GoScouter/GoScouter/main/scripts/install.ps1 | iex

.EXAMPLE
    .\install.ps1 -Version v1.2.3 -InstallDir C:\tools
#>
[CmdletBinding()]
param(
    [string]$Version = $env:GS_VERSION,
    [string]$InstallDir = $env:GS_INSTALL_DIR,
    [switch]$NoVerify,
    [switch]$NoPath
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Repo = 'GoScouter/GoScouter'
$Binary = 'gs.exe'

function Write-Info { param([string]$Message) Write-Host "  $Message" }
function Write-Warn { param([string]$Message) Write-Host "  warning: $Message" -ForegroundColor Yellow }
function Write-Err {
    param([string]$Message)
    Write-Host "  error: $Message" -ForegroundColor Red
    exit 1
}

# TLS 1.2 is not the default on older Windows PowerShell hosts; GitHub requires it.
try {
    [Net.ServicePointManager]::SecurityProtocol =
        [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
}
catch { }

# --- Platform detection -----------------------------------------------------

# The release matrix builds windows/amd64 only.
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'amd64' }  # runs under x64 emulation on Windows 11 ARM
    'x86' {
        # A 32-bit shell on a 64-bit OS still reports x86.
        if ($env:PROCESSOR_ARCHITEW6432) { 'amd64' }
        else { Write-Err '32-bit Windows is not supported.' }
    }
    default { Write-Err "unsupported architecture: $($env:PROCESSOR_ARCHITECTURE)" }
}

if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {
    Write-Warn 'No native ARM64 build is published; installing the amd64 build (runs under emulation).'
}

$asset = "gs-windows-$arch.exe"

# --- Version resolution -----------------------------------------------------

if (-not $Version) {
    Write-Info 'Resolving latest release...'
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" `
            -Headers @{ 'User-Agent' = 'goscouter-installer' }
        $Version = $release.tag_name
    }
    catch {
        Write-Err @"
could not determine the latest release.
  The repository may not have published one yet - see
  https://github.com/$Repo/releases
  You can build from source instead (needs Go and make):
    git clone https://github.com/$Repo.git; cd GoScouter; make build
"@
    }
}

$baseUrl = "https://github.com/$Repo/releases/download/$Version"

# --- Install directory ------------------------------------------------------

if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\GoScouter'
}

try {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
catch {
    Write-Err "cannot create install directory: $InstallDir"
}

# --- Download ---------------------------------------------------------------

$tmp = Join-Path ([IO.Path]::GetTempPath()) ("gs-install-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $tmp -Force | Out-Null

try {
    Write-Info "Installing GoScouter $Version (windows/$arch) to $InstallDir"

    $downloaded = Join-Path $tmp $asset
    $progress = $ProgressPreference
    $ProgressPreference = 'SilentlyContinue'  # a visible progress bar makes downloads far slower
    try {
        Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile $downloaded -UseBasicParsing
    }
    catch {
        Write-Err @"
download failed: $baseUrl/$asset
  Check that $Version exists and publishes a windows/$arch build.
"@
    }
    finally {
        $ProgressPreference = $progress
    }

    # --- Checksum verification ----------------------------------------------

    if ($NoVerify) {
        Write-Warn 'skipping checksum verification (-NoVerify)'
    }
    else {
        $expected = $null
        try {
            $checksums = (Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -UseBasicParsing).Content
            foreach ($line in $checksums -split "`n") {
                $fields = ($line.Trim() -split '\s+')
                if ($fields.Count -ge 2 -and $fields[1].TrimStart('*') -eq $asset) {
                    $expected = $fields[0]
                    break
                }
            }
        }
        catch { }

        if (-not $expected) {
            Write-Warn "no checksum published for $asset; skipping verification"
        }
        else {
            $actual = (Get-FileHash -Path $downloaded -Algorithm SHA256).Hash
            if ($actual -ne $expected.ToUpperInvariant()) {
                Write-Err @"
checksum mismatch for $asset
  expected: $expected
  actual:   $actual
  The download may be corrupt or tampered with - not installing.
"@
            }
            Write-Info 'Checksum verified.'
        }
    }

    # --- Install ------------------------------------------------------------

    # Unblock so SmartScreen does not flag the binary as web-downloaded.
    try { Unblock-File -Path $downloaded -ErrorAction SilentlyContinue } catch { }

    $target = Join-Path $InstallDir $Binary
    try {
        Move-Item -Path $downloaded -Destination $target -Force
    }
    catch {
        Write-Err @"
failed to install to $target
  If GoScouter is currently running, close it and re-run this installer.
"@
    }

    Write-Info "Installed $target"
}
finally {
    Remove-Item -Path $tmp -Recurse -Force -ErrorAction SilentlyContinue
}

# --- PATH -------------------------------------------------------------------

if ($NoPath) {
    Write-Info "Skipping PATH update (-NoPath). Run the binary at $target"
}
else {
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $entries = @()
    if ($userPath) { $entries = $userPath -split ';' | Where-Object { $_ } }

    if ($entries -notcontains $InstallDir) {
        $newPath = (@($entries) + $InstallDir) -join ';'
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        $env:Path = "$env:Path;$InstallDir"
        Write-Info "Added $InstallDir to your user PATH."
        Write-Info 'Open a new terminal for it to take effect in other sessions.'
    }
}

Write-Info "Run 'gs --help' to get started."
