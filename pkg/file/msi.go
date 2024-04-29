package file

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/sassoftware/relic/v7/lib/comdoc"
)

func ExtractMSIMetadata(r io.Reader) (name, version string, shaSum []byte, err error) {
	//doc, err := mscfb.New(r)
	//if err != nil {
	//	return "", "", fmt.Errorf("parsing table: %w", err)
	//}
	//for _, f := range doc.File {
	//	fmt.Println("====================", f.Name)
	//	var b []byte
	//	_, err := f.Read(b)
	//	if err != nil {
	//		if err == io.EOF {
	//			fmt.Println("EOF")
	//			continue
	//		}
	//		return "", "", fmt.Errorf("rrrrrrrrrrrrrread table: %w", err)
	//	}
	//	fmt.Println(">>>", strconv.Quote(string(b)), "<<<")
	//}

	h := sha256.New()
	r = io.TeeReader(r, h)
	b, err := io.ReadAll(r)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to read all content: %w", err)
	}

	rr := bytes.NewReader(b)
	c, err := comdoc.ReadFile(rr)
	if err != nil {
		return "", "", nil, fmt.Errorf("reading msi file: %w", err)
	}
	defer c.Close()

	e, err := c.ListDir(nil)
	if err != nil {
		return "", "", nil, fmt.Errorf("listing files in msi: %w", err)
	}
	for _, ee := range e {
		if ee.Type != comdoc.DirStream {
			continue
		}

		name := msiDecodeName(ee.Name())
		fmt.Println(name, ee.Type)

		//if name == "Table._StringData" {
		//	rr, err := c.ReadStream(ee)
		//	if err != nil {
		//		return "", "", fmt.Errorf("opening file stream %s: %w", name, err)
		//	}

		//	b, err := io.ReadAll(rr)
		//	if err != nil {
		//		return "", "", fmt.Errorf("reading file stream %s: %w", name, err)
		//	}
		//	fmt.Println(string(b))

		//	br := bytes.NewReader(b)
		//	doc, err := mscfb.New(br)
		//	if err != nil {
		//		fmt.Println(">>>> failed parsing table ", name, err)
		//		continue
		//		//return "", "", fmt.Errorf("parsing table: %w", err)
		//	}
		//	_ = doc
		//}
		//if bytes.Contains(b, []byte("ProductVersion")) {
		//	fmt.Println("ProductVersion found")
		//}
		//if bytes.Contains(b, []byte("ProductName")) {
		//	fmt.Println("ProductName found")
		//}
		//fmt.Printf("%x\n", b[:9])

		//br := bytes.NewReader(b)
		//doc, err := mscfb.New(br)
		//if err != nil {
		//	fmt.Println(">>>> failed parsing table ", name, err)
		//	continue
		//	//return "", "", fmt.Errorf("parsing table: %w", err)
		//}
		//for _, f := range doc.File {
		//	fmt.Println("=========stream file=====", f.Name)
		//	var b []byte
		//	_, err := f.Read(b)
		//	if err != nil {
		//		if err == io.EOF {
		//			fmt.Println("EOF")
		//			continue
		//		}
		//		return "", "", fmt.Errorf("rrrrrrrrrrrrrread table: %w", err)
		//	}
		//	fmt.Println(">>>", strconv.Quote(string(b)), "<<<")
		//}

		//fmt.Println(doc)
	}

	return "", "", h.Sum(nil), nil
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
