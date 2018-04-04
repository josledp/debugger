package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type DebugPod struct {
	targetPod       string
	targetNamespace string
	targetNode      string
	podName         string
	pod             *v1.Pod
	k8s             *kubernetes.Clientset
	ctx             context.Context
}

func NewDebugPod(ctx context.Context, k8sClient *kubernetes.Clientset, namespace, targetPod string) (*DebugPod, error) {

	dp := &DebugPod{
		targetPod:       targetPod,
		targetNamespace: namespace,
		podName:         fmt.Sprintf("debug-%s-%d", targetPod, rand.Int63()),
		k8s:             k8sClient,
		ctx:             ctx,
	}

	pod, err := dp.k8s.CoreV1().Pods(namespace).Get(dp.targetPod, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get pod %s: %v", dp.targetPod, err)
	}

	dp.targetNode = pod.Spec.NodeName

	privilegeEscalation := true
	privileged := true

	dp.pod = &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dp.podName,
			Namespace: dp.targetNamespace,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				v1.Container{
					Name:  "debugpod",
					Image: "josledp/debugpod",
					SecurityContext: &v1.SecurityContext{
						AllowPrivilegeEscalation: &privilegeEscalation,
						Privileged:               &privileged,
					},
				},
			},
			NodeSelector: map[string]string{
				"kubernetes.io/hostname": dp.targetNode,
			},
		},
	}

	return dp, nil
}

func (dp *DebugPod) waitForPod(timeout int) error {
	var i int
	for i = 0; i < timeout; i++ {
		status, err := dp.k8s.CoreV1().Pods(dp.targetNamespace).Get(dp.podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("unable to retrieve pod status: %v", err)
		}
		if status.Status.Phase == "Running" {
			break
		}
		select {
		case <-dp.ctx.Done():
			dp.Clean()
			return fmt.Errorf("exited because requested")
		default:
		}
		time.Sleep(1 * time.Second)
	}
	for ; i < timeout; i++ {
		status, err := dp.k8s.CoreV1().Pods(dp.targetNamespace).Get(dp.podName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("unable to retrieve pod status: %v", err)
		}
		if status.Status.ContainerStatuses[0].Ready {
			return nil
		}
		select {
		case <-dp.ctx.Done():
			dp.Clean()
			return fmt.Errorf("exited because requested")
		default:
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("Pod not ready after %d seconds", timeout)
}

func (dp *DebugPod) Create() error {
	var err error
	dp.pod, err = dp.k8s.CoreV1().Pods(dp.targetNamespace).Create(dp.pod)
	if err != nil {
		return fmt.Errorf("error creating debugPod: %v", err)
	}

	err = dp.waitForPod(30)

	if err != nil {
		err2 := dp.Clean()
		if err2 != nil {
			return fmt.Errorf("debugPod did not get ready status: %v\nFurthermore there was an error cleaning the pod %s: %v", err, dp.podName, err2)
		}
		return fmt.Errorf("debugPod did not get ready status: %v", err)
	}
	go func() {
		<-dp.ctx.Done()
		dp.Clean()
	}()
	return nil
}

func (dp *DebugPod) Clean() error {
	return dp.k8s.CoreV1().Pods(dp.targetNamespace).Delete(dp.podName, &metav1.DeleteOptions{})
}
