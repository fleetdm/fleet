package removednames

func init() { register(packEntries...) }

// packEntries are the 2017 packs features that Fleet 5 removed.
var packEntries = []Entry{
	{Kind: KindSpecKind, Scope: ScopeRequest, Old: "pack"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get packs"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get pack"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl get p"},
	{Kind: KindCommand, Scope: ScopeRequest, Old: "fleetctl upgrade-packs"},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/packs", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/packs/schedule", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/packs/schedule/{id}", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/packs/id/{id}", Format: EndpointFormat{}},
	// Also covers DELETE /packs/{name}, which has the same shape.
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/packs/{id}", Format: EndpointFormat{}},
	// Only GET: other methods on this shape never existed, and a wildcard for every method would collide with the paths above.
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/packs/{id}/scheduled", Context: "GET", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/spec/packs", Format: EndpointFormat{}},
	{Kind: KindPath, Scope: ScopeRequest, Old: pathPrefix + "/spec/packs/{name}", Format: EndpointFormat{}},
}
