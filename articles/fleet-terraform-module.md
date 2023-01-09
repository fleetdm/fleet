# Keep Fleet running smoothly on AWS with the new Terraform module

The #1 way we see Fleet deployed today is via Terraform to AWS. In the past, we used our Dogfood environment as an example of this deployment. Some customers chose to pull this example code from the Dogfood segment of our repo. This article addresses upcoming changes to our Terraform pattern that will disrupt usage of the code in Dogfood, the reasons for these changes, and how you can use our new pattern to avoid breaking changes.

## Reasons for the change

We’re glad the Dogfood environment has been helpful. But Dogfood's Terraform code wasn’t meant for production. Since it was being used for deployment, we decided to test new features in a way that didn’t cause breaking changes, which required extra resources that could’ve gone toward other improvements.

And because we didn’t intend for people to use Dogfood’s Terraform, there were no means in place to see who used the code for deployment. So we wouldn’t know whether to contact you about breaking changes or updates.

We needed a solution that makes it easier to test new features and easier to deploy Fleet to unique environments.

## Introducing the Fleet Terraform module

Modules are the classic solution to this Terraform issue. Making a module with a streamlined interface would let teams deploy Fleet in AWS ASAP. You’d have minimal code to maintain and you could keep environments updated as Fleet evolves.

The basic install had to be as simple as possible. Would you use a module if you had to pass in every single thing to make it work? Probably not.

The new module also had to be flexible enough to support all your needs. That’s why we nested modules within a module. The outermost module deploys Fleet ASAP. The inner modules give you more and more control over how Fleet installs.

This lets lean companies deploy Fleet without customization, while larger organizations can pass in variables for specific departments, such as building an RDS database or a VPC.

Either way, it’s easy to update your infrastructure to Fleet’s new recommendations. All you have to do is pull and apply the new version of the module.

This approach also makes it easier for Fleet to communicate with you. We can clearly indicate any changes with semantic versioning.

## Laying out the module

The module is laid out with an increasing degree of Bring Your Own (BYO). We start with BYO-Nothing (the root module) and work our way to BYO-ECS.

-   BYO-Nothing
    -   BYO-VPC
        -   BYO-Database
            -   BYO-ECS

You can select the level of BYO you want depending on your deployment environment.

We also had to consider the module lifecycle. You might start with a BYO-Nothing level install, but decide to customize the module’s internals later. To make this simpler, we use variables as objects:

```hcl
variable "fleet_config" {
  type = object({
    mem                         = optional(number, 512)
    cpu                         = optional(number, 256)
    image                       = optional(string, "fleetdm/fleet:v4.22.1")
    extra_environment_variables = optional(map(string), {})
    extra_secrets               = optional(map(string), {})
    security_groups             = optional(list(string), null)
    iam_role_arn                = optional(string, null)
    database = object({
      password_secret_arn = string
      user                = string
      database            = string
      address             = string
      rr_address          = optional(string, null)
    })
    redis = object({
      address = string
      use_tls = optional(bool, true)
    })
    awslogs = optional(object({
      name      = optional(string, null)
      region    = optional(string, null)
      prefix    = optional(string, "fleet")
      retention = optional(number, 5)
      }), {
      name      = null
      region    = null
      prefix    = "fleet"
      retention = 5
    })
    loadbalancer = object({
      arn = string
    })
    networking = object({
      subnets         = list(string)
      security_groups = optional(list(string), null)
    })
    autoscaling = optional(object({
      max_capacity                 = optional(number, 5)
      min_capacity                 = optional(number, 1)
      memory_tracking_target_value = optional(number, 80)
      cpu_tracking_target_value    = optional(number, 80)
      }), {
      max_capacity                 = 5
      min_capacity                 = 1
      memory_tracking_target_value = 80
      cpu_tracking_target_value    = 80
    })
  })
  default = {
    mem                         = 512
    cpu                         = 256
    image                       = "fleetdm/fleet:v4.22.1"
    extra_environment_variables = {}
    extra_secrets               = {}
    security_groups             = null
    iam_role_arn                = null
    database = {
      password_secret_arn = null
      user                = null
      database            = null
      address             = null
      rr_address          = null
    }
    redis = {
      address = null
      use_tls = true
    }
    awslogs = {
      name      = null
      region    = null
      prefix    = "fleet"
      retention = 5
    }
    loadbalancer = {
      arn = null
    }
    networking = {
      subnets         = null
      security_groups = null
    }
    autoscaling = {
      max_capacity                 = 5
      min_capacity                 = 1
      memory_tracking_target_value = 80
      cpu_tracking_target_value    = 80
    }
  }
  description = "The configuration object for Fleet itself. Fields that default 
to null will have their respective resources created if not specified."
  nullable    = false
}
```

With an object like this, you can expose it all the way to the root level and allow full customization of internals — no matter the level of BYO at the time of deployment. This also makes variables more organized and understandable compared to a long list of flat variables.

## How to use the module

Here’s a minimal example of using the module:

```hcl
module "main" {
  source          = "git::https://github.com/fleetdm/fleet.git//terraform/"
  certificate_arn = module.acm.acm_certificate_arn
}

module "acm" {
  source  = "terraform-aws-modules/acm/aws"
  version = "4.3.1"

  domain_name = "fleet.loadtest.fleetdm.com"
  zone_id     = data.aws_route53_zone.main.id

  wait_for_validation = true
}

resource "aws_route53_record" "main" {
  zone_id = data.aws_route53_zone.main.id
  name    = "fleet.loadtest.fleetdm.com"
  type    = "A"

  alias {
    name                   = module.main.byo-vpc.byo-db.alb.lb_dns_name
    zone_id                = module.main.byo-vpc.byo-db.alb.lb_zone_id
    evaluate_target_health = true
  }
}

data "aws_route53_zone" "main" {
  name         = "loadtest.fleetdm.com."
  private_zone = false
}
```

You can start with this code and customize it to meet your needs, adding extra
resources on the side or refining the installation for deployment at scale. See the [Terraform GitHub repo](https://github.com/fleetdm/fleet/tree/main/terraform) for a full list of variables.

<meta name="category" value="announcements">
<meta name="authorFullName" value="Zachary Winnerman">
<meta name="authorGitHubUsername" value="zwinnerman-fleetdm">
<meta name="publishedOn" value="2023-01-09">
<meta name="articleTitle" value="Keep Fleet running smoothly on AWS with the new Terraform module">
