# Kilo Peer Validation Webhook

Validation of [Kilo](https://github.com/squat/kilo) Peers with a Validation Webhook.

## About

The Kubernetes API Server only talks to Validation Webhooks via tls connections.
So for the webhook to work, we need to create certificates with openssl.

To make this easy, the `set-up.sh` script can be used.

## Getting started

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
 - create the file `deployment.yaml` with the kubernetes manifest
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
