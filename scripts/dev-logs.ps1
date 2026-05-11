[CmdletBinding()]
param(
    [string[]]$Service
)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$DefaultServices = @("postgres", "backend-api", "admin-web", "prometheus", "grafana")
$TargetServices = if ($Service -and $Service.Count -gt 0) { $Service } else { $DefaultServices }

Set-Location $RepoRoot

& docker compose logs -f --tail 100 @TargetServices
if ($LASTEXITCODE -ne 0) {
    throw "Command failed: docker compose logs -f --tail 100 $($TargetServices -join ' ')"
}
