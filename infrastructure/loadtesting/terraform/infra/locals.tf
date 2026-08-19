locals {
  customer    = "fleet-${terraform.workspace}"
  prefix      = "fleet-${terraform.workspace}"
  fleet_image = "${aws_ecr_repository.fleet.repository_url}:${var.tag}-${split(":", data.docker_registry_image.dockerhub.sha256_digest)[1]}"

  # Tracing configuration - either OTEL or Elastic APM
  otel_environment_variables = var.enable_otel ? {
    OTEL_SERVICE_NAME             = "fleet"
    OTEL_RESOURCE_ATTRIBUTES      = "deployment.environment.name=${terraform.workspace},deployment.environment=${terraform.workspace}"
    OTEL_EXPORTER_OTLP_ENDPOINT   = "http://${data.terraform_remote_state.signoz[0].outputs.otel_collector_endpoint}"
    FLEET_LOGGING_TRACING_ENABLED = "true"
    FLEET_LOGGING_TRACING_TYPE    = "opentelemetry"
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

  # Single label under loadtest.fleetdm.com so the *.loadtest.fleetdm.com
  # wildcard cert would cover it if the mock ever moves to the HTTPS listener.
  # A nested name would not: wildcards match exactly one label.
  #
  # This name resolves to the shared internal ALB and is now an operator door
  # only -- /healthz, /stats, /memstats from the VPN, where the Cloud Map name
  # below does not resolve. No load-bearing traffic goes through it; see
  # apple_apns_mock.tf for why.
  apple_apns_mock_hostname = "${local.customer}-apns-mock.loadtest.fleetdm.com"
  apple_apns_mock_port     = 8378

  # Cloud Map private DNS: resolves straight to the mock's task IP, no load
  # balancer in the path. Every simulated device holds an SSE stream open for
  # the life of the test, and an ALB has to hold one connection per stream from
  # its own node IPs to a single target IP:port. That tops out around 55k
  # connections per ALB node against one target (source ephemeral ports), so a
  # 75k-host run -- two streams per host, device plus user channel -- collapsed
  # into a 39k-count TargetConnectionErrorCount/504 storm at ~75-100k streams
  # while the mock itself sat at ~0ms response time and never went unhealthy.
  # Dialing the task directly moves the port space to the osquery-perf side,
  # where it is spread over every loadtest container's ENI instead of a handful
  # of ALB nodes.
  #
  # Built from locals rather than from the aws_service_discovery_service
  # resource so the Fleet task definition does not take a dependency on the
  # mock. Nothing resolves this name until a push actually happens.
  apple_apns_mock_namespace = "${local.customer}.internal"
  apple_apns_mock_dns_name  = "apns-mock.${local.customer}.internal"
  apple_apns_mock_url       = "http://apns-mock.${local.customer}.internal:8378"

  # MDM behaviours we always want in a loadtest. These were previously buried
  # in the Elastic APM branch above, which meant they silently vanished
  # whenever enable_otel was true.
  mdm_apple_environment_variables = merge(
    {
      # Skip verification of Apple certificates for OTA enrollments.
      FLEET_DEV_MDM_APPLE_DISABLE_DEVICE_INFO_CERT_VERIFY = "1"
    },
    # Push traffic must never reach real Apple infrastructure, which would
    # rate-limit us for pushing to thousands of fake device UUIDs. Either it
    # goes to the mock, or it goes nowhere.
    #
    # These two are mutually exclusive: DISABLE_PUSH short-circuits
    # initAppleMDMPushService to a nopPusher before the push URL is ever read,
    # so setting both would silently make the mock unreachable.
    var.enable_apple_mdm ? {
      FLEET_DEV_MDM_APPLE_PUSH_SERVER_URL = local.apple_apns_mock_url
      } : {
      FLEET_DEV_MDM_APPLE_DISABLE_PUSH = "1"
    }
  )

  extra_environment_variables = merge(
    {
      CLOUDWATCH_NAMESPACE = "fleet-loadtest-migration"
      CLOUDWATCH_REGION    = "us-east-2"
      # PROMETHEUS_SCRAPE_URL = "http://localhost:8080/metrics"

      FLEET_VULNERABILITIES_DATABASES_PATH       = "/home/fleet"
      FLEET_OSQUERY_ENABLE_ASYNC_HOST_PROCESSING = "false"
      FLEET_LOGGING_JSON                         = "true"
      FLEET_LOGGING_DEBUG                        = "true"
      FLEET_OSQUERY_STATUS_LOG_PLUGIN            = "filesystem"
      FLEET_FILESYSTEM_STATUS_LOG_FILE           = "/dev/null"
      FLEET_OSQUERY_RESULT_LOG_PLUGIN            = "filesystem"
      FLEET_FILESYSTEM_RESULT_LOG_FILE           = "/dev/null"
      FLEET_MYSQL_MAX_OPEN_CONNS                 = tostring(var.mysql_max_open_conns)
      FLEET_MYSQL_READ_REPLICA_MAX_OPEN_CONNS    = tostring(var.mysql_max_open_conns)
      # 30 min: recycle connections often enough that pooled reader connections re-spread across replicas after a
      # replica reboot/failover, and proactively drop bad idles.
      FLEET_MYSQL_CONN_MAX_LIFETIME                  = "1800"
      FLEET_MYSQL_READ_REPLICA_CONN_MAX_LIFETIME     = "1800"
      FLEET_OSQUERY_ASYNC_HOST_REDIS_SCAN_KEYS_COUNT = "10000"
      FLEET_REDIS_MAX_OPEN_CONNS                     = "500"
      FLEET_REDIS_MAX_IDLE_CONNS                     = "500"
      FLEET_AUTH_SSO_SESSION_VALIDITY_PERIOD         = "15m"
      FLEET_MDM_SSO_RATE_LIMIT_PER_MINUTE            = "500"
      FLEET_SERVER_GZIP_RESPONSES                    = "true"
      FLEET_DEV_ANDROID_PROXY_ENDPOINT               = "http://${resource.aws_lb.internal.dns_name}/"

      # Load TLS Certificate for RDS Authentication
      FLEET_MYSQL_TLS_CA                  = local.cert_path
      FLEET_MYSQL_READ_REPLICA_TLS_CA     = local.cert_path
      FLEET_MYSQL_READ_REPLICA_TLS_CONFIG = "custom"

      # Skip backfilling S3 config with dev values for load testing
      FLEET_DEV_SKIP_S3_CONFIG = "1"
    },
    local.otel_environment_variables,
    local.elastic_apm_environment_variables,
    local.mdm_apple_environment_variables
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
