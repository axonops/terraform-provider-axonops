variable "pagerduty_integration_key" {
  type      = string
  sensitive = true
}

resource "axonops_pagerduty_integration" "oncall" {
  cluster_name    = "production-kafka"
  cluster_type    = "kafka"
  name            = "pagerduty-oncall"
  integration_key = var.pagerduty_integration_key
}

# Route all error-level alerts to PagerDuty
resource "axonops_alert_route" "pagerduty_global" {
  cluster_name     = axonops_pagerduty_integration.oncall.cluster_name
  cluster_type     = axonops_pagerduty_integration.oncall.cluster_type
  integration_name = axonops_pagerduty_integration.oncall.name
  integration_type = "pagerduty"
  type             = "global"
  severity         = "error"
}
