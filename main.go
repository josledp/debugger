package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	var kubeconfig *string
	var config *rest.Config
	var err error
	if home := os.Getenv("HOME"); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	inCluster := flag.Bool("in-cluster", false, "configure in cluster")
	podName := flag.String("pod", "", "pod to debug")
	namespace := flag.String("namespace", "default", "(optional) namespace of the pod")

	flag.Parse()

	if *podName == "" {
		log.Fatalln("pod option must be specified")
	}

	if !*inCluster {
		if *kubeconfig == "" {
			log.Fatalf("kubeconfig path must be specified")
		}
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Fatalf("unable to load kubeconfig: %v", err)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Fatalf("unable to get Kubernetes config: %v", err)
		}

	}
	// creates the clientset
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("unable to setup client: %v", err)

	}

	pod, err := k8sClient.CoreV1().Pods(*namespace).Get(*podName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("unable to get pod %s: %v", *podName, err)
	}

	nodeName := pod.Spec.NodeName
	log.Printf("pod %s is on node %s", *podName, nodeName)
}
