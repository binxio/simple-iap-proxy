simple IAP proxy for accessing a private GKE master control plane
=================================================================
GKE CIS security control 6.6.4 requires clusters to be created with private endpoint enabled
and public access disabled. When public access is disabled, CI/CD pipelines which need
to deploy objects in the k8s cluster, not be able to.

This simple IAP proxy allows you to access a private GKE master control plane
via the Identity Aware Proxy.

## prerequisites
To deploy the IAP proxy you need the following:

- a google project id with a default network
- a GKE cluster with a private endpoint
- a Google DNS managed zone, which is publicly accessible
- a user you want to grant access

To configure your deployment, create a file `.auto.tfvars` with the following content:

```hcl
# project and region to deploy to IAP proxy into
project = "my-project"
region = "europe-west4"

# target cluster to forward the requests to
target_cluster = {
    name = "cluster-1"
    location = "europe-west4-c"
}

## DNS managed zone accessible from the public internet
dns_managed_zone = "my-managed-zone"
accessors = [
    "user:markvanholsteijn@binx.io",
]

# support email address for the IAP brand.
# if there is an IAP brand in your project, make this empty string: ""
# To check whether you already have a brand, type `gcloud alpha iap oauth-brands list`
iap_support_email = "markvanholsteijn@binx.io"
```

## deploying the IAP proxy
To deploy the IAP proxy for GKE, type:

```
$ git clone https://github.com/binxio/simple-iap-proxy.git
$ cp .auto.tfvars simple-iap-proxy/terraform
$ terraform init
$ terraform apply
```

After the apply, the required IAP proxy command is printed:
```
iap_proxy_command = <<EOT
simple-iap-proxy  \
  --rename-auth-header \
  --target-url https://iap-proxy.my.cloud.dev \
  --iap-audience 1234567890-j9onig1ofcgle7iogv8fceu04v8hriuv.apps.googleusercontent.com \
  --service-account iap-proxy-accessor@my-project.iam.gserviceaccount.com \
  --certificate-file server.crt \
  --key-file server.key

EOT
```

## start the IAP proxy
To start the IAP proxy, you need a certificate. To generate a self-signed certificate, type:

```shell-terminal
$ openssl genrsa -out server.key 2048
$ openssl req -new -x509 -sha256 \
    -key server.key \
    -subj "/CN=localhost" \
    -addext "subjectAltName = DNS:localhost" \
    -days 3650 \
    -out server.crt
```
Now you can start the proxy, by copying the outputted command:

```shell-terminal
$ go install github.com/binxio/simple-iap-proxy@0.2.0
```
The reason for the self-signed certificate is that kubectl will not send the credentials over HTTP.

## get credentials for your cluster
To get the credentials for your cluster, type:

```shell-terminal
$ gcloud container clusters \
   get-credentials cluster-1
````

## configure kubectl access via IAP proxy
To configure the kubectl access via the IAP proxy, type:

```$shell-terminal
gcloud container clusters \
   get-credentials cluster-1
context_name=$(kubectl config current-context)
kubectl config set clusters.$context_name.certificate-authority-data $(base64 < server.crt)
kubectl config set clusters.$context_name.server https://localhost:8443
```

This points the context to the proxy and configure the self-signed certificate for the server.

## use kubectl over IAP
Now you can use kubectl over IAP!

```shell-terminal
$ kubectl cluster-info dump
```

## todo
- support proxying to multiple k8s clusters in the project
- deploy across multiple regions
