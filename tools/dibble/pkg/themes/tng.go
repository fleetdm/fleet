package themes

func init() {
	Register(Theme{
		Name:    "tng",
		Display: "Star Trek: The Next Generation",
		Domain:  "ussenterprise.test",
		Users: []Person{
			{"Jean-Luc", "Picard", "picard"},
			{"William", "Riker", "riker"},
			{"Data", "", "data"},
			{"Worf", "", "worf"},
			{"Geordi", "La Forge", "geordi"},
			{"Deanna", "Troi", "troi"},
			{"Beverly", "Crusher", "crusher"},
			{"Wesley", "Crusher", "wesley"},
		},
		Teams: []string{
			"USS Enterprise NCC-1701-D", "Klingon Empire", "Borg Collective",
			"Romulan Star Empire", "Q Continuum", "Ten Forward Staff",
		},
		Policies: []Named{
			{"Earl Grey availability", "Replicator must offer hot tea"},
			{"Prime Directive compliance", "Non-interference posture"},
			{"Shields up readiness", "Default to raised when scanning"},
			{"Make-it-so cadence", "Daily directive execution"},
		},
		Software: []Named{
			{"Holodeck Programs", "Caution: program 9 caused incidents"},
			{"LCARS Console", "Library Computer Access/Retrieval System"},
			{"Tricorder firmware", "v1701.D"},
		},
		Labels: []string{
			"engage-ready", "tea-earl-grey-hot",
			"resistance-is-futile", "klingon-honorable",
		},
		Scripts: []Named{
			{"raise-shields.sh", "Defensive posture"},
			{"engage-warp.sh", "Set course, warp 9"},
		},
	})
}
