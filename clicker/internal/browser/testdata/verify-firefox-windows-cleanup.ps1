# Manual Windows verification for issue #622: Firefox processes left running
# after `vibium stop`. Must run on Windows with a real Firefox (auto-install is
# unsupported on Windows, so set VIBIUM_ENGINE_PATH to firefox.exe). CI is
# Linux-only and cannot exercise this; run it by hand on a Windows host or VM.
#
# PASS when the firefox.exe count returns to its pre-run baseline after
# `vibium stop`. Before the fix the run leaves ~11 firefox.exe behind and they
# accumulate across runs.
#
# Usage (from the repo root, with vibium.exe built or on PATH):
#   $env:VIBIUM_ENGINE = "firefox"
#   $env:VIBIUM_ENGINE_PATH = "C:\Program Files\Mozilla Firefox\firefox.exe"
#   powershell -ExecutionPolicy Bypass -File clicker\internal\browser\testdata\verify-firefox-windows-cleanup.ps1 -Vibium .\clicker\bin\vibium.exe

param(
    [string]$Vibium = "vibium",
    [string]$Url = "https://example.com"
)

$ErrorActionPreference = "Stop"

function Count-Firefox {
    @(Get-Process -Name firefox -ErrorAction SilentlyContinue).Count
}

$session = "ff622"
$before = Count-Firefox
Write-Host "firefox.exe before: $before"

& $Vibium --session $session go $Url            | Out-Host
$during = Count-Firefox
Write-Host "firefox.exe during session: $during"

& $Vibium --session $session stop               | Out-Host

# Close returns once the stop command is acknowledged; give the OS a moment to
# reap the terminated processes before counting.
Start-Sleep -Seconds 3
$after = Count-Firefox
Write-Host "firefox.exe after stop: $after"

if ($during -le $before) {
    Write-Warning "Firefox never started ($during <= $before); check VIBIUM_ENGINE_PATH. Test inconclusive."
    exit 2
}

if ($after -le $before) {
    Write-Host "PASS: firefox.exe returned to baseline ($after <= $before) after stop." -ForegroundColor Green
    exit 0
}

Write-Error "FAIL: $($after - $before) firefox.exe survived `vibium stop` (before=$before, after=$after)."
exit 1
