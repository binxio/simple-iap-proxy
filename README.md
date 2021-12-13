simple IAP proxy for accessing a private GKE master control plane
=================================================================
GKE CIS security control 6.6.4 requires clusters to be created with private endpoint enabled
and public access disabled. When public access is disabled, CI/CD pipelines which need
to deploy objects in the k8s cluster, not be able to.

This simple IAP proxy allows you to access a private GKE master control plane
via the Identity Aware Proxy.

![iap proxy to gke private endpoint](https://binx.io/wp-content/uploads/2021/12/simple-iap-proxy-2-1800x937.png)

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

## DNS managed zone accessible from the public internet
dns_managed_zone = "my-managed-zone"

## users you want to grant access via the IAP proxy
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

```sh
git clone https://github.com/binxio/simple-iap-proxy.git
cp .auto.tfvars simple-iap-proxy/terraform
terraform init
terraform apply
```

After the apply, the required IAP proxy command is printed:
```
iap_proxy_command = <<EOT
simple-iap-proxy gke-client \
  --target-url https://iap-proxy.google.binx.dev \
  --iap-audience 712731707077-j9onig1ofcgle7iogv8fceu04v8hriuv.apps.googleusercontent.com \
  --service-account iap-proxy-accessor@speeltuin-mvanholsteijn.iam.gserviceaccount.com \
  --key-file server.key \
  --certificate-file server.crt

EOT
```

## start the IAP proxy
To start the IAP proxy, you need a certificate. To generate a self-signed certificate, type:

```bash
simple-iap-proxy generate-certificate \
  --key-file server.key \
  --certificate-file server.crt
 ```

Or alternatively, use openssl:
```bash
openssl genrsa -out server.key 2048
openssl req -new -x509 -sha256 \
    -key server.key \
    -subj "/CN=localhost" \
    -addext "subjectAltName = DNS:localhost" \
    -days 3650 \
    -out server.crt
```
Now you can start the proxy, by copying the command printed by terraform:

```sh
$ go install github.com/binxio/simple-iap-proxy@0.4.1
$ terraform output -raw iap_proxy_command | sh
```
The reason for the self-signed certificate is that kubectl will not send the credentials over HTTP.

## get credentials for your cluster
To get the credentials for your cluster, type:

```sh
$ gcloud container clusters \
   get-credentials cluster-1
````

## configure kubectl access via IAP proxy
To configure the kubectl access via the IAP proxy, type:

```sh
context_name=$(kubectl config current-context)
kubectl config set clusters.$context_name.certificate-authority-data $(base64 < server.crt)
kubectl config set clusters.$context_name.proxy-url https://localhost:8080
```

This points the context to the proxy and configure the self-signed certificate for the server.

## use kubectl over IAP
Now you can use kubectl over IAP!

```sh
$ kubectl cluster-info dump
```

## using the proxy with other clients
If you want to trust the proxy from other clients than kubectl, add the certificate to the trust store. On MacOS, type:

```bash
sudo security add-trusted-cert -d -p ssl -p basic -k /Library/Keychains/System.keychain ./server.crt
```

On Linux, type:
```bash
cp server.crt /etc/ssl/certs/
c_rehash
```

## Caveats
- The IAP protocol does not support websockets as Authorization header cannot be passed in. Commands which rely
  on websockets will fail (ie kubectl exec).
- the --debug flag is not very verbose.
- The proxy has not been tested yet in the field, so I am happy to hear your feedback!

[Read the blog](https://binx.io/blog/2021/12/11/how-to-connect-to-a-gke-private-endpoint-using-iap/)
