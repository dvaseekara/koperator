module github.com/banzaicloud/koperator

go 1.16

require (
	emperror.dev/errors v0.8.0
	github.com/Shopify/sarama v1.30.0
	github.com/banzaicloud/istio-client-go v0.0.11
	github.com/banzaicloud/istio-operator/pkg/apis v0.10.6
	github.com/banzaicloud/k8s-objectmatcher v1.6.1
	github.com/banzaicloud/koperator/api v0.0.0
	github.com/banzaicloud/koperator/properties v0.1.0
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cncf/xds/go v0.0.0-20211011173535-cb28da3451f1 // indirect
	github.com/envoyproxy/go-control-plane v0.10.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.4.0
	github.com/imdario/mergo v0.3.12
	github.com/jetstack/cert-manager v1.6.0
	github.com/lestrrat-go/backoff v1.0.1
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/pavel-v-chernykh/keystore-go/v4 v4.2.0
	github.com/prometheus/common v0.32.1
	go.uber.org/zap v1.19.1
	golang.org/x/oauth2 v0.0.0-20211028175245-ba495a64dcb5 // indirect
	golang.org/x/sys v0.0.0-20211029165221-6e7872819dc8 // indirect
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	google.golang.org/protobuf v1.27.1
	gopkg.in/inf.v0 v0.9.1
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.22.3
	k8s.io/apiextensions-apiserver v0.22.3
	k8s.io/apimachinery v0.22.3
	k8s.io/client-go v0.22.3
	k8s.io/kube-openapi v0.0.0-20211029090450-ec1f4c89925a // indirect
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b // indirect
	sigs.k8s.io/controller-runtime v0.10.2
)

replace (
	github.com/banzaicloud/koperator/api => ./api
	github.com/banzaicloud/koperator/properties => ./properties
)
