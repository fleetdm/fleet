package themes

func init() {
	Register(Theme{
		Name:    "robin_williams",
		Display: "Robin Williams characters",
		Domain:  "naunau.test",
		Users: []Person{
			{"Genie", "of the Lamp", "genie"},
			{"Mrs.", "Doubtfire", "doubtfire"},
			{"John", "Keating", "keating"},
			{"Sean", "Maguire", "sean"},
			{"Peter", "Banning", "peterpan"},
			{"Alan", "Parrish", "alan"},
			{"Patch", "Adams", "patch"},
			{"Adrian", "Cronauer", "adrian"},
			{"Mork", "of Ork", "mork"},
		},
		Teams: []string{
			"Cave of Wonders", "Welton Academy", "Jumanji Survivors",
			"Hillside Hospital", "Saigon Radio AFRS", "Mount Hope",
		},
		Policies: []Named{
			{"Carpe Diem reminder", "Seize the day, daily"},
			{"No phenomenal cosmic power without itty-bitty living space", "Privilege bounds check"},
			{"Nanu nanu greeting required", "Mork-compliant SSH banner"},
		},
		Software: []Named{
			{"Lamp Polish Pro", "Three wishes maximum"},
			{"Jumanji Board Sim", "Do not run on weekends"},
			{"Patch.MD Clinic Suite", "Laughter is the best metric"},
		},
		Labels: []string{
			"oh-captain-my-captain", "second-star-to-the-right",
			"good-morning-vietnam", "shazbot",
		},
		Scripts: []Named{
			{"rub-the-lamp.sh", "Yields one (1) wish"},
			{"goooooood-morning.sh", "Radio broadcast bootstrap"},
		},
	})
}
