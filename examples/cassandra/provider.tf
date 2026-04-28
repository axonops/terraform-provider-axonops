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
  description = "Cassandra cluster name in AxonOps"
  type        = string
  default     = "my-cassandra-cluster"
}

variable "slack_webhook_url" {
  description = "Slack incoming webhook URL"
  type        = string
  sensitive   = true
}

variable "teams_webhook_url" {
  description = "Microsoft Teams incoming webhook URL"
  type        = string
  sensitive   = true
}

variable "pagerduty_integration_key" {
  description = "PagerDuty Events API v2 integration key"
  type        = string
  sensitive   = true
}

variable "opsgenie_api_key" {
  description = "OpsGenie API key URL"
  type        = string
  sensitive   = true
}

variable "servicenow_password" {
  description = "ServiceNow user password"
  type        = string
  sensitive   = true
}
