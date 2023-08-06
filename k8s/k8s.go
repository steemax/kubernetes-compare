package k8s

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Cluster struct {
	Name       string
	ConfigPath string
}

// cluster_name:path_to_kubeconfig (map[dev-pcidss:conf/cloud.paa.kubeconfig new-test-dss:conf/new.paa.kubeconfig])
var clusterConfigPaths = make(map[string]string)

func getClusterConfig(configDir string) []Cluster {
	configFiles, err := filepath.Glob(filepath.Join(configDir, "*.kubeconfig"))
	if err != nil {
		log.Println("Error reading config files in conf catalog:", err)
		return nil
	}

	clusters1 := make([]Cluster, 0, len(configFiles))
	for _, configFile := range configFiles {
		config, err := clientcmd.LoadFromFile(configFile)
		if err != nil {
			log.Println("Error loading config file:", err)
			continue
		}

		contextName := config.CurrentContext
		context, ok := config.Contexts[contextName]
		if !ok {
			log.Println("Context not found in config:", contextName)
			continue
		}

		clusterName := context.Cluster
		cluster := Cluster{Name: clusterName, ConfigPath: configFile}
		clusters1 = append(clusters1, cluster)
		// print path to kubeconfig file
		fmt.Println("Config Path for Cluster", clusterName+":", configFile)

	}

	return clusters1
}

func getClusterConfigHome(conf1 string) []Cluster {
	config, err := clientcmd.LoadFromFile(conf1)
	if err != nil {
		log.Fatalf("Error loading config file: %v", err)
	}

	clusters := make([]Cluster, 0, len(config.Contexts))
	for _, context := range config.Contexts {
		clusterName := context.Cluster
		cluster := Cluster{Name: clusterName, ConfigPath: conf1}
		clusters = append(clusters, cluster)
		// print path to kubeconfig file
		fmt.Println("Config Path for Cluster", clusterName+":", conf1)
	}

	return clusters
}

func SetClusterConfig() map[string]string {
	clusterConfigPaths = make(map[string]string)
	Clusters1 := getClusterConfig("./conf/kubeconfig")
	for _, cluster := range Clusters1 {
		clusterConfigPaths[cluster.Name] = cluster.ConfigPath
	}
	return clusterConfigPaths
}

func SetClusterConfigHome() map[string]string {
	clusterConfigPaths = make(map[string]string)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	kubeConfigPath := filepath.Join(homeDir, ".kube", "config")
	Clusters1 := getClusterConfigHome(kubeConfigPath)
	for _, cluster := range Clusters1 {
		clusterConfigPaths[cluster.Name] = cluster.ConfigPath
	}
	return clusterConfigPaths
}

func ClusterVersion(cluster, configPath string, returnSlice bool) (interface{}, error) {

	config, err := clientcmd.LoadFromFile(configPath)
	if err != nil {
		fmt.Printf("Failed to load kubeconfig: %v\n", err)
		return nil, err
	}

	// Find the context corresponding to the given cluster name
	for contextName, context := range config.Contexts {
		if context.Cluster == cluster {
			// Set the current context to the found context
			config.CurrentContext = contextName
			break
		}
	}

	// create api client configuration
	clientcmdapiConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{})
	clientConfig, err := clientcmdapiConfig.ClientConfig()
	if err != nil {
		fmt.Printf("Failed to create client config: %v\n", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		fmt.Println("Failed to create clientset from config")
		return nil, err
	}

	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		log.Println("Failed to get server version:", err)
		return nil, err
	}

	clusterV := version.String()
	if returnSlice {
		return []string{clusterV}, nil
	}
	return clusterV, nil
}

func GetAllClusterVersions(configPaths map[string]string) map[string]string {
	clusterVersions := make(map[string]string)
	for cluster, configPath := range configPaths {

		versionInterface, err := ClusterVersion(cluster, configPath, true)
		if err != nil {
			log.Println("Failed to get version for cluster", cluster, err)
			log.Println("Delete unavailable cluster", cluster, configPath, "from list")
			delete(configPaths, cluster)
			continue
		}
		if version, ok := versionInterface.([]string); ok {
			clusterVersions[cluster] = version[0]
		} else if version, ok := versionInterface.(string); ok {
			clusterVersions[cluster] = version
		}
	}

	return clusterVersions
}

func getNamespaces(cluster, configPath string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()

	config, err := clientcmd.LoadFromFile(configPath)
	if err != nil {
		fmt.Printf("Failed to load kubeconfig: %v\n", err)
		return nil, err
	}

	// Find the context corresponding to the given cluster name
	for contextName, context := range config.Contexts {
		if context.Cluster == cluster {
			// Set the current context to the found context
			config.CurrentContext = contextName
			break
		}
	}

	// create api clients configuration
	clientcmdapiConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{})
	clientConfig, err := clientcmdapiConfig.ClientConfig()
	if err != nil {
		fmt.Printf("Failed to create client config: %v\n", err)
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		fmt.Println("Failed to create clientset from config when get NAMESPACES, cluster:", cluster)
		return nil, err
	}

	namespaceList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get namespace list for", cluster)
		return nil, err
	}

	namespaces := make([]string, len(namespaceList.Items))
	for i, namespace := range namespaceList.Items {
		namespaces[i] = namespace.Name
	}

	return namespaces, nil
}

func FillNamespaces(cluster, configPath string) []string {
	var clusterNamespaces []string
	ns, err := getNamespaces(cluster, configPath)
	if err != nil {
		log.Println("Error getting namespaces for cluster", cluster, err)
	}
	clusterNamespaces = ns
	return clusterNamespaces
}

func GetNodesInfo(cluster, configPath string) (int, int, int, int64, int, error) {
	totalCPUs := 0
	totalMemory := 0
	totalStorage := int64(0)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()
	config, err := clientcmd.LoadFromFile(configPath)
	if err != nil {
		fmt.Printf("Failed to load kubeconfig: %v\n", err)
		return 0, 0, 0, 0, 0, err
	}

	// Find the context corresponding to the given cluster name
	for contextName, context := range config.Contexts {
		if context.Cluster == cluster {
			// Set the current context to the found context
			config.CurrentContext = contextName
			break
		}
	}

	// create config API client
	clientcmdapiConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{})
	clientConfig, err := clientcmdapiConfig.ClientConfig()
	if err != nil {
		fmt.Printf("Failed to create client config: %v\n", err)
		return 0, 0, 0, 0, 0, err
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		fmt.Println("Failed to create clientset from config when get Nodes, cluster:", cluster)
		return 0, 0, 0, 0, 0, err
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
		return 0, 0, 0, 0, 0, err
	}

	for _, node := range nodes.Items {
		cpuStr := node.Status.Capacity[corev1.ResourceCPU]
		cpuQuantity, _ := cpuStr.AsInt64()
		totalCPUs += int(cpuQuantity)
	}
	nodesNum := len(nodes.Items)
	for _, node := range nodes.Items {
		memStr := node.Status.Capacity[corev1.ResourceMemory]
		memQuantity, _ := memStr.AsInt64() // bytes
		totalMemory += int(memQuantity)
	}
	totalMemoryGB := totalMemory / 1024 / 1024 / 1024

	for _, node := range nodes.Items {
		storage := node.Status.Capacity[corev1.ResourceEphemeralStorage]
		storageQuantity := storage.Value() // bytes
		totalStorage += storageQuantity
	}

	totalStorageGB := totalStorage / 1024 / 1024 / 1024

	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get pods")
		return 0, 0, 0, 0, 0, err
	}

	totalPods := len(pods.Items)

	return nodesNum, totalCPUs, totalMemoryGB, totalStorageGB, totalPods, nil
}

func GetAPIinfo(cluster, configPath string) (int, *metav1.APIGroupList, error) {

	config, err := clientcmd.LoadFromFile(configPath)
	if err != nil {
		fmt.Printf("Failed to load kubeconfig: %v\n", err)
	}

	// Find the context corresponding to the given cluster name
	for contextName, context := range config.Contexts {
		if context.Cluster == cluster {
			// Set the current context to the found context
			config.CurrentContext = contextName
			break
		}
	}

	// create config API client
	clientcmdapiConfig := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{})
	clientConfig, err := clientcmdapiConfig.ClientConfig()
	if err != nil {
		fmt.Printf("Failed to create client config: %v\n", err)
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		fmt.Println("Failed to create clientset from config when get ApiResources, cluster:", cluster)
	}
	resources, err := clientset.Discovery().ServerPreferredResources()
	if err != nil {
		fmt.Println("Failed to get API resources")
	}

	resourceMap := make(map[string]bool)
	for _, list := range resources {
		for _, resource := range list.APIResources {
			resourceMap[resource.Kind] = true
		}
	}

	groups, err := clientset.Discovery().ServerGroups()
	if err != nil {
		fmt.Printf("error")
	}
	apiNums := len(resourceMap)

	return apiNums, groups, err
}
func GetCanaryPerNs(cluster, configPath string, namespace string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		fmt.Println("Failed to build config from kubeconfig file when get ApiResources, cluster:", cluster)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to create client:", err)
		return nil
	}

	gvr := schema.GroupVersionResource{Group: "flagger.app", Version: "v1beta1", Resource: "canaries"}
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get resources:", err)
		return nil
	}
	// convert []unstructured.Unstructured to []string and return canary names
	var canaryNames []string
	for _, unstructuredObj := range unstructuredList.Items {
		canaryNames = append(canaryNames, unstructuredObj.GetName())
	}
	return canaryNames
}

func GetCanaryObjectsPerNs(cluster, configPath string, namespace string) []unstructured.Unstructured {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		fmt.Println("Failed to build config from kubeconfig file when get ApiResources, cluster:", cluster)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to create client:", err)
		return nil
	}

	gvr := schema.GroupVersionResource{Group: "flagger.app", Version: "v1beta1", Resource: "canaries"}
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get resources:", err)
		return nil
	}

	// return canary as []unstructured.Unstructured
	return unstructuredList.Items
}

func GetPerCluster(cluster, configPath string) (int, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		fmt.Println("Failed to build config from kubeconfig file when get ApiResources, cluster:", cluster)
		return 1, 1
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to create client:", err)
		return 0, 0
	}
	// get canary from Cluster scope
	gvr := schema.GroupVersionResource{Group: "flagger.app", Version: "v1beta1", Resource: "canaries"}
	unstructuredList, err := dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get resources Flagger:", err)
		return 0, 0
	}

	canaryNamesByNamespace := make(map[string][]string)
	totalCanaryCount := 0
	for _, unstructuredObj := range unstructuredList.Items {
		namespace := unstructuredObj.GetNamespace()
		name := unstructuredObj.GetName()
		canaryNamesByNamespace[namespace] = append(canaryNamesByNamespace[namespace], name)
		totalCanaryCount++
	}
	// get Traefik ingressroutes from Cluster scope
	ing := schema.GroupVersionResource{Group: "traefik.containo.us", Version: "v1alpha1", Resource: "ingressroutes"}
	ingList, err := dynamicClient.Resource(ing).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get resources Traefik IngressRoutes:", err)
		return 0, 0
	}
	totalIngCount := len(ingList.Items)

	return totalCanaryCount, totalIngCount
}

func GetCanaryMetricTemplateObjectsPerNs(cluster, configPath string, namespace string) []unstructured.Unstructured {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		fmt.Println("Failed to build config from kubeconfig file when get MT for canary, cluster:", cluster)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to create client for get MT in flagger.app::", err)
		return nil
	}

	gvr := schema.GroupVersionResource{Group: "flagger.app", Version: "v1beta1", Resource: "metrictemplates"}
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get resources metrictemplates in flagger.app:", err)
		return nil
	}

	// return metrictemplates list as []unstructured.Unstructured
	return unstructuredList.Items
}

func GetMTPerNs(cluster, configPath string, namespace string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		fmt.Println("Failed to build config from kubeconfig file when get MT for canary, cluster:", cluster)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to create client for get MT in flagger.app:", err)
		return nil
	}

	gvr := schema.GroupVersionResource{Group: "flagger.app", Version: "v1beta1", Resource: "metrictemplates"}
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get resources metrictemplates in flagger.app:", err)
		return nil
	}

	// convert []unstructured.Unstructured to []string return names for metrictemplates
	var mtNames []string
	for _, unstructuredObj := range unstructuredList.Items {
		mtNames = append(mtNames, unstructuredObj.GetName())
	}
	return mtNames
}

func GetDeployPerNs(cluster, configPath string, namespace string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		fmt.Println("Failed to build config from kubeconfig file when get Deployments:", cluster)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to create client for get Deployments:", err)
		return nil
	}

	gvr := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get resources deployments:", err)
		return nil
	}

	// convert []unstructured.Unstructured to []string return deployments names
	var deployNames []string
	for _, unstructuredObj := range unstructuredList.Items {
		deployNames = append(deployNames, unstructuredObj.GetName())
	}
	return deployNames
}

func GetUniversalObjectsPerNsUnstruct(cluster, configPath string, namespace string, group string, version string, resource string) []unstructured.Unstructured {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		fmt.Println("Failed to build config from kubeconfig file when get ApiResources, cluster:", cluster)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to create client:", err)
		return nil
	}

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get resources:", err)
		return nil
	}

	// return objects as []unstructured.Unstructured
	return unstructuredList.Items
}

func GetUniversalObjectPerNsAsString(cluster, configPath string, namespace string, group string, version string, resource string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // timeout wait cluster response
	defer cancel()
	config, err := clientcmd.BuildConfigFromFlags("", configPath)
	if err != nil {
		fmt.Println("Failed to build config from kubeconfig file in k8s.GetUniversalObjectPerNsAsString func:", cluster)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		fmt.Println("Failed to create client in k8s.GetUniversalObjectPerNsAsString func:", err)
		return nil
	}

	gvr := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	unstructuredList, err := dynamicClient.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		fmt.Println("Failed to get resources in k8s.GetUniversalObjectPerNsAsString func):", err)
		return nil
	}

	// convert []unstructured.Unstructured to []string return objects as strings
	var universalObjectNames []string
	for _, unstructuredObj := range unstructuredList.Items {
		universalObjectNames = append(universalObjectNames, unstructuredObj.GetName())
	}
	return universalObjectNames
}
