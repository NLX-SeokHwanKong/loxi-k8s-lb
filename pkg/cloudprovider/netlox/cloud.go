package netlox

import (
	"io"

	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog"
)

const ProviderName = "netlox"

// The netlox cloud provider implementation. Encapsulates a client to talk to our cloud provider
// and the interfaces needed to satisfy the cloudprovider.Interface interface.
type netlox struct {
	providerName  string
	instances     cloudprovider.Instances
	zones         cloudprovider.Zones
	loadbalancers cloudprovider.LoadBalancer
}

// Register the cloud provider
func init() {
	cloudprovider.RegisterCloudProvider(ProviderName, func(io.Reader) (cloudprovider.Interface, error) {
		return newCloud()
	})
}

// newCloud returns a cloudprovider.Interface
func newCloud() (cloudprovider.Interface, error) {
	// Bootstrap HTTP client here
	cc := newnetloxClient()

	return &netlox{
		instances:     newInstances(cc),
		zones:         newZones(cc),
		loadbalancers: newLoadBalancers(cc),
	}, nil
}

// Note that all methods below makes netlox satisfy the cloudprovider.Interface interface!

// Initialize starts any custom cloud controller loops needed for our cloud and
// performs various kinds of housekeeping
func (c *netlox) Initialize(clientBuilder cloudprovider.ControllerClientBuilder, stop <-chan struct{}) {
	// Start your own controllers here
	klog.V(5).Info("Initialize()")
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
