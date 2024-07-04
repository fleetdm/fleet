$exeFilePath = "${env:INSTALLER_PATH}"

# extract the name of the executable to use as the sub-directory name
$exeName = [System.IO.Path]::GetFileName($exeFilePath)
$subDir = [System.IO.Path]::GetFileNameWithoutExtension($exeFilePath)

# determine the correct Program Files directory based on OS architecture
$destinationPath = Join-Path -Path $env:ProgramFiles -ChildPath $subDir
$destinationExePath = Join-Path -Path $destinationPath -ChildPath $exeName

# remove only the exe file, while at runtime other files could have been
# created in this folder, this is a naive approach to prevent forcing us to
# remove important folders by crafting a malicious file name.
Remove-Item -Path $destinationExePath
