package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	dockerterm "github.com/docker/docker/pkg/term"
	"k8s.io/kubernetes/pkg/kubectl/util/term"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type DebugPod struct {
	targetPod       string
	targetNamespace string
	targetNode      string
	podName         string
	pod             *v1.Pod
	k8sConfig       *rest.Config
	k8s             *kubernetes.Clientset
	ctx             context.Context
}

func NewDebugPod(ctx context.Context, k8sConfig *rest.Config, namespace, targetPod string) (*DebugPod, error) {

	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to setup client: %v", err)
	}
	r := rand.New(rand.NewSource(int64(time.Now().UnixNano())))

	dp := &DebugPod{
		targetPod:       targetPod,
		targetNamespace: namespace,
		podName:         fmt.Sprintf("debug-%s-%d", targetPod, r.Int63()),
		k8s:             k8sClient,
		k8sConfig:       k8sConfig,
		ctx:             ctx,
	}

	pod, err := dp.k8s.CoreV1().Pods(namespace).Get(dp.targetPod, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get pod %s: %v", dp.targetPod, err)
	}

	dp.targetNode = pod.Spec.NodeName

	privilegeEscalation := true
	privileged := true
	containerID := pod.Status.ContainerStatuses[0].ContainerID
	hostPathType := v1.HostPathFile

	dp.pod = &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dp.podName,
			Namespace: dp.targetNamespace,
		},
		Spec: v1.PodSpec{
			HostIPC:     true,
			HostPID:     true,
			HostNetwork: true,
			Containers: []v1.Container{
				v1.Container{
					Name:            "debugpod",
					Image:           "josledp/debugpod",
					ImagePullPolicy: v1.PullAlways,
					//					Args:  []string{containerID},
					Env: []v1.EnvVar{
						v1.EnvVar{Name: "CONTAINER_ID", Value: containerID},
					},
					SecurityContext: &v1.SecurityContext{
						AllowPrivilegeEscalation: &privilegeEscalation,
						Privileged:               &privileged,
					},
					VolumeMounts: []v1.VolumeMount{
						v1.VolumeMount{Name: "dockersock", MountPath: "/var/run/docker.sock"},
					},
				},
			},
			Volumes: []v1.Volume{
				v1.Volume{Name: "dockersock", VolumeSource: v1.VolumeSource{HostPath: &v1.HostPathVolumeSource{Path: "/var/run/docker.sock", Type: &hostPathType}}},
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
			dp.Clean(nil)
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
			dp.Clean(nil)
			return fmt.Errorf("exited because requested")
		default:
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("Pod not ready after %d seconds", timeout)
}

func (dp *DebugPod) Create() (<-chan struct{}, error) {
	var err error
	dp.pod, err = dp.k8s.CoreV1().Pods(dp.targetNamespace).Create(dp.pod)
	if err != nil {
		return nil, fmt.Errorf("error creating debugPod: %v", err)
	}

	err = dp.waitForPod(60)

	if err != nil {
		err2 := dp.Clean(nil)
		if err2 != nil {
			return nil, fmt.Errorf("debugPod did not get ready status: %v\nFurthermore there was an error cleaning the pod %s: %v", err, dp.podName, err2)
		}
		return nil, fmt.Errorf("debugPod did not get ready status: %v", err)
	}
	end := make(chan struct{})
	go func() {
		<-dp.ctx.Done()
		dp.Clean(end)
	}()
	return end, nil
}

func (dp *DebugPod) Clean(end chan<- struct{}) error {
	err := dp.k8s.CoreV1().Pods(dp.targetNamespace).Delete(dp.podName, &metav1.DeleteOptions{})
	if end != nil {
		close(end)
	}
	return err
}

func (dp *DebugPod) Attach() error {

	req := dp.k8s.CoreV1().RESTClient().Post().Resource("pods").Name(dp.podName).Namespace(dp.targetNamespace).SubResource("exec")
	req = req.Param("container", "debugpod")
	req = req.Param("command", "/entrypoint.sh")
	req = req.Param("stdin", "true")
	req = req.Param("stdout", "true")
	req = req.Param("tty", "true")

	executor, err := remotecommand.NewSPDYExecutor(dp.k8sConfig, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("unable to create executor: %v", err)
	}
	stdin, stdout, _ := dockerterm.StdStreams()

	t := term.TTY{
		Out: stdout,
		In:  stdin,
	}

	terminalSize := t.MonitorSize(t.GetSize())

	return executor.Stream(remotecommand.StreamOptions{Tty: true, Stdin: t.In, Stdout: t.Out, TerminalSizeQueue: terminalSize})
}
