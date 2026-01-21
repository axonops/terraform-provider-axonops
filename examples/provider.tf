# Provider Configuration
# This file shows how to configure the AxonOps provider

provider "axonops" {
  # API key for authentication (required for SaaS, optional for self-hosted)
  api_key = "your-api-key-here"

  # AxonOps server hostname
  axonops_host = "your-axonops-host.com"

  # Protocol (http or https)
  axonops_protocol = "https"

  # Organization ID (required)
  org_id = "your-org-id"

  # Token type for Authorization header: 'AxonApi' (default) or 'Bearer'
  token_type = "AxonApi"
}
