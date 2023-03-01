# Migrating FleetDM from Dogfood to Terraform on AWS

The community feedback on the new Terraform module for Fleet has been fantastic. In that feedback, we discovered a need to clarify migrating from the current Dogfood code to the Terraform module. Due to the large variety of situations, there is no hard and fast "this is how to do it" that we can provide, but I'll present two methods that should work in most situations.

# Snapshot method (easiest, with downtime)

The easiest method is to take a snapshot of the existing database and then pass that into the module when you apply. The snapshot will cause downtime since it will recreate all resources, but based on customer feedback, this is acceptable in most situations.

Here is a step-by-step guide on how to migrate using this method:

1.  Comb through the Terraform code, removing all code from Dogfood and keeping the code that your team added. Do not apply until step 6
2.  Add in the module code from the example here: [https://github.com/fleetdm/fleet/tree/main/terraform/example](https://github.com/fleetdm/fleet/tree/main/terraform/example)
3.  Rewrite the changes your team made so that its compatible with the module (hint: you can use `terraform validate` to ensure it will work)
4.  Take a database snapshot: [https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_CreateSnapshot.html](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/USER_CreateSnapshot.html)
5.  Make an edit to the module block so that you pass the snapshot ARN into the module. It should be the `snapshot_identifier` field of the `rds_config` variable.
6.  Now run `terraform apply -target module.main.module.vpc`. Change the target to match what you named the module if you changed the name.
7.  Now run `terraform apply` to apply the rest of the changes.
8.  Check out our library of addons to Fleet here: [https://github.com/fleetdm/fleet/tree/main/terraform/addons](https://github.com/fleetdm/fleet/tree/main/terraform/addons)
9.  Add any addons that you want.
10.  Terraform apply after adding the addons. You might need to target the addon with something like `terraform apply -target module.<addon>` first so that IAM policies are created. Terraform can get confused if the policies are not available at plan time.

# Resource-based migration method (hardest, minimal to no downtime)

Resource-based migration is a more complicated method but can result in little to no downtime. In short, terraform added a new feature that lets users rename resources: [https://developer.hashicorp.com/terraform/language/modules/develop/refactoring](https://developer.hashicorp.com/terraform/language/modules/develop/refactoring)

We can use this feature to rename most or even all resources, resulting in less downtime.

Below is the code we have written to migrate the "heavy" resources to the module:

```
moved {
  from = module.vpc
  to   = module.main.module.vpc
}

moved {
  from = module.aurora_mysql
  to = module.main.module.byo-vpc.module.rds
}

moved {
  from = aws_elasticache_replication_group.default
  to = module.main.module.byo-vpc.module.redis.aws_elasticache_replication_group.default
}
```

This method does not target all resources and will still result in downtime, but it should be much lower. Here is a step-by-step guide to help you through this process:

1.  Comb through the Terraform code, removing all code from Dogfood and keeping the code that your team added. Do not apply until step 4.
2.  Add in the module code from the example here: [https://github.com/fleetdm/fleet/tree/main/terraform/example](https://github.com/fleetdm/fleet/tree/main/terraform/example)
3.  Add in the provided code anywhere in the code base.
4.  Run `terraform apply -target module.main.module.vpc`.
5.  Run `terraform plan`, carefully examining the provided plan. Do not actually run the apply yet.
6.  Find and note down any resources being destroyed that you do not want to be destroyed.
7.  Write a block like the above that migrates it to the module format. Some resources might be in an add-on module. In this case, add the add-on to your code and go to step 4 again.
8.  Go back to step 4, and repeat until everything is correct for your needs.
9.  Now run `terraform apply` to apply the rest of the changes.
10.  Check out our library of addons to Fleet here: [https://github.com/fleetdm/fleet/tree/main/terraform/addons](https://github.com/fleetdm/fleet/tree/main/terraform/addons)
11.  Add any addons that you want.
12.  Terraform apply after adding the addons. You might need to target the addon with something like `terraform apply -target module.<addon>` first so that IAM policies are created. Terraform can get confused if the policies are not available at plan time.
