# AWS licensing integration

This specification documents the changes we would need in Fleet to integrate with AWS licensing.

## Why

> AWS requires us to offer a pricing option native to AWS Marketplace ~90 days after launch. To achieve this, we need to go through various technical reviews and build an integration to their pricing system.

## Documentation

- https://docs.aws.amazon.com/marketplace/latest/userguide/container-license-manager-integration.html
- https://aws.amazon.com/blogs/awsmarketplace/aws-license-manager-integration-aws-marketplace-ami-containers-contract-pricing/

## Successful integration?

How does AWS determine whether we integrated to their licensing system the way they expect? (TBD!)

## Product requirements

Following are the requirements from the [epic issue](https://github.com/fleetdm/fleet/issues/8004).
1. Use the contract-based pricing option w/ our current pricing ($1/host/month)
2. Enable private offers.

We believe that once we implement requirement #1 in Fleet, requirement #2 only needs a AWS/cloud configuration (it's just a different agreed upon rate than the listed one).

In other words:
- #1 requires code changes in Fleet.
- #2 requires #1, and,
- #2 requires no code changes in Fleet.

## How

Assumption: To integrate with the AWS licensing model, the application (Fleet) must use the AWS License Manager API (instead of using own/current licensing functionality).

## Some TBDs around requirements

- What do we want Fleet to do when a AWS license expires? (Currently we don't restrict usage much when a license expires.)
- Do we want to delete hosts if the number of hosts is reduced in a purchased license? (not sure how feasible of a scenario this is.)

### Must have

Limit the number of hosts allowed to enroll to Fleet depending on the license "resource unit" (host count) defined in the purchased AWS license.

### Nice to have (can be done in a later iteration)

The following features wouldn't block the AWS licensing integration (thus they are nice to have):

- Show license about to expire banner on the UI.
- Show banner when a user has reached the maximum number of hosts allowed in their purchased license.

## Implementation of must have

Following are three ways we can integrate the available AWS license models to Fleet (based on the [following guide](https://aws.amazon.com/blogs/awsmarketplace/aws-license-manager-integration-aws-marketplace-ami-containers-contract-pricing/)).

The options are listed in the order of preference (considering implementation complexity+efficiency).

### Option A: use `GetLicense` and `CheckoutLicense` APIs

By calling `GetLicense` API to get the `"LicenseArn"` followed by the `CheckoutLicense` API with entitlement name `"AWS::Marketplace::Usage"` Fleet can fetch the license details, which includes the licensing status and the maximum number of units (hosts) allowed by the license. With such information Fleet can allow/disallow adding more hosts (re-using the same functionality that we have in our licensing implementation). Fleet would execute the `GetLicense`+`CheckoutLicense` APIs every 15m.

This would be the simpler in implementation and efficiency (in terms of number of AWS API calls), but we need to confirm this is indeed "integrating AWS licensing model to Fleet".

### Option B: Use configurable floating license model

See "Floating licenses" in https://docs.aws.amazon.com/marketplace/latest/userguide/ami-license-manager-integration.html.

Changes required in Fleet to use this model:
- When enrolling a host, Fleet will call the `CheckoutLicense` API with unit=1.
- Every 1 hour we have to perform one call to `ExtendLicenseConsumption` (or `CheckoutLicense`) API to extend the license for every enrolled host.
- Deleting a host will call `CheckinLicense` (aka return license to the pool).
- This new solution wouldn't re-use what we implemented (EnrollHostLimiter), we'd just use AWS License Manager for checking if a device can be enrolled or not.

- We'll need to implement a rate limit of enroll hosts (maybe via UUID+enroll secret?) to not DDOS Fleet and AWS Licensing API with hosts trying to enroll and failing when entitlemnts have been consumed.

As shown in the docs above, this model makes sense for modeling user sessions but not so much for Fleet's host enrollments. In Fleet, once a device is enrolled, it has a permanent session (node key).

### Option C: Use tiered license model

The tiered model has a constraint: Customers can not define the quantity of units they want to purchase.

A simplification that would allow us to use this model (if we needed to) would be to define multiple tiers, e.g.:
- "basic" up to 100 hosts
- "standard" up to 500 hosts
- "premium" up to 100k hosts.

## Development and testing

The developer/tester can run Fleet in its workstation with IAM credentials that can access the AWS License Manager API and use test licenses created in AWS. (Both the IAM credentials and test licenses would be provided by the infrastructure team.)