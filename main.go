package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

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

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		exit(cancel, 1)
	}()

	debugPod, err = NewDebugPod(ctx, config, *namespace, *podName)
	if err != nil {
		log.Printf("%v", err)
		exit(cancel, 1)
	}

	log.Println("creating debugPod ")
	err = debugPod.Create()
	if err != nil {
		log.Printf("%v", err)
		exit(cancel, 1)
	}

	log.Println("attaching to debugPod")
	err = debugPod.Attach()
	if err != nil {
		log.Printf("%v", err)
		exit(cancel, 1)
	}
	exit(cancel, 0)

}

func exit(cancel context.CancelFunc, code int) {
	cancel()
	time.Sleep(2 * time.Second)
	os.Exit(1)
}
