# Restores AI coding agents on Windows that stop-ai-agents.ps1 disabled.

$Suffix = ".fleet-disabled"
$Profiles = Get-ChildItem -Path "C:\Users" -Directory -ErrorAction SilentlyContinue

$Dirs = foreach ($p in $Profiles) {
    Join-Path $p.FullName ".local\bin"
    Join-Path $p.FullName "AppData\Roaming\npm"
    Join-Path $p.FullName "AppData\Local\Programs"
}

$Restored = 0
foreach ($dir in $Dirs) {
    if (-not (Test-Path -LiteralPath $dir)) { continue }
    Get-ChildItem -LiteralPath $dir -Filter "*$Suffix" -Force -ErrorAction SilentlyContinue | ForEach-Object {
        $original = $_.FullName.Substring(0, $_.FullName.Length - $Suffix.Length)
        if (Test-Path -LiteralPath $original) {
            Write-Output "Skipped, $original already exists: $($_.FullName)"
            return
        }
        try {
            Rename-Item -LiteralPath $_.FullName -NewName (Split-Path $original -Leaf) -ErrorAction Stop
            Write-Output "Restored: $original"
            $Restored++
        } catch {
            Write-Output "Failed to restore: $($_.FullName) ($($_.Exception.Message))"
        }
    }
}

Write-Output "Restored $Restored AI agent install(s)."
exit 0
