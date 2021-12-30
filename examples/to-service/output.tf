resource "google_secret_manager_secret" "httpbin" {
  for_each  = local.exports
  secret_id = format("httpbin-%s", each.key)

  replication {
    automatic = true
  }
  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "httpbin" {
  for_each    = local.exports
  secret      = google_secret_manager_secret.httpbin[each.key].id
  secret_data = each.value
}

data "google_iam_policy" "httpbin" {
  binding {
    role    = "roles/secretmanager.secretAccessor"
    members = local.iap_accessors
  }
}

resource "google_secret_manager_secret_iam_policy" "policy" {
  for_each    = local.exports
  project     = google_secret_manager_secret.httpbin[each.key].project
  secret_id   = google_secret_manager_secret.httpbin[each.key].secret_id
  policy_data = data.google_iam_policy.httpbin.policy_data
}

locals {
  exports = {
    target-url      = format("https://%s", trimsuffix(google_dns_record_set.httpbin.name, "."))
    audience        = google_iap_client.httpbin.client_id
    service-account = google_service_account.httpbin_accessor.email
  }
}

output "iap_proxy_command" {
  value = <<EOF
simple-iap-proxy client \
  --target-url ${local.exports.target-url} \
  --iap-audience ${local.exports.audience} \
  --service-account ${local.exports.service-account} \
  --key-file server.key \
  --certificate-file server.crt \
  --to-host ${trimsuffix(google_dns_record_set.httpbin.name, ".")}
EOF
}

output "httpbin_command" {
  value = <<EOF
export HTTPS_PROXY=https://localhost:8080
curl ${local.exports.target-url}/anything
EOF
}

resource "google_project_service" "secretmanager" {
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}
