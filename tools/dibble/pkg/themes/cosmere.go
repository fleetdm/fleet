package themes

func init() {
	Register(Theme{
		Name:    "cosmere",
		Display: "Brandon Sanderson's Cosmere",
		Domain:  "cosmere.test",
		Users: []Person{
			{"Kaladin", "Stormblessed", "kaladin"},
			{"Shallan", "Davar", "shallan"},
			{"Dalinar", "Kholin", "dalinar"},
			{"Vin", "", "vin"},
			{"Kelsier", "", "kelsier"},
			{"Sazed", "", "sazed"},
			{"Wax", "Ladrian", "wax"},
			{"Wayne", "", "wayne"},
			{"Hoid", "", "hoid"},
			{"Lift", "", "lift"},
		},
		Teams: []string{
			"Bridge Four", "Survivors of the Final Empire", "Knights Radiant",
			"Worldsingers", "Ghostbloods", "Skybreakers", "Mistborn",
		},
		Policies: []Named{
			{"Stormlight reserve check", "Spheres dun → noncompliant"},
			{"Journey before destination", "Process maturity policy"},
			{"No metal in cabinets", "Allomantic safety"},
			{"Honor the oaths", "Audit log integrity"},
		},
		Software: []Named{
			{"Stormlight Manager", "Bind to highstorms only"},
			{"Allomantic Burn Console", "Eight basic metals"},
			{"Shardplate Diagnostics", "Pre-flight checks"},
		},
		Labels: []string{
			"windrunner-eligible", "mistborn-active",
			"radiant-aspirant", "ghostblood-affiliated",
		},
		Scripts: []Named{
			{"swear-the-oaths.sh", "Idempotent — second-ideal-aware"},
			{"burn-pewter.ps1", "Windows variant for Allomancers"},
		},
	})
}
