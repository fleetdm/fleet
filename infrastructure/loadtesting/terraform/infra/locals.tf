locals {
  customer    = "fleet-${terraform.workspace}"
  prefix      = "fleet-${terraform.workspace}"
  fleet_image = "${aws_ecr_repository.fleet.repository_url}:${var.tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"
  # Tracing configuration - either OTEL or Elastic APM
  otel_environment_variables = var.enable_otel ? {
    OTEL_SERVICE_NAME               = terraform.workspace
    OTEL_EXPORTER_OTLP_ENDPOINT     = "http://${data.terraform_remote_state.signoz[0].outputs.otel_collector_endpoint}"
    FLEET_LOGGING_TRACING_ENABLED   = "true"
    FLEET_LOGGING_TRACING_TYPE      = "opentelemetry"
  } : {}

  elastic_apm_environment_variables = var.enable_otel ? {} : {
    ELASTIC_APM_SERVER_URL              = "https://loadtest.fleetdm.com:8200"
    ELASTIC_APM_SERVICE_NAME            = "fleet"
    ELASTIC_APM_ENVIRONMENT             = "${terraform.workspace}"
    ELASTIC_APM_TRANSACTION_SAMPLE_RATE = "0.004"
    ELASTIC_APM_SERVICE_VERSION         = "${var.tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"
    FLEET_LOGGING_TRACING_ENABLED       = "true"
    FLEET_LOGGING_TRACING_TYPE          = "elasticapm"
  }

  extra_environment_variables = merge(
    {
      CLOUDWATCH_NAMESPACE = "fleet-loadtest-migration"
      CLOUDWATCH_REGION    = "us-east-2"
      # PROMETHEUS_SCRAPE_URL = "http://localhost:8080/metrics"

      FLEET_VULNERABILITIES_DATABASES_PATH           = "/home/fleet"
      FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING     = "false"
      FLEET_LOGGING_JSON                             = "true"
      FLEET_LOGGING_DEBUG                            = "true"
      FLEET_MYSQL_MAX_OPEN_CONNS                     = "10"
      FLEET_MYSQL_READ_REPLICA_MAX_OPEN_CONNS        = "10"
      FLEET_OSQUERY_ASYNC_HOST_REDIS_SCAN_KEYS_COUNT = "10000"
      FLEET_REDIS_MAX_OPEN_CONNS                     = "500"
      FLEET_REDIS_MAX_IDLE_CONNS                     = "500"

      # Load TLS Certificate for RDS Authentication
      FLEET_MYSQL_TLS_CA              = local.cert_path
      FLEET_MYSQL_READ_REPLICA_TLS_CA = local.cert_path
    },
    local.otel_environment_variables,
    local.elastic_apm_environment_variables
  )
  extra_secrets = {
    FLEET_LICENSE_KEY = data.aws_secretsmanager_secret.license.arn
  }
  # Private Subnets from VPN VPC
  vpn_cidr_blocks = [
    "10.255.1.0/24",
    "10.255.2.0/24",
    "10.255.3.0/24",
  ]

  /* 
    configurations below are necessary for MySQL TLS authentication
    MySQL TLS Settings to download and store TLS Certificate

    ca_thumbprint is maintained in the infrastructure/cloud/shared/
    ca_thumbprint is the sha1 thumbprint value of the following certificate: aws rds describe-db-instances --filters='Name=db-cluster-id,Values='${cluster_name}'' | jq '.DBInstances.[0].CACertificateIdentifier' | sed 's/\"//g'
    You can retrieve the value with the following command: aws rds describe-certificates --certificate-identifier=${ca_cert_val} | jq '.Certificates.[].Thumbprint' | sed 's/\"//g'
  */
  ca_cert_thumbprint = "8cf85e3e2bdbcbe2c4a34c1e85828fb29833e87f"
  rds_container_path = "/tmp/rds-tls"
  cert_path          = "${local.rds_container_path}/${data.aws_region.current.region}.pem"

  # load the certificate with a side car into a volume mount
  sidecars = [
    {
      name       = "rds-tls-ca-retriever"
      image      = "public.ecr.aws/docker/library/alpine@sha256:8a1f59ffb675680d47db6337b49d22281a139e9d709335b492be023728e11715"
      entrypoint = ["/bin/sh", "-c"]
      command = [templatefile("./template/mysql_ca_tls_retrieval.sh.tpl", {
        aws_region         = data.aws_region.current.region
        container_path     = local.rds_container_path
        ca_cert_thumbprint = local.ca_cert_thumbprint
      })]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = local.customer
          "awslogs-region"        = data.aws_region.current.region
          "awslogs-stream-prefix" = "rds-tls-ca-retriever"
        }
      }
      environment = []
      mountPoints = [
        {
          sourceVolume  = "rds-tls-certs",
          containerPath = local.rds_container_path
        }
      ]
      essential = false
    },
    # {
    #   name      = "prometheus-exporter"
    #   image     = "${data.terraform_remote_state.shared.outputs.ecr.repository_url}:latest"
    #   entrypoint = []
    #   command = ["sleep"]
    #   logConfiguration = {
    #     logDriver = "awslogs"
    #     options = {
    #       "awslogs-group"         = local.customer
    #       "awslogs-region"        = data.aws_region.current.region
    #       "awslogs-stream-prefix" = "fleet-prometheus-exporter"
    #     }
    #   }
    #   environment = [
    #     {
    #       name  = "CLOUDWATCH_NAMESPACE"
    #       value = "fleet-loadtest"
    #     },
    #     {
    #       name  = "CLOUDWATCH_REGION"
    #       value = "us-east-2"
    #     },
    #     {
    #       name  = "PROMETHEUS_SCRAPE_URL"
    #       value = "http://localhost:8080/metrics"
    #     },
    #   ]
    #   mountPoints = []
    #   essential = false
    # }
  ]
}