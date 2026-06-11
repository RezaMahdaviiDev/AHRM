# Run on Iranian internet from project root:  powershell -File scripts\test-sourcearena.ps1
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
            [System.Environment]::GetEnvironmentVariable("Path", "User")

$envFile = Join-Path (Get-Location) ".env"
if (-not (Test-Path $envFile)) {
    Write-Error ".env not found at $envFile — run from D:\ahrd"
}

$token = (Get-Content $envFile -Encoding UTF8 |
    Where-Object { $_ -match '^\s*SOURCEARENA_API_TOKEN=' } |
    Select-Object -First 1) -replace '^\s*SOURCEARENA_API_TOKEN=', ''
$token = $token.Trim()

if ([string]::IsNullOrWhiteSpace($token)) {
    Write-Error "SOURCEARENA_API_TOKEN is empty in .env"
}

Write-Host "=== SourceArena candle test (Iran only) ===" -ForegroundColor Cyan
Write-Host "Token length: $($token.Length)"

$urls = @(
    "https://api3.sourcearena.ir/api/v2/candle/1m?from=1700000000&to=1710000000&symbol=فملی&resolution=1D&type=1",
    "https://api3.sourcearena.ir/api/v2/candle/1m?from=1700000000&to=1710000000&symbol=اهرم&resolution=1D&type=1"
)

foreach ($url in $urls) {
    $sym = if ($url -match 'symbol=([^&]+)') { [uri]::UnescapeDataString($matches[1]) } else { "?" }
    Write-Host "`n--- symbol: $sym ---" -ForegroundColor Yellow
    $out = curl.exe -s -H "X-Header-Token:$token" $url
    if ($out.Length -gt 400) { $out.Substring(0, 400) + "..." } else { $out }
    Write-Host ""
}

Write-Host "=== Expected ===" -ForegroundColor Green
Write-Host "JSON array like [{""c"":...,""t"":...}] OR wrapped data with candles."
Write-Host "If you see error_code 1001 => token missing/invalid in header."
