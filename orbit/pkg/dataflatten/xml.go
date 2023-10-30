package dataflatten

import (
	"fmt"
	"os"

	"github.com/clbanning/mxj"
)

func XmlFile(file string, opts ...FlattenOpts) ([]Row, error) {
	rdr, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	mv, err := mxj.NewMapXmlReader(rdr)
	if err != nil {
		return nil, err
	}

	return Flatten(mv.Old(), opts...)
}

func Xml(rawdata []byte, opts ...FlattenOpts) ([]Row, error) {
	mv, err := mxj.NewMapXml(rawdata)

	if err != nil {
		return nil, fmt.Errorf("mxj parse: %w", err)
	}

	return Flatten(mv.Old(), opts...)
}
