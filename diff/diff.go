package diff

import (
	"compareapp/helm"
	"compareapp/k8s"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func GetDiff(list1, list2 []string) ([]string, []string) {
	m1 := make(map[string]bool)
	m2 := make(map[string]bool)

	for _, item := range list1 {
		m1[item] = true
	}

	for _, item := range list2 {
		m2[item] = true
	}

	diff1 := []string{} // Элементы из list1, которых нет в list2
	for _, item := range list1 {
		if _, ok := m2[item]; !ok {
			diff1 = append(diff1, item)
		}
	}

	diff2 := []string{} // Элементы из list2, которых нет в list1
	for _, item := range list2 {
		if _, ok := m1[item]; !ok {
			diff2 = append(diff2, item)
		}
	}

	return diff1, diff2
}

type CanarySpecDiff struct {
	CanaryName   string
	SpecCluster1 interface{}
	SpecCluster2 interface{}
	Difference   string
	Cluster1     string
	Cluster2     string
}

type MTSpecDiff struct {
	MTName       string
	SpecCluster1 interface{}
	SpecCluster2 interface{}
	Difference   string
	Cluster1     string
	Cluster2     string
}

type DeploySpecDiff struct {
	DeployName   string
	SpecCluster1 interface{}
	SpecCluster2 interface{}
	Difference   string
	Cluster1     string
	Cluster2     string
}

type DmnSetsSpecDiff struct {
	DmnSetName   string
	SpecCluster1 interface{}
	SpecCluster2 interface{}
	Difference   string
	Cluster1     string
	Cluster2     string
}

type ServicesSpecDiff struct {
	ServiceName  string
	SpecCluster1 interface{}
	SpecCluster2 interface{}
	Difference   string
	Cluster1     string
	Cluster2     string
}

type TingressSpecDiff struct {
	IngName      string
	SpecCluster1 interface{}
	SpecCluster2 interface{}
	Difference   string
	Cluster1     string
	Cluster2     string
}
type HelmValuesDiff struct {
	ReleaseName    string
	ValuesCluster1 interface{}
	ValuesCluster2 interface{}
	Difference     string
	Cluster1       string
	Cluster2       string
}

func DiffCanarySpecs(spec1, spec2 interface{}) map[string]interface{} {
	diff := make(map[string]interface{})

	map1, ok1 := spec1.(map[string]interface{})
	map2, ok2 := spec2.(map[string]interface{})

	if ok1 && ok2 {
		for k, v1 := range map1 {
			if v2, ok := map2[k]; ok {
				if !reflect.DeepEqual(v1, v2) {
					diff[k] = DiffCanarySpecs(v1, v2)
				}
			} else {
				diff[k] = v1
			}
		}
		for k, v2 := range map2 {
			if _, ok := map1[k]; !ok {
				diff[k] = v2
			}
		}
	} else if !reflect.DeepEqual(spec1, spec2) {
		diff["spec1"] = spec1
		diff["spec2"] = spec2
	}

	return diff
}

func GetDiffCanarySpecs(cluster1, configPath1, cluster2, configPath2, namespace string) []CanarySpecDiff {
	// Получаем Canary-объекты из двух кластеров
	canariesCluster1 := k8s.GetCanaryObjectsPerNs(cluster1, configPath1, namespace)
	canariesCluster2 := k8s.GetCanaryObjectsPerNs(cluster2, configPath2, namespace)

	// Слайс для хранения объектов с различиями
	diffSpecs := []CanarySpecDiff{}

	// Проверяем каждую Canary в первом кластере
	for _, canary1 := range canariesCluster1 {
		// Получаем спецификацию для Canary
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(canary1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}

		// Ищем соответствующую Canary во втором кластере
		for _, canary2 := range canariesCluster2 {
			// Если имена Canary совпадают
			if canary1.GetName() == canary2.GetName() {
				// Получаем спецификацию для Canary
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(canary2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				// Сравниваем спецификации
				if !reflect.DeepEqual(spec1, spec2) {
					// Если спецификации различны, добавляем их в слайс
					diffMap := DiffCanarySpecs(spec1.(map[string]interface{}), spec2.(map[string]interface{})) // вызываем функцию Diff
					diffBytes, err := json.Marshal(diffMap)
					if err != nil {
						log.Printf("Failed to marshal difference map: %v", err)
						continue
					}

					diff := string(diffBytes)

					diffSpecs = append(diffSpecs, CanarySpecDiff{
						CanaryName:   canary1.GetName(),
						SpecCluster1: spec1,
						SpecCluster2: spec2,
						Difference:   diff,
						Cluster1:     cluster1,
						Cluster2:     cluster2,
					})
				}
				break
			}
		}
	}
	return diffSpecs
}

func GetDiffMetricTemplatesSpecs(cluster1, configPath1, cluster2, configPath2, namespace string) []MTSpecDiff {

	// Получаем metrictemplates-объекты из двух кластеров
	cmpCluster1 := k8s.GetCanaryMetricTemplateObjectsPerNs(cluster1, configPath1, namespace)
	cmpCluster2 := k8s.GetCanaryMetricTemplateObjectsPerNs(cluster2, configPath2, namespace)

	// Слайс для хранения объектов с различиями
	diffSpecs := []MTSpecDiff{}

	// Проверяем каждую mt в первом кластере
	for _, tmpl1 := range cmpCluster1 {
		// Получаем спецификацию для Canary
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(tmpl1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}

		// Ищем соответствующую Canary во втором кластере
		for _, tmpl2 := range cmpCluster2 {
			// Если имена Canary совпадают
			if tmpl1.GetName() == tmpl2.GetName() {
				// Получаем спецификацию для Canary
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(tmpl2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				// Сравниваем спецификации
				if !reflect.DeepEqual(spec1, spec2) {
					// Если спецификации различны, добавляем их в слайс
					diffMap := DiffCanarySpecs(spec1.(map[string]interface{}), spec2.(map[string]interface{})) // вызываем функцию Diff
					diffBytes, err := json.Marshal(diffMap)
					if err != nil {
						log.Printf("Failed to marshal difference map: %v", err)
						continue
					}

					diff := string(diffBytes)

					diffSpecs = append(diffSpecs, MTSpecDiff{
						MTName:       tmpl1.GetName(),
						SpecCluster1: spec1,
						SpecCluster2: spec2,
						Difference:   diff,
						Cluster1:     cluster1,
						Cluster2:     cluster2,
					})
				}
				break
			}
		}
	}
	return diffSpecs
}

func GetDiffDeploymentsSpecs(cluster1, configPath1, cluster2, configPath2, namespace string) []DeploySpecDiff {
	// Получаем Deployments-объекты из двух кластеров
	group := "apps"
	version := "v1"
	resource := "deployments"
	deployCluster1 := k8s.GetUniversalObjectsPerNsUnstruct(cluster1, configPath1, namespace, group, version, resource)
	deployCluster2 := k8s.GetUniversalObjectsPerNsUnstruct(cluster2, configPath2, namespace, group, version, resource)

	// Слайс для хранения объектов с различиями
	diffSpecs := []DeploySpecDiff{}

	// Проверяем каждую Canary в первом кластере
	for _, deploy1 := range deployCluster1 {
		// Получаем спецификацию для Deployments
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(deploy1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		// Удаляем "template.metadata.annotations" из spec1
		if specMap1, ok := spec1.(map[string]interface{}); ok {
			if template, ok := specMap1["template"].(map[string]interface{}); ok {
				if metadata, ok := template["metadata"].(map[string]interface{}); ok {
					delete(metadata, "annotations")
				}
			}
		}

		// Ищем соответствующую Deployments во втором кластере
		for _, deploy2 := range deployCluster2 {
			// Если имена Canary совпадают
			if deploy1.GetName() == deploy2.GetName() {
				// Получаем спецификацию для Deployments
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(deploy2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				// Удаляем "template.metadata.annotations" из spec2
				if specMap2, ok := spec2.(map[string]interface{}); ok {
					if template, ok := specMap2["template"].(map[string]interface{}); ok {
						if metadata, ok := template["metadata"].(map[string]interface{}); ok {
							delete(metadata, "annotations")
						}
					}
				}
				// Сравниваем спецификации
				if !reflect.DeepEqual(spec1, spec2) {
					// Если спецификации различны, добавляем их в слайс
					diffMap := DiffCanarySpecs(spec1.(map[string]interface{}), spec2.(map[string]interface{})) // вызываем функцию Diff
					diffBytes, err := json.Marshal(diffMap)
					if err != nil {
						log.Printf("Failed to marshal difference map: %v", err)
						continue
					}

					diff := string(diffBytes)

					diffSpecs = append(diffSpecs, DeploySpecDiff{
						DeployName:   deploy1.GetName(),
						SpecCluster1: spec1,
						SpecCluster2: spec2,
						Difference:   diff,
						Cluster1:     cluster1,
						Cluster2:     cluster2,
					})
				}
				break
			}
		}
	}
	return diffSpecs
}

func GetDiffDmnSetsSpecs(cluster1, configPath1, cluster2, configPath2, namespace string) []DmnSetsSpecDiff {
	// Получаем Dmnsets-объекты из двух кластеров
	dmnsetCluster1 := k8s.GetUniversalObjectsPerNsUnstruct(cluster1, configPath1, namespace, "apps", "v1", "daemonsets")
	dmnsetCluster2 := k8s.GetUniversalObjectsPerNsUnstruct(cluster2, configPath2, namespace, "apps", "v1", "daemonsets")

	// Слайс для хранения объектов с различиями
	diffSpecs := []DmnSetsSpecDiff{}

	// Проверяем каждую Dmnsets в первом кластере
	for _, dmnsets1 := range dmnsetCluster1 {
		// Получаем спецификацию для Dmnsets
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(dmnsets1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		// Удаляем "template.metadata.annotations" из spec1
		if specMap1, ok := spec1.(map[string]interface{}); ok {
			if template, ok := specMap1["template"].(map[string]interface{}); ok {
				if metadata, ok := template["metadata"].(map[string]interface{}); ok {
					delete(metadata, "annotations")
				}
			}
		}

		// Ищем соответствующую Dmnsets во втором кластере
		for _, dmnsets2 := range dmnsetCluster2 {
			// Если имена Dmnsets совпадают
			if dmnsets1.GetName() == dmnsets2.GetName() {
				// Получаем спецификацию для Dmnsets
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(dmnsets2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				// Удаляем "template.metadata.annotations" из spec2
				if specMap2, ok := spec2.(map[string]interface{}); ok {
					if template, ok := specMap2["template"].(map[string]interface{}); ok {
						if metadata, ok := template["metadata"].(map[string]interface{}); ok {
							delete(metadata, "annotations")
						}
					}
				}
				// Сравниваем спецификации
				if !reflect.DeepEqual(spec1, spec2) {
					// Если спецификации различны, добавляем их в слайс
					diffMap := DiffCanarySpecs(spec1.(map[string]interface{}), spec2.(map[string]interface{})) // вызываем функцию Diff
					diffBytes, err := json.Marshal(diffMap)
					if err != nil {
						log.Printf("Failed to marshal difference map: %v", err)
						continue
					}

					diff := string(diffBytes)

					diffSpecs = append(diffSpecs, DmnSetsSpecDiff{
						DmnSetName:   dmnsets1.GetName(),
						SpecCluster1: spec1,
						SpecCluster2: spec2,
						Difference:   diff,
						Cluster1:     cluster1,
						Cluster2:     cluster2,
					})
				}
				break
			}
		}
	}
	return diffSpecs
}

func GetDiffServicesSpecs(cluster1, configPath1, cluster2, configPath2, namespace string) []ServicesSpecDiff {
	// Получаем Dmnsets-объекты из двух кластеров
	servicesCluster1 := k8s.GetUniversalObjectsPerNsUnstruct(cluster1, configPath1, namespace, "", "v1", "services")
	servicesCluster2 := k8s.GetUniversalObjectsPerNsUnstruct(cluster2, configPath2, namespace, "", "v1", "services")

	// Слайс для хранения объектов с различиями
	diffSpecs := []ServicesSpecDiff{}

	// Проверяем каждую Dmnsets в первом кластере
	for _, svcs1 := range servicesCluster1 {
		// Получаем спецификацию для Dmnsets
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(svcs1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		// Удаляем "clusterIP" из spec1
		if specMap1, ok := spec1.(map[string]interface{}); ok {
			delete(specMap1, "clusterIP")
			delete(specMap1, "clusterIPs")
			delete(specMap1, "healthCheckNodePort")
			delete(specMap1, "loadBalancerIP")
			if ports, ok := specMap1["ports"].([]interface{}); ok {
				for i, port := range ports {
					if portMap, ok := port.(map[string]interface{}); ok {

						delete(portMap, "nodePort")
						// Обновляем порт в срезе после удаления nodePort
						ports[i] = portMap

					}
				}
			}
		}

		// Ищем соответствующую Dmnsets во втором кластере
		for _, svcs2 := range servicesCluster2 {
			// Если имена Dmnsets совпадают
			if svcs1.GetName() == svcs2.GetName() {
				// Получаем спецификацию для Dmnsets
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(svcs2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				// Удаляем "clusterIP" из spec2
				if specMap2, ok := spec2.(map[string]interface{}); ok {
					delete(specMap2, "clusterIP")
					delete(specMap2, "clusterIPs")
					delete(specMap2, "healthCheckNodePort")
					delete(specMap2, "loadBalancerIP")
					if ports, ok := specMap2["ports"].([]interface{}); ok {
						for i, port := range ports {
							if portMap, ok := port.(map[string]interface{}); ok {

								delete(portMap, "nodePort")
								// Обновляем порт в срезе после удаления nodePort
								ports[i] = portMap

							}
						}
					}
				}
				// Сравниваем спецификации
				if !reflect.DeepEqual(spec1, spec2) {
					// Если спецификации различны, добавляем их в слайс
					diffMap := DiffCanarySpecs(spec1.(map[string]interface{}), spec2.(map[string]interface{})) // вызываем функцию Diff
					diffBytes, err := json.Marshal(diffMap)
					if err != nil {
						log.Printf("Failed to marshal difference map: %v", err)
						continue
					}

					diff := string(diffBytes)

					diffSpecs = append(diffSpecs, ServicesSpecDiff{
						ServiceName:  svcs1.GetName(),
						SpecCluster1: spec1,
						SpecCluster2: spec2,
						Difference:   diff,
						Cluster1:     cluster1,
						Cluster2:     cluster2,
					})
				}
				break
			}
		}
	}
	return diffSpecs
}

func GetDiffHelmTemplates(cluster1, configPath1, cluster2, configPath2, namespace string) []HelmValuesDiff {
	helmSpec1, _ := helm.GetHelmReleasesJsonPerNS(cluster1, configPath1, namespace)
	helmSpec2, _ := helm.GetHelmReleasesJsonPerNS(cluster2, configPath2, namespace)

	diffSpecs := []HelmValuesDiff{}

	for _, helmValues1 := range helmSpec1 {
		values1 := helmValues1.Object

		// Adapt the "image" field
		if imageStr, ok := values1["image"].(string); ok {
			imageParts := strings.Split(imageStr, "/")
			values1["image"] = imageParts[len(imageParts)-1]
		}

		for _, helmValues2 := range helmSpec2 {
			if helmValues1.Object["releaseName"] == helmValues2.Object["releaseName"] {
				values2 := helmValues2.Object

				// Adapt the "image" field
				if imageStr, ok := values2["image"].(string); ok {
					imageParts := strings.Split(imageStr, "/")
					values2["image"] = imageParts[len(imageParts)-1]
				}

				// Compare the values
				if !reflect.DeepEqual(values1, values2) {
					diffMap := DiffCanarySpecs(values1, values2) // call the Diff function
					diffBytes, err := json.Marshal(diffMap)
					if err != nil {
						log.Printf("Failed to marshal difference map: %v", err)
						continue
					}

					diff := string(diffBytes)
					name, ok := helmValues1.Object["releaseName"].(string)
					if !ok {
						// Обработать ситуацию, когда fullnameOverride отсутствует или не является строкой
						fmt.Print("Не найдено имя хельм релиза")
						continue
					}
					//name := helmValues1.Object["fullnameOverride"].(string)
					diffSpecs = append(diffSpecs, HelmValuesDiff{
						ReleaseName:    name,
						ValuesCluster1: values1,
						ValuesCluster2: values2,
						Difference:     diff,
						Cluster1:       cluster1,
						Cluster2:       cluster2,
					})
				}
				break
			}
		}
	}
	return diffSpecs
}

func GetDiffTingressSpecs(cluster1, configPath1, cluster2, configPath2, namespace string) []TingressSpecDiff {
	// Получаем Dmnsets-объекты из двух кластеров
	ingressCluster1 := k8s.GetUniversalObjectsPerNsUnstruct(cluster1, configPath1, namespace, "traefik.containo.us", "v1alpha1", "ingressroutes")
	ingressCluster2 := k8s.GetUniversalObjectsPerNsUnstruct(cluster2, configPath2, namespace, "traefik.containo.us", "v1alpha1", "ingressroutes")

	// Слайс для хранения объектов с различиями
	diffSpecs := []TingressSpecDiff{}

	// Проверяем каждую Dmnsets в первом кластере
	for _, items1 := range ingressCluster1 {
		// Получаем спецификацию для Dmnsets
		spec1, found1, err1 := unstructured.NestedFieldNoCopy(items1.Object, "spec")
		if err1 != nil || !found1 {
			// Обрабатываем ошибку или случай, когда спецификация не найдена
			continue
		}
		if specMap1, ok := spec1.(map[string]interface{}); ok {
			if routes, ok := specMap1["routes"].([]interface{}); ok {
				for i, dnsname := range routes {
					if routesMap, ok := dnsname.(map[string]interface{}); ok {

						delete(routesMap, "match")
						// Обновляем порт в срезе после удаления nodePort
						routes[i] = routesMap

					}
				}
			}
		}

		// Ищем соответствующую Dmnsets во втором кластере
		for _, items2 := range ingressCluster2 {
			// Если имена Dmnsets совпадают
			if items1.GetName() == items2.GetName() {
				// Получаем спецификацию для Dmnsets
				spec2, found2, err2 := unstructured.NestedFieldNoCopy(items2.Object, "spec")
				if err2 != nil || !found2 {
					// Обрабатываем ошибку или случай, когда спецификация не найдена
					continue
				}
				if specMap2, ok := spec2.(map[string]interface{}); ok {
					if routes, ok := specMap2["routes"].([]interface{}); ok {
						for i, dnsname := range routes {
							if routesMap, ok := dnsname.(map[string]interface{}); ok {

								delete(routesMap, "match")
								// Обновляем порт в срезе после удаления nodePort
								routes[i] = routesMap

							}
						}
					}
				}
				// Сравниваем спецификации
				if !reflect.DeepEqual(spec1, spec2) {
					// Если спецификации различны, добавляем их в слайс
					diffMap := DiffCanarySpecs(spec1.(map[string]interface{}), spec2.(map[string]interface{})) // вызываем функцию Diff
					diffBytes, err := json.Marshal(diffMap)
					if err != nil {
						log.Printf("Failed to marshal difference map: %v", err)
						continue
					}

					diff := string(diffBytes)

					diffSpecs = append(diffSpecs, TingressSpecDiff{
						IngName:      items1.GetName(),
						SpecCluster1: spec1,
						SpecCluster2: spec2,
						Difference:   diff,
						Cluster1:     cluster1,
						Cluster2:     cluster2,
					})
				}
				break
			}
		}
	}
	return diffSpecs
}
