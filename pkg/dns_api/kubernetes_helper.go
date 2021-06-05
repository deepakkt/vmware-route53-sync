package dns_api

import (
	"flag"
	"fmt"
	"context"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pkg/errors"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Builds the configuration for the kubernetes client.
// It first tries to read it from CLUSTER_KUBECONFIG env variable.
// If the var is empty, then it will read it from the cluster.
func GetClusterConfig() (*rest.Config, error) {
	var clusterConfig *rest.Config
	var err error
	if GetEnv("CLUSTER_KUBECONFIG") == "" {
		fmt.Println("Using incluster kubeconfig")
		clusterConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, errors.Wrap(err, "Error getting cluster config.")
		}
	} else {
		fmt.Printf("Using kubeconfig from %s\n", GetEnv("CLUSTER_KUBECONFIG"))
		kubeconfigFlag := flag.Lookup("kubeconfig")
		if kubeconfigFlag == nil {
			kubeConfig := GetEnv("CLUSTER_KUBECONFIG")
			clusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfig)
			if err != nil {
				return nil, errors.Wrap(err, "Error getting cluster config.")
			}
		} else {
			clusterConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfigFlag.Value.String())
			if err != nil {
				return nil, errors.Wrap(err, "Error getting cluster config.")
			}
		}
	}
	return clusterConfig, err
}

// GetKubernetesClient - returns client that gives access to the cluster
func GetKubernetesClient() (kubernetes.Interface, error) {
	clusterConfig, err := GetClusterConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(clusterConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create a ClientSet: %v. Exiting", err)
	}
	return kubeClient, err
}

// GetConfigmaps - lists configmaps we are interested in
func getConfigmaps() ([]v1.ConfigMap, error) {
	kubeClient, err := GetKubernetesClient()
	if err != nil {
		return nil, errors.Wrap(err, "Error getting kubernetes client.")
	}

	configMaps, err := kubeClient.CoreV1().ConfigMaps("").List(context.TODO(),
			metav1.ListOptions{
				 LabelSelector: "kind=vm-status",
			},
		)

	return configMaps.Items, nil
}


func GetDNStoVMMapping() map[string]string {
	dnsMap := make(map[string]string)

	log.Println("Syncing Kubernetes configmaps")

	configmaps, err := getConfigmaps()

	if err != nil {
		log.Println("Configmap fetch was unsuccessful")
		log.Println("This is an unrecoverable error. No point proceeding")
		return dnsMap
	}

	log.Printf("Fetched %d configmaps\n", len(configmaps))

	for _, cm := range configmaps {
		cmName := fmt.Sprintf("%s/%s", cm.ObjectMeta.Namespace, cm.Name)
		log.Printf("%s\n", cmName)

		if _, ok := cm.Data["VM_NAME"]; !ok {
			log.Printf("VM_NAME was not present in cm %s. We will skip this\n", cmName)
			continue
		}

		if cm.Data["STATUS"] != "deployed" {
			log.Printf("Status indicates '%s', not 'deployed' in %s. Let's skip",
				cm.Data["status"], cmName)
			continue
		}

		dnsMap[cm.Data["URL"]] = cm.Data["VM_NAME"]
	}

	return dnsMap
}