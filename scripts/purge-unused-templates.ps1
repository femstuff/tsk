# Удаляет шаблоны без заявок и связанных файлов (нужен запущенный backend на :8080).
$base = "http://localhost:8080/api/v1"
$templates = (Invoke-RestMethod "$base/document-templates").items
$jobs = (Invoke-RestMethod "$base/document-jobs").items
$used = $jobs | ForEach-Object { $_.templateId } | Select-Object -Unique

foreach ($t in $templates) {
    if ($used -contains $t.id) {
        Write-Host "SKIP (in use): $($t.name) [$($t.id)]"
        continue
    }
    try {
        Invoke-WebRequest -Method DELETE -Uri "$base/document-templates/$($t.id)" | Out-Null
        Write-Host "DELETED: $($t.name) [$($t.id)]"
    } catch {
        Write-Warning "FAILED $($t.id): $_"
    }
}
