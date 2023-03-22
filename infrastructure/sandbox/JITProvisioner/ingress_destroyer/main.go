package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	instanceID := getOrPanic("INSTANCE_ID")
	clusterName := getOrPanic("CLUSTER_NAME")
	region := getOrDefault("REGION", "us-east-2")

	deleteIngress(instanceID, clusterName, region)
}

func getOrDefault(s, d string) string {
	r, ok := os.LookupEnv(s)
	if !ok {
		return d
	}
	return r
}

func getOrPanic(env string) string {
	s, ok := os.LookupEnv(env)
	if !ok {
		panic(fmt.Sprintf("%s not found", env))
	}
	return s
}

func deleteIngress(id, name, region string) {

	//AWS_PROFILE=Sandbox aws eks --region us-east-2 update-kubeconfig --name sandbox-prod
	conf := os.TempDir() + "/kube-config"
	cmd := exec.Command("aws", "eks", "--region", region, "update-kubeconfig", "--name", name, "--kubeconfig", conf)
	cmd.Env = os.Environ()
	buf, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err)
		fmt.Println(cmd.String())
		fmt.Println(string(buf))
		return
	}

	config, err := clientcmd.BuildConfigFromFlags("", conf)
	if err != nil {
		fmt.Println(err)
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Delete the ingress using the Kubernetes clientset
	err = clientset.NetworkingV1().Ingresses("default").Delete(context.Background(), id, v1.DeleteOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Ingress %s deleted\n", id)
}
