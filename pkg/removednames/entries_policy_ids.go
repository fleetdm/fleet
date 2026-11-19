package removednames

func init() { register(policyIDsEntries...) }

// policyIDsEntries are the failing policies webhook's policy list, replaced by webhooks_and_tickets_enabled on each policy.
var policyIDsEntries = []Entry{
	yamlEntry("org_settings.webhook_settings.failing_policies_webhook.policy_ids", ""),
	yamlEntry("settings.webhook_settings.failing_policies_webhook.policy_ids", ""),
}
