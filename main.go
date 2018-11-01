package main

import (
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"

	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"

	loggregator "code.cloudfoundry.org/go-loggregator"
)

type PodMetricsList struct {
	Metadata Metadata      `json:"metadata"`
	Items    []*PodMetrics `json:"items"`
}

type PodMetrics struct {
	Metadata   Metadata      `json:"metadata"`
	Containers []*Containers `json:"containers"`
}

type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type Containers struct {
	Name  string `json:"name"`
	Usage Usage  `json:"usage"`
}

type Usage struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

func main() {
	tlsConfig, err := loggregator.NewIngressTLSConfig(
		os.Getenv("CA_CERT_PATH"),
		os.Getenv("CERT_PATH"),
		os.Getenv("KEY_PATH"),
	)
	if err != nil {
		log.Fatal("Could not create TLS config", err)
	}

	client, err := loggregator.NewIngressClient(
		tlsConfig,
		loggregator.WithAddr(os.Getenv("DOPPLER_ADDR")),
	)

	if err != nil {
		log.Fatal("Could not create client", err)
	}

	defer client.CloseSend()

	for {
		collectAppMetrics(client)
		time.Sleep(15 * time.Second)
	}
}

func collectAppMetrics(client *loggregator.IngressClient) {
	resp, err := http.Get("http://heapster.kube-system.svc.cluster.local/apis/metrics/v1alpha1/namespaces/opi/pods")
	if err != nil {
		fmt.Println("Failed to create get metrics for pods", err)
		return
	}

	metricList := &PodMetricsList{}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Failed to parse d body: ", err)
		return
	}
	fmt.Println("Kube metrics response body: ", string(body))
	if err = json.Unmarshal(body, metricList); err != nil {
		fmt.Println("Failed to decode metrics response", err)
		return
	}

	for _, podMetric := range metricList.Items {
		fmt.Println("Emitting metrics for pod: ", podMetric)
		podName := podMetric.Metadata.Name
		appID, indexID, err := parsePodName(podName)
		if err != nil {
			fmt.Println("Pod has no index id: ", podName, err)
			return
		}

		re := regexp.MustCompile("[a-zA-Z]+")
		match := re.FindStringSubmatch(podMetric.Containers[0].Usage.Memory)
		unit := match[0]
		index, err := strconv.Atoi(indexID)
		if err != nil {
			fmt.Println("Failed to convert index id", indexID)
			return
		}
		cpuValue, err := strconv.Atoi(podMetric.Containers[0].Usage.CPU)
		if err != nil {
			fmt.Println("Failed to convert cpu value", cpuValue)
			return
		}
		memoryValue, _ := strconv.Atoi(strings.Trim(podMetric.Containers[0].Usage.Memory, unit))
		if err != nil {
			fmt.Println("Failed to convert memory value", memoryValue)
			return
		}

		fmt.Println("Trying to send the following request: ", appID, indexID, index, cpuValue, memoryValue, unit)
		client.EmitGauge(
			loggregator.WithGaugeAppInfo(appID, index),
			loggregator.WithGaugeSourceInfo(appID, indexID),
			loggregator.WithGaugeValue("cpu", float64(cpuValue), "%"),
			loggregator.WithGaugeValue("memory", float64(memoryValue), unit),
			loggregator.WithGaugeValue("disk", 123, "Mb"),
			loggregator.WithGaugeValue("memory_quota", 123, "Mb"),
			loggregator.WithGaugeValue("disk_quota", 123, "Mb"),
		)
	}
}
func parsePodName(podName string) (string, string, error) {
	sl := strings.Split(podName, "-")

	if len(sl) <= 1 {
		return "", "", fmt.Errorf("Could not parse pod name from %s", podName)
	}

	return strings.Join(sl[:len(sl)-1], "-"), sl[len(sl)-1], nil
}
