package netlox

import (
	"context"
	"fmt"

	"github.com/plunder-app/plndr-cloud-provider/pkg/ipam"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	cloudprovider "k8s.io/cloud-provider"
	"k8s.io/klog"
)

type loxiServices struct {
	Services []services `json:"services"`
}

type services struct {
	Vip         string `json:"vip"`
	Port        int    `json:"port"`
	Type        string `json:"type"`
	UID         string `json:"uid"`
	ServiceName string `json:"serviceName"`
}

type loadbalancers struct {
	kubeClient     *kubernetes.Clientset
	nameSpace      string
	cloudConfigMap string
}

func newLoadBalancers(kubeClient *kubernetes.Clientset, ns, cm, serviceCidr string) cloudprovider.LoadBalancer {
	return &loadbalancers{
		kubeClient:     kubeClient,
		nameSpace:      ns,
		cloudConfigMap: cm,
	}
}

// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (lb *loadbalancers) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {
	klog.V(5).Info("GetLoadBalancer()")

	// Retrieve the netlox configuration from it's namespace
	cm, err := lb.GetConfigMap(ctx, NetloxClientConfig, service.Namespace)
	if err != nil {
		return nil, true, nil
	}

	// Find the services configuration in the configMap
	svc, err := lb.GetServices(cm)
	if err != nil {
		return nil, false, err
	}

	for x := range svc.Services {
		if svc.Services[x].UID == string(service.UID) {
			return &service.Status.LoadBalancer, true, nil
		}
	}
	return nil, false, nil
}

// GetLoadBalancerName returns the name of the load balancer. Implementations must treat the
// *v1.Service parameter as read-only and not modify it.
func (lb *loadbalancers) GetLoadBalancerName(ctx context.Context, clusterName string, service *v1.Service) string {
	klog.V(5).Info("GetLoadBalancerName()")
	return cloudprovider.DefaultLoadBalancerName(service)
}

// EnsureLoadBalancer creates a new load balancer 'name', or updates the existing one. Returns the status of the balancer
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (lb *loadbalancers) EnsureLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (*v1.LoadBalancerStatus, error) {
	klog.V(5).Info("EnsureLoadBalancer()")
	return lb.syncLoadBalancer(ctx, service)
}

// UpdateLoadBalancer updates hosts under the specified load balancer.
// Implementations must treat the *v1.Service and *v1.Node
// parameters as read-only and not modify them.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (lb *loadbalancers) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (err error) {
	klog.V(5).Info("UpdateLoadBalancer()")
	_, err = lb.syncLoadBalancer(ctx, service)
	return err
}

// EnsureLoadBalancerDeleted deletes the specified load balancer if it
// exists, returning nil if the load balancer specified either didn't exist or
// was successfully deleted.
// This construction is useful because many cloud providers' load balancers
// have multiple underlying components, meaning a Get could say that the LB
// doesn't exist even if some part of it is still laying around.
// Implementations must treat the *v1.Service parameter as read-only and not modify it.
// Parameter 'clusterName' is the name of the cluster as presented to kube-controller-manager
func (lb *loadbalancers) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	klog.V(5).Info("EnsureLoadBalancerDeleted()")
	return lb.deleteLoadBalancer(ctx, service)
}

func (lb *loadbalancers) deleteLoadBalancer(ctx context.Context, service *v1.Service) error {
	klog.Infof("deleting service '%s' (%s)", service.Name, service.UID)

	// Get the netlox (client) configuration from it's namespace
	cm, err := lb.GetConfigMap(ctx, NetloxClientConfig, service.Namespace)
	if err != nil {
		klog.Errorf("The configMap [%s] doensn't exist", NetloxClientConfig)
		return nil
	}
	// Find the services configuraiton in the configMap
	svc, err := lb.GetServices(cm)
	if err != nil {
		klog.Errorf("The service [%s] in configMap [%s] doensn't exist", service.Name, NetloxClientConfig)
		return nil
	}

	// Update the services configuration, by removing the  service
	updatedSvc := svc.delServiceFromUID(string(service.UID))
	if len(service.Status.LoadBalancer.Ingress) != 0 {
		err = ipam.ReleaseAddress(service.Namespace, service.Spec.LoadBalancerIP)
		if err != nil {
			klog.Errorln(err)
		}
	}
	// Update the configMap
	_, err = lb.UpdateConfigMap(ctx, cm, updatedSvc)
	return err
}

func (lb *loadbalancers) syncLoadBalancer(ctx context.Context, service *v1.Service) (*v1.LoadBalancerStatus, error) {

	// CREATE / UPDATE LOAD BALANCER LOGIC (and return updated load balancer IP)

	// Get the clound controller configuration map
	controllerCM, err := lb.GetConfigMap(ctx, NetloxCloudConfig, "kube-system")
	if err != nil {
		klog.Errorf("Unable to retrieve netlox ipam config from configMap [%s] in kube-system", NetloxClientConfig)
		// TODO - determine best course of action, create one if it doesn't exist
		controllerCM, err = lb.CreateConfigMap(ctx, NetloxCloudConfig, "kube-system")
		if err != nil {
			return nil, err
		}
	}

	// Retrieve the netlox configuration map
	namespaceCM, err := lb.GetConfigMap(ctx, NetloxClientConfig, service.Namespace)
	if err != nil {
		klog.Errorf("Unable to retrieve netlox service cache from configMap [%s] in [%s]", NetloxClientConfig, service.Namespace)
		// TODO - determine best course of action
		namespaceCM, err = lb.CreateConfigMap(ctx, NetloxClientConfig, service.Namespace)
		if err != nil {
			return nil, err
		}
	}

	// This function reconciles the load balancer state
	klog.Infof("syncing service '%s' (%s)", service.Name, service.UID)

	// Find the services configuraiton in the configMap
	svc, err := lb.GetServices(namespaceCM)
	if err != nil {
		klog.Errorf("Unable to retrieve services from configMap [%s], [%s]", NetloxClientConfig, err.Error())

		// TODO best course of action, currently we create a new services config
		svc = &loxiServices{}
	}

	// Check for existing configuration

	existing := svc.findService(string(service.UID))
	if existing != nil {
		klog.Infof("found existing service '%s' (%s) with vip %s", service.Name, service.UID, existing.Vip)
		return &service.Status.LoadBalancer, nil
	}

	if service.Spec.LoadBalancerIP == "" {
		service.Spec.LoadBalancerIP, err = discoverAddress(controllerCM, service.Namespace, lb.cloudConfigMap)
		if err != nil {
			return nil, err
		}
	}

	// TODO - manage more than one set of ports
	newSvc := services{
		ServiceName: service.Name,
		UID:         string(service.UID),
		Type:        string(service.Spec.Ports[0].Protocol),
		Vip:         service.Spec.LoadBalancerIP,
		Port:        int(service.Spec.Ports[0].Port),
	}

	klog.Infof("Updating service [%s], with load balancer address [%s]", service.Name, service.Spec.LoadBalancerIP)

	// FIXME: Need to modify go.mod
	_, err = lb.kubeClient.CoreV1().Services(service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	if err != nil {
		// release the address internally as we failed to update service
		ipamerr := ipam.ReleaseAddress(service.Namespace, service.Spec.LoadBalancerIP)
		if ipamerr != nil {
			klog.Errorln(ipamerr)
		}
		return nil, fmt.Errorf("Error updating Service Spec [%s] : %v", service.Name, err)
	}

	svc.addService(newSvc)

	namespaceCM, err = lb.UpdateConfigMap(ctx, namespaceCM, svc)
	if err != nil {
		return nil, err
	}
	return &service.Status.LoadBalancer, nil
}

func discoverAddress(cm *v1.ConfigMap, namespace, configMapName string) (vip string, err error) {
	var cidr, ipRange string
	var ok bool

	// Find Cidr
	cidrKey := fmt.Sprintf("cidr-%s", namespace)
	// Lookup current namespace
	if cidr, ok = cm.Data[cidrKey]; !ok {
		klog.Info(fmt.Errorf("No cidr config for namespace [%s] exists in key [%s] configmap [%s]", namespace, cidrKey, configMapName))
		// Lookup global cidr configmap data
		if cidr, ok = cm.Data["cidr-global"]; !ok {
			klog.Info(fmt.Errorf("No global cidr config exists [cidr-global]"))
		} else {
			klog.Infof("Taking address from [cidr-global] pool")
		}
	} else {
		klog.Infof("Taking address from [%s] pool", cidrKey)
	}
	if ok {
		vip, err = ipam.FindAvailableHostFromCidr(namespace, cidr)
		if err != nil {
			return "", err
		}
		return
	}

	// Find Range
	rangeKey := fmt.Sprintf("range-%s", namespace)
	// Lookup current namespace
	if ipRange, ok = cm.Data[rangeKey]; !ok {
		klog.Info(fmt.Errorf("No range config for namespace [%s] exists in key [%s] configmap [%s]", namespace, rangeKey, configMapName))
		// Lookup global range configmap data
		if ipRange, ok = cm.Data["range-global"]; !ok {
			klog.Info(fmt.Errorf("No global range config exists [range-global]"))
		} else {
			klog.Infof("Taking address from [range-global] pool")
		}
	} else {
		klog.Infof("Taking address from [%s] pool", rangeKey)
	}
	if ok {
		vip, err = ipam.FindAvailableHostFromRange(namespace, ipRange)
		if err != nil {
			return vip, err
		}
		return
	}
	return "", fmt.Errorf("No IP address ranges could be found either range-global or range-<namespace>")
}

//////////////////////////////////////////// sample lb with explanation of lifecycle ///////////////////////////////////////////////
/*

package netlox

import (
	“context”
	“fmt”

	v1 “k8s.io/api/core/v1”
	“k8s.io/client-go/kubernetes”
	cloudprovider “k8s.io/cloud-provider”
	“k8s.io/klog”
)

//netloxLBManager -
type netloxLBManager struct {
	kubeClient     *kubernetes.Clientset
	nameSpace      string
}

func newLoadBalancer() cloudprovider.LoadBalancer {
	// Needs code to get a kubeclient => client
	// Needs code to get a namespace to operate in => namespace

	return &netloxLBManager{
		kubeClient: client,
		namespace: ns,}
}

func (tlb *netloxLBManager) EnsureLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (lbs *v1.LoadBalancerStatus, err error) {
	return tlb.syncLoadBalancer(service)
}
func (tlb *netloxLBManager) UpdateLoadBalancer(ctx context.Context, clusterName string, service *v1.Service, nodes []*v1.Node) (err error) {
	_, err = tlb.syncLoadBalancer(service)
	return err
}

func (tlb *netloxLBManager) EnsureLoadBalancerDeleted(ctx context.Context, clusterName string, service *v1.Service) error {
	return tlb.deleteLoadBalancer(service)
}

func (tlb *netloxLBManager) GetLoadBalancer(ctx context.Context, clusterName string, service *v1.Service) (status *v1.LoadBalancerStatus, exists bool, err error) {

	// RETRIEVE EXISTING LOAD BALANCER STATUS

	return &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{
			{
				IP: vip,
			},
		},
	}, nil
}

// GetLoadBalancerName returns the name of the load balancer. Implementations must treat the
// *v1.Service parameter as read-only and not modify it.
func (tlb *netloxLBManager) GetLoadBalancerName(_ context.Context, clusterName string, service *v1.Service) string {
	return getDefaultLoadBalancerName(service)
}

func getDefaultLoadBalancerName(service *v1.Service) string {
	return cloudprovider.DefaultLoadBalancerName(service)
}
func (tlb *netloxLBManager) deleteLoadBalancer(service *v1.Service) error {
	klog.Infof(“deleting service ‘%s’ (%s)”, service.Name, service.UID)

	// DELETE LOAD BALANCER LOGIC

	return err
}

func (tlb *netloxLBManager) syncLoadBalancer(service *v1.Service) (*v1.LoadBalancerStatus, error) {

	// CREATE / UPDATE LOAD BALANCER LOGIC (and return updated load balancer IP)

	return &v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{
			{
				IP: vip,
			},
		},
	}, nil
}

*/
