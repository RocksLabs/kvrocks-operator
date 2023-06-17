#!/usr/bin/env bash

set -euo pipefail

action=$1
profile=$3

KUBE_VERSION=${KUBE_VERSION:-"v1.22.3"}

if ! command -v minikube &>/dev/null; then
  echo "Minikube is not installed or not in the PATH"
  exit 1
fi

case "$action" in
up)
  minikube start --driver=docker --force --kubernetes-version="${KUBE_VERSION}" --nodes 3 -p "${profile}"
  ;;
down)
  minikube delete --profile "${profile}"
  ;;
*)
  echo "Unknown action: ${action}"
  exit 1
  ;;
esac
