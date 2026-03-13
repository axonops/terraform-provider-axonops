# Provider Configuration for AxonOps SaaS
provider "axonops" {
  # Organization ID (required)
  org_id = "your-org-id"

  # API key for authentication (required for SaaS)
  api_key = "your-api-key-here"
}

# Provider Configuration for AxonOps SaaS with SAML
# provider "axonops" {
#   # Organization ID (required)
#   org_id = "your-org-id"
#
#   # API key for authentication
#   api_key = "your-api-key-here"
#
#   # Enable SAML authentication mode
#   # Uses tenant-specific URL: https://{org_id}.axonops.cloud/dashboard
#   use_saml = true
# }

# Provider Configuration for Self-Hosted AxonOps
# provider "axonops" {
#   # Organization ID (required)
#   org_id = "your-org-id"
#
#   # API key for authentication
#   api_key = "your-api-key-here"
#
#   # Self-hosted server hostname
#   axonops_host = "axonops.example.com"
#
#   # Protocol (http or https)
#   axonops_protocol = "https"
#
#   # Token type for Authorization header: 'Bearer' (default) or 'AxonApi'
#   token_type = "Bearer"
#
#   # Skip TLS certificate verification (for self-signed certificates)
#   tls_skip_verify = false
#
#   # Enable SAML if using SAML authentication on self-hosted
#   # use_saml = true
# }
