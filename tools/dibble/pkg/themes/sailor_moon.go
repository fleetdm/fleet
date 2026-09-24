package themes

func init() {
	Register(Theme{
		Name:    "sailor_moon",
		Display: "Sailor Moon",
		Domain:  "moonkingdom.test",
		Users: []Person{
			{"Usagi", "Tsukino", "usagi"},
			{"Ami", "Mizuno", "ami"},
			{"Rei", "Hino", "rei"},
			{"Makoto", "Kino", "makoto"},
			{"Minako", "Aino", "minako"},
			{"Mamoru", "Chiba", "mamoru"},
			{"Chibiusa", "", "chibiusa"},
			{"Luna", "", "luna"},
			{"Artemis", "", "artemis"},
		},
		Teams: []string{
			"Inner Senshi", "Outer Senshi", "Dark Kingdom",
			"Black Moon Clan", "Death Busters", "Moon Kingdom",
		},
		Policies: []Named{
			{"Moon Tiara armed", "Frisbee-ready"},
			{"No youma in inventory", "Block demonic processes"},
			{"In the name of the Moon", "Banner-text compliance"},
		},
		Software: []Named{
			{"Moon Tiara Action", "Throw-and-recall enabled"},
			{"Disguise Pen", "Costume changes per minute"},
			{"Luna-P Chat", "Talking-cat console"},
		},
		Labels: []string{
			"crystal-tokyo-ready", "moon-prism-active",
			"talking-cat-detected", "tuxedo-mask-fan",
		},
		Scripts: []Named{
			{"transform.sh", "Idempotent magical girl transformation"},
			{"summon-tuxedo-mask.ps1", "Convenient last-second arrival"},
		},
	})
}
