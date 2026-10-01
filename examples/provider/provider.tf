terraform {
  required_providers {
    axonops = {
      source  = "axonops/axonops"
      version = "~> 0.1"
    }
  }
}

variable "axonops_api_key" {
  description = "AxonOps API key for authentication"
  type        = string
  sensitive   = true
}

variable "axonops_org_id" {
  description = "AxonOps organization ID"
  type        = string
}

# Provider Configuration for AxonOps SaaS (SAML auto-detected)
provider "axonops" {
  org_id  = var.axonops_org_id
  api_key = var.axonops_api_key
}

# Provider Configuration for AxonOps Self-Hosted Deployment
# SAML is automatically detected; no explicit configuration needed.
# provider "axonops" {
#   org_id             = var.axonops_org_id
#   api_key            = var.axonops_api_key
#   axonops_host       = "axonops.example.com"
#   axonops_protocol   = "https"
#   token_type         = "Bearer"
#   tls_skip_verify    = false  # Only for self-signed certificates in non-production
# }
