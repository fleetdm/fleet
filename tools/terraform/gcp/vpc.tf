module "vpc" {
  source       = "terraform-google-modules/network/google"
  version      = "~> 4.1.0"
  project_id   = var.project_id
  network_name = "fleet-network"
  mtu          = 1460

  subnets = [
    {
      subnet_name   = "serverless-subnet"
      subnet_ip     = "10.10.10.0/28"
      subnet_region = "us-central1"
    }
  ]
}

module "serverless-connector" {
  source     = "terraform-google-modules/network/google//modules/vpc-serverless-connector-beta"
  project_id = var.project_id
  vpc_connectors = [{
    name        = "central-serverless"
    region      = "us-central1"
    subnet_name = module.vpc.subnets["us-central1/serverless-subnet"].name
    # host_project_id = var.host_project_id # Specify a host_project_id for shared VPC
    machine_type  = "f1-micro"
    min_instances = 1
    max_instances = 2
    }
    # Uncomment to specify an ip_cidr_range
    #   , {
    #     name          = "central-serverless2"
    #     region        = "us-central1"
    #     network       = module.test-vpc-module.network_name
    #     ip_cidr_range = "10.10.11.0/28"
    #     subnet_name   = null
    #     machine_type  = "e2-standard-4"
    #     min_instances = 2
    #   max_instances = 7 }
  ]
  depends_on = [
    google_project_service.vpcaccess-api
  ]
}

resource "google_vpc_access_connector" "connector_beta" {
  provider = google-beta
  name     = "foobar"
  project  = var.project_id
  region   = var.region
  subnet {
    name       = module.vpc.subnets["us-central1/serverless-subnet"].name
    project_id = var.project_id
  }
  machine_type  = "f1-micro"
  min_instances = 1
  max_instances = 2
}