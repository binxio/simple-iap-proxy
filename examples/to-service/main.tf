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
resource "google_compute_backend_service" "httpbin" {
  name             = "httpbin"
  description      = "httpbin service via IAP proxy"
  protocol         = "HTTP"
  port_name        = "httpbin"
  timeout_sec      = 10
  session_affinity = "NONE"

  iap {
    oauth2_client_id     = google_iap_client.httpbin.client_id
    oauth2_client_secret = google_iap_client.httpbin.secret
  }

  backend {
    group = google_compute_region_instance_group_manager.httpbin.instance_group
  }

  health_checks = [
    google_compute_health_check.httpbin.id
  ]
}

resource "google_compute_region_instance_group_manager" "httpbin" {
  name = "httpbin"

  base_instance_name = "httpbin"
  region             = data.google_client_config.current.region

  version {
    instance_template = google_compute_instance_template.httpbin.id
  }

  target_size = 1

  named_port {
    name = "httpbin"
    port = 80
  }

  auto_healing_policies {
    health_check      = google_compute_health_check.httpbin.id
    initial_delay_sec = 300
  }

  update_policy {
    minimal_action               = "REPLACE"
    type                         = "PROACTIVE"
    instance_redistribution_type = "PROACTIVE"
    max_surge_fixed              = 3
  }
}

resource "google_compute_health_check" "httpbin" {
  name                = "httpbin"
  check_interval_sec  = 10
  timeout_sec         = 5
  healthy_threshold   = 2
  unhealthy_threshold = 10 # 100 seconds

  http_health_check {
    port         = "80"
    request_path = "/anything"
  }

  log_config {
    enable = true
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_instance_template" "httpbin" {
  name_prefix  = "httpbin-"
  machine_type = "n2-standard-2"
  region       = data.google_client_config.current.region

  disk {
    source_image = "cos-cloud/cos-stable"
    auto_delete  = true
    boot         = true
  }

  network_interface {
    network = "default"
  }

  tags = ["httpbin"]

  metadata = {
    user-data              = format("#cloud-config\n%s", yamlencode(local.cloud_config))
    enable-oslogin         = "TRUE"
    block-project-ssh-keys = "TRUE"
    serial-port-enable     = "FALSE"
  }

  service_account {
    email = google_service_account.httpbin.email
    scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "google_compute_firewall" "default_allow_ssh_to_httpbin" {
  name    = "default-allow-ssh-to-httpbin"
  network = "default"

  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  target_tags = ["httpbin"]
  source_ranges = [
    "35.235.240.0/20", ## IAP
  ]
}

resource "google_compute_firewall" "default_allow_htto_to_httpbin_from_lb_and_iap" {
  name    = "default-allow-http-to-httpbin-from-lb-and-iap"
  network = "default"

  direction = "INGRESS"

  allow {
    protocol = "tcp"
    ports    = ["80"]
  }

  target_tags = ["httpbin"]
  source_ranges = [
    "35.235.240.0/20", ## IAP
    "35.191.0.0/16",   ## GLB health checks
    "130.211.0.0/22",  ## GLB health checks
  ]
}

resource "google_service_account" "httpbin" {
  account_id   = "httpbin"
  display_name = "httpbin service"
}

resource "google_project_iam_member" "httpbin" {
  for_each = { for r in [
    "roles/logging.logWriter",
    "roles/monitoring.metricWriter",
    "roles/container.clusterViewer",
  ] : r => r }
  member  = format("serviceAccount:%s", google_service_account.httpbin.email)
  role    = each.key
  project = data.google_client_config.current.project
}



locals {
  cloud_config = {
    runcmd = [
      "c_rehash > /dev/null",
      "iptables -I INPUT -p tcp -j ACCEPT",
      "i6ptables -I INPUT -p tcp -j ACCEPT",
      "systemctl daemon-reload",
      "systemctl enable --now httpbin.service"
    ]

    write_files = [{
      path        = "/etc/systemd/system/httpbin.service"
      permissions = "0644"
      owner       = "root:root"
      content     = file("${path.module}/httpbin.service")
      },
    ]
  }
}

