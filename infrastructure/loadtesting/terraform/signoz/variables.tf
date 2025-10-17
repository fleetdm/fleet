variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-2"
}

variable "cluster_name" {
  description = "Name of the EKS cluster for SigNoz"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID for SigNoz EKS cluster (shared fleet VPC)"
  type        = string
}

variable "subnet_ids" {
  description = "Subnet IDs for SigNoz EKS cluster (private subnets from fleet VPC)"
  type        = list(string)
}
