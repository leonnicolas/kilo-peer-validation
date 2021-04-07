#!/usr/bin/env bash

keyDir=$(mktemp -d ./certXXX)
baseDir=$(pwd)
secret_name=peer-validation-webhook-tls
csr_conf=../csr.conf

siginit_handler(){
	echo "caught SIGINT, cleaning up..."
	rm -r $keyDir > /dev/null
	exit 1
}

trap siginit_handler SIGINT

cd $keyDir
echo "Entering dir $(pwd)"

# Generate self signed certificate and private key.
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt -subj "/CN=Kilo Peer Validation CA" -newkey 4096

# Genrate private RSA key.
openssl genrsa -out webhook-server-tls.key 4096

# Generate Certificate singing request for public key of previously generated private key.
openssl req -new -key webhook-server-tls.key -subj "/CN=Kilo Peer Validation Webhook"  -out ca.req

# Sign the certificate with a conf file. This is neccessary because openssl will not copy subjectAltNames by default.
openssl x509 -req  -in ca.req -CA ca.crt -CAkey ca.key -CAcreateserial -extensions v3_req -extfile $csr_conf -out webhook-server-tls.crt

cd ..

kubectl  get namespace kilo -o name | grep -xq namespace/kilo
if [ $? -ne 0 ]; then
	kubectl create namespace kilo
fi

kubectl -n kilo create secret tls $secret_name --cert $keyDir/webhook-server-tls.crt --key $keyDir/webhook-server-tls.key
if [ $? -ne 0 ]; then
	echo ""
	echo -n "Do you want to delete the secret $secret_name? [Y/n] "
	read -n1 yes
	if [[ "$yes" != "" && "$yes" != "y" && "$yes" != "Y" ]]; then
		echo ""
		echo aborting
		rm -r $keyDir
		exit 1
	else
		echo ""
		kubectl delete secret $secret_name
		kubectl create secret tls $secret_name --cert $keyDir/webhook-server-tls.crt --key $keyDir/webhook-server-tls.key
	fi
fi

crt_base64=$(openssl base64 -A <$keyDir/ca.crt)
sed -e s%'${CA_BUNDLE_BASE64}'%$crt_base64%g <$baseDir/deployment-template.yaml > deployment.yaml

echo "run kubectl apply -f deployment.yaml to apply the configuration to your cluster"
rm -r $keyDir
