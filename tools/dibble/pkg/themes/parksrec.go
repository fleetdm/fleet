package themes

func init() {
	Register(Theme{
		Name:    "parksrec",
		Display: "Parks and Recreation",
		Domain:  "pawneeparks.test",
		Users: []Person{
			{"Leslie", "Knope", "leslie"},
			{"Ron", "Swanson", "ron"},
			{"Tom", "Haverford", "tom"},
			{"April", "Ludgate", "april"},
			{"Andy", "Dwyer", "andy"},
			{"Ben", "Wyatt", "ben"},
			{"Donna", "Meagle", "donna"},
			{"Jerry", "Gergich", "jerry"},
		},
		Teams: []string{
			"Pawnee Parks Department", "Eagleton", "Snakehole Lounge Staff",
			"Entertainment 720", "Mouse Rat", "Pawnee Goddesses",
		},
		Policies: []Named{
			{"Breakfast food only check", "Bacon, eggs, and waffles required"},
			{"No Eagleton software", "Block all rival-town origin packages"},
			{"Treat Yo Self readiness", "Once a year only"},
			{"Ron's privacy posture", "No tracking software allowed"},
		},
		Software: []Named{
			{"Pawnee Government Portal", "More efficient than Eagleton's"},
			{"Rent-a-Swag", "Tom's first business"},
			{"DJ Roomba", "Just keeps roaming"},
		},
		Labels: []string{
			"loves-breakfast", "hates-eagleton",
			"mouse-rat-fan", "literally-treat-yo-self",
		},
		Scripts: []Named{
			{"build-pit-park.sh", "Lot 48 mobilization"},
			{"deploy-li_l-sebastian.sh", "5000 candles in the wind"},
		},
	})
}
