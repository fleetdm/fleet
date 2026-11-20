# Installs the runtimes this app needs when they're missing or too old.
& {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'
    $appName = {{APP_NAME}}
    $dependencies = @(
{{DEPENDENCIES}}
    )

    function Get-OSArch {
        $arch = (Get-ItemProperty 'HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment' -ErrorAction SilentlyContinue).PROCESSOR_ARCHITECTURE
        switch ($arch) { 'ARM64' { 'arm64' } 'AMD64' { 'x64' } default { 'x86' } }
    }

    function ConvertTo-Version($text) {
        if ("$text" -match '(\d+(\.\d+){1,3})') { try { return [version]$Matches[1] } catch { } }
        return $null
    }

    # Runtimes register in the 64-bit or the 32-bit registry view depending on the installer.
    function Get-MachineRegistryKeys($subKey) {
        foreach ($view in 'Registry64', 'Registry32') {
            $base = [Microsoft.Win32.RegistryKey]::OpenBaseKey('LocalMachine', $view)
            $key = $base.OpenSubKey($subKey)
            if ($key) { $key }
        }
    }

    function Get-InstalledVersion($dep) {
        $versions = @()
        switch ($dep.Kind) {
            'vcredist' {
                foreach ($key in Get-MachineRegistryKeys "SOFTWARE\Microsoft\VisualStudio\14.0\VC\Runtimes\$($dep.Arch)") {
                    if ($key.GetValue('Installed') -eq 1) { $versions += ConvertTo-Version $key.GetValue('Version') }
                }
            }
            'webview2' {
                foreach ($key in Get-MachineRegistryKeys 'SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}') {
                    $version = ConvertTo-Version $key.GetValue('pv')
                    if ($version -and $version -gt [version]'0.0.0.0') { $versions += $version }
                }
            }
            'dotnet-desktop' {
                $programFiles = if ($env:ProgramW6432) { $env:ProgramW6432 } else { $env:ProgramFiles }
                $roots = @()
                foreach ($key in Get-MachineRegistryKeys "SOFTWARE\dotnet\Setup\InstalledVersions\$($dep.Arch)") {
                    $roots += $key.GetValue('InstallLocation')
                }
                if ($dep.Arch -eq 'x86' -and ${env:ProgramFiles(x86)}) {
                    $roots += Join-Path ${env:ProgramFiles(x86)} 'dotnet'
                } elseif ($dep.Arch -eq (Get-OSArch)) {
                    $roots += Join-Path $programFiles 'dotnet'
                } else {
                    $roots += Join-Path $programFiles "dotnet\$($dep.Arch)"
                }
                foreach ($root in ($roots | Where-Object { $_ } | Select-Object -Unique)) {
                    $shared = Join-Path $root 'shared\Microsoft.WindowsDesktop.App'
                    if (-not (Test-Path -LiteralPath $shared)) { continue }
                    foreach ($dir in Get-ChildItem -LiteralPath $shared -Directory -ErrorAction SilentlyContinue) {
                        # Preview builds ("10.0.0-rc.1") don't satisfy the dependency.
                        if ($dir.Name -notmatch '^\d+(\.\d+){1,3}$') { continue }
                        $version = ConvertTo-Version $dir.Name
                        if ($version -and "$($version.Major)" -eq $dep.Major) { $versions += $version }
                    }
                }
            }
        }
        $versions | Where-Object { $_ } | Sort-Object -Descending | Select-Object -First 1
    }

    function Test-Dependency($dep) {
        $installed = Get-InstalledVersion $dep
        if (-not $installed) { return $false }
        if ($dep.MinVersion -and $installed -lt (ConvertTo-Version $dep.MinVersion)) { return $false }
        return $true
    }

    function Install-Dependency($dep) {
        $arch = if ($dep.Arch -eq 'host') { Get-OSArch } else { $dep.Arch }
        $installer = $dep.Installers[$arch]
        if (-not $installer) { throw "there is no $arch installer for it" }

        # Stage the download where only SYSTEM and Administrators can write.
        $security = New-Object System.Security.AccessControl.DirectorySecurity
        $security.SetAccessRuleProtection($true, $false)
        foreach ($sid in 'S-1-5-18', 'S-1-5-32-544') {
            $security.AddAccessRule((New-Object System.Security.AccessControl.FileSystemAccessRule(
                (New-Object System.Security.Principal.SecurityIdentifier($sid)), 'FullControl',
                'ContainerInherit, ObjectInherit', 'None', 'Allow')))
        }
        $stageDir = Join-Path $env:ProgramData ('fleet-dependency-' + [guid]::NewGuid().ToString('N'))
        [void][System.IO.Directory]::CreateDirectory($stageDir, $security)
        try {
            $file = Join-Path $stageDir 'installer.exe'
            [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
            for ($attempt = 1; ; $attempt++) {
                try {
                    (New-Object System.Net.WebClient).DownloadFile($installer.Url, $file)
                    break
                } catch {
                    if ($attempt -ge 3) {
                        $reason = if ($_.Exception.InnerException) { $_.Exception.InnerException.Message } else { $_.Exception.Message }
                        throw "downloading $($installer.Url) failed: $reason"
                    }
                    Start-Sleep -Seconds 10
                }
            }
            $hash = (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash
            if ($hash -ne $installer.Sha256) {
                throw "the download from $($installer.Url) has SHA-256 $hash, expected $($installer.Sha256)"
            }

            for ($attempt = 1; ; $attempt++) {
                $process = Start-Process -FilePath $file -ArgumentList $dep.Arguments -PassThru -WindowStyle Hidden
                $null = $process.Handle
                if (-not $process.WaitForExit(1800000)) {
                    try { $process.Kill() } catch { }
                    throw 'its installer did not finish within 30 minutes'
                }
                $exitCode = $process.ExitCode
                # 1618: another installation is in progress.
                if ($exitCode -ne 1618 -or $attempt -ge 10) { break }
                Start-Sleep -Seconds 30
            }
        } finally {
            # The installer can stay locked for a few seconds after it exits.
            for ($attempt = 1; $attempt -le 10 -and (Test-Path -LiteralPath $stageDir); $attempt++) {
                Remove-Item -LiteralPath $stageDir -Recurse -Force -ErrorAction SilentlyContinue
                if (Test-Path -LiteralPath $stageDir) { Start-Sleep -Seconds 3 }
            }
        }

        if (-not (Test-Dependency $dep)) { throw "its installer exited with code $exitCode and it is still missing" }
        Write-Host "Installed $($dep.Name) $(Get-InstalledVersion $dep) (exit code $exitCode)."
    }

    foreach ($dep in $dependencies) {
        try {
            if (Test-Dependency $dep) {
                Write-Host "$($dep.Name) $(Get-InstalledVersion $dep) is already installed."
                continue
            }
            Write-Host "$appName needs $($dep.Name). Installing it."
            Install-Dependency $dep
        } catch {
            Write-Host "Error: couldn't install $($dep.Name), which $appName needs: $($_.Exception.Message)"
            Exit 1
        }
    }
}

