package dataflatten

import (
	"fmt"
	"os"

	"howett.net/plist"
)

func PlistFile(file string, opts ...FlattenOpts) ([]Row, error) {
	rawdata, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return Plist(rawdata, opts...)
}

func Plist(rawdata []byte, opts ...FlattenOpts) ([]Row, error) {
	var data interface{}

	if _, err := plist.Unmarshal(rawdata, &data); err != nil {
		return nil, fmt.Errorf("unmarshalling plist: %w", err)
	}

	return Flatten(data, opts...)
}
