# Stops common AI coding agents on Windows and keeps them from starting again
# until restore-ai-agents.ps1 runs. Each agent's command is renamed with a
# .fleet-disabled suffix, so the restore script can put it back.
#
# Runs as SYSTEM through Fleet. A user who owns an install (for example, in
# their profile) can rename it back, so pair this with conditional access.

$Suffix = ".fleet-disabled"
$Commands = @("claude", "codex", "copilot", "cursor-agent", "gemini")
$Profiles = Get-ChildItem -Path "C:\Users" -Directory -ErrorAction SilentlyContinue

$Candidates = foreach ($p in $Profiles) {
    foreach ($cmd in $Commands) {
        Join-Path $p.FullName ".local\bin\$cmd.exe"
        foreach ($ext in @("", ".cmd", ".ps1")) {
            Join-Path $p.FullName "AppData\Roaming\npm\$cmd$ext"
        }
    }
    Join-Path $p.FullName "AppData\Local\Programs\cursor"
}

# Stop processes first so their files aren't locked when renamed.
Get-Process -Name ($Commands + "Cursor") -ErrorAction SilentlyContinue |
    Stop-Process -Force -ErrorAction SilentlyContinue

# npm installs run as "node.exe ...\<package>\...", so match on the package path.
$Pattern = '@anthropic-ai[\\/]claude-code|@openai[\\/]codex|@github[\\/]copilot|@google[\\/]gemini-cli|cursor-agent'
Get-CimInstance Win32_Process -Filter "Name = 'node.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.CommandLine -match $Pattern } |
    ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }

Start-Sleep -Seconds 2

$Disabled = 0
foreach ($path in $Candidates) {
    if (Test-Path -LiteralPath $path) {
        try {
            Rename-Item -LiteralPath $path -NewName ((Split-Path $path -Leaf) + $Suffix) -ErrorAction Stop
            Write-Output "Disabled: $path"
            $Disabled++
        } catch {
            Write-Output "Failed to disable: $path ($($_.Exception.Message))"
        }
    }
}

Write-Output "Disabled $Disabled AI agent install(s) and stopped running agents."
exit 0
