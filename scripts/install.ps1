$ErrorActionPreference = "Stop"

$Repo = "traiproject/yaml-schema-router"
$ProjectName = "yaml-schema-router"

# Detect Architecture
$Arch = $env:PROCESSOR_ARCHITECTURE
if ($Arch -eq "AMD64") {
    $ArchName = "x86_64"
} elseif ($Arch -eq "ARM64") {
    $ArchName = "arm64"
} else {
    Write-Host "Unsupported architecture: $Arch" -ForegroundColor Red
    exit 1
}

Write-Host "Detecting latest version for $ProjectName..."
# Fetch latest version
$LatestReleaseUrl = "https://api.github.com/repos/$Repo/releases/latest"
$Release = Invoke-RestMethod -Uri $LatestReleaseUrl
$Version = $Release.tag_name

if (-not $Version) {
    Write-Host "Error: Could not determine the latest version." -ForegroundColor Red
    exit 1
}

$VersionWithoutV = $Version.TrimStart('v')
# Example: yaml-schema-router_1.0.0_windows_x86_64.tar.gz
$FileName = "${ProjectName}_${VersionWithoutV}_windows_${ArchName}.tar.gz"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$FileName"

$InstallDir = "$env:LOCALAPPDATA\$ProjectName"
$TempDir = Join-Path $env:TEMP $ProjectName
$ArchivePath = Join-Path $TempDir $FileName

if (-not (Test-Path $TempDir)) {
    New-Item -ItemType Directory -Path $TempDir | Out-Null
}

Write-Host "Downloading $ProjectName $Version for Windows $ArchName..."
Invoke-WebRequest -Uri $DownloadUrl -OutFile $ArchivePath

Write-Host "Extracting..."
# Using tar which is built into modern Windows 10/11
tar -xzf $ArchivePath -C $TempDir

if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

Write-Host "Installing to $InstallDir..."
Move-Item -Path (Join-Path $TempDir "$ProjectName.exe") -Destination (Join-Path $InstallDir "$ProjectName.exe") -Force

# Clean up
Remove-Item -Path $TempDir -Recurse -Force

# Check if it is in PATH
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notmatch [regex]::Escape($InstallDir)) {
    Write-Host "Adding $InstallDir to your User PATH..."
    $NewPath = "$UserPath;$InstallDir"
    [Environment]::SetEnvironmentVariable("PATH", $NewPath, "User")
    Write-Host "PATH updated. You may need to restart your PowerShell session for the changes to take effect." -ForegroundColor Yellow
}

Write-Host "Successfully installed $ProjectName $Version!" -ForegroundColor Green
