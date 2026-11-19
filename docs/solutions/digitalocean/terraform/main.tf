terraform {
  required_version = ">= 1.5"

  required_providers {
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "digitalocean" {
  # Reads the DIGITALOCEAN_TOKEN environment variable.
}

locals {
  # Writes the database CA to a file, then starts Fleet. App Platform's run_command
  # replaces the image entrypoint, so the command is spelled out. The doubled
  # percent sign is format()'s escape for a literal one.
  fleet_cmd = "sh -c 'printf \"%%s\\n\" \"$FLEET_MYSQL_CA_PEM\" > /tmp/mysql-ca.pem && exec fleet %s'"

  use_s3 = var.installers_bucket != ""

  # The $${...} syntax is Terraform's escape for App Platform's own ${...} bindings,
  # which App Platform resolves at runtime.
  mysql_env = {
    FLEET_MYSQL_ADDRESS  = "$${fleet-mysql.HOSTNAME}:$${fleet-mysql.PORT}"
    FLEET_MYSQL_USERNAME = "$${fleet-mysql.USERNAME}"
    FLEET_MYSQL_PASSWORD = "$${fleet-mysql.PASSWORD}"
    FLEET_MYSQL_DATABASE = "$${fleet-mysql.DATABASE}"
    FLEET_MYSQL_TLS_CA   = "/tmp/mysql-ca.pem"
    FLEET_MYSQL_CA_PEM   = "$${fleet-mysql.CA_CERT}"
  }

  server_env = merge(
    local.mysql_env,
    {
      FLEET_SERVER_ADDRESS                          = "0.0.0.0:8080"
      FLEET_SERVER_TLS                              = "false"
      FLEET_REDIS_ADDRESS                           = "redis:6379"
      FLEET_LOGGING_JSON                            = "true"
      FLEET_VULNERABILITIES_DATABASES_PATH          = "/tmp/vulndb"
      FLEET_VULNERABILITIES_PERIODICITY             = "1h"
      FLEET_VULNERABILITIES_CURRENT_INSTANCE_CHECKS = "yes"
    },
    local.use_s3 ? {
      FLEET_S3_SOFTWARE_INSTALLERS_BUCKET       = var.installers_bucket
      FLEET_S3_SOFTWARE_INSTALLERS_ENDPOINT_URL = "https://${var.installers_region}.digitaloceanspaces.com"
      FLEET_S3_SOFTWARE_INSTALLERS_REGION       = var.installers_region
    } : {}
  )

  # Secrets live at the app level. App Platform returns SECRET values encrypted, so
  # Terraform can't diff them, and the lifecycle block below ignores them after
  # creation. To rotate one, remove `spec[0].env` from ignore_changes, apply,
  # and put it back.
  secrets = merge(
    {
      FLEET_SERVER_PRIVATE_KEY = random_id.private_key.b64_std
    },
    var.fleet_license_key != "" ? { FLEET_LICENSE_KEY = var.fleet_license_key } : {},
    local.use_s3 ? {
      FLEET_S3_SOFTWARE_INSTALLERS_ACCESS_KEY_ID     = var.installers_access_key_id
      FLEET_S3_SOFTWARE_INSTALLERS_SECRET_ACCESS_KEY = var.installers_secret_access_key
    } : {}
  )
}

resource "random_id" "private_key" {
  byte_length = 32
}

resource "digitalocean_database_cluster" "mysql" {
  name       = "${var.name}-mysql"
  engine     = "mysql"
  version    = "8.4"
  size       = var.mysql_size
  region     = var.db_region
  node_count = var.mysql_node_count
}

# DigitalOcean's defaults reject Fleet's schema: it requires primary keys on every
# table, and its sql_mode adds ANSI, which changes how identifiers are quoted.
resource "digitalocean_database_mysql_config" "mysql" {
  cluster_id              = digitalocean_database_cluster.mysql.id
  sql_require_primary_key = false
  sql_mode                = "ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION"
}

resource "digitalocean_database_db" "fleet" {
  cluster_id = digitalocean_database_cluster.mysql.id
  name       = "fleet"
}

resource "digitalocean_database_user" "fleet" {
  cluster_id = digitalocean_database_cluster.mysql.id
  name       = "fleet"

  # The provider reports an empty settings block as drift.
  lifecycle {
    ignore_changes = [settings]
  }
}

resource "digitalocean_app" "fleet" {
  lifecycle {
    ignore_changes = [spec[0].env]
  }

  # Apply the MySQL settings before the first deploy so migrations run against them.
  depends_on = [digitalocean_database_mysql_config.mysql]

  spec {
    name   = var.name
    region = var.app_region

    alert {
      rule = "DEPLOYMENT_FAILED"
    }

    dynamic "domain" {
      for_each = var.domain != "" ? [var.domain] : []
      content {
        name = domain.value
        type = "PRIMARY"
      }
    }

    database {
      name         = "fleet-mysql"
      engine       = "MYSQL"
      production   = true
      cluster_name = digitalocean_database_cluster.mysql.name
      db_name      = digitalocean_database_db.fleet.name
      db_user      = digitalocean_database_user.fleet.name
    }

    dynamic "env" {
      for_each = local.secrets
      content {
        key   = env.key
        value = env.value
        scope = "RUN_TIME"
        type  = "SECRET"
      }
    }

    ingress {
      rule {
        match {
          path {
            prefix = "/"
          }
        }
        component {
          name = "fleet"
        }
      }
    }

    # Internal only: no http_port and no ingress rule. Fleet uses Redis as a
    # cache, so nothing in it needs to survive a restart.
    service {
      name               = "redis"
      instance_size_slug = "apps-s-1vcpu-0.5gb"
      instance_count     = 1
      internal_ports     = [6379]

      image {
        registry_type = "DOCKER_HUB"
        registry      = "library"
        repository    = "redis"
        tag           = "7-alpine"
      }
    }

    # Runs before every deploy. Fleet won't serve with pending migrations.
    job {
      name               = "fleet-migrate"
      kind               = "PRE_DEPLOY"
      instance_size_slug = "apps-s-1vcpu-0.5gb"
      instance_count     = 1
      run_command        = format(local.fleet_cmd, "prepare db --no-prompt")

      image {
        registry_type = "DOCKER_HUB"
        registry      = "fleetdm"
        repository    = "fleet"
        tag           = var.fleet_version
      }

      dynamic "env" {
        for_each = local.mysql_env
        content {
          key   = env.key
          value = env.value
          scope = "RUN_TIME"
          type  = "GENERAL"
        }
      }
    }

    service {
      name               = "fleet"
      instance_size_slug = var.fleet_instance_size
      instance_count     = var.fleet_instance_count
      http_port          = 8080
      run_command        = format(local.fleet_cmd, "serve")

      image {
        registry_type = "DOCKER_HUB"
        registry      = "fleetdm"
        repository    = "fleet"
        tag           = var.fleet_version
      }

      health_check {
        http_path             = "/healthz"
        port                  = 8080
        initial_delay_seconds = 60
        period_seconds        = 15
        timeout_seconds       = 10
        failure_threshold     = 10
      }

      dynamic "env" {
        for_each = local.server_env
        content {
          key   = env.key
          value = env.value
          scope = "RUN_TIME"
          type  = "GENERAL"
        }
      }
    }
  }
}

# Created after the app so the first deploy can reach the database. Once this
# applies, only the app can connect.
resource "digitalocean_database_firewall" "mysql" {
  cluster_id = digitalocean_database_cluster.mysql.id

  rule {
    type  = "app"
    value = digitalocean_app.fleet.id
  }
}
