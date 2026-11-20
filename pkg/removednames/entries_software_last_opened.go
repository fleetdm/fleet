package removednames

func init() { register(softwareLastOpenedEntries...) }

var softwareLastOpenedEntries = []Entry{
	serverConfigEntry("osquery.min_software_last_opened_at_diff", ""),
}
