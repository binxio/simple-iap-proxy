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