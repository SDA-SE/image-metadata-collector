#!/bin/bash
set -e

source library.bash

kubectl apply -k argocd
wait_for_pods_ready "argocd" "argocd" 5 10 120


kubectl apply -k ./minio
wait_for_pods_ready "minio operator" "minio-operator" 3 10 120
kubectl apply -k ./minio-tenant


if ! which mc > /dev/null 2>&1; then
  mkdir tmp || true
  curl -sL https://dl.min.io/client/mc/release/linux-amd64/mc --output ./tmp/mc
  chmod +x ./tmp/mc
  PATH=${PATH}:./tmp
fi

sleep 3
wait_for_pods_ready "minio tenant" "cluster-image-scanner-image-collector" 1 10 60

sleep 10 #

for i in $(ps -ef | grep 9000 | grep -v grep | awk '{print $2}'); do
  kill $i
done
kubectl -n cluster-image-scanner-image-collector port-forward svc/minio-hl 9000:9000 > ${TMP_FOLDER}/port-forward.log &
sleep 2 # wait for log
if [ "$(cat ${TMP_FOLDER}/port-forward.log | grep -c 'Forwarding from 127.0.0.1:9000 -> 9000')" -ne 1 ]; then
  echo "Port forwarding didn't work"
  cat ${TMP_FOLDER}/port-forward.log
  exit 1
fi
sleep 1
mc alias set local http://127.0.0.1:9000 testtesttest testtesttest || true
mc mb local/local/lord-of-the-rings/imagecollector || true

sed -i "s~###VERSION###~${VERSION}~g" job.yml

kubectl apply -k ./application
wait_for_pods_ready "test deployment of image" "shire" 1 10 60

kubectl apply -k ./application-pr
wait_for_pods_ready "test deployment of image" "shire-pr-80" 1 10 60

kubectl apply -k .
wait_for_pods_ready "collector" "cluster-image-scanner-image-collector" 2 10 60

if [ "$(kubectl get pods -n cluster-image-scanner-image-collector | grep -c Running)" -ne 2 ]; then
  kubectl get pods -n cluster-image-scanner-image-collector
  for pod in $(kubectl get pods -n cluster-image-scanner-image-collector | grep -v NAME | awk '{print $1}'); do
    echo "####"
    kubectl get pod ${pod} -n cluster-image-scanner-image-collector -o yaml
    echo "####"
    kubectl logs ${pod} -n cluster-image-scanner-image-collector
  done
  echo "Collector is broken"
  exit 1
fi

rm tmp || true

for pod in $(kubectl get pods -n cluster-image-scanner-image-collector | grep -v NAME | awk '{print $1}'); do
  if [ "$(kubectl logs ${pod} -n cluster-image-scanner-image-collector | grep -c 'Done, sleep')" -eq 1 ]; then
    break
  fi
  sleep 1
done
filesToCheck="lord-of-the-rings-output.json lord-of-the-rings-service-description.json" # missing-service-description.txt
for fileToCheck in ${filesToCheck}; do
  ./s3download.bash "${fileToCheck}"
  #compareDownloadedFileWithExpected "${fileToCheck}" # Disabled due to deacitiviation of labels
  rm ${TMP_FOLDER}/${fileToCheck}
done
