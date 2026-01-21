param(
  [int]$Iterations = 10
)

$ErrorActionPreference = "Stop"

Push-Location (Split-Path -Parent $MyInvocation.MyCommand.Path)
try {
  for ($i = 1; $i -le $Iterations; $i++) {
    Write-Host "=== Vibium Java integration soak: run $i/$Iterations ==="
    Push-Location ..
    try {
      mvn -q verify -DskipITs=false
    } finally {
      Pop-Location
    }
  }
  Write-Host "OK: $Iterations integration runs passed"
} finally {
  Pop-Location
}

