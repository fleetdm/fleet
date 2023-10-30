package dataflatten

import (
	"encoding/json"
	"fmt"
	"os"
)

func JsonFile(file string, opts ...FlattenOpts) ([]Row, error) {
	rawdata, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return Json(rawdata, opts...)
}

func Json(rawdata []byte, opts ...FlattenOpts) ([]Row, error) {
	var data interface{}

	if err := json.Unmarshal(rawdata, &data); err != nil {
		return nil, fmt.Errorf("unmarshalling json: %w", err)
	}

	return Flatten(data, opts...)
}
