#!/bin/bash
# Run E2E tests (Groovy + Spock + Fabric8 + Gradle)
set -eux
source "qa-tests-backend-single-step/scripts/common.sh"
source "qa-tests-backend-single-step/scripts/config.sh"
cd "$STACKROX_SOURCE_ROOT"  # all paths should be relative to here

SCRIPT_ROOT=$(realpath "$(dirname "$0")")  # brew install coreutils
echo "SCRIPT_ROOT          : $SCRIPT_ROOT"
echo "QA_TESTS_BACKEND_DIR : $QA_TESTS_BACKEND_DIR"

echo "Creating $QA_TESTS_BACKEND_DIR/qa-test-settings.properties"
pass show qa-test-settings.properties.v2 \
    > "$QA_TESTS_BACKEND_DIR/qa-test-settings.properties"

cd "$QA_TESTS_BACKEND_DIR"
REGISTRY_USERNAME="$(pass quay-io-ro-username)"; export REGISTRY_USERNAME
REGISTRY_PASSWORD="$(pass quay-io-ro-password)"; export REGISTRY_PASSWORD

# Disabling build to accelerate dev loop -- takes 3-5 minutes on my laptop
if false; then
    make style proto-generated-srcs
else
    echo "SKIPPING BUILD TO SPEEDUP DEV LOOP"
fi

export CLUSTER="OPENSHIFT"
export AWS_ECR_REGISTRY_NAME="051999192406"
export AWS_ECR_REGISTRY_REGION="us-east-2"

AWS_ECR_DOCKER_PULL_PASSWORD="$(aws ecr get-login-password)" || true
export AWS_ECR_DOCKER_PULL_PASSWORD

QUAY_USERNAME="$(pass quay-io-ro-username)"
QUAY_PASSWORD="$(pass quay-io-ro-password)"
export QUAY_USERNAME QUAY_PASSWORD

export KUBECONFIG="/tmp/kubeconfig"
pkill -f 'port-forward.*svc/central' || true
sleep 2
kubectl port-forward -n stackrox svc/central 8000:443 &> /tmp/central.log &
sleep 3

# The Groovy e2e api tests require these two variables are set
export API_HOSTNAME="localhost"
export API_PORT="8000"

nc -vz "$API_HOSTNAME" "$API_PORT" \
    || error "FAILED: [nc -vz $API_HOSTNAME $API_PORT]"

PASSWORD_FILE_PATH="$GOPATH/src/github.com/stackrox/stackrox/deploy/openshift/central-deploy/password"
ROX_USERNAME="admin"
ROX_PASSWORD=$(cat "$PASSWORD_FILE_PATH")
export ROX_USERNAME ROX_PASSWORD
echo "Access Central console at localhost:8000"
echo "Login with ($ROX_USERNAME, $ROX_PASSWORD)"

gradle test --tests='ImageScanningTest'
#gradle test --tests='ImageScanningTest.Image metadata from registry test'
