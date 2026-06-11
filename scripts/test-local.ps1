# Local tests (no SourceArena network):  powershell -File scripts\test-local.ps1
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
            [System.Environment]::GetEnvironmentVariable("Path", "User")

Write-Host "=== Go version ===" -ForegroundColor Cyan
go version

Write-Host "`n=== Unit tests ===" -ForegroundColor Cyan
go test ./... -count=1 2>&1 | ForEach-Object { $_ }
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "`n=== Build server ===" -ForegroundColor Cyan
go build -o bin/server.exe ./cmd/server
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "`n=== Config load ===" -ForegroundColor Cyan
go test ./internal/config/... -count=1 -v -run TestLoadDefaults

Write-Host "`nOK - local tests passed. Next: run scripts/test-sourcearena.ps1 on Iranian internet." -ForegroundColor Green
