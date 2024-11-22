#! /bin/bash
kind delete clusters e2e-kind
echo "-------------------------------------------------------------------------"
kind create cluster --config=/Users/dvaseeka/Documents/adobe/kRaft-migration/pipeline-kraft-migration/koperators/koperator/tests/e2e/platforms/kind/kind_config.yaml --name=e2e-kind
echo "-------------------------------------------------------------------------"

## Start cloud-provider=kind
kubectl label node e2e-kind-control-plane node.kubernetes.io/exclude-from-external-load-balancers-
echo "-------------------------------------------------------------------------"

## Prep Kind Cluster for E2E tests
kind load docker-image docker-pipeline-upstream-mirror.dr-uw2.adobeitc.com/adobe/cruise-control:2.5.133-adbe-20240313 --name e2e-kind
echo "-------------------------------------------------------------------------"

kind load docker-image docker-pipeline-upstream-mirror.dr-uw2.adobeitc.com/adobe/kafka:2.13-3.7.0 --name e2e-kind
echo "-------------------------------------------------------------------------"

docker build . -t koperator_e2e_test
echo "-------------------------------------------------------------------------"

kind load docker-image koperator_e2e_test:latest --name e2e-kind
echo "-------------------------------------------------------------------------"

## Rolebindings
kubectl apply -f rolebinding.yaml
echo "-------------------------------------------------------------------------"

go test ./tests/e2e \
                -v \
                -timeout 20m \
                -tags e2e \
                --ginkgo.show-node-events \
                --ginkgo.trace \
                --ginkgo.v