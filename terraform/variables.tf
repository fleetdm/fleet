variable "vpc" {
  type = object({
    name                = string
    cidr                = string
    azs                 = list(string)
    private_subnets     = list(string)
    public_subnets      = list(string)
    database_subnets    = list(string)
    elasticache_subnets = list(string)

    create_database_subnet_group          = bool
    create_database_subnet_route_table    = bool
    create_elasticache_subnet_group       = bool
    create_elasticache_subnet_route_table = bool
    enable_vpn_gateway                    = bool
    one_nat_gateway_per_az                = bool
    single_nat_gateway                    = bool
    enable_nat_gateway                    = bool
  })
  default = {
    name                = "fleet"
    cidr                = "10.20.0.0/16"
    azs                 = ["us-east-2a", "us-east-2b", "us-east-2c"]
    private_subnets     = ["10.10.1.0/24", "10.10.2.0/24", "10.10.3.0/24"]
    public_subnets      = ["10.10.11.0/24", "10.10.12.0/24", "10.10.13.0/24"]
    database_subnets    = ["10.10.21.0/24", "10.10.22.0/24", "10.10.23.0/24"]
    elasticache_subnets = ["10.10.31.0/24", "10.10.32.0/24", "10.10.33.0/24"]

    create_database_subnet_group          = true
    create_database_subnet_route_table    = true
    create_elasticache_subnet_group       = true
    create_elasticache_subnet_route_table = true
    enable_vpn_gateway                    = false
    one_nat_gateway_per_az                = false
    single_nat_gateway                    = true
    enable_nat_gateway                    = true
  }
}

variable "vpc_cidr" {
  type        = string
  default     = "10.20.0.0/16"
  description = "The CIDR of the fleet VPC."
  validation {
    condition     = endswith(var.vpc_cidr, "/16")
    error_message = "VPC Cidr must be a /16"
  }
  nullable = false
}

variable "db_instance_size" {
  type        = string
  default     = "db.r6g.xlarge"
  description = "Size of the RDS Database Instances"
  validation {
    condition = contains([
      "db.r6g.xlarge",
      "db.r6g.2xlarge",
      "db.r6g.4xlarge",
    ], var.db_instance_size)
    error_message = "Must use one of the allowed db instance sizes"
  }
  nullable = false
}

variable "cache_node_type" {
  type        = string
  default     = "cache.m6g.large"
  description = "Redis Replication Group Node Type"
  validation {
    condition = contains([
      "cache.m6g.large",
      "cache.m6g.xlarge",
      "cache.m6g.2xlarge",
      "cache.m6g.4xlarge",
    ], var.cache_node_type)
    error_message = "Must use one of the allowed cache node types"
  }
  nullable = false
}

variable "fleet_container_cpu" {
  type        = number
  default     = 512
  description = "CPU resources allocated per container. 1024 per VCPU"
  validation {
    condition = contains([
      512,
      1024,
      2048,
      4096,
      8192,
      16384,
    ], var.fleet_container_cpu)
  }
  nullable = false
}

variable "fleet_container_memory" {
  type        = number
  default     = 4096
  description = "Memory Allocated to Fleet Container in MB.  See https://docs.aws.amazon.com/AmazonECS/latest/developerguide/AWS_Fargate.html to ensure the value matches the selected CPU count above."
  validation {
    condition = contains([
      4096,
      6144,
      8192,
      16384,
      32768,
    ], var.fleet_container_memory)
  }
  nullable = false
}

variable "fleet_ingress_cidr" {
  type        = string
  default     = "0.0.0.0/0"
  description = "IP Addresses allowed to access the Fleet Ingress in CIDR Format.  Use 0.0.0.0/0 to allow access from the entire Internet"
  nullable    = false
}

variable "fleet_dns_name" {
  type        = string
  description = "The DNS Name for your Fleet instance. Example: fleet.fleetdm.com if your hosted zone is fleetdm.com."
  nullable    = false
}

variable "fleet_license" {
  type        = string
  description = "Bringing your own license?  Enter it here.  Leave blank for the free version."
  sensitive   = true
  default     = null
}

variable "route53_hosted_zone_id" {
  type        = string
  description = "Route53 Hosted Zone Id for the FleetDNSName"
  nullable    = false
}

variable "fleet_logging_debug" {
  type        = bool
  description = "Enable Fleet Debug Logging"
  default     = false
  nullable    = false
}

variable "fleet_logging_json" {
  type        = bool
  description = "Enable Fleet Logging in JSON Format"
  default     = false
  nullable    = false
}

variable "fleet_async_host_processing" {
  type        = bool
  description = "Enable Fleet osquery async host processing."
  default     = false
  nullable    = false
}
