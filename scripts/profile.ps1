$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

New-Item -ItemType Directory -Force -Path profiles | Out-Null

$server = Start-Process -FilePath ".\shortener.exe" -ArgumentList "-a","127.0.0.1:8080" -PassThru -WindowStyle Hidden
Start-Sleep -Seconds 1

try {
    .\loadgen.exe
    Invoke-WebRequest -Uri "http://127.0.0.1:6060/debug/pprof/heap" -OutFile "profiles\result.pprof"
    Write-Host "Profile saved to profiles\result.pprof"
} finally {
    Stop-Process -Id $server.Id -Force -ErrorAction SilentlyContinue
}
