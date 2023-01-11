# Download latest LTS rootfs
param(
    [Parameter(Mandatory)][string]$Path
)

$rootfsDir="${Path}"
$sha256file="${rootfsDir}\SHA256SUMS.txt"
$tarball="${rootfsDir}\jammy.tar.gz"

if ( ! $(Test-Path -Path "${rootfsDir}") ) {
    New-Item -Path "${rootfsDir}" -ItemType "directory" | Out-Null
}

# Testing if a rootfs exists already
$inCache=$False
if ( $(Test-Path -Path "${sha256file}") -and $(Test-Path -Path "${tarball}") ) {
    $inCache=$True
}

Invoke-WebRequest                                                       `
-Uri "https://cloud-images.ubuntu.com/wsl/jammy/current/SHA256SUMS" `
-OutFile "${sha256file}.tmp"

if ( $inCache ) {
    # Testing if the existing cache is up to date
    $oldSha256=$(Get-Content "${sha256file}")
    $newSha256=$(Get-Content "${sha256file}.tmp")

    if ( "${oldSha256}" -eq "${newSha256}" ) {
        Write-Output "Cache hit"
        Remove-Item -Path "${sha256file}.tmp" 2>&1 | Out-Null
        Exit(0)
    }
}

Invoke-WebRequest                                                                                     `
    -Uri "https://cloud-images.ubuntu.com/wsl/jammy/current/ubuntu-jammy-wsl-amd64-wsl.rootfs.tar.gz" `
    -OutFile "${tarball}.tmp"
if ( ! "$?" ) {
    Remove-Item -Path "${sha256file}.tmp" 2>&1 | Out-Null
    Remove-Item -Path "${tarball}.tmp" 2>&1 | Out-Null
    Exit(1)
}

Move-Item "${tarball}.tmp" "${tarball}" -Force
if ( ! "$?" ) {
    Remove-Item -Path "${sha256file}.tmp" 2>&1 | Out-Null
    Remove-Item -Path "${tarball}.tmp" 2>&1 | Out-Null
    Exit(2)
}

Move-Item "${sha256file}.tmp" "${sha256file}" -Force
