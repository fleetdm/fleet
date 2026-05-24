package themes

func init() {
	Register(Theme{
		Name:    "goodplace",
		Display: "The Good Place",
		Domain:  "thegoodplace.test",
		Users: []Person{
			{"Eleanor", "Shellstrop", "eleanor"},
			{"Chidi", "Anagonye", "chidi"},
			{"Tahani", "Al-Jamil", "tahani"},
			{"Jason", "Mendoza", "jason"},
			{"Michael", "Demon", "michael"},
			{"Janet", "", "janet"},
			{"Doug", "Forcett", "doug"},
		},
		Teams: []string{
			"The Good Place", "The Bad Place", "The Medium Place",
			"Mindy St. Claire's Cabin", "Judge Gen's Chambers",
		},
		Policies: []Named{
			{"Ethical compliance check", "Has the host done a good deed today?"},
			{"No frozen yogurt detected", "Hard pass on frozen yogurt installs"},
			{"Trolley problem readiness", "Chidi-grade decision-making"},
			{"Forking filter enabled", "All ' fork ' substring detection"},
		},
		Software: []Named{
			{"Janet Void Browser", "Returns anything you ask for"},
			{"Tahani's Charity Tracker", "Name-dropping included"},
			{"Jason's Molotov Manual", "Use with discretion"},
		},
		Labels: []string{
			"good-place-resident", "bad-place-architect",
			"truly-good", "trolley-survivor",
		},
		Scripts: []Named{
			{"compute-ethical-score.sh", "Sum of moral acts since boot"},
			{"summon-janet.sh", "Snap your fingers"},
		},
	})
}
