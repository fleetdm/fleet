package msrc

type MicrosftProduct struct {
	ID   uint
	Name string
}

func NewMicrosoftProduct(pID uint, pName string) MicrosftProduct {
	return MicrosftProduct{
		ID:   pID,
		Name: pName,
	}
}
