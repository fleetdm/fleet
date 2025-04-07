locals {
  subnet_name        = "${var.prefix}-serverless-subnet"
  vpc_connector_name = "${var.prefix}-serverless-con"
}

module "vpc" {
  source       = "terraform-google-modules/network/google"
  version      = "~> 4.1.0"
  project_id   = var.project_id
  network_name = "${var.prefix}-network"
  mtu          = 1460

  subnets = [
    {
      subnet_name   = local.subnet_name
      subnet_ip     = var.vpc_subnet
      subnet_region = var.region
    }
  ]
}

module "serverless-connector" {
  source     = "terraform-google-modules/network/google//modules/vpc-serverless-connector-beta"
  project_id = var.project_id
  vpc_connectors = [{
    name          = local.vpc_connector_name
    region        = var.region
    subnet_name   = module.vpc.subnets["${var.region}/${local.subnet_name}"].name
    machine_type  = var.serverless_connector_instance_type
    min_instances = var.serverless_connector_min_instances
    max_instances = var.serverless_connector_max_instances
    }
  ]
  depends_on = [
    google_project_service.vpcaccess-api
  ]
}

module "private-service-access" {
  source      = "GoogleCloudPlatform/sql-db/google//modules/private_service_access"
  version     = "9.0.0"
  project_id  = var.project_id
  vpc_network = module.vpc.network_name
  depends_on  = [module.vpc]
}

module "cloud_router" {
  source  = "terraform-google-modules/cloud-router/google"
  version = "~> 6.0"
  name    = "fleet-cloud-router"
  project = var.project_id
  network = module.vpc.network_name
  region  = var.region

  nats = [{
    name = "fleet-vpc-nat"
  }]
}
