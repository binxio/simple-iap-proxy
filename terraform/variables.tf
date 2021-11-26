variable "project" {
  description = "google project id to deploy the proxy in"
  type        = string
}

variable "region" {
  description = "name of the region to deploy the proxy in"
  type        = string
}

variable "target_cluster" {
  description = "target GKE cluster to forward requests to"
  type = object({
    name     = string
    location = string
  })
}

variable "accessors" {
  description = "list of google user ids allowed to use the IAP proxy"
  type        = list(string)
  validation {
    condition     = length([for a in var.accessors : a if length(regexall("^(user:.*|group:.*|serviceAccount:.*)$", a)) == 0]) == 0
    error_message = "Accessors must be a IAM user, group or service account."
  }
}


variable "dns_managed_zone" {
  description = "name of the DNS managed zone to insert the record into\n\t-> this must be accessible from the public internet"
  type        = string
}

variable "iap_support_email" {
  description = "support email address for IAP brand creation\n\t -> leave empty if it already exists in your project"
  type        = string
}

