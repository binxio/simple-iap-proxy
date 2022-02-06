//   Copyright 2021 binx.io B.V.
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//
resource "google_compute_backend_service" "iap_proxy" {
  name             = "iap-proxy"
  description      = "IAP proxy into the VPC"
  protocol         = "HTTPS"
  port_name        = "iap-proxy-tls"
  timeout_sec      = 10
  session_affinity = "NONE"

  iap {
    oauth2_client_id     = google_iap_client.iap_proxy.client_id
    oauth2_client_secret = google_iap_client.iap_proxy.secret
  }

  backend {
    group = google_compute_region_instance_group_manager.iap_proxy.instance_group
  }

  health_checks = [
    google_compute_health_check.iap_proxy_https.id
  ]
}

resource "google_compute_region_instance_group_manager" "iap_proxy" {
  name = "iap-proxy"

  base_instance_name = "iap-proxy"
  region             = data.google_client_config.current.region

  version {
    instance_template = google_compute_instance_template.iap_proxy.id
  }

  target_size = 1

  named_port {
    name = "iap-proxy-tls"
    port = 8443
  }

  auto_healing_policies {
    health_check      = google_compute_health_check.iap_proxy_https.id
    initial_delay_sec = 300
  }

  update_policy {
    minimal_action               = "REPLACE"
    type                         = "PROACTIVE"
    instance_redistribution_type = "PROACTIVE"
    max_surge_fixed              = 3
  }
}

resource "google_compute_health_check" "iap_proxy_https" {
  name                = "iap-proxy-https"
  check_interval_sec  = 10
  timeout_sec         = 5
  healthy_threshold   = 2
  unhealthy_threshold = 10 # 100 seconds

  https_health_check {
    port         = "8443"
    request_path = "/__health"
  }

  log_config {
    enable = true
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_instance_template" "iap_proxy" {
  name_prefix  = "iap-proxy-"
  machine_type = "e2-micro"
  region       = data.google_client_config.current.region

  disk {
    source_image = "cos-cloud/cos-stable"
    auto_delete  = true
    boot         = true
  }

  network_interface {
    network = "default"
  }

  tags = ["iap-proxy"]

  metadata = {
    user-data              = format("#cloud-config\n%s", yamlencode(local.cloud_config))
    enable-oslogin         = "TRUE"
    block-project-ssh-keys = "TRUE"
    serial-port-enable     = "FALSE"
  }

  service_account {
    email = google_service_account.iap_proxy.email
    scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_firewall" "default_allow_ssh_to_iap_proxy" {
  name    = "default-allow-ssh-to-iap-proxy"
  network = "default"

  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  target_tags = ["iap-proxy"]
  source_ranges = [
    "35.235.240.0/20", ## IAP
  ]
}

resource "google_compute_firewall" "default_allow_8443_to_iap_proxy_from_lb_and_iap" {
  name    = "default-allow-8443-to-iap-proxy-from-lb-and-iap"
  network = "default"

  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["8443"]
  }

  target_tags = ["iap-proxy"]
  source_ranges = [
    "35.235.240.0/20", ## IAP
    "35.191.0.0/16",   ## GLB health checks
    "130.211.0.0/22",  ## GLB health checks
  ]
}

resource "google_service_account" "iap_proxy" {
  account_id   = "iap-proxy"
  display_name = "IAP proxy"
}

resource "google_project_iam_member" "iap_proxy_sa" {
  for_each = { for r in [
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/container.clusterViewer",
  ] : r => r }
  member  = format("serviceAccount:%s", google_service_account.iap_proxy.email)
  role    = each.key
  project = data.google_client_config.current.project
}


resource "tls_private_key" "iap_proxy" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "tls_self_signed_cert" "iap_proxy" {
  key_algorithm   = "RSA"
  private_key_pem = tls_private_key.iap_proxy.private_key_pem

  subject {
    common_name  = "localhost"
    organization = "Binx.io B.V."
  }

  validity_period_hours = 24 * 365

  allowed_uses = [
    "key_encipherment",
    "digital_signature",
    "server_auth",
  ]

  early_renewal_hours = 24 * 30
}

locals {
  cloud_config = {
    runcmd = [
      "c_rehash > /dev/null",
      "iptables -I INPUT -p tcp -j ACCEPT --dport 8443",
      "i6ptables -I INPUT -p tcp -j ACCEPT --dport 8443",
      "systemctl daemon-reload",
      "systemctl enable --now iap-proxy.service"
    ]

    write_files = [{
      path        = "/etc/ssl/private/iap-proxy.key"
      owner       = "root:root"
      permissions = "0600"
      encoding    = "gzip+base64"
      content     = base64gzip(tls_private_key.iap_proxy.private_key_pem)
      }, {
      path        = "/etc/ssl/certs/iap-proxy.cert.pem"
      owner       = "root:root"
      permissions = "0644"
      encoding    = "gzip+base64"
      content     = base64gzip(tls_self_signed_cert.iap_proxy.cert_pem)
      }, {
      path        = "/etc/systemd/system/iap-proxy.service"
      permissions = "0644"
      owner       = "root:root"
      content     = file("${path.module}/iap-proxy.service")
      },
    ]
  }
}

