package themes

func init() {
	Register(Theme{
		Name:    "ghibli",
		Display: "Studio Ghibli",
		Domain:  "spiritedaway.test",
		Users: []Person{
			{"Totoro", "", "totoro"},
			{"Chihiro", "Ogino", "chihiro"},
			{"Howl", "Jenkins", "howl"},
			{"Sophie", "Hatter", "sophie"},
			{"Kiki", "", "kiki"},
			{"San", "", "san"},
			{"Ashitaka", "", "ashitaka"},
			{"Ponyo", "", "ponyo"},
			{"Calcifer", "", "calcifer"},
		},
		Teams: []string{
			"Spirited Bathhouse", "Moving Castle", "Iron Town",
			"Witch's Delivery Service", "Laputa Sky Castle", "Forest Spirits",
		},
		Policies: []Named{
			{"No name-stealing", "Yubaba contract guard"},
			{"Catbus availability", "On-demand transit ready"},
			{"Soot-sprite hygiene", "Konpeito stock check"},
		},
		Software: []Named{
			{"Catbus Transit", "12 legs, very punctual"},
			{"Calcifer Heater", "Don't move out of the hearth"},
			{"Howl's Door Compass", "Four destinations, color-coded"},
		},
		Labels: []string{
			"soot-sprite-detected", "name-not-stolen",
			"forest-spirit-blessed", "ponyo-friendly",
		},
		Scripts: []Named{
			{"summon-catbus.sh", "Wait 12 seconds at the stop"},
			{"feed-calcifer.sh", "Use bacon and eggs"},
		},
	})
}
