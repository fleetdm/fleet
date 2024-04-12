package file

import (
	"log"

	peparser "github.com/saferwall/pe"
)

func GetPEInfo(r []byte) (string, string, error) {
	pep, err := peparser.NewBytes(r, &peparser.Options{})
	if err != nil {
		log.Fatalf("Error while opening file: %s, reason: %v", "", err)
	}
	pep.Parse()
	v, err := pep.ParseVersionResources()
	if err != nil {
		log.Fatalf("Error while opening file: %s, reason: %v", "", err)
	}
	return v["ProductName"], v["ProductVersion"], nil

	// f, err := pe.NewFile(r)
	//if err != nil {
	//	return "", "", err
	//}
	//for _, s := range f.Symbols {
	//	fmt.Println(s.Name)
	//}
	//for _, section := range f.Sections {
	//	fmt.Println(section.Name)
	//	if section.Name == ".rsrc" {
	//		fmt.Printf("Found %s section\n", section.Name)
	//		//section.VirtualAddress
	//		//d, err := section.Data()
	//		//if err != nil {
	//		//	return "", "", err
	//		//}
	//		// fmt.Println(string(d))
	//	}
	//}
	return "", "", nil
}
