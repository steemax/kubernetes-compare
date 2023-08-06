package helm

import (
	"fmt"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/klog/v2"
)

func GetHelmReleasesPerNS(cluster, kubeconfig string, namespace string) ([]string, error) {
	var releases []string

	// Create a new config flags object
	configFlags := genericclioptions.NewConfigFlags(true)
	// Set the kubeconfig path
	configFlags.KubeConfig = &kubeconfig

	// Create the Helm client configuration
	actionConfig := new(action.Configuration)

	// Initialize the Helm client configuration
	if err := actionConfig.Init(configFlags, namespace, os.Getenv("HELM_DRIVER"), klog.Infof); err != nil {
		fmt.Println("Failed to initialize Helm client configuration in GetHelmReleasesPerNS")
		return nil, err
	}

	// Create the Helm list client
	listClient := action.NewList(actionConfig)

	// Configure the Helm list client
	listClient.All = true

	// Run the Helm list command
	results, err := listClient.Run()
	if err != nil {
		fmt.Println("Failed to list Helm releases in GetHelmReleasesPerNS")
		return nil, err
	}

	// Output the results
	for _, release := range results {
		releases = append(releases, release.Name)
	}

	return releases, nil
}

func GetHelmReleasesJsonPerNS(cluster, kubeconfig string, namespace string) ([]unstructured.Unstructured, error) {
	releaseNames, err := GetHelmReleasesPerNS(cluster, kubeconfig, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get helm release names for GetHelmReleasesJsonPerNS: %v", err)
	}

	// Create a new config flags object
	configFlags := genericclioptions.NewConfigFlags(true)
	// Set the kubeconfig path
	configFlags.KubeConfig = &kubeconfig

	// Create the Helm client configuration
	actionConfig := new(action.Configuration)

	// Initialize the Helm client configuration
	if err := actionConfig.Init(configFlags, namespace, os.Getenv("HELM_DRIVER"), klog.Infof); err != nil {
		fmt.Println("Failed to initialize Helm client configuration for GetHelmReleasesJsonPerNS")
		return nil, err
	}

	// Create the Helm GetValues client
	getValuesClient := action.NewGetValues(actionConfig)

	var unstructuredValues []unstructured.Unstructured
	for _, releaseName := range releaseNames {
		releaseValues, err := getValuesClient.Run(releaseName)
		if err != nil {
			fmt.Println("Failed to get values for Helm release in GetHelmReleasesJsonPerNS", releaseName)
			return nil, err
		}
		if releaseValues == nil {
			releaseValues = make(map[string]interface{})
		}
		// Add the release name to the values
		releaseValues["releaseName"] = releaseName
		unstructuredValue := unstructured.Unstructured{Object: releaseValues}
		unstructuredValues = append(unstructuredValues, unstructuredValue)
	}
	return unstructuredValues, nil
}
