package netlox

import (
	"context"
	"encoding/json"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Services functions - once the service data is taken from teh configMap, these functions will interact with the data

func (s *loxiServices) addService(newSvc services) {
	s.Services = append(s.Services, newSvc)
}

func (s *loxiServices) findService(UID string) *services {
	for x := range s.Services {
		if s.Services[x].UID == UID {
			return &s.Services[x]
		}
	}
	return nil
}

func (s *loxiServices) delServiceFromUID(UID string) *loxiServices {
	// New Services list
	updatedServices := &loxiServices{}
	// Add all [BUT] the removed service
	for x := range s.Services {
		if s.Services[x].UID != UID {
			updatedServices.Services = append(updatedServices.Services, s.Services[x])
		}
	}
	// Return the updated service list (without the mentioned service)
	return updatedServices
}

func (s *loxiServices) updateServices(vip, name, uid string) string {
	newsvc := services{
		Vip:         vip,
		UID:         uid,
		ServiceName: name,
	}
	s.Services = append(s.Services, newsvc)
	b, _ := json.Marshal(s)
	return string(b)
}

// ConfigMap functions - these wrap all interactions with the kubernetes configmaps

func (lb *loadbalancers) GetServices(cm *v1.ConfigMap) (svcs *loxiServices, err error) {
	// Attempt to retrieve the config map
	b := cm.Data[NetloxServicesKey]
	// Unmarshall raw data into struct
	err = json.Unmarshal([]byte(b), &svcs)
	return
}

func (lb *loadbalancers) GetConfigMap(ctx context.Context, cm, nm string) (*v1.ConfigMap, error) {
	// Attempt to retrieve the config map
	return lb.kubeClient.CoreV1().ConfigMaps(nm).Get(ctx, lb.cloudConfigMap, metav1.GetOptions{})
}

func (lb *loadbalancers) CreateConfigMap(ctx context.Context, cm, nm string) (*v1.ConfigMap, error) {

	// Create new configuration map in the correct namespace
	newConfigMap := v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lb.cloudConfigMap,
			Namespace: nm,
		},
	}
	// Return results of configMap create
	return lb.kubeClient.CoreV1().ConfigMaps(nm).Create(ctx, &newConfigMap, metav1.CreateOptions{})
}

func (lb *loadbalancers) UpdateConfigMap(ctx context.Context, cm *v1.ConfigMap, s *loxiServices) (*v1.ConfigMap, error) {
	// Create new configuration map in the correct namespace

	// If the cm.Data / cm.Annotations haven't been initialised
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}
	if cm.Annotations == nil {
		cm.Annotations = map[string]string{}
		cm.Annotations["provider"] = ProviderName
	}

	// Set ConfigMap data
	b, _ := json.Marshal(s)
	cm.Data[NetloxServicesKey] = string(b)

	// Return results of configMap create
	return lb.kubeClient.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
}
