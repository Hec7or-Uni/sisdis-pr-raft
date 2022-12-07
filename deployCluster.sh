#!/bin/sh

kubectl delete statefulset ss
kubectl delete service ss-service
kubectl delete pod c1
kubectl create -f statefulset_go.yaml
