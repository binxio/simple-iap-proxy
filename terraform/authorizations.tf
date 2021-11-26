
resource "google_iap_web_backend_service_iam_binding" "iap_proxy_https_resource_accessor" {
  web_backend_service = google_compute_backend_service.iap_proxy.name
  role                = "roles/iap.httpsResourceAccessor"
  members = flatten([
    format("serviceAccount:%s", google_service_account.iap_proxy_accessor.email),
    var.accessors,
  ])
}

resource "google_service_account" "iap_proxy_accessor" {
  account_id  = "iap-proxy-accessor"
  description = "IAP proxy accessors"
}

resource "google_service_account_iam_binding" "iap_proxy_accessor_service_account_token_creator" {
  service_account_id = google_service_account.iap_proxy_accessor.id
  role               = "roles/iam.serviceAccountTokenCreator"
  members            = var.accessors
}