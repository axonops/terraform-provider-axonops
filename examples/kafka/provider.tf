terraform {
  required_version = ">= 1.0"
  required_providers {
    axonops = {
      source  = "axonops/axonops"
      version = ">= 1.0"
    }
  }
}

provider "axonops" {
  api_key          = var.axonops_api_key
  axonops_host     = var.axonops_host
  axonops_protocol = "https"
  org_id           = var.org_id
}

variable "axonops_api_key" {
  description = "AxonOps API key"
  type        = string
  sensitive   = true
}

variable "axonops_host" {
  description = "AxonOps server hostname"
  type        = string
  default     = "axonops.com"
}

variable "org_id" {
  description = "AxonOps organisation ID"
  type        = string
}

variable "cluster_name" {
  description = "Kafka cluster name in AxonOps"
  type        = string
  default     = "my-kafka-cluster"
}
