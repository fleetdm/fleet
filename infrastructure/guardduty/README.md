basing the architecture off of https://docs.aws.amazon.com/prescriptive-guidance/latest/patterns/use-terraform-to-automatically-enable-amazon-guardduty-for-an-organization.html but using workspaces instead of templates.

Use apply.sh to automatically apply the terraform code in all regions. There is an apply.sh in both this folder and the members folder. The findings folder exists in only one region, so just do a normal apply there.
