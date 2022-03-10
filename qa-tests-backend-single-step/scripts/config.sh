#!/bin/bash
export CLUSTER_NAME="sb-03-09-osdgcp"
export MAIN_IMAGE_TAG="3.68.0"
export KUBECONFIG="/tmp/kubeconfig"
export STACKROX_SOURCE_ROOT="$GOPATH/src/github.com/stackrox/stackrox"
export WORKFLOW_SOURCE_ROOT="$GOPATH/src/github.com/stackrox/workflow"
export STACKROX_TEARDOWN_SCRIPT="$WORKFLOW_SOURCE_ROOT/bin/teardown"
export STACKROX_NAMESPACE="stackrox"
export CENTRAL_BUNDLE_DPATH="/tmp/central-bundle"
