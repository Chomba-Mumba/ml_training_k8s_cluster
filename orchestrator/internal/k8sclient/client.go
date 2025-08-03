package k8sclient

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func checkErr(err error) {
	if err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}
}

type K8sClient struct {
	client kubernetes.Interface
}

func NewK8sClient() (*K8sClient, error) {
	var kubeconfig *rest.Config

	// use cluster service account to get config object
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to load in-cluster config: %v", err)
	}
	kubeconfig = config

	// create the client
	client, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create a client: %v", err)
	}

	return &K8sClient{client: client}, nil
}

func (k8s *K8sClient) CreatePod(podName string, image string) error {
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{
				v1.Container{
					Name:    "main",
					Image:   image,
					Command: []string{"python"},
					Args:    []string{"-c", "print('hello world')"}, //TODO - run EA image and add hyperparameters
				},
			},
		},
	}

	//create pod
	var err error

	_, err = k8s.client.CoreV1().Pods("default").Create(
		context.Background(),
		pod,
		metav1.CreateOptions{},
	)

	if err != nil {
		return fmt.Errorf("ran into a problem creating pod %s: %v", podName, err)
	} else {
		return err
	}
}

func (k8s *K8sClient) DeletePod(podName string) {
	err := k8s.client.CoreV1().
		Pods("default").
		Delete(
			context.Background(),
			podName,
			metav1.DeleteOptions{},
		)
	checkErr(err)
}

func (k8s *K8sClient) GetPodExitCode(ctx context.Context, name string, podName string) (int32, error) {
	var exitCode int32
	cli := k8s.client.CoreV1().Pods("default")
	err := wait.PollUntilContextTimeout(ctx, 3*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		p, err := cli.Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if len(p.Status.ContainerStatuses) == 0 {
			return false, nil
		}
		state := p.Status.ContainerStatuses[0].State
		if state.Terminated != nil {
			exitCode = state.Terminated.ExitCode
			return true, nil
		}
		return false, nil
	})
	checkErr(err)
	return exitCode, nil
}
