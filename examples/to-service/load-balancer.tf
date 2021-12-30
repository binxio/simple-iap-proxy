resource "google_compute_global_forwarding_rule" "httpbin" {
  name       = "httpbin-port-443"
  target     = google_compute_target_https_proxy.httpbin.self_link
  ip_address = google_compute_global_address.httpbin.address
  port_range = "443"
}

resource "google_compute_target_https_proxy" "httpbin" {
  name = "httpbin"

  ssl_certificates = [
    google_compute_managed_ssl_certificate.httpbin.id,
  ]

  url_map = google_compute_url_map.httpbin.self_link
}

resource "google_compute_managed_ssl_certificate" "httpbin" {
  name = random_pet.httpbin_certificate_name.id
  managed {
    domains = [
      google_dns_record_set.httpbin.name
    ]
  }
  lifecycle {
    create_before_destroy = true
  }
}

resource "random_pet" "httpbin_certificate_name" {
  prefix = "httpbin"

  keepers = {
    domains = google_dns_record_set.httpbin.name
  }
}

resource "google_compute_url_map" "httpbin" {
  name = "httpbin"

  default_service = google_compute_backend_service.httpbin.self_link
}

resource "google_compute_global_address" "httpbin" {
  name = "httpbin"
}

resource "google_dns_record_set" "httpbin" {
  name = format("httpbin.%s", data.google_dns_managed_zone.tld.dns_name)
  type = "A"
  ttl  = 300

  managed_zone = data.google_dns_managed_zone.tld.name

  rrdatas = [google_compute_global_address.httpbin.address]
}

data "google_dns_managed_zone" "tld" {
  name = var.dns_managed_zone
}
