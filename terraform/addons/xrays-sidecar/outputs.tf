output "fleet_extra_iam_policies" {
  value = [aws_iam_policy.main.arn]
}

output "fleet_extra_execution_iam_policies" {
  value = [aws_iam_policy.execution.arn]
}

output "fleet_sidecars" {
  value = [
    {
      "name" : "aws-otel-collector",
      "image" : "public.ecr.aws/aws-observability/aws-otel-collector:v0.26.1",
      "essential" : true,
      "command" : [
        "--config=/etc/ecs/ecs-default-config.yaml"
      ],
      "logConfiguration" : {
        "logDriver" : "awslogs",
        "options" : {
          "awslogs-create-group" : "True",
          "awslogs-group" : "/ecs/ecs-aws-otel-sidecar-collector",
          "awslogs-region" : data.aws_region.current.name,
          "awslogs-stream-prefix" : "ecs"
        }
      }
    }
  ]
}

output "fleet_extra_environment_variables" {
  value = {
    FLEET_LOGGING_TRACING_ENABLED = "true"
    FLEET_LOGGING_TRACING_TYPE    = "opentelemetry"
    OTEL_SERVICE_NAME             = "fleet"
    OTEL_EXPORTER_OTLP_ENDPOINT   = "http://localhost:4317"
  }
}
