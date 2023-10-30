package dataflatten

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func JsonlFile(file string, opts ...FlattenOpts) ([]Row, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Jsonl(f, opts...)
}

func Jsonl(r io.Reader, opts ...FlattenOpts) ([]Row, error) {
	decoder := json.NewDecoder(r)
	var objects []interface{}

	for {
		var object interface{}
		err := decoder.Decode(&object)

		switch {
		case err == nil:
			objects = append(objects, object)
		case err == io.EOF:
			return Flatten(objects, opts...)
		default:
			return nil, fmt.Errorf("unmarshalling jsonl: %w", err)
		}
	}
}
