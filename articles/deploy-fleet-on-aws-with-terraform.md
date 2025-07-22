# Deploy Fleet on AWS with Terraform

The simplest way to get started with Fleet at scale is to use AWS with Terraform.

This workflow takes about 30 minutes to complete and supports between 10 and 350,000 hosts.


### Prerequisites

- A new or existing Amazon Web Services (AWS) account

- An AWS Identity and Access Management (IAM) user with administrator privileges

- The latest version of AWS Command Line Interface `awscli`

- The latest version of HashiCorp Terraform

- A fully qualified domain name (FQDN) for hosting Fleet

### Instructions

1. [Download](https://github.com/fleetdm/fleet-terraform/blob/main/example/main.tf) the Fleet `main.tf` Terraform file.

2. Edit the following variables in the `main.tf` Terraform file you just downloaded to match your environment:
    
```
# Change these to match your environment.
domain_name = "fleet.example.com"
vpc_name = "fleet-vpc"
```

> **Note:** Terraform modules for Fleet features can be enabled and disabled by commenting or uncommenting sections of the code as needed. To learn more about the modules, check out our [AWS with Terraform advanced guide](https://fleetdm.com/docs/deploy/deploy-on-aws-with-terraform).

> **Add a license key:** You can include your [license key as an environment variable](https://fleetdm.com/docs/configuration/fleet-server-configuration#license-key) during this step.

3. Log in to [your AWS account](https://aws.amazon.com/iam/) using your IAM identity.

4. Run a command like the following in Terminal:
    
```
% terraform init ~/Downloads/main.tf
```

> If the file was not downloaded to the downloads folder, ensure that you adjust the file path in the command.

> This step will take around 15 minutes.

5. Run the following command in Terminal:

```
terraform apply -target module.fleet.module.vpc
```

6. Run the following command in Terminal:
    
```
terraform apply -target module.osquery-carve -target module.firehose-logging
```

7. Log in to your AWS Route 53 instance

8. Run the following command in Terminal:

```
terraform apply -target aws_route53_zone.main
```

9. From the Terminal output, obtain the NS records created for the zone and add them to the parent DNS zone in the AWS Route 53 GUI. Ensure you're *adding* the subdomain and its NS records to the parent DNS, not changing the NS records for the parent. For example: if the subdomain is `fleet.acme.com` and the NS record is `ns-420.awsdns-52.com`, *add* this record to the parent domain. 

10. Run the following command in Terminal:
    
```
terraform apply -target module.fleet
```

11. Run the following command in Terminal:
    
```
terraform apply
```

12. That’s it! You should now be able to log in to Fleet and [enroll a host](https://fleetdm.com/docs/using-fleet/enroll-hosts).

<meta name="articleTitle" value="Deploy Fleet on AWS with Terraform">
<meta name="authorGitHubUsername" value="edwardsb">
<meta name="authorFullName" value="Ben Edwards">
<meta name="publishedOn" value="2025-07-17">
<meta name="category" value="guides">
<meta name="articleImageUrl" value="../website/assets/images/articles/deploy-fleet-on-aws-with-terraform-800x450@2x.png">
<meta name="description" value="Learn how to deploy Fleet on AWS.">
