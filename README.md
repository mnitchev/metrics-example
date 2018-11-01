# Sending container metrics from Kubernetes to Doppler

*_NOTE:_* This is example code from this [spike](https://www.pivotaltracker.com/story/show/160848366)

### Prerequisites

1. Kube cluster with [heapster](https://github.com/kubernetes/heapster) installed (should be installed by default if the cluster is version <=1.10)
2. A doppler running somewhere that's reachable from inside the cluster

### Example code

The example gathers the metrics for all pods running in the `opi` namespace. Unfortunately the metrics that are returned are just `cpu` and `memory` and doppler [requires](https://github.com/cloudfoundry/loggregator-api#containermetric) also `disk`, `disk_quota` and `memory_quota` to also be set for it to interpret the Envelope as a ContainerMetric. 
To emit the envelopes the [go-loggregator](https://github.com/cloudfoundry/go-loggregator/) client is used. The client has a [hardcoded value](https://github.com/cloudfoundry/go-loggregator/blob/master/tls.go#L13) for the "serverName" for which the doppler certificates are valid. In this example the code in `vendor` has been changed but a more permanent solution would be to add the "metron" alternative name to the certificates.
