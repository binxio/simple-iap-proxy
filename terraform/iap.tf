## you can only create one brand per project
resource "google_iap_brand" "default" {
  count = var.iap_support_email != "" ? 1 : 0

  support_email = var.iap_support_email

  application_title = "IAP Proxy into the VPC"
}

resource "google_iap_client" "iap_proxy" {
  display_name = "IAP GKE proxy"
  brand = (
    var.iap_support_email != "" ?
    google_iap_brand.default[0].name :
    format("projects/%s/brands/%s",
      data.google_project.current.number,
      data.google_project.current.number,
  ))
  depends_on = [google_iap_brand.default]
  ## if a brand is not yet created, specify a brand
}