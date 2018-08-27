package commands

import (
	"k8s.io/api/core/v1"
)

// filter out the pods with GPU
func gpuPods(pods []v1.Pod) (podsWithGPU []v1.Pod) {
	for _, pod := range pods {
		if gpuInPod(pod) > 0 {
			podsWithGPU = append(podsWithGPU, pod)
		}
	}
	return podsWithGPU
}

// The way to get GPU Count of Node: nvidia.com/gpu
func gpuInNode(node v1.Node) int64 {
	val, ok := node.Status.Capacity[NVIDIAGPUResourceName]

	if !ok {
		return gpuInNodeDeprecated(node)
	}

	return val.Value()
}

// The way to get GPU Count of Node: alpha.kubernetes.io/nvidia-gpu
func gpuInNodeDeprecated(node v1.Node) int64 {
	val, ok := node.Status.Capacity[DeprecatedNVIDIAGPUResourceName]

	if !ok {
		return 0
	}

	return val.Value()
}

func gpuInPod(pod v1.Pod) (gpuCount int64) {
	containers := pod.Spec.Containers
	for _, container := range containers {
		gpuCount += gpuInContainer(container)
	}

	return gpuCount
}

func gpuInActivePod(pod v1.Pod) (gpuCount int64) {
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return 0
	}

	containers := pod.Spec.Containers
	for _, container := range containers {
		gpuCount += gpuInContainer(container)
	}

	return gpuCount
}

func gpuInContainer(container v1.Container) int64 {
	val, ok := container.Resources.Limits[NVIDIAGPUResourceName]

	if !ok {
		return gpuInContainerDeprecated(container)
	}

	return val.Value()
}

func gpuInContainerDeprecated(container v1.Container) int64 {
	val, ok := container.Resources.Limits[DeprecatedNVIDIAGPUResourceName]

	if !ok {
		return 0
	}

	return val.Value()
}

// filter out the pods which use resources
func resourcePods(heapsterEndpointUrl string, pods []v1.Pod) (podsWithResource []v1.Pod) {
	for _, pod := range pods {
		if gpuInPod(pod) > 0 || cpuInPod(heapsterEndpointUrl, pod) > 0 || memoryInPod(heapsterEndpointUrl, pod) > 0 {
			podsWithResource = append(podsWithResource, pod)
		}
	}
	return podsWithResource
}

// The way to get CPU Count of Node
func cpuInNode(node v1.Node) int64 {
	val, ok := node.Status.Capacity[v1.ResourceCPU]

	if !ok {
		return 0
	}

	return val.MilliValue()
}

func cpuInPod(heapsterEndpointUrl string, pod v1.Pod) (cpuCount int64) {
	return getPodCPUUsage(heapsterEndpointUrl, pod.Name, pod.Namespace).MilliValue()
}

// The way to get Memory Count of Node
func memoryInNode(node v1.Node) int64 {
	val, ok := node.Status.Capacity[v1.ResourceMemory]

	if !ok {
		return 0
	}

	return val.Value()
}

func memoryInPod(heapsterEndpointUrl string, pod v1.Pod) (memoryCount int64) {
	return getPodMemoryUsage(heapsterEndpointUrl, pod.Name, pod.Namespace).Value()
}
