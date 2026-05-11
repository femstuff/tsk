[CmdletBinding()]
param(
    [switch]$RemoveContainers
)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$DefaultServices = @("postgres", "backend-api", "admin-web", "prometheus", "grafana")

Set-Location $RepoRoot

& docker compose stop @DefaultServices
if ($LASTEXITCODE -ne 0) {
    throw "Command failed: docker compose stop $($DefaultServices -join ' ')"
}

if ($RemoveContainers) {
    & docker compose rm -f @DefaultServices
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed: docker compose rm -f $($DefaultServices -join ' ')"
    }
}
