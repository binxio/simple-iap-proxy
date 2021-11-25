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