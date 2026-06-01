# Проверяет, что в API есть шаблон сметы (category=estimate, желательно .docx)
$base = "http://localhost:8080/api/v1"
$health = Invoke-RestMethod "$base/health" -ErrorAction SilentlyContinue
if (-not $health) {
    Write-Host "Backend недоступен на $base — сначала: docker compose up -d"
    exit 1
}
$templates = (Invoke-RestMethod "$base/document-templates").items
$estimate = $templates | Where-Object {
    $c = $_.category.ToLower()
    $c -eq "estimate" -or $c -eq "smeta" -or $_.name -match "смет"
}
if ($estimate) {
    Write-Host "OK: найдено шаблонов сметы: $($estimate.Count)"
    $estimate | ForEach-Object { Write-Host "  - $($_.name) [$($_.fileName)] id=$($_.id)" }
} else {
    Write-Host "Шаблон сметы не найден. Пересоберите backend:"
    Write-Host "  docker compose build backend-api"
    Write-Host "  docker compose up -d backend-api"
}
