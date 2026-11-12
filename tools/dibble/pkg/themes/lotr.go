package themes

func init() {
	Register(Theme{
		Name:    "lotr",
		Display: "The Lord of the Rings",
		Domain:  "middleearth.test",
		Users: []Person{
			{"Frodo", "Baggins", "frodo"},
			{"Samwise", "Gamgee", "sam"},
			{"Meriadoc", "Brandybuck", "merry"},
			{"Peregrin", "Took", "pippin"},
			{"Gandalf", "the Grey", "gandalf"},
			{"Aragorn", "Elessar", "aragorn"},
			{"Legolas", "Greenleaf", "legolas"},
			{"Gimli", "son of Glóin", "gimli"},
			{"Boromir", "of Gondor", "boromir"},
		},
		Teams: []string{
			"The Fellowship", "Rivendell", "Rohan", "Gondor",
			"Lothlórien", "Erebor", "Mordor",
		},
		Policies: []Named{
			{"No One Ring in registry", "Inventory must not contain the One Ring"},
			{"Second breakfast permitted", "Hobbit dietary compliance"},
			{"Path of Caradhras blocked", "Use Moria pass instead"},
			{"You shall not pass", "Balrog firewall rule"},
		},
		Software: []Named{
			{"Palantír Viewer", "Use with extreme caution"},
			{"Sting Sensor", "Glows blue near orcs"},
			{"Mithril Patch Manager", "Light yet strong"},
		},
		Labels: []string{
			"ring-bearer", "took-took",
			"speaks-elvish", "knows-the-old-songs",
		},
		Scripts: []Named{
			{"cast-into-mount-doom.sh", "Final disposal procedure"},
			{"summon-eagles.ps1", "Late but effective"},
		},
	})
}
