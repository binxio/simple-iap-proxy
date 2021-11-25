resource "google_compute_global_forwarding_rule" "iap_proxy" {
  name       = "iap-proxy-port-443"
  target     = google_compute_target_https_proxy.iap_proxy.self_link
  ip_address = google_compute_global_address.iap_proxy.address
  port_range = "443"
}

resource "google_compute_target_https_proxy" "iap_proxy" {
  name = "iap-proxy"

  ssl_certificates = [
    google_compute_managed_ssl_certificate.iap_proxy.id,
  ]

  url_map = google_compute_url_map.iap_proxy.self_link
}

resource "google_compute_managed_ssl_certificate" "iap_proxy" {
  name = random_pet.iap_proxy_certificate_name.id
  managed {
    domains = [
      google_dns_record_set.iap_proxy.name
    ]
  }
  lifecycle {
    create_before_destroy = true
  }
}

resource "random_pet" "iap_proxy_certificate_name" {
  prefix = "iap-proxy"

  keepers = {
    domains = google_dns_record_set.iap_proxy.name
  }
}

resource "google_compute_url_map" "iap_proxy" {
  name = "iap-proxy"

  default_service = google_compute_backend_service.iap_proxy.self_link
}

resource "google_compute_global_address" "iap_proxy" {
  name = "iap-proxy"
}

resource "google_dns_record_set" "iap_proxy" {
  name = format("iap-proxy.%s", data.google_dns_managed_zone.tld.dns_name)
  type = "A"
  ttl  = 300

  managed_zone = data.google_dns_managed_zone.tld.name

  rrdatas = [google_compute_global_address.iap_proxy.address]
}

data "google_dns_managed_zone" "tld" {
  name = var.dns_managed_zone
}