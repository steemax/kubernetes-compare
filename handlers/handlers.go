package handlers

import (
	"compareapp/diff"
	"compareapp/helm"
	"compareapp/k8s"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var compar string
var selectedConfig string
var configPaths map[string]string
var Kubeconfig1, Kubeconfig2 string
var Cluster1, Cluster2 string
var Namespaces1, Namespaces2 []string
var Namespace1, Namespace2 string
var Resources []string
var ClusterVersion1, ClusterVersion2 interface{}
var ConfigType string

type aboutCluster struct {
	Cluster1    string
	Cluster2    string
	Kubeconfig1 string
	Kubeconfig2 string
	Namespaces1 []string
	Namespaces2 []string
	Namespace1  string
	Namespace2  string
	Resources   []string
}

type tableInfra struct {
	ClusterName string
	KubeVersion string
	NodeCount   int
	NsCount     int
	CpuTotal    int
	MemTotal    int
	DiskTotal   int64
	PodTotal    int
	ApiNums     int
	Traefik     string
	TraefikNum  int
	Flagger     string
	FlaggerNum  int
	Gatekeeper  string
	Jaeger      string
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(10 << 20) // Максимум 10 MB файлов
	if err != nil {
		fmt.Print("error reading values in IndexHandler: ", err)
		http.Redirect(w, r, "/select_config", http.StatusSeeOther)
		return
	}
	selectedConfig = r.FormValue("connectionSelect")
	if selectedConfig == "" {
		http.Redirect(w, r, "/select_config", http.StatusSeeOther)

	} else {
		if selectedConfig == "internal" {
			ConfigType = selectedConfig
			configPaths = k8s.SetClusterConfig()
			// Вывод глаыной странички из темплейта
			err := renderPage(w, "templates/index.html", configPaths)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if selectedConfig == "external" {
			ConfigType = selectedConfig
			file1, handler1, err := r.FormFile("cluster1Input")
			if err != nil {
				fmt.Println("Error Retrieving the File1")
				fmt.Println(err)
				return
			}
			defer file1.Close()
			fmt.Printf("Uploaded File: %+v\n", handler1.Filename)
			fmt.Printf("File Size: %+v\n", handler1.Size)
			fmt.Printf("MIME Header: %+v\n", handler1.Header)

			file2, handler2, err := r.FormFile("cluster2Input")
			if err != nil {
				fmt.Println("Error Retrieving the File2")
				fmt.Println(err)
				return
			}
			defer file2.Close()
			fmt.Printf("Uploaded File: %+v\n", handler2.Filename)
			fmt.Printf("File Size: %+v\n", handler2.Size)
			fmt.Printf("MIME Header: %+v\n", handler2.Header)
		} else if selectedConfig == "home" {
			ConfigType = selectedConfig
			configPaths = k8s.SetClusterConfigHome()
			// Вывод глаыной странички из темплейта
			err := renderPage(w, "templates/index.html", configPaths)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
}

func SelectConfigHandler(w http.ResponseWriter, r *http.Request) {

	// Вывод глаыной странички из темплейта
	err := renderPage(w, "templates/pre_index.html", configPaths)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ClearSelectHandler(w http.ResponseWriter, r *http.Request) {
	selectedConfig = ""
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func renderPage(w http.ResponseWriter, page string, data interface{}) error {
	// Подготовка шаблона.
	t, err := template.ParseFiles(page)
	if err != nil {
		return err
	}

	// Выполнение шаблона с переданными данными.
	err = t.Execute(w, data)
	if err != nil {
		return err
	}

	return nil
}

func renderCanaryPage(w http.ResponseWriter, page string, data interface{}) error {
	t, err := template.New(filepath.Base(page)).Funcs(template.FuncMap{
		"formatAsJSON":       formatAsJSON,
		"UnstructuredToJSON": UnstructuredToJSON,
	}).ParseFiles(page)

	if err != nil {
		return fmt.Errorf("error parsing template file %s: %v", page, err)
	}

	err = t.Execute(w, data)
	if err != nil {
		return fmt.Errorf("error executing template: %v", err)
	}
	return nil
}

func UnstructuredToJSON(u interface{}) (string, error) {
	// Попытка привести данные к типу Unstructured
	unstr, ok := u.(*unstructured.Unstructured)
	if !ok {
		// Если данные не являются типом Unstructured, обработать их стандартным образом
		b, err := json.Marshal(u)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	// Если данные являются типом Unstructured, обработать их особым образом
	b, err := json.MarshalIndent(unstr.Object, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func formatAsJSON(s string) template.HTML {
	var data interface{}
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		// handle error
		return template.HTML(err.Error())
	}
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		// handle error
		return template.HTML(err.Error())
	}
	return template.HTML(formatted)
}

func NamespaceHandler(w http.ResponseWriter, r *http.Request) {

	// Получаем имена выбранных кластеров из веб формы странички
	Cluster1 = r.FormValue("cluster1")
	Cluster2 = r.FormValue("cluster2")

	// Получаем неймспейсы и кубконфиги для выбранных кластеров.
	Kubeconfig1 = configPaths[Cluster1]
	Kubeconfig2 = configPaths[Cluster2]
	Namespaces1 := k8s.FillNamespaces(Cluster1, Kubeconfig1)
	Namespaces2 := k8s.FillNamespaces(Cluster2, Kubeconfig2)
	//Namespaces1 = clusterNamespaces[Cluster1]
	//Namespaces2 = clusterNamespaces[Cluster2]

	data := aboutCluster{
		Cluster1:    Cluster1,
		Cluster2:    Cluster2,
		Namespaces1: Namespaces1,
		Namespaces2: Namespaces2,
	}

	// Формируем страницу из шаблона для выбора неймспейса
	err := renderPage(w, "templates/namespaces.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func ResourceHandler(w http.ResponseWriter, r *http.Request) {
	Namespace1 = r.FormValue("namespace1")
	Namespace2 = r.FormValue("namespace2")
	Resources = nil // set to null every time when page requested
	Resources = append(Resources, "ClusterInfra")
	Resources = append(Resources, "HelmValues")
	canaryNum1, ingNum1 := k8s.GetPerCluster(Cluster1, Kubeconfig1)
	canaryNum2, ingNum2 := k8s.GetPerCluster(Cluster2, Kubeconfig2)
	deployNum1 := len(k8s.GetDeployPerNs(Cluster1, Kubeconfig1, Namespace1))
	deployNum2 := len(k8s.GetDeployPerNs(Cluster2, Kubeconfig2, Namespace2))
	daemonSet1 := len(k8s.GetUniversalObjectPerNsAsString(Cluster1, Kubeconfig1, Namespace1, "apps", "v1", "daemonsets"))
	daemonSet2 := len(k8s.GetUniversalObjectPerNsAsString(Cluster2, Kubeconfig2, Namespace2, "apps", "v1", "daemonsets"))
	if deployNum1 >= 1 || deployNum2 >= 1 {
		Resources = append(Resources, "Deployments")
	}
	if daemonSet1 >= 1 || daemonSet2 >= 1 {
		Resources = append(Resources, "Daemonsets")
	}
	if canaryNum1 >= 1 || canaryNum2 >= 1 {
		Resources = append(Resources, "Flagger (Canary)")
	}
	if ingNum1 >= 1 || ingNum2 >= 1 {
		Resources = append(Resources, "IngressRoutes (Traefik)")
	}
	Resources = append(Resources, "Services")

	data := aboutCluster{
		Cluster1:   Cluster1,
		Cluster2:   Cluster2,
		Namespace1: Namespace1,
		Namespace2: Namespace2,
		Resources:  Resources,
	}
	err := renderPage(w, "templates/resources.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var clusterVersion1, clusterVersion2 interface{}
	clusterVersion1, err = k8s.ClusterVersion(Cluster1, Kubeconfig1, false)
	if err != nil {
		log.Println("Failed to get version for", Cluster1, err)
		return
	}

	clusterVersion2, err = k8s.ClusterVersion(Cluster2, Kubeconfig2, false)
	if err != nil {
		log.Println("Failed to get version for", Cluster2, err)
		return
	}
	ClusterVersion1 = clusterVersion1.(string)
	ClusterVersion2 = clusterVersion2.(string)
}

func CompareClusterHandler(w http.ResponseWriter, r *http.Request) {
	compar = r.FormValue("resource")

	if compar == "ClusterInfra" {
		var tableData []tableInfra
		tableData = make([]tableInfra, 0)
		nodeCount1, cpuTotal1, memTotal1, diskTotal1, podsTotal1, _ := k8s.GetNodesInfo(Cluster1, Kubeconfig1)
		nodeCount2, cpuTotal2, memTotal2, diskTotal2, podsTotal2, _ := k8s.GetNodesInfo(Cluster2, Kubeconfig2)
		nsCount1, nsCount2 := len(Namespaces1), len(Namespaces2)
		var apiNums1, apiNums2 int
		var apiresources1, apiresources2 *metav1.APIGroupList
		apiNums1, apiresources1, _ = k8s.GetAPIinfo(Cluster1, Kubeconfig1)
		apiNums2, apiresources2, _ = k8s.GetAPIinfo(Cluster2, Kubeconfig2)
		// проверяем формат значения возвращенного в версии кластера поскольку стоит {intarface}
		str1 := ClusterVersion1.(string)
		str2 := ClusterVersion2.(string)
		var traefikStatus1 string = "Not Installed"
		for _, group := range apiresources1.Groups {
			for _, version := range group.Versions {
				if version.GroupVersion == "traefik.containo.us/v1alpha1" {
					traefikStatus1 = "Installed"
					break
				}
			}
		}
		var traefikStatus2 string = "Not Installed"
		for _, group := range apiresources2.Groups {
			for _, version := range group.Versions {
				if version.GroupVersion == "traefik.containo.us/v1alpha1" {
					traefikStatus2 = "Installed"
					break
				}
			}
		}
		var canaryStatus1 string = "Not Installed"
		for _, group := range apiresources1.Groups {
			for _, version := range group.Versions {
				if version.GroupVersion == "flagger.app/v1beta1" {
					canaryStatus1 = "Installed"
					break
				}
			}
		}
		var canaryStatus2 string = "Not Installed"
		for _, group := range apiresources2.Groups {
			for _, version := range group.Versions {
				if version.GroupVersion == "flagger.app/v1beta1" {
					canaryStatus2 = "Installed"
					break
				}
			}
		}
		var jaegerStatus1 string = "Not Installed"
		for _, group := range apiresources1.Groups {
			for _, version := range group.Versions {
				if version.GroupVersion == "jaegertracing.io/v1" {
					jaegerStatus1 = "Installed"
					break
				}
			}
		}
		var jaegerStatus2 string = "Not Installed"
		for _, group := range apiresources2.Groups {
			for _, version := range group.Versions {
				if version.GroupVersion == "jaegertracing.io/v1" {
					jaegerStatus2 = "Installed"
					break
				}
			}
		}

		canaryNum1, ingNum1 := k8s.GetPerCluster(Cluster1, Kubeconfig1)
		canaryNum2, ingNum2 := k8s.GetPerCluster(Cluster2, Kubeconfig2)

		tableData = append(tableData, tableInfra{
			ClusterName: Cluster1,
			KubeVersion: str1,
			NodeCount:   nodeCount1,
			NsCount:     nsCount1,
			CpuTotal:    cpuTotal1,
			MemTotal:    memTotal1,
			DiskTotal:   diskTotal1,
			PodTotal:    podsTotal1,
			ApiNums:     apiNums1,
			Traefik:     traefikStatus1,
			TraefikNum:  ingNum1,
			Flagger:     canaryStatus1,
			FlaggerNum:  canaryNum1,
			Jaeger:      jaegerStatus1,
		})
		tableData = append(tableData, tableInfra{
			ClusterName: Cluster2,
			KubeVersion: str2,
			NodeCount:   nodeCount2,
			NsCount:     nsCount2,
			CpuTotal:    cpuTotal2,
			MemTotal:    memTotal2,
			DiskTotal:   diskTotal2,
			PodTotal:    podsTotal2,
			ApiNums:     apiNums2,
			Traefik:     traefikStatus2,
			TraefikNum:  ingNum2,
			Flagger:     canaryStatus2,
			FlaggerNum:  canaryNum2,
			Jaeger:      jaegerStatus2,
		})
		// Формируем страницу из шаблона для выбора неймспейса
		err := renderPage(w, "templates/compare_cluster.html", tableData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if compar == "Flagger (Canary)" {
		//var tableData []tableCanary
		type ClusterNamespaceCanaries struct {
			ClusterName string
			Namespace   string
			Canaries    []string
		}
		type Data struct {
			Clusters  []ClusterNamespaceCanaries
			Diffs     map[string][]string
			DiffSpecs map[string][]diff.CanarySpecDiff
		}

		diff1, diff2 := diff.GetDiff(k8s.GetCanaryPerNs(Cluster1, Kubeconfig1, Namespace1), k8s.GetCanaryPerNs(Cluster2, Kubeconfig2, Namespace2))

		data := Data{
			Clusters: []ClusterNamespaceCanaries{
				{
					ClusterName: Cluster1,
					Namespace:   Namespace1,
					Canaries:    k8s.GetCanaryPerNs(Cluster1, Kubeconfig1, Namespace1),
				},
				{
					ClusterName: Cluster2,
					Namespace:   Namespace2,
					Canaries:    k8s.GetCanaryPerNs(Cluster2, Kubeconfig2, Namespace2),
				},
			},
			Diffs: map[string][]string{
				Cluster1: diff1,
				Cluster2: diff2,
			},
			DiffSpecs: map[string][]diff.CanarySpecDiff{
				"Cluster1": diff.GetDiffCanarySpecs(Cluster1, Kubeconfig1, Cluster2, Kubeconfig2, Namespace1),
			},
		}

		// Формируем страницу из шаблона для выбора неймспейса если в указанных НС нет выбранного типа ресурса то выводи пустую страницу
		isEmpty := true
		for _, cluster := range data.Clusters {
			if len(cluster.Canaries) > 0 {
				isEmpty = false
				break
			}
		}
		if !isEmpty {
			err := renderCanaryPage(w, "templates/compare_canary.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			err := renderPage(w, "templates/blank.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else if compar == "Deployments" {
		//var tableData []tableCanary
		type ClusterNamespaceDeployments struct {
			ClusterName string
			Namespace   string
			Deployments []string
		}
		type Data struct {
			Clusters  []ClusterNamespaceDeployments
			Diffs     map[string][]string
			DiffSpecs map[string][]diff.DeploySpecDiff
		}

		diff1, diff2 := diff.GetDiff(k8s.GetDeployPerNs(Cluster1, Kubeconfig1, Namespace1), k8s.GetDeployPerNs(Cluster2, Kubeconfig2, Namespace2))

		data := Data{
			Clusters: []ClusterNamespaceDeployments{
				{
					ClusterName: Cluster1,
					Namespace:   Namespace1,
					Deployments: k8s.GetDeployPerNs(Cluster1, Kubeconfig1, Namespace1),
				},
				{
					ClusterName: Cluster2,
					Namespace:   Namespace2,
					Deployments: k8s.GetDeployPerNs(Cluster2, Kubeconfig2, Namespace2),
				},
			},
			Diffs: map[string][]string{
				Cluster1: diff1,
				Cluster2: diff2,
			},
			DiffSpecs: map[string][]diff.DeploySpecDiff{
				"Cluster1": diff.GetDiffDeploymentsSpecs(Cluster1, Kubeconfig1, Cluster2, Kubeconfig2, Namespace1),
			},
		}

		// Формируем страницу из шаблона для выбора неймспейса если в указанных НС нет выбранного типа ресурса то выводи пустую страницу
		isEmpty := true
		for _, cluster := range data.Clusters {
			if len(cluster.Deployments) > 0 {
				isEmpty = false
				break
			}
		}
		if !isEmpty {
			err := renderCanaryPage(w, "templates/compare_deployments.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			err := renderPage(w, "templates/blank.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	} else if compar == "Daemonsets" {
		//var tableData []tableCanary
		type ClusterNamespaceDmnsets struct {
			ClusterName string
			Namespace   string
			DmnSets     []string
		}
		type Data struct {
			Clusters  []ClusterNamespaceDmnsets
			Diffs     map[string][]string
			DiffSpecs map[string][]diff.DmnSetsSpecDiff
		}

		diff1, diff2 := diff.GetDiff(k8s.GetUniversalObjectPerNsAsString(Cluster1, Kubeconfig1, Namespace1, "apps", "v1", "daemonsets"), k8s.GetUniversalObjectPerNsAsString(Cluster2, Kubeconfig2, Namespace2, "apps", "v1", "daemonsets"))

		data := Data{
			Clusters: []ClusterNamespaceDmnsets{
				{
					ClusterName: Cluster1,
					Namespace:   Namespace1,
					DmnSets:     k8s.GetUniversalObjectPerNsAsString(Cluster1, Kubeconfig1, Namespace1, "apps", "v1", "daemonsets"),
				},
				{
					ClusterName: Cluster2,
					Namespace:   Namespace2,
					DmnSets:     k8s.GetUniversalObjectPerNsAsString(Cluster2, Kubeconfig2, Namespace2, "apps", "v1", "daemonsets"),
				},
			},
			Diffs: map[string][]string{
				Cluster1: diff1,
				Cluster2: diff2,
			},
			DiffSpecs: map[string][]diff.DmnSetsSpecDiff{
				"Cluster1": diff.GetDiffDmnSetsSpecs(Cluster1, Kubeconfig1, Cluster2, Kubeconfig2, Namespace1),
			},
		}

		// Формируем страницу из шаблона для выбора неймспейса если в указанных НС нет выбранного типа ресурса то выводи пустую страницу
		isEmpty := true
		for _, cluster := range data.Clusters {
			if len(cluster.DmnSets) > 0 {
				isEmpty = false
				break
			}
		}
		if !isEmpty {
			err := renderCanaryPage(w, "templates/compare_daemonset.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			err := renderPage(w, "templates/blank.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	} else if compar == "Services" {

		type ClusterNamespaceServices struct {
			ClusterName string
			Namespace   string
			Services    []string
		}
		type Data struct {
			Clusters  []ClusterNamespaceServices
			Diffs     map[string][]string
			DiffSpecs map[string][]diff.ServicesSpecDiff
		}

		diff1, diff2 := diff.GetDiff(k8s.GetUniversalObjectPerNsAsString(Cluster1, Kubeconfig1, Namespace1, "", "v1", "services"), k8s.GetUniversalObjectPerNsAsString(Cluster2, Kubeconfig2, Namespace2, "", "v1", "services"))

		data := Data{
			Clusters: []ClusterNamespaceServices{
				{
					ClusterName: Cluster1,
					Namespace:   Namespace1,
					Services:    k8s.GetUniversalObjectPerNsAsString(Cluster1, Kubeconfig1, Namespace1, "", "v1", "services"),
				},
				{
					ClusterName: Cluster2,
					Namespace:   Namespace2,
					Services:    k8s.GetUniversalObjectPerNsAsString(Cluster2, Kubeconfig2, Namespace2, "", "v1", "services"),
				},
			},
			Diffs: map[string][]string{
				Cluster1: diff1,
				Cluster2: diff2,
			},
			DiffSpecs: map[string][]diff.ServicesSpecDiff{
				"Cluster1": diff.GetDiffServicesSpecs(Cluster1, Kubeconfig1, Cluster2, Kubeconfig2, Namespace1),
			},
		}

		// Формируем страницу из шаблона для выбора неймспейса если в указанных НС нет выбранного типа ресурса то выводи пустую страницу
		isEmpty := true
		for _, cluster := range data.Clusters {
			if len(cluster.Services) > 0 {
				isEmpty = false
				break
			}
		}
		if !isEmpty {
			err := renderCanaryPage(w, "templates/compare_services.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			err := renderPage(w, "templates/blank.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	} else if compar == "HelmValues" {

		helmReleases1, _ := helm.GetHelmReleasesPerNS(Cluster1, Kubeconfig1, Namespace1)
		helmReleases2, _ := helm.GetHelmReleasesPerNS(Cluster2, Kubeconfig2, Namespace2)

		//var tableData
		type ClusterNamespaceHelmReleases struct {
			ClusterName  string
			Namespace    string
			HelmReleases []string
		}
		type Data struct {
			Clusters  []ClusterNamespaceHelmReleases
			Diffs     map[string][]string
			DiffSpecs map[string][]diff.HelmValuesDiff
		}

		diff1, diff2 := diff.GetDiff(helmReleases1, helmReleases2)

		data := Data{
			Clusters: []ClusterNamespaceHelmReleases{
				{
					ClusterName:  Cluster1,
					Namespace:    Namespace1,
					HelmReleases: helmReleases1,
				},
				{
					ClusterName:  Cluster2,
					Namespace:    Namespace2,
					HelmReleases: helmReleases2,
				},
			},
			Diffs: map[string][]string{
				Cluster1: diff1,
				Cluster2: diff2,
			},
			DiffSpecs: map[string][]diff.HelmValuesDiff{
				"Cluster1": diff.GetDiffHelmTemplates(Cluster1, Kubeconfig1, Cluster2, Kubeconfig2, Namespace1),
			},
		}

		// Формируем страницу из шаблона для выбора неймспейса если в указанных НС нет выбранного типа ресурса то выводи пустую страницу
		isEmpty := true
		for _, cluster := range data.Clusters {
			if len(cluster.HelmReleases) > 0 {
				isEmpty = false
				break
			}
		}
		if !isEmpty {

			err := renderCanaryPage(w, "templates/compare_values.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			err := renderPage(w, "templates/blank.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else if compar == "IngressRoutes (Traefik)" {

		type ClusterNamespaceTingress struct {
			ClusterName string
			Namespace   string
			TraefikIng  []string
		}
		type Data struct {
			Clusters  []ClusterNamespaceTingress
			Diffs     map[string][]string
			DiffSpecs map[string][]diff.TingressSpecDiff
		}

		diff1, diff2 := diff.GetDiff(k8s.GetUniversalObjectPerNsAsString(Cluster1, Kubeconfig1, Namespace1, "traefik.containo.us", "v1alpha1", "ingressroutes"), k8s.GetUniversalObjectPerNsAsString(Cluster2, Kubeconfig2, Namespace2, "traefik.containo.us", "v1alpha1", "ingressroutes"))

		data := Data{
			Clusters: []ClusterNamespaceTingress{
				{
					ClusterName: Cluster1,
					Namespace:   Namespace1,
					TraefikIng:  k8s.GetUniversalObjectPerNsAsString(Cluster1, Kubeconfig1, Namespace1, "traefik.containo.us", "v1alpha1", "ingressroutes"),
				},
				{
					ClusterName: Cluster2,
					Namespace:   Namespace2,
					TraefikIng:  k8s.GetUniversalObjectPerNsAsString(Cluster2, Kubeconfig2, Namespace2, "traefik.containo.us", "v1alpha1", "ingressroutes"),
				},
			},
			Diffs: map[string][]string{
				Cluster1: diff1,
				Cluster2: diff2,
			},
			DiffSpecs: map[string][]diff.TingressSpecDiff{
				"Cluster1": diff.GetDiffTingressSpecs(Cluster1, Kubeconfig1, Cluster2, Kubeconfig2, Namespace1),
			},
		}
		isEmpty := true
		for _, cluster := range data.Clusters {
			if len(cluster.TraefikIng) > 0 {
				isEmpty = false
				break
			}
		}
		if !isEmpty {
			err := renderCanaryPage(w, "templates/compare_tingress.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			err := renderPage(w, "templates/blank.html", data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	}
}

func DisplayCanaryJSONHandler(w http.ResponseWriter, r *http.Request) {
	canaryObjects1 := k8s.GetCanaryObjectsPerNs(Cluster1, Kubeconfig1, Namespace1)
	canaryObjects2 := k8s.GetCanaryObjectsPerNs(Cluster2, Kubeconfig2, Namespace2)
	clusterCanaries := make(map[string]map[string]map[string]interface{})

	for _, canary1 := range canaryObjects1 {
		// Получаем спецификацию для Canary
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(canary1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		for _, canary2 := range canaryObjects2 {
			// Если имена Canary совпадают
			if canary1.GetName() == canary2.GetName() {
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(canary2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				clusterCanaries[canary1.GetName()] = map[string]map[string]interface{}{
					Cluster1: spec1.(map[string]interface{}),
					Cluster2: spec2.(map[string]interface{}),
				}
			}
		}
	}

	// Загрузить шаблон страницы
	tmpl, err := template.New("canary_json.html").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "    ")
			return string(a)
		},
	}).ParseFiles("templates/canary_json.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Выполнить шаблон с данными canaryJSON
	err = tmpl.Execute(w, clusterCanaries)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DisplayDeployJSONHandler(w http.ResponseWriter, r *http.Request) {
	group := "apps"
	version := "v1"
	resource := "deployments"
	deployObjects1 := k8s.GetUniversalObjectsPerNsUnstruct(Cluster1, Kubeconfig1, Namespace1, group, version, resource)
	deployObjects2 := k8s.GetUniversalObjectsPerNsUnstruct(Cluster2, Kubeconfig2, Namespace2, group, version, resource)
	clusterDeployments := make(map[string]map[string]map[string]interface{})

	for _, deploy1 := range deployObjects1 {
		// Получаем спецификацию для Deployment
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(deploy1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		for _, deploy2 := range deployObjects2 {
			if deploy1.GetName() == deploy2.GetName() {
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(deploy2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				clusterDeployments[deploy1.GetName()] = map[string]map[string]interface{}{
					Cluster1: spec1.(map[string]interface{}),
					Cluster2: spec2.(map[string]interface{}),
				}
			}
		}
	}

	// Загрузить шаблон страницы
	tmpl, err := template.New("deployments_json.html").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "    ")
			return string(a)
		},
	}).ParseFiles("templates/deployments_json.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Выполнить шаблон с данными deployJSON
	err = tmpl.Execute(w, clusterDeployments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DisplayDmnSetJSONHandler(w http.ResponseWriter, r *http.Request) {
	dmnSetObjects1 := k8s.GetUniversalObjectsPerNsUnstruct(Cluster1, Kubeconfig1, Namespace1, "apps", "v1", "daemonsets")
	dmnSetObjects2 := k8s.GetUniversalObjectsPerNsUnstruct(Cluster2, Kubeconfig2, Namespace2, "apps", "v1", "daemonsets")
	clusterDmnsets := make(map[string]map[string]map[string]interface{})

	for _, dmnset1 := range dmnSetObjects1 {
		// Получаем спецификацию для Deployment
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(dmnset1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		for _, dmnset2 := range dmnSetObjects2 {
			if dmnset1.GetName() == dmnset2.GetName() {
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(dmnset2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				clusterDmnsets[dmnset1.GetName()] = map[string]map[string]interface{}{
					Cluster1: spec1.(map[string]interface{}),
					Cluster2: spec2.(map[string]interface{}),
				}
			}
		}
	}

	// Загрузить шаблон страницы
	tmpl, err := template.New("dmnsets_json.html").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "    ")
			return string(a)
		},
	}).ParseFiles("templates/dmnsets_json.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Выполнить шаблон с данными deployJSON
	err = tmpl.Execute(w, clusterDmnsets)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DisplaySvsJSONHandler(w http.ResponseWriter, r *http.Request) {
	svcObjects1 := k8s.GetUniversalObjectsPerNsUnstruct(Cluster1, Kubeconfig1, Namespace1, "", "v1", "services")
	svcObjects2 := k8s.GetUniversalObjectsPerNsUnstruct(Cluster2, Kubeconfig2, Namespace2, "", "v1", "services")
	clusterSvcs := make(map[string]map[string]map[string]interface{})

	for _, svcs1 := range svcObjects1 {
		// Получаем спецификацию для Deployment
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(svcs1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		for _, svcs2 := range svcObjects2 {
			if svcs1.GetName() == svcs2.GetName() {
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(svcs2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				clusterSvcs[svcs1.GetName()] = map[string]map[string]interface{}{
					Cluster1: spec1.(map[string]interface{}),
					Cluster2: spec2.(map[string]interface{}),
				}
			}
		}
	}

	// Загрузить шаблон страницы
	tmpl, err := template.New("services_json.html").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "    ")
			return string(a)
		},
	}).ParseFiles("templates/services_json.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Выполнить шаблон с данными deployJSON
	err = tmpl.Execute(w, clusterSvcs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DisplayHelmJSONHandler(w http.ResponseWriter, r *http.Request) {
	helmSpec1, _ := helm.GetHelmReleasesJsonPerNS(Cluster1, Kubeconfig1, Namespace1)
	helmSpec2, _ := helm.GetHelmReleasesJsonPerNS(Cluster2, Kubeconfig2, Namespace2)
	clusterReleases := make(map[string]map[string]map[string]interface{})

	for _, values1 := range helmSpec1 {
		// Получаем спецификацию для Deployment
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(values1.Object)
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		for _, values2 := range helmSpec2 {
			if values1.Object["releaseName"] == values2.Object["releaseName"] {
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(values2.Object)
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				name := values1.Object["releaseName"].(string)
				clusterReleases[name] = map[string]map[string]interface{}{
					Cluster1: spec1.(map[string]interface{}),
					Cluster2: spec2.(map[string]interface{}),
				}
			}
		}
	}

	// Загрузить шаблон страницы
	tmpl, err := template.New("helm_json.html").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "    ")
			return string(a)
		},
	}).ParseFiles("templates/helm_json.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Выполнить шаблон с данными deployJSON
	err = tmpl.Execute(w, clusterReleases)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func CompareClusterCMTHandler(w http.ResponseWriter, r *http.Request) {
	type ClusterNamespaceMT struct {
		ClusterName string
		Namespace   string
		MetricTpl   []string
	}
	type Data struct {
		Clusters  []ClusterNamespaceMT
		Diffs     map[string][]string
		DiffSpecs map[string][]diff.MTSpecDiff
	}
	diff1, diff2 := diff.GetDiff(k8s.GetMTPerNs(Cluster1, Kubeconfig1, Namespace1), k8s.GetMTPerNs(Cluster2, Kubeconfig2, Namespace2))
	data := Data{
		Clusters: []ClusterNamespaceMT{
			{
				ClusterName: Cluster1,
				Namespace:   Namespace1,
				MetricTpl:   k8s.GetMTPerNs(Cluster1, Kubeconfig1, Namespace1),
			},
			{
				ClusterName: Cluster2,
				Namespace:   Namespace2,
				MetricTpl:   k8s.GetMTPerNs(Cluster2, Kubeconfig2, Namespace2),
			},
		},
		Diffs: map[string][]string{
			Cluster1: diff1,
			Cluster2: diff2,
		},
		DiffSpecs: map[string][]diff.MTSpecDiff{
			"Cluster1": diff.GetDiffMetricTemplatesSpecs(Cluster1, Kubeconfig1, Cluster2, Kubeconfig2, Namespace1),
		},
	}

	err := renderCanaryPage(w, "templates/canary_mt.html", data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func DisplayTingJSONHandler(w http.ResponseWriter, r *http.Request) {
	ingObjects1 := k8s.GetUniversalObjectsPerNsUnstruct(Cluster1, Kubeconfig1, Namespace1, "traefik.containo.us", "v1alpha1", "ingressroutes")
	ingObjects2 := k8s.GetUniversalObjectsPerNsUnstruct(Cluster2, Kubeconfig2, Namespace2, "traefik.containo.us", "v1alpha1", "ingressroutes")
	clusterIngs := make(map[string]map[string]map[string]interface{})

	for _, item1 := range ingObjects1 {
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(item1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		for _, item2 := range ingObjects2 {
			if item1.GetName() == item2.GetName() {
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(item2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				clusterIngs[item1.GetName()] = map[string]map[string]interface{}{
					Cluster1: spec1.(map[string]interface{}),
					Cluster2: spec2.(map[string]interface{}),
				}
			}
		}
	}

	// Загрузить шаблон страницы
	tmpl, err := template.New("tingress_json.html").Funcs(template.FuncMap{
		"toJSON": func(v interface{}) string {
			a, _ := json.MarshalIndent(v, "", "    ")
			return string(a)
		},
	}).ParseFiles("templates/tingress_json.html")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Выполнить шаблон с данными deployJSON
	err = tmpl.Execute(w, clusterIngs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
