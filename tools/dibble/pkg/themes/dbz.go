package themes

func init() {
	Register(Theme{
		Name:    "dbz",
		Display: "Dragon Ball Z",
		Domain:  "capsulecorp.test",
		Users: []Person{
			{"Son", "Goku", "goku"},
			{"Vegeta", "", "vegeta"},
			{"Piccolo", "", "piccolo"},
			{"Son", "Gohan", "gohan"},
			{"Krillin", "", "krillin"},
			{"Bulma", "Briefs", "bulma"},
			{"Trunks", "", "trunks"},
			{"Tien", "Shinhan", "tien"},
		},
		Teams: []string{
			"Z Fighters", "Capsule Corp", "Frieza Force",
			"Ginyu Force", "Cell Saga Survivors", "Saiyan Royal Family",
		},
		Policies: []Named{
			{"Power level > 9000", "Scouter reading threshold"},
			{"Senzu bean inventory", "At least one bean on file"},
			{"No Cell installations", "Block bio-android packages"},
			{"Kamehameha rate limit", "Per-host energy quota"},
		},
		Software: []Named{
			{"Scouter Firmware", "Don't trust readings over 9000"},
			{"Dragon Radar", "Locates all seven balls"},
			{"Hyperbolic Time Chamber", "1-day-per-year scheduler"},
		},
		Labels: []string{
			"super-saiyan", "namekian",
			"earthling", "androids-allowed",
		},
		Scripts: []Named{
			{"charge-kamehameha.sh", "Five-second windup"},
			{"open-capsule.ps1", "Vehicle deployment"},
		},
	})
}
