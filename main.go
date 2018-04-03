package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"k8s.io/api/core/v1"
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
	log.Println("creating k8s client")
	// creates the clientset
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("unable to setup client: %v", err)

	}
	log.Printf("searching pod %s", *podName)
	pod, err := k8sClient.CoreV1().Pods(*namespace).Get(*podName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("unable to get pod %s: %v", *podName, err)
	}

	nodeName := pod.Spec.NodeName
	log.Printf("pod %s is on node %s", *podName, nodeName)

	privilegeEscalation := true
	privileged := true

	log.Println("creating debugPod")
	debugPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "debugger",
			Namespace: *namespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:  "debugger",
					Image: "debugger",
					SecurityContext: &v1.SecurityContext{
						AllowPrivilegeEscalation: &privilegeEscalation,
						Privileged:               &privileged,
					},
				},
			},
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": nodeName,
			},
		},
	}

	debugPod, err = k8sClient.CoreV1().Pods(*namespace).Create(debugPod)
	if err != nil {
		log.Fatalf("error creating debugPod: %v", err)
	}
	for {
		status, err := k8sClient.CoreV1().Pods(*namespace).Get("debugger", metav1.GetOptions{})
		if err != nil {
			log.Println("unable to retrieve debug pod status")
		}
		log.Printf("Status is: %s", status.Status.Phase)
		time.Sleep(1 * time.Second)
	}

}
