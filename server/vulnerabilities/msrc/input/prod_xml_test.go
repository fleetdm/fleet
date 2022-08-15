package msrc_input

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProdXML(t *testing.T) {
	t.Run("ProductBranchXML", func(t *testing.T) {
		t.Run("#ContainsWinProducts", func(t *testing.T) {
			testCases := []struct {
				nameP    string
				typeP    string
				expected bool
			}{
				{nameP: "Microsoft", typeP: "Vendor", expected: false},
				{nameP: "Windows", typeP: "Product Family", expected: true},
				{typeP: "Product Family", nameP: "ESU", expected: false},
				{typeP: "Product Family", nameP: "Developer Tools", expected: false},
				{typeP: "Product Family", nameP: "Browser", expected: false},
				{typeP: "Product Family", nameP: "Microsoft Office", expected: false},
				{typeP: "Product Family", nameP: "Azure", expected: false},
			}

			for _, tCase := range testCases {
				sut := ProductBranchXML{Name: tCase.nameP, Type: tCase.typeP}
				require.Equal(t, sut.ContainsWinProducts(), tCase.expected)
			}
		})
	})
}
