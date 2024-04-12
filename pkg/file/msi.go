package file

import (
	"bytes"
	"fmt"
	"io"

	"github.com/richardlehane/mscfb"
	"github.com/sassoftware/relic/v7/lib/comdoc"
)

func GetMSIInfo(r io.ReaderAt, size int64) (string, string, error) {

	doc, err := mscfb.New(r)
	if err != nil {
		return "", "", fmt.Errorf("parsing table: %w", err)
	}
	for _, f := range doc.File {
		fmt.Println("====================", f.Name)
		var b []byte
		_, err := f.Read(b)
		if err != nil {
			return "", "", fmt.Errorf("rrrrrrrrrrrrrread table: %w", err)
		}
		fmt.Println(string(b))
	}
	c, err := comdoc.ReadFile(r)
	if err != nil {
		return "", "", fmt.Errorf("reading comdoc file: %w", err)
	}
	for _, f := range c.Files {
		name := msiDecodeName(f.Name())
		fmt.Println(name)
	}

	e, err := c.ListDir(nil)
	if err != nil {
		return "", "", fmt.Errorf("listing dir: %w", err)
	}
	for _, ee := range e {
		name := msiDecodeName(ee.Name())
		// fmt.Println(name)
		if name == "Table.File" {
			fmt.Println("====================")

			//if strings.Contains(ee.Name(), "SummaryInformation") {
			o, err := c.ReadStream(ee)
			if err != nil {
				return "", "", fmt.Errorf("reading file stream: %w", err)
			}

			b, err := io.ReadAll(o)
			if err != nil {
				return "", "", fmt.Errorf("reading bytes: %w", err)
			}

			br := bytes.NewReader(b)
			doc, err := mscfb.New(br)
			if err != nil {
				return "", "", fmt.Errorf("parsing table: %w", err)
			}

			fmt.Println(doc)
		}

	}
	return "", "", nil
}

func msiDecodeName(msiName string) string {
	out := ""
	for _, x := range msiName {
		if x >= 0x3800 && x < 0x4800 {
			x -= 0x3800
			out += string(msiDecodeRune(x&0x3f)) + string(msiDecodeRune(x>>6))
		} else if x >= 0x4800 && x < 0x4840 {
			x -= 0x4800
			out += string(msiDecodeRune(x))
		} else if x == 0x4840 {
			out += "Table."
		} else {
			out += string(x)
		}
	}
	return out
}

func msiDecodeRune(x rune) rune {
	if x < 10 {
		return x + '0'
	} else if x < 10+26 {
		return x - 10 + 'A'
	} else if x < 10+26+26 {
		return x - 10 - 26 + 'a'
	} else if x == 10+26+26 {
		return '.'
	} else {
		return '_'
	}
}
