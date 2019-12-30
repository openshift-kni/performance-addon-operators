package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/openshift-kni/performance-addon-operators/pkg/apis"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	"github.com/openshift-kni/performance-addon-operators/version"

	configv1 "github.com/openshift/api/config/v1"
	tunedv1 "github.com/openshift/cluster-node-tuning-operator/pkg/apis/tuned/v1"
	mcov1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

func printVersion() {
	klog.Infof("Operator Version: %s", version.Version)
	klog.Infof("Go Version: %s", runtime.Version())
	klog.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	klog.Infof("Version of operator-sdk: %v", sdkVersion.Version)
}

func main() {
	// Add klog flags
	klog.InitFlags(nil)

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	// we have two namespaces that we need to watch
	// 1. tuned namespace - for tuned resources
	// 2. None namespace - for cluster wide resources
	namespaces := []string{
		components.NamespaceNodeTuningOperator,
		metav1.NamespaceNone,
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		NewCache:                cache.MultiNamespacedCacheBuilder(namespaces),
		LeaderElection:          true,
		LeaderElectionID:        "performance-addon-operators",
		LeaderElectionNamespace: namespace,
		Namespace:               namespace,
		MapperProvider:          restmapper.NewDynamicRESTMapper,
		MetricsBindAddress:      fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	klog.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	if err := configv1.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	if err := mcov1.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	if err := tunedv1.AddToScheme(mgr.GetScheme()); err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	if err = serveCRMetrics(cfg); err != nil {
		klog.Errorf("Could not generate and serve custom resource metrics: %v", err)
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(context.TODO(), cfg, servicePorts)
	if err != nil {
		klog.Errorf("Could not create metrics Service: %v", err)
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		klog.Errorf("Could not create ServiceMonitor object: %v", err)
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			klog.Errorf("Install prometheus-operator in your cluster to create ServiceMonitor objects: %v", err)
		}
	}

	klog.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		klog.Errorf("Manager exited with non-zero code: %v", err)
		os.Exit(1)
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}

	// To generate metrics in other namespaces, add the values below.
	ns := []string{metav1.NamespaceNone}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}
