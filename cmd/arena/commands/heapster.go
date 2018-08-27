package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/bitly/go-simplejson"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	DefaultHeapsterScheme      = "http"
	DefaultHeapsterServiceName = "heapster"
	DefaultHeapsterPort        = "80"
	HeapsterApiPath            = "/api/v1/model"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

func getHeapsterEndpointUrl(client *kubernetes.Clientset) (string, error) {
	heapsterService, err := client.CoreV1().Services(metav1.NamespaceSystem).Get(DefaultHeapsterServiceName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	heapsterEndpointUrl := DefaultHeapsterScheme + "://" + heapsterService.Spec.ClusterIP + ":" + DefaultHeapsterPort + HeapsterApiPath
	return heapsterEndpointUrl, nil
}

func getHeapsterMetrics(heapsterEndpointUrl, url string) (*simplejson.Json, error) {
	response, err := httpClient.Get(heapsterEndpointUrl + url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := simplejson.NewFromReader(response.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func getNodeCPUUsage(heapsterEndpointUrl, nodeName string) *resource.Quantity {
	nodeCPUUsageRate, err := getHeapsterMetrics(heapsterEndpointUrl, "/nodes/" + nodeName + "/metrics/cpu/usage_rate")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return getLatestCPUUsage(nodeCPUUsageRate)
}

func getNodeMemoryUsage(heapsterEndpointUrl, nodeName string) *resource.Quantity {
	nodeMemoryUsage, err := getHeapsterMetrics(heapsterEndpointUrl, "/nodes/" + nodeName + "/metrics/memory/usage")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return getLatestMemoryUsage(nodeMemoryUsage)
}

func getPodCPUUsage(heapsterEndpointUrl, podName, namespace string) *resource.Quantity {
	podCPUUsageRate, err := getHeapsterMetrics(heapsterEndpointUrl, "/namespaces/" + namespace + "/pods/" + podName + "/metrics/cpu/usage_rate")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return getLatestCPUUsage(podCPUUsageRate)
}

func getPodMemoryUsage(heapsterEndpointUrl, podName, namespace string) *resource.Quantity {
	podMemoryUsage, err := getHeapsterMetrics(heapsterEndpointUrl, "/namespaces/" + namespace + "/pods/" + podName + "/metrics/memory/working_set")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return getLatestMemoryUsage(podMemoryUsage)
}

func getLatestCPUUsage(cpuUsageRate *simplejson.Json) *resource.Quantity {
	cpuUsageMetrics := cpuUsageRate.Get("metrics")
	usageArray, err := cpuUsageMetrics.Array()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// get cpu usage of the latest timestamp
	cpuUsageMap := cpuUsageMetrics.GetIndex(len(usageArray)-1).MustMap()
	value, err := cpuUsageMap["value"].(json.Number).Int64()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return resource.NewMilliQuantity(value, resource.DecimalSI)
}

func getLatestMemoryUsage(memoryUsage *simplejson.Json) *resource.Quantity {
	memoryUsageMetrics := memoryUsage.Get("metrics")
	usageArray, err := memoryUsageMetrics.Array()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// get memory usage of the latest timestamp
	memoryUsageMap := memoryUsageMetrics.GetIndex(len(usageArray)-1).MustMap()
	value, err := memoryUsageMap["value"].(json.Number).Int64()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return resource.NewQuantity(value, resource.BinarySI)
}
