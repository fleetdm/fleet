package themes

func init() {
	Register(Theme{
		Name:    "hitchhikers",
		Display: "Hitchhiker's Guide to the Galaxy",
		Domain:  "magrathea.test",
		Users: []Person{
			{"Arthur", "Dent", "arthur"},
			{"Ford", "Prefect", "ford"},
			{"Zaphod", "Beeblebrox", "zaphod"},
			{"Trillian", "McMillan", "trillian"},
			{"Marvin", "Android", "marvin"},
			{"Slartibartfast", "", "slarti"},
			{"Fenchurch", "", "fenchurch"},
			{"Agrajag", "", "agrajag"},
		},
		Teams: []string{
			"Heart of Gold", "Magrathea", "Vogon Constructor Fleet",
			"Restaurant at the End of the Universe", "Mostly Harmless",
		},
		Policies: []Named{
			{"Towel readiness check", "Verifies the host has a towel attached"},
			{"Don't Panic banner", "Asserts the desktop wallpaper says DON'T PANIC"},
			{"Babel fish installed", "Required for galactic-language support"},
			{"Improbability drive disabled in prod", "No accidental whale-summoning"},
			{"Answer-to-life check", "Result must equal 42"},
		},
		Software: []Named{
			{"Babel Fish", "Universal translator"},
			{"Pan Galactic Gargle Blaster", "Do not deploy on Fridays"},
			{"Sub-Etha Sens-O-Matic", "Catch interstellar transit"},
			{"Eddie the Shipboard Computer", "Cheerful and very annoying"},
		},
		Labels: []string{
			"knows-where-towel-is", "vogon-poetry-resistant",
			"improbability-tolerant", "froody-cats",
		},
		Scripts: []Named{
			{"summon-towel.sh", "Ensures a towel is in $HOME"},
			{"play-vogon-poetry.sh", "For interrogation purposes only"},
			{"engage-improbability.ps1", "Windows variant. Use sparingly."},
		},
	})
}
