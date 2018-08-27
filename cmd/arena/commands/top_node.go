package commands

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	showDetails bool
)

type NodeInfo struct {
	node v1.Node
	pods []v1.Pod
}

func NewTopNodeCommand() *cobra.Command {

	var command = &cobra.Command{
		Use:   "node",
		Short: "Display Resource (GPU) usage of nodes.",
		Run: func(cmd *cobra.Command, args []string) {
			setupKubeconfig()
			client, err := initKubeClient()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			allPods, err := acquireAllActivePods(client)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			nd := newNodeDescriber(client, allPods)
			nodeInfos, err := nd.getAllNodeInfos()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			heapsterEndpointUrl, err := nd.getHeapsterEndpointUrl()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			displayTopNode(nodeInfos, heapsterEndpointUrl)
		},
	}

	command.Flags().BoolVarP(&showDetails, "details", "d", false, "Display details")
	return command
}

type NodeDescriber struct {
	client  *kubernetes.Clientset
	allPods []v1.Pod
}

func newNodeDescriber(client *kubernetes.Clientset, pods []v1.Pod) *NodeDescriber {
	return &NodeDescriber{
		client:  client,
		allPods: pods,
	}
}

func (d *NodeDescriber) getHeapsterEndpointUrl() (string, error) {
	return getHeapsterEndpointUrl(d.client)
}

func (d *NodeDescriber) getAllNodeInfos() ([]NodeInfo, error) {
	nodeInfoList := []NodeInfo{}
	nodeList, err := d.client.CoreV1().Nodes().List(metav1.ListOptions{})

	if err != nil {
		return nodeInfoList, err
	}

	for _, node := range nodeList.Items {
		pods := d.getPodsFromNode(node)
		nodeInfoList = append(nodeInfoList, NodeInfo{
			node: node,
			pods: pods,
		})
	}

	return nodeInfoList, nil
}

func (d *NodeDescriber) getPodsFromNode(node v1.Node) []v1.Pod {
	pods := []v1.Pod{}
	for _, pod := range d.allPods {
		if pod.Spec.NodeName == node.Name {
			pods = append(pods, pod)
		}
	}

	return pods
}

func displayTopNode(nodes []NodeInfo, heapsterEndpointUrl string) {
	if showDetails {
		displayTopNodeDetails(nodes, heapsterEndpointUrl)
	} else {
		displayTopNodeSummary(nodes, heapsterEndpointUrl)
	}
}

func displayTopNodeSummary(nodeInfos []NodeInfo, heapsterEndpointUrl string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	var (
		totalGPUsInCluster     int64
		allocatedGPUsInCluster int64
		totalCPUsInCluster     int64
		usedCPUsInCluster      int64
		totalMemoryInCluster   int64
		usedMemoryInCluster    int64
	)

	fmt.Fprintf(w, "NAME\tIPADDRESS\tROLE\tGPU(Total)\tGPU(Allocated)\tCPU(Total)\tCPU(Usage)\tMemory(Total)\tMemory(Usage)\n")
	for _, nodeInfo := range nodeInfos {
		totalGPU, allocatedGPU := calculateNodeGPU(nodeInfo)
		totalCPU, usedCPU := calculateNodeCPU(heapsterEndpointUrl, nodeInfo)
		totalMemory, usedMemory := calculateNodeMemory(heapsterEndpointUrl, nodeInfo)
		totalGPUsInCluster += totalGPU
		allocatedGPUsInCluster += allocatedGPU
		totalCPUsInCluster += totalCPU
		usedCPUsInCluster += usedCPU
		totalMemoryInCluster += totalMemory
		usedMemoryInCluster += usedMemory

		address := "unknown"
		if len(nodeInfo.node.Status.Addresses) > 0 {
			// address = nodeInfo.node.Status.Addresses[0].Address
			for _, addr := range nodeInfo.node.Status.Addresses {
				if addr.Type == v1.NodeInternalIP {
					address = addr.Address
					break
				}
			}
		}

		role := "worker"
		if isMasterNode(nodeInfo.node) {
			role = "master"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", nodeInfo.node.Name,
			address,
			role,
			strconv.FormatInt(totalGPU, 10),
			strconv.FormatInt(allocatedGPU, 10),
			resource.NewMilliQuantity(totalCPU, resource.DecimalSI).String(),
			resource.NewMilliQuantity(usedCPU, resource.DecimalSI).String(),
			resource.NewQuantity(totalMemory, resource.BinarySI).String(),
			resource.NewQuantity(usedMemory, resource.BinarySI).String())
	}
	fmt.Fprintf(w, "--------------------------------------------------------------------------------------------------------------------------------------\n")
	fmt.Fprintf(w, "Allocated/Total GPUs In Cluster:\n")
	log.Debugf("gpu: %s, allocated GPUs %s", strconv.FormatInt(totalGPUsInCluster, 10),
		strconv.FormatInt(allocatedGPUsInCluster, 10))
	var gpuUsage float64 = 0
	if totalGPUsInCluster > 0 {
		gpuUsage = float64(allocatedGPUsInCluster) / float64(totalGPUsInCluster) * 100
	}
	fmt.Fprintf(w, "%s/%s (%d%%)\t\n",
		strconv.FormatInt(allocatedGPUsInCluster, 10),
		strconv.FormatInt(totalGPUsInCluster, 10),
		int64(gpuUsage))

	fmt.Fprintf(w, "Used/Total CPUs In Cluster:\n")
	log.Debugf("cpu: %s, used CPUs %s", resource.NewMilliQuantity(totalCPUsInCluster, resource.DecimalSI).String(),
		resource.NewMilliQuantity(usedCPUsInCluster, resource.DecimalSI).String())
	var cpuUsage float64 = 0
	if totalCPUsInCluster > 0 {
		cpuUsage = float64(usedCPUsInCluster) / float64(totalCPUsInCluster) * 100
	}
	fmt.Fprintf(w, "%s/%s (%d%%)\t\n",
		resource.NewMilliQuantity(usedCPUsInCluster, resource.DecimalSI).String(),
		resource.NewMilliQuantity(totalCPUsInCluster, resource.DecimalSI).String(),
		int64(cpuUsage))

	fmt.Fprintf(w, "Used/Total Memory In Cluster:\n")
	log.Debugf("Memory: %s, used Memory %s", resource.NewQuantity(usedMemoryInCluster, resource.BinarySI).String(),
		resource.NewQuantity(totalMemoryInCluster, resource.BinarySI).String())
	var memoryUsage float64 = 0
	if totalMemoryInCluster > 0 {
		memoryUsage = float64(usedMemoryInCluster) / float64(totalMemoryInCluster) * 100
	}
	fmt.Fprintf(w, "%s/%s (%d%%)\t\n",
		resource.NewQuantity(usedMemoryInCluster, resource.BinarySI).String(),
		resource.NewQuantity(totalMemoryInCluster, resource.BinarySI).String(),
		int64(memoryUsage))

	_ = w.Flush()
}

func displayTopNodeDetails(nodeInfos []NodeInfo, heapsterEndpointUrl string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	var (
		totalGPUsInCluster     int64
		allocatedGPUsInCluster int64
		totalCPUsInCluster     int64
		usedCPUsInCluster      int64
		totalMemoryInCluster   int64
		usedMemoryInCluster    int64
	)

	fmt.Fprintf(w, "\n")
	for _, nodeInfo := range nodeInfos {
		totalGPU, allocatedGPU := calculateNodeGPU(nodeInfo)
		totalCPU, usedCPU := calculateNodeCPU(heapsterEndpointUrl, nodeInfo)
		totalMemory, usedMemory := calculateNodeMemory(heapsterEndpointUrl, nodeInfo)
		totalGPUsInCluster += totalGPU
		allocatedGPUsInCluster += allocatedGPU
		totalCPUsInCluster += totalCPU
		usedCPUsInCluster += usedCPU
		totalMemoryInCluster += totalMemory
		usedMemoryInCluster += usedMemory

		address := "unknown"
		if len(nodeInfo.node.Status.Addresses) > 0 {
			address = nodeInfo.node.Status.Addresses[0].Address
		}

		role := "worker"
		if isMasterNode(nodeInfo.node) {
			role = "master"
		}
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "NAME:\t%s\n", nodeInfo.node.Name)
		fmt.Fprintf(w, "IPADDRESS:\t%s\n", address)
		fmt.Fprintf(w, "ROLE:\t%s\n", role)

		pods := resourcePods(heapsterEndpointUrl, nodeInfo.pods)
		if len(pods) > 0 {
			fmt.Fprintf(w, "\n")
			fmt.Fprintf(w, "NAMESPACE\tNAME\tGPU REQUESTS\tGPU LIMITS\tCPU USAGE\tMEM USAGE\n")
			for _, pod := range pods {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", pod.Namespace,
					pod.Name,
					strconv.FormatInt(gpuInPod(pod), 10),
					strconv.FormatInt(gpuInPod(pod), 10),
					resource.NewMilliQuantity(cpuInPod(heapsterEndpointUrl, pod), resource.DecimalSI).String(),
					resource.NewQuantity(memoryInPod(heapsterEndpointUrl, pod), resource.BinarySI).String())
			}
			fmt.Fprintf(w, "\n")
		}
		var gpuUsageInNode float64 = 0
		if totalGPU > 0 {
			gpuUsageInNode = float64(allocatedGPU) / float64(totalGPU) * 100
		} else {
			fmt.Fprintf(w, "\n")
		}

		var cpuUsageInNode float64 = 0
		if totalCPU > 0 {
			cpuUsageInNode = float64(usedCPU) / float64(totalCPU) * 100
		} else {
			fmt.Fprintf(w, "\n")
		}
		var memoryUsageInNode float64 = 0
		if totalMemory > 0 {
			memoryUsageInNode = float64(usedMemory) / float64(totalMemory) * 100
		} else {
			fmt.Fprintf(w, "\n")
		}

		fmt.Fprintf(w, "Total GPUs In Node %s:\t%s \t\n", nodeInfo.node.Name, strconv.FormatInt(totalGPU, 10))
		fmt.Fprintf(w, "Allocated GPUs In Node %s:\t%s (%d%%)\t\n", nodeInfo.node.Name, strconv.FormatInt(allocatedGPU, 10), int64(gpuUsageInNode))
		log.Debugf("gpu: %s, allocated GPUs %s", strconv.FormatInt(totalGPU, 10),
			strconv.FormatInt(allocatedGPU, 10))

		fmt.Fprintf(w, "Total CPUs In Node %s:\t%s \t\n", nodeInfo.node.Name, resource.NewMilliQuantity(totalCPU, resource.DecimalSI).String())
		fmt.Fprintf(w, "Used CPUs In Node %s:\t%s (%d%%)\t\n", nodeInfo.node.Name, resource.NewMilliQuantity(usedCPU, resource.DecimalSI).String(), int64(cpuUsageInNode))
		log.Debugf("cpu: %s, allocated CPUs %s", resource.NewMilliQuantity(totalCPU, resource.DecimalSI).String(),
			resource.NewMilliQuantity(usedCPU, resource.DecimalSI).String())
		fmt.Fprintf(w, "Total Memory In Node %s:\t%s \t\n", nodeInfo.node.Name, resource.NewQuantity(totalMemory, resource.BinarySI).String())
		fmt.Fprintf(w, "Used Memory In Node %s:\t%s (%d%%)\t\n", nodeInfo.node.Name, resource.NewQuantity(usedMemory, resource.BinarySI).String(), int64(memoryUsageInNode))
		log.Debugf("memory: %s, allocated Memory %s", resource.NewQuantity(totalMemory, resource.BinarySI).String(),
			resource.NewQuantity(usedMemory, resource.BinarySI).String())
		fmt.Fprintf(w, "--------------------------------------------------------------------------------------------------------------------------------------\n")
	}
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "Allocated/Total GPUs In Cluster:\t")
	log.Debugf("gpu: %s, allocated GPUs %s", strconv.FormatInt(totalGPUsInCluster, 10),
		strconv.FormatInt(allocatedGPUsInCluster, 10))

	var gpuUsage float64 = 0
	if totalGPUsInCluster > 0 {
		gpuUsage = float64(allocatedGPUsInCluster) / float64(totalGPUsInCluster) * 100
	}
	fmt.Fprintf(w, "%s/%s (%d%%)\t\n",
		strconv.FormatInt(allocatedGPUsInCluster, 10),
		strconv.FormatInt(totalGPUsInCluster, 10),
		int64(gpuUsage))

	fmt.Fprintf(w, "Used/Total CPUs In Cluster:\t")
	log.Debugf("cpu: %s, used CPUs %s", resource.NewMilliQuantity(totalCPUsInCluster, resource.DecimalSI).String(),
		resource.NewMilliQuantity(usedCPUsInCluster, resource.DecimalSI).String())
	var cpuUsage float64 = 0
	if totalCPUsInCluster > 0 {
		cpuUsage = float64(usedCPUsInCluster) / float64(totalCPUsInCluster) * 100
	}
	fmt.Fprintf(w, "%s/%s (%d%%)\t\n",
		resource.NewMilliQuantity(usedCPUsInCluster, resource.DecimalSI).String(),
		resource.NewMilliQuantity(totalCPUsInCluster, resource.DecimalSI).String(),
		int64(cpuUsage))

	fmt.Fprintf(w, "Used/Total Memory In Cluster:\t")
	log.Debugf("memory: %s, used Memory %s", resource.NewQuantity(totalMemoryInCluster, resource.BinarySI).String(),
		resource.NewQuantity(usedMemoryInCluster, resource.BinarySI).String())
	var memoryUsage float64 = 0
	if totalMemoryInCluster > 0 {
		memoryUsage = float64(usedMemoryInCluster) / float64(totalMemoryInCluster) * 100
	}
	fmt.Fprintf(w, "%s/%s (%d%%)\t\n",
		resource.NewQuantity(usedMemoryInCluster, resource.BinarySI).String(),
		resource.NewQuantity(totalMemoryInCluster, resource.BinarySI).String(),
		int64(memoryUsage))

	_ = w.Flush()
}

// calculate the GPU count of each node
func calculateNodeGPU(nodeInfo NodeInfo) (totalGPU int64, allocatedGPU int64) {
	node := nodeInfo.node
	totalGPU = gpuInNode(node)
	// allocatedGPU = gpuInPod()

	for _, pod := range nodeInfo.pods {
		allocatedGPU += gpuInPod(pod)
	}

	return totalGPU, allocatedGPU
}

// calculate the CPU count of each node
func calculateNodeCPU(heapsterEndpointUrl string, nodeInfo NodeInfo) (totalCPU int64, usedCPU int64) {
	node := nodeInfo.node
	totalCPU = cpuInNode(node)

	for _, pod := range nodeInfo.pods {
		usedCPU += cpuInPod(heapsterEndpointUrl, pod)
	}

	return totalCPU, usedCPU
}

// calculate the Memory count of each node
func calculateNodeMemory(heapsterEndpointUrl string, nodeInfo NodeInfo) (totalMemory int64, usedMemory int64) {
	node := nodeInfo.node
	totalMemory = memoryInNode(node)

	for _, pod := range nodeInfo.pods {
		usedMemory += memoryInPod(heapsterEndpointUrl, pod)
	}

	return totalMemory, usedMemory
}

func isMasterNode(node v1.Node) bool {
	if _, ok := node.Labels[masterLabelRole]; ok {
		return true
	}

	return false
}
