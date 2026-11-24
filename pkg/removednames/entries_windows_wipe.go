package removednames

func init() {
	register(WipeHostWindowsParam)
}

// WipeHostWindowsParam is the windows object Wipe host took to pick doWipe or doWipeProtected. Windows hosts are now always wiped with
// doWipeProtected, and Reset host replaces doWipe for repurposing. Context keeps it to Wipe host: windows is a common key name.
var WipeHostWindowsParam = Entry{
	Kind: KindJSONKey, Scope: ScopeRequest, Old: "windows", Context: "POST " + pathPrefix + "/hosts/{id}/wipe",
}
