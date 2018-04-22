package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

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

	var debugPod *DebugPod

	ctx, cancel := context.WithCancel(context.Background())

	debugPod, err = NewDebugPod(ctx, config, *namespace, *podName)
	if err != nil {
		log.Printf("%v", err)
		exit(cancel, nil, 1)
	}

	log.Println("creating debugPod ")
	end, err := debugPod.Create()
	if err != nil {
		log.Printf("%v", err)
		exit(cancel, end, 1)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		exit(cancel, end, 1)
	}()

	log.Println("attaching to debugPod")
	err = debugPod.Attach()
	if err != nil {
		log.Printf("%v", err)
		exit(cancel, end, 1)
	}
	exit(cancel, end, 0)

}

func exit(cancel context.CancelFunc, end <-chan struct{}, code int) {
	cancel()
	if end != nil {
		<-end
	}
	os.Exit(code)
}
