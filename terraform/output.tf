resource "google_secret_manager_secret" "simple_iap_policy" {
  for_each  = local.exports
  secret_id = format("simple-iap-proxy-%s", each.key)

  replication {
    automatic = true
  }
  depends_on = [google_project_service.secretmanager]
}

resource "google_secret_manager_secret_version" "simple_iap_policy" {
  for_each    = local.exports
  secret      = google_secret_manager_secret.simple_iap_policy[each.key].id
  secret_data = each.value
}

data "google_iam_policy" "simple_iap_policy" {
  binding {
    role    = "roles/secretmanager.secretAccessor"
    members = local.iap_accessors
  }
}

resource "google_secret_manager_secret_iam_policy" "policy" {
  for_each    = local.exports
  project     = google_secret_manager_secret.simple_iap_policy[each.key].project
  secret_id   = google_secret_manager_secret.simple_iap_policy[each.key].secret_id
  policy_data = data.google_iam_policy.simple_iap_policy.policy_data
}

locals {
  exports = {
    target-url      = format("https://%s", trimsuffix(google_dns_record_set.iap_proxy.name, "."))
    audience        = google_iap_client.iap_proxy.client_id
    service-account = google_service_account.iap_proxy_accessor.email
  }
}

output "curl_command" {
  value = <<EOF
ID_TOKEN=$(
   gcloud auth print-identity-token \
   --audiences  ${local.exports.audience} \
   --include-email \
   --impersonate-service-account ${local.exports.service-account}
)
echo -n "Cluster endpoint: " && read CLUSTER_ENDPOINT
curl --header "Host: $CLUSTER_ENDPOINT" \
     --header "Authorization: Bearer $(gcloud auth print-access-token)" \
     --header "Proxy-Authorization: Bearer $ID_TOKEN"  ${local.exports.target-url}
EOF
}

output "iap_proxy_command" {
  value = <<EOF
simple-iap-proxy gke-client \
  --target-url ${local.exports.target-url} \
  --iap-audience ${local.exports.audience} \
  --service-account ${local.exports.service-account} \
  --key-file server.key \
  --certificate-file server.crt
EOF
}

resource "google_project_service" "secretmanager" {
  service            = "secretmanager.googleapis.com"
  disable_on_destroy = false
}
