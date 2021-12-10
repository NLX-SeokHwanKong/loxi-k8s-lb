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