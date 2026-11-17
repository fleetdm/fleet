package removednames

func init() {
	register(orgLogoKeyEntries...)
}

// orgLogoKeyEntries are the deprecated organization logo URL settings, replaced by the dark and light mode variants.
var orgLogoKeyEntries = []Entry{
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "org_logo_url", New: "org_logo_url_dark_mode"},
	{Kind: KindJSONKey, Scope: ScopeRequest | ScopeResponse, Old: "org_logo_url_light_background", New: "org_logo_url_light_mode"},
	{Kind: KindYAMLKey, Scope: ScopeRequest, Old: "org_settings.org_info.org_logo_url", New: "org_settings.org_info.org_logo_url_dark_mode"},
	{
		Kind: KindYAMLKey, Scope: ScopeRequest,
		Old: "org_settings.org_info.org_logo_url_light_background", New: "org_settings.org_info.org_logo_url_light_mode",
	},
}
