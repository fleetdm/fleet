package efi

const (
	BootUUID       = "8be4df61-93ca-11d2-aa0d-00e098032b8c"
	BootLoaderUUID = "4a67b082-0a4c-41cf-b6c7-440b29bb8c4f"
)

func ReadVarAsBool(uuid, name string) (bool, error) {
	ev, err := ReadVar(uuid, name)
	if err != nil {
		return false, err
	}
	return ev.AsBool()
}

func ReadVarAsUTF16(uuid, name string) (string, error) {
	ev, err := ReadVar(uuid, name)
	if err != nil {
		return "", err
	}
	return ev.AsUTF16()
}

func ReadSecureBoot() (bool, error) {
	return ReadVarAsBool(BootUUID, "SecureBoot")
}

func ReadSetupMode() (bool, error) {
	return ReadVarAsBool(BootUUID, "SetupMode")
}

func ReadLoaderEntrySelected() (string, error) {
	return ReadVarAsUTF16(BootLoaderUUID, "LoaderEntrySelected")
}
