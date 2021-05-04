# Kilo Peer Validation Webhook

Validation of [Kilo](https://github.com/squat/kilo) Peers with a Validation Webhook.

## Note

The Webhook is very simple.
The only annoying thing is, that the Kubernetes API Server only talks to Validation Webhooks via tls connections.
So for the webhook to work, we need to create tls certificates.

To make this easy, you can use one of the two following methods.

### Use [kube-webhook-certgen](https://github.com/jet/kube-webhook-certgen)

You wil need `docker` installed.
Note, that [kube-webhook-certgen](https://github.com/jet/kube-webhook-certgen) uses a depricated API to sign the certificate and will be unavailable in v1.22+.

Apply the ValidatingWebhookConfiguration, Deployment and Service without a CaBundle with
```bash
kubectl apply -f https://raw.githubusercontent.com/leonnicolas/kilo-peer-validation/main/deployment-no-cabundle.yaml
```

Then create a CA and tls certificates and apply them to your cluster with
```bash
docker run -v ~/.kube/k3s.yaml:/kubeconfig.yaml:ro jettech/kube-webhook-certgen:v1.5.2 --kubeconfig /kubeconfig.yaml create --namespace kilo --secret-name peer-validation-webhook-tls --host peer-validation,peer-validation.kilo.svc,peer-validation.kilo.svc.cluster.local --key-name tls.key --cert-name tls.crt
```
_(Don't forget the change the path to your kubeconfig!)_

Then patch the ValidatingWebhookConfiguration with the CaBundle and the Deployment with the tls certificates with 
```bash
docker run -v ~/.kube/k3s.yaml:/kubeconfig.yaml:ro jettech/kube-webhook-certgen:v1.5.2 --kubeconfig /kubeconfig.yaml patch --webhook-name peer-validation.kilo.svc --secret-name peer-validation-webhook-tls --namespace kilo --patch-mutating=false
```

__Note__: The created secret contains the CA.
It will be mounted into the container of the webhook server, though the CA is not needed by the server.

### Use the set up script

You need `openssl` and `kubectl` installed and the $KUBECONFIG variable set.

Clone the repository and `cd` into the `kilo-peer-valdation` repository
```bash
git clone https://github.com/leonnicolas/kilo-peer-validation.git
cd kilo-peer-validation
```

Edit the `deployment-template.yaml` file, if you like.
If you change the secret's name, you will also have to change the `set-up.sh` script.

Run the `set-up.sh` script to:
 - create a self signed certificate and use it to sign the certificate later used by the webhook
 - create a secret in your cluster with the tls certificate and private key that will be mounted into the webhook's container
 - create the file `deployment.yaml` with the Kubernetes manifest
```
./set-up.sh
```

If you are happy with the manifest, deploy it with
```bash
kubectl apply -f deployment.yaml
```

This will create a
 - ValidatingWebhookConfiguration
 - Deployment with the webook server
 - Service

 __Note__: After the signing the CA is discarded so it cannot be used to sign more certificates.
