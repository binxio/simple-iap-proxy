variable "dns_managed_zone" {
  description = "name of the DNS managed zone to insert the record into"
  type        = string
}

variable "region" {
  description = "to deploy the proxy in"
  type        = string
}

variable "project" {
  description = "to deploy the proxy in"
  type        = string
}

variable "accessors" {
  description = "additional google identities allows to use the IAP proxy"
  type        = list(string)
  default     = []
  validation {
    condition     = length([for a in var.accessors : a if length(regexall("^(user:.*|group:.*|serviceAccount:.*)$", a)) == 0]) == 0
    error_message = "Accessors must be a IAM user, group or service account."
  }
}
variable "target_cluster" {
  description = "to forward requests to"
  type = object({
    name     = string
    location = string
  })
}

variable "iap_support_email" {
  description = "support email address for IAP brand creation"
  type        = string
}

output "iap_proxy_command" {
  value = <<EOF
simple-iap-proxy  \
  --rename-auth-header \
  --target-url https://iap-proxy.${trimsuffix(data.google_dns_managed_zone.tld.dns_name, ".")} \
  --iap-audience ${google_iap_client.iap_proxy.client_id} \
  --service-account ${google_service_account.iap_proxy_accessor.email} \
  --certificate-file server.crt \
  --key-file server.key
EOF
}