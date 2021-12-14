package netlox

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog"
)

// OutSideCluster allows the controller to be started using a local kubeConfig for testing
var OutSideCluster bool

// The netlox cloud provider implementation. Encapsulates a client to talk to our cloud provider
// and the interfaces needed to satisfy the cloudprovider.Interface interface.
type netlox struct {
	providerName  string
	instances     cloudprovider.Instances
	zones         cloudprovider.Zones
	loadbalancers cloudprovider.LoadBalancer
}

const (
	//ProviderName is the name of the cloud provider
	ProviderName = "netlox"

	//PlunderCloudConfig is the default name of the load balancer config Map
	NetloxCloudConfig = "netlox"

	//PlunderClientConfig is the default name of the load balancer config Map
	NetloxClientConfig = "netlox"

	//PlunderServicesKey is the key in the ConfigMap that has the services configuration
	NetloxServicesKey = "netlox-services"
)

// Register the cloud provider
func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(io.Reader) (cloudprovider.Interface, error) {
		return newCloud()
	})
}

// newCloud returns a cloudprovider.Interface
func newCloud() (cloudprovider.Interface, error) {
	ns := os.Getenv("NETLOX_NAMESPACE")
	cm := os.Getenv("NETLOX_CONFIG_MAP")
	cidr := os.Getenv("NETLOX_SERVICE_CIDR")

	if cm == "" {
		cm = NetloxCloudConfig
	}

	if ns == "" {
		ns = "default"
	}

	var cl *kubernetes.Clientset
	if OutSideCluster == false {
		// This will attempt to load the configuration when running within a POD
		cfg, err := rest.InClusterConfig()
		if err != nil {
			klog.Error("error creating kubernetes client config: %s", err.Error())
			return nil, fmt.Errorf("error creating kubernetes client config: %s", err.Error())
		}
		cl, err = kubernetes.NewForConfig(cfg)

		if err != nil {
			klog.Error("error creating kubernetes client: %s", err.Error())
			return nil, fmt.Errorf("error creating kubernetes client: %s", err.Error())
		}
		// use the current context in kubeconfig
	} else {
		config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(os.Getenv("HOME"), ".kube", "config"))
		if err != nil {
			panic(err.Error())
		}
		cl, err = kubernetes.NewForConfig(config)

		if err != nil {
			klog.Error("error creating kubernetes client: %s", err.Error())
			return nil, fmt.Errorf("error creating kubernetes client: %s", err.Error())
		}
	}

	// Bootstrap HTTP client here
	cc := newnetloxClient()

	return &netlox{
		instances:     newInstances(cc),
		zones:         newZones(cc),
		loadbalancers: newLoadBalancers(cl, ns, cm, cidr),
	}, nil
}

// Note that all methods below makes netlox satisfy the cloudprovider.Interface interface!

// Initialize starts any custom cloud controller loops needed for our cloud and
// performs various kinds of housekeeping
func (c *netlox) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
	// Start your own controllers here
	klog.V(5).Info("Initialize()")

	clientset := clientBuilder.ClientOrDie("do-shared-informers")
	sharedInformer := informers.NewSharedInformerFactory(clientset, 0)

	//res := NewResourcesController(c.resources, sharedInformer.Core().V1().Services(), clientset)

	sharedInformer.Start(nil)
	sharedInformer.WaitForCacheSync(nil)
	//go res.Run(stop)
	//go c.serveDebug(stop)
}

func (c *netlox) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	klog.V(5).Info("LoadBalancer()")
	return nil, true
}

func (c *netlox) Instances() (cloudprovider.Instances, bool) {
	klog.V(5).Info("Instances()")
	return c.instances, true
}

func (c *netlox) Zones() (cloudprovider.Zones, bool) {
	klog.V(5).Info("Zones()")
	return c.zones, true
}

// Clusters is not implemented
func (c *netlox) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Routes is not implemented
func (c *netlox) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

// ProviderName returns this cloud providers name
func (c *netlox) ProviderName() string {
	klog.V(5).Infof("ProviderName() returned %s", ProviderName)
	return ProviderName
}

func (c *netlox) HasClusterID() bool {
	klog.V(5).Info("HasClusterID()")
	return true
}
