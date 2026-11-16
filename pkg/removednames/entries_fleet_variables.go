package removednames

func init() { register(fleetVariableEntries...) }

// fleetVariableEntries are the Fleet variables removed from configuration profiles, managed app configurations, and certificate
// authority fields.
var fleetVariableEntries = []Entry{
	{Kind: KindFleetVariable, Scope: ScopeRequest, Old: "$FLEET_VAR_HOST_END_USER_EMAIL_IDP", New: "$FLEET_VAR_HOST_END_USER_IDP_USERNAME"},
}
