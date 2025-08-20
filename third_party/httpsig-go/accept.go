package httpsig

import (
	"fmt"

	sfv "github.com/dunglas/httpsfv"
)

type AcceptSignature struct {
	Profile   SigningProfile
	MetaNonce string // 'nonce'
	MetaKeyID string // 'keyid'
	MetaTag   string // 'tag' - No default. A value must be provided if the parameter is in Metadata.
}

func ParseAcceptSignature(acceptHeader string) (AcceptSignature, error) {
	as := AcceptSignature{}
	acceptDict, err := sfv.UnmarshalDictionary([]string{acceptHeader})
	if err != nil {
		return as, newError(ErrInvalidAcceptSignature, "Unable to parse Accept-Signature value", err)
	}
	profiles := acceptDict.Names()
	if len(profiles) == 0 {
		return as, newError(ErrMissingAcceptSignature, "No Accept-Signature value")
	}

	label := profiles[0]
	profileItems, _ := acceptDict.Get(label)
	profileList, isList := profileItems.(sfv.InnerList)
	if !isList {
		return as, newError(ErrInvalidAcceptSignature, "Unable to parse Accept-Signature value. Accept-Signature must be a dictionary.")
	}

	fields := []string{}
	for _, componentItem := range profileList.Items {
		field, ok := componentItem.Value.(string)
		if !ok {
			return as, newError(ErrInvalidAcceptSignature, fmt.Sprintf("Invalid signature component '%v', Components must be strings", componentItem.Value))

		}
		fields = append(fields, field)
	}
	as.Profile = SigningProfile{
		Fields:   Fields(fields...),
		Label:    label,
		Metadata: []Metadata{},
	}

	md := metadataProviderFromParams{profileList.Params}
	for _, meta := range profileList.Params.Names() {
		as.Profile.Metadata = append(as.Profile.Metadata, Metadata(meta))
		switch Metadata(meta) {
		case MetaNonce:
			as.MetaNonce, _ = md.Nonce()
		case MetaAlgorithm:
			alg, _ := md.Alg()
			as.Profile.Algorithm = Algorithm(alg)
		case MetaKeyID:
			as.MetaKeyID, _ = md.KeyID()
		case MetaTag:
			as.MetaTag, _ = md.Tag()

		}
	}

	return as, nil

}
