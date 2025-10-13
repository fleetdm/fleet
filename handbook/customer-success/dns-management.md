# DNS management

**Responsible team:** [🌦️ Infrastructure Engineer](https://fleetdm.com/handbook/customer-success#team)

---

Fleet manages DNS in Cloudflare using Terraform.  
This page explains how and why we do that.

---

## Purpose

DNS connects everything Fleet runs on the internet.  
We manage it as code to keep it reliable, secure, and transparent.

Infrastructure defined in Terraform can be reviewed, tested, and rolled back.  
That helps us spot mistakes early and prevents silent configuration drift.  
This process also reduces the risk of dangling DNS records that could be abused.

---

## Where DNS lives

All Fleet-managed DNS records are hosted in **Cloudflare**.

The source of truth is the Terraform configuration in:

<https://github.com/fleetdm/confidential/tree/main/infrastructure/cloudflare>

Any record managed by Fleet belongs there.

Subdomain delegations for specific environments live in their own Terraform projects:

- **Load testing:** <https://github.com/fleetdm/fleet/tree/main/infrastructure/loadtesting/terraform>  
- **Fleet managed cloud:** <https://github.com/fleetdm/confidential/tree/main/infrastructure/cloud>

Those delegated zones remain responsible for records inside their scope,  
but the top-level delegation (the NS record in Cloudflare) stays managed in the main Cloudflare repo.

---

## Example record

Terraform keeps each zone’s DNS records in a separate file.  
Here’s an example from `fleetdm_com.tf` that manages a Slack domain verification TXT record.

```hcl
resource "cloudflare_record" "fleetdm_com_txt_slack_domain_verification" {
  zone_id = cloudflare_zone.fleetdm_com.id
  name    = "fleetdm.com"
  type    = "TXT"
  content = "slack-domain-verification=RpK2KmiKKmjmAXayjIhla9FCQfTQLUExoiJAvTVx"
  proxied = false
  comment = "Slack domain verification https://github.com/fleetdm/confidential/issues/12505"
  tags    = []
}
```

Each record includes a descriptive name, record type, and comment linking to the related GitHub issue.  
The comment provides context and traceability for anyone reviewing or debugging later.

---

## How to change a DNS record

1. Create a new branch in `fleetdm/confidential`.  
2. Edit the Terraform in `infrastructure/cloudflare` to add, remove, or update records.  
3. Open a **pull request** to `main`.

   - The GitHub Action runs `terraform plan` automatically.  
   - The plan output appears in the PR checks.

4. When the PR is merged, the same workflow runs `terraform apply` automatically.  
   No one needs to run Terraform manually.

> ⚠️ Changes made directly in the Cloudflare UI are not persistent.  
> They will be lost the next time automation runs.

---

## Continuous checks

A nightly job validates Cloudflare against Terraform.  
It flags:

- Records that drift from the declared state.  
- Dangling or orphaned records that no longer point to active infrastructure.

These checks help catch potential subdomain takeover risks before they become incidents.

---

## Why this way?

Managing DNS through code and automation reflects Fleet’s values.

- **🟠 Ownership:** Every record has a clear history and reviewer.  
- **🟢 Results:** Automation applies approved changes quickly and safely.  
- **🔵 Objectivity:** Drift detection shows the real state, not assumptions.  
- **🟣 Openness:** All changes are public inside Fleet’s GitHub org.  
- **🔴 Empathy:** The process makes life easier for anyone debugging DNS issues later.

---

## Best practices

- Write clear commit messages describing what changed and why.  
- Remove DNS records when infrastructure is retired.  
- Keep TTLs short (300–900 seconds) for records that change often.  
- Avoid editing records in the Cloudflare UI except for emergencies.  
  If you must, follow up with a matching Terraform change in the next PR.  
- Use PR descriptions to give reviewers context, especially for delegations or migration work.  

---

## Emergency override

If a DNS change is needed to fix an outage, you can edit Cloudflare directly.  
After the emergency, update Terraform to reflect the change so drift detection returns to green.

---

## Summary

| Concern | How Fleet handles it |
|:--|:--|
| Hosting | Cloudflare |
| Source of truth | Terraform in `fleetdm/confidential` |
| Change process | Pull request → plan → apply (automated) |
| Manual changes | Discouraged; overwritten later |
| Drift detection | Nightly check + dangling record scan |
| Delegated zones | Managed in environment repos |
| Responsible team | [🌦️ Infrastructure Engineer](https://fleetdm.com/handbook/customer-success#team) |

<meta name="maintainedBy" value="rfairburn">
<meta name="title" value="DNS Management">

