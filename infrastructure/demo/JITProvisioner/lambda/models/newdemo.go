package models

type NewDemo struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
	Breed string `json:"breed"`
}

type NewDemoParams struct {
	Name string `query:"name"`
}
