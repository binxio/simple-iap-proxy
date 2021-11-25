
resource "google_iap_web_backend_service_iam_binding" "iap_proxy_https_resource_accessor" {
  web_backend_service = google_compute_backend_service.iap_proxy.name
  role                = "roles/iap.httpsResourceAccessor"
  members = flatten([
    format("serviceAccount:%s", google_service_account.iap_proxy.email),
    var.accessors,
  ])
}

resource "google_service_account" "iap_proxy_accessor" {
  account_id  = "iap-proxy-accessor"
  description = "IAP proxy accessors"
}
