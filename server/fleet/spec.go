package fleet

// BaseItem provides path/paths fields for types that can reference external
// files in GitOps YAML configurations.
type BaseItem struct {
	Path  *string `json:"path"`
	Paths *string `json:"paths"`
}

// SupportsFileInclude is implemented by types that can reference external
// files via path/paths fields in GitOps YAML.
type SupportsFileInclude interface {
	GetBaseItem() BaseItem
	SetBaseItem(v BaseItem)
}

// GetBaseItem returns the current BaseItem value.
// Types that embed BaseItem inherit this method via promotion.
func (b *BaseItem) GetBaseItem() BaseItem {
	return *b
}

// SetBaseItem sets the BaseItem value.
// Types that embed BaseItem inherit this method via promotion.
func (b *BaseItem) SetBaseItem(v BaseItem) {
	*b = v
}
