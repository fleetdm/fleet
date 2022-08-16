package msrc

type MicrosoftProduct struct {
	ID       uint
	FullName string
}

func NewMicrosoftProduct(pID uint, fullName string) MicrosoftProduct {
	return MicrosoftProduct{
		ID:       pID,
		FullName: fullName,
	}
}
