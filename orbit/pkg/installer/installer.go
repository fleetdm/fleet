package installer

import (
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func InstallSoftware(installer *fleet.OrbitSoftwareInstaller) (installOutput, postInstallOutput string, err error) {
	shouldInstall, err := PreConditionCheck(installer.PreInstallCondition)
	if err != nil {
		return "", "", err
	}

	if !shouldInstall {
		return "", "", nil
	}

	installScriptPath, err := FetchScript(installer.InstallScript)
	if err != nil {
		return "", "", err
	}

	postInstallScriptPath, err := FetchScript(installer.PostInstallScript)
	if err != nil {
		return "", "", err
	}

	installerPath, err := FetchInstaller(installer.SoftwareId)
	if err != nil {
		return "", "", err
	}
	defer func() {}() // remove tmp directory and installer

	installOutput, err = RunInstallerScript(installScriptPath, installerPath)
	if err != nil {
		return installOutput, "", err
	}

	postInstallOutput, err = RunInstallerScript(postInstallScriptPath, installerPath)
	if err != nil {
		return installOutput, postInstallOutput, err
	}

	return installOutput, postInstallOutput, nil
}

func PreConditionCheck(query string) (bool, error) {
	return false, nil
}

func FetchInstaller(softwareId string) (path string, err error) {
	// put it in a tmp directory
	return "", nil
}

func FetchScript(scriptId string) (path string, err error) {
	return "", nil
}

func RunInstallerScript(scriptPath, installerPath string) (string, error) {
	// run script in installer directory
	return "", nil
}
