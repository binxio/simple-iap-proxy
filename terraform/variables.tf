variable "dns_managed_zone" {
  description = "name of the DNS managed zone to insert the record into"
  type        = string
}

variable "oauth_client_id" {
  description = "IAP Oauth client id"
  type        = string
}

variable "oauth_client_secret" {
  description = "IAP Oauth client secret"
  type        = string
  sensitive   = true
}

variable "region" {
  description = "to deploy the proxy in"
  type        = string
}

variable "project" {
  description = "to deploy the proxy in"
  type        = string
}
