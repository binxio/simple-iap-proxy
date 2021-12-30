
resource "google_iap_web_backend_service_iam_binding" "httpbin_https_resource_accessor" {
  web_backend_service = google_compute_backend_service.httpbin.name
  role                = "roles/iap.httpsResourceAccessor"
  members             = local.iap_accessors
}

resource "google_service_account" "httpbin_accessor" {
  account_id  = "httpbin-accessor"
  description = "IAP proxy accessors"
}

resource "google_service_account_iam_binding" "httpbin_accessor_service_account_token_creator" {
  service_account_id = google_service_account.httpbin_accessor.id
  role               = "roles/iam.serviceAccountTokenCreator"
  members            = local.iap_accessors
}


locals {
  iap_accessors = toset(flatten([
    format("serviceAccount:%s", google_service_account.httpbin_accessor.email),
    var.accessors,
  ]))
}
