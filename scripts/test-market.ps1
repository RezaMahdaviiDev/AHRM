# Test market API (query token only) — run on Iranian internet from project root.
#   powershell -File scripts\test-market.ps1
$ErrorActionPreference = "Stop"
Set-Location (Join-Path $PSScriptRoot "..")

$envFile = Join-Path (Get-Location) ".env"
$token = (Get-Content $envFile -Encoding UTF8 |
    Where-Object { $_ -match '^\s*SOURCEARENA_API_TOKEN=' } |
    Select-Object -First 1) -replace '^\s*SOURCEARENA_API_TOKEN=', ''
$token = $token.Trim()
if ([string]::IsNullOrWhiteSpace($token)) { Write-Error "SOURCEARENA_API_TOKEN empty in .env" }

Write-Host "=== SourceArena market API (token in query only) ===" -ForegroundColor Cyan

$tests = @(
    @{ Name = "options (all=e)"; Url = "https://apis.sourcearena.ir/api/?token=$token&all=e" },
    @{ Name = "all symbols (all&type=2)"; Url = "https://apis.sourcearena.ir/api/?token=$token&all&type=2" },
    @{ Name = "underlying (name=اهرم)"; Url = "https://apis.sourcearena.ir/api/?token=$token&name=اهرم" }
)

foreach ($t in $tests) {
    Write-Host "`n--- $($t.Name) ---" -ForegroundColor Yellow
    $out = curl.exe -s $t.Url
    if ($out.Length -gt 500) { $out.Substring(0, 500) + "..." } else { $out }
    Write-Host ""
}

Write-Host "=== Expected ===" -ForegroundColor Green
Write-Host "JSON data. NOT: invalid method"
