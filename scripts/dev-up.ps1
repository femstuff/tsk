[CmdletBinding()]
param(
    [switch]$SkipBuild,
    [switch]$NoWait
)

$ErrorActionPreference = "Stop"

$RepoRoot = Split-Path -Parent $PSScriptRoot
$DefaultServices = @("postgres", "backend-api", "admin-web", "prometheus", "grafana")

function Invoke-DockerCommand {
    param(
        [string[]]$Arguments,
        [string]$Description
    )

    Write-Host "==> $Description" -ForegroundColor Cyan
    & docker @Arguments

    if ($LASTEXITCODE -ne 0) {
        throw "Command failed: docker $($Arguments -join ' ')"
    }
}

function Write-NetworkHint {
    Write-Warning "Docker still needs network access the first time it pulls required base images from Docker Hub."
    Write-Host "If the images were built successfully before, retry without rebuilding:" -ForegroundColor Yellow
    Write-Host "  .\scripts\dev-up.ps1 -SkipBuild" -ForegroundColor Yellow
}

Set-Location $RepoRoot

try {
    Get-Command docker -ErrorAction Stop | Out-Null
    Invoke-DockerCommand -Arguments @("compose", "version") -Description "Checking Docker Compose availability"
    Invoke-DockerCommand -Arguments @("info") -Description "Checking Docker Desktop daemon"

    if (-not $SkipBuild) {
        Invoke-DockerCommand -Arguments @("compose", "build", "backend-api", "admin-web") -Description "Building backend and admin images"
    }

    $UpArguments = @("compose", "up", "-d", "--remove-orphans")

    if ($SkipBuild) {
        $UpArguments += "--no-build"
    }

    if (-not $NoWait) {
        $UpArguments += "--wait"
    }

    $UpArguments += $DefaultServices
    Invoke-DockerCommand -Arguments $UpArguments -Description "Starting backend, admin, Prometheus, and Grafana"

    Write-Host ""
    Write-Host "Local services are available at:" -ForegroundColor Green
    Write-Host "  PostgreSQL: postgres://tsk:tsk@localhost:5432/tsk"
    Write-Host "  API:        http://localhost:8080"
    Write-Host "  Admin web:  http://localhost:5173"
    Write-Host "  Prometheus: http://localhost:9090"
    Write-Host "  Grafana:    http://localhost:3000 (admin / admin)"
}
catch {
    $message = $_.Exception.Message

    if ($message -match "TLS handshake timeout" -or $message -match "context deadline exceeded" -or $message -match "failed to resolve source metadata") {
        Write-NetworkHint
    }

    throw
}
