## Deploy Fleet on AWS ECS

Terraform reference architecture can be found [here](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws)

### Infrastructure dependencies

#### MySQL

In AWS we recommend running Aurora with MySQL Engine, see [here for terraform details](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/rds.tf#L64).

#### Redis

In AWS we recommend running ElastiCache (Redis Engine) see [here for terraform details](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/redis.tf#L13)

#### Fleet server

Running Fleet in ECS consists of two main components the [ECS Service](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/ecs.tf#L84) & [Load Balancer](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/ecs.tf#L59). In our example the ALB is [handling TLS termination](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/ecs.tf#L46)

#### Fleet migrations

Migrations in ECS can be achieved (and is recommended) by running [dedicated ECS tasks](https://github.com/fleetdm/fleet/tree/main/infrastructure/dogfood/terraform/aws#migrating-the-db) that run the `fleet prepare --no-prompt=true db` command. See [terraform for more details](https://github.com/fleetdm/fleet/blob/main/infrastructure/dogfood/terraform/aws/ecs.tf#L261)

Alternatively you can bake the prepare command into the same task definition see [here for a discussion](https://github.com/fleetdm/fleet/pull/1761#discussion_r697599457), but this not recommended for production environments.

<meta name="title" value="AWS ECS">
<meta name="pageOrderInSection" value="400">
<meta name="description" value="Information for deploying Fleet on AWS ECS.">
<meta name="navSection" value="Deployment guides">