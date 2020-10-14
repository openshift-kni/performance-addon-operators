module github.com/openshift-kni/performance-addon-operators

go 1.13

require (
	cloud.google.com/go v0.56.0 // indirect
	github.com/RHsyseng/operator-utils v0.0.0-20200213165520-1a022eb07a43
	github.com/blang/semver v3.5.1+incompatible
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f
	github.com/coreos/ignition v0.35.0
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/zapr v0.1.1 // indirect
	github.com/go-openapi/errors v0.19.4 // indirect
	github.com/go-openapi/loads v0.19.5 // indirect
	github.com/go-openapi/runtime v0.19.15 // indirect
	github.com/go-openapi/spec v0.19.7 // indirect
	github.com/go-openapi/strfmt v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.9 // indirect
	github.com/go-openapi/validate v0.19.8 // indirect
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/kr/pretty v0.2.0 // indirect
	github.com/mitchellh/mapstructure v1.2.2 // indirect
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v3.9.1-0.20191111211345-a27ff30ebf09+incompatible
	github.com/openshift/cluster-node-tuning-operator v0.0.0-20200914165052-a39511828cf0
	github.com/openshift/custom-resource-status v0.0.0-20200602122900-c002fd1547ca
	github.com/openshift/machine-config-operator v4.2.0-alpha.0.0.20190917115525-033375cbe820+incompatible
	github.com/operator-framework/operator-lifecycle-manager v0.0.0-20191115003340-16619cd27fa5
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_golang v1.6.0 // indirect
	github.com/prometheus/common v0.10.0 // indirect
	github.com/stretchr/testify v1.5.1 // indirect
	github.com/vincent-petithory/dataurl v0.0.0-20191104211930-d1553a71de50 // indirect
	go.mongodb.org/mongo-driver v1.3.2 // indirect
	go.uber.org/zap v1.14.1 // indirect
	golang.org/x/crypto v0.0.0-20200422194213-44a606286825 // indirect
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/sys v0.0.0-20200602225109-6fdc65e7d980 // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	google.golang.org/appengine v1.6.6 // indirect
	google.golang.org/protobuf v1.24.0 // indirect
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kubelet v0.18.3
	k8s.io/kubernetes v1.18.3
	k8s.io/utils v0.0.0-20200603063816-c1c6865ac451
	kubevirt.io/qe-tools v0.1.6
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/controller-tools v0.3.0
)

// Pinned to kubernetes-1.18.3
replace (
	k8s.io/api => k8s.io/api v0.18.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.3
	k8s.io/apiserver => k8s.io/apiserver v0.18.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.3
	k8s.io/client-go => k8s.io/client-go v0.18.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.3
	k8s.io/code-generator => k8s.io/code-generator v0.18.3
	k8s.io/component-base => k8s.io/component-base v0.18.3
	k8s.io/cri-api => k8s.io/cri-api v0.18.3
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.3
	k8s.io/kubectl => k8s.io/kubectl v0.18.3
	k8s.io/kubelet => k8s.io/kubelet v0.18.3
	k8s.io/kubernetes => k8s.io/kubernetes v1.18.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.3
	k8s.io/metrics => k8s.io/metrics v0.18.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.3
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.3.0
)

// Other pinned deps
replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.1.1+incompatible
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.40.0
	github.com/cri-o/cri-o => github.com/cri-o/cri-o v1.18.1
	github.com/go-log/log => github.com/go-log/log v0.1.0
	github.com/mtrmac/gpgme => github.com/mtrmac/gpgme v0.1.1
	github.com/openshift/api => github.com/openshift/api v0.0.0-20200526144822-34f54f12813a // release-4.5
	github.com/openshift/client-go => github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c // release-4.5
	github.com/openshift/cluster-node-tuning-operator => github.com/openshift/cluster-node-tuning-operator v0.0.0-20200408190329-b227599f61b0 // release-4.5
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20200421122923-c1de486c7d47 // fix bitbucket dependency https://github.com/openshift/library-go/pull/776
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20200618004043-7b1eb84e0083 // release-4.5
	github.com/operator-framework/operator-sdk => github.com/operator-framework/operator-sdk v0.18.1
	golang.org/x/tools => golang.org/x/tools v0.0.0-20191206213732-070c9d21b343
)

replace vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
