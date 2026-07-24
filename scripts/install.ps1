$ErrorActionPreference = "Stop"

$manifestURL = "https://raw.githubusercontent.com/dorkitude/scoop-bucket/main/deadlore.json"
$architecture = if ($env:PROCESSOR_ARCHITEW6432 -eq "ARM64" -or $env:PROCESSOR_ARCHITECTURE -eq "ARM64") {
    "arm64"
} else {
    "64bit"
}

$manifest = Invoke-RestMethod -Uri $manifestURL
$asset = $manifest.architecture.PSObject.Properties[$architecture].Value
if ($null -eq $asset) {
    throw "No Deadlore release is available for Windows $architecture."
}

$temporaryDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("deadlore-" + [guid]::NewGuid())
$archive = Join-Path $temporaryDirectory "deadlore.zip"
$installDirectory = Join-Path $env:LOCALAPPDATA "Programs\deadlore"

try {
    New-Item -ItemType Directory -Path $temporaryDirectory -Force | Out-Null
    Invoke-WebRequest -Uri $asset.url -OutFile $archive

    $actualHash = (Get-FileHash -Path $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne $asset.hash.ToLowerInvariant()) {
        throw "Deadlore download failed SHA-256 verification."
    }

    Expand-Archive -Path $archive -DestinationPath $temporaryDirectory -Force
    New-Item -ItemType Directory -Path $installDirectory -Force | Out-Null
    Copy-Item -Path (Join-Path $temporaryDirectory "deadlore.exe") -Destination (Join-Path $installDirectory "deadlore.exe") -Force
} finally {
    if (Test-Path $temporaryDirectory) {
        Remove-Item -Path $temporaryDirectory -Recurse -Force
    }
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
$pathEntries = @($userPath -split ";" | Where-Object { $_ })
if ($pathEntries -notcontains $installDirectory) {
    $updatedPath = @($pathEntries + $installDirectory) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $updatedPath, "User")
}

$env:Path = "$installDirectory;$env:Path"
Write-Host "Deadlore $($manifest.version) installed to $installDirectory"
Write-Host "Open a new terminal, then run: deadlore Haze"
