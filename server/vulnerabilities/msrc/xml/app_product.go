package xml

import "strings"

// AppProducts traverses the ProductBranch tree returning Microsoft application products
// like Edge, Office 365, etc.
func (b *ProductBranch) AppProducts() []Product {
	var r []Product
	queue := []ProductBranch{*b}

	// Product families we care about for apps
	// Note: "Browser" contains Microsoft Edge, "Microsoft Office" contains Office products
	appFamilies := map[string]bool{
		"Browser":          true,
		"Microsoft Office": true,
		"Developer Tools":  true,
		"Apps":             true,
	}

	// Product name prefixes we want to include
	appPrefixes := []string{
		"Microsoft Edge",
		"Microsoft 365 Apps",
		"Microsoft Office",
	}

	for len(queue) > 0 {
		next := queue[0]

		if next.Type == "Product Family" && appFamilies[next.Name] {
			for _, p := range next.Products {
				for _, prefix := range appPrefixes {
					if strings.HasPrefix(p.FullName, prefix) {
						r = append(r, p)
						break
					}
				}
			}
		}

		queue = queue[1:]
		queue = append(queue, next.Branches...)
	}

	return r
}
