package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	//"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
	instanceID := getOrPanic("INSTANCE_ID")
	ddbTable := getOrPanic("DYNAMODB_LIFECYCLE_TABLE")
	clusterName := getOrPanic("CLUSTER_NAME")

	deleteIngress(instanceID, clusterName, ddbTable)
}

func getOrPanic(env string) string {
	s, ok := os.LookupEnv(env)
	if !ok {
		panic(fmt.Sprintf("%s not found", env))
	}
	return s
}

func deleteIngress(id, name, ddbTable string) {

	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	// AWS_PROFILE=Sandbox aws eks --region us-east-2 update-kubeconfig --name sandbox-prod
	conf := os.TempDir() + "/kube-config"
	cmd := exec.Command("aws", "eks", "update-kubeconfig", "--name", name, "--kubeconfig", conf)
	cmd.Env = os.Environ()
	buf, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(cmd.String())
		log.Println(string(buf))
		log.Fatal(err)
	}

	config, err := clientcmd.BuildConfigFromFlags("", conf)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	// Delete the ingress using the Kubernetes clientset
	err = clientset.NetworkingV1().Ingresses("default").Delete(context.Background(), id, v1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}

	/*
	// Delete the cronjob so we don't spam the database for stuff that's not running
	err = clientset.BatchV1().CronJobs("default").Delete(context.Background(), id, v1.DeleteOptions{})
	if err != nil {
		log.Fatal(err)
	}

	// Scale it down to save money
	time.Sleep(60)
    s, err := clientset.AppsV1().Deployments("default").GetScale(context.Background(), id, v1.GetOptions{})
    if err != nil {
        log.Fatal(err)
    }

    sc := *s
    sc.Spec.Replicas = 0
	_, err = clientset.AppsV1().Deployments("default").UpdateScale(context.Background(), id, &sc, v1.UpdateOptions{})
	if err != nil {
		log.Fatal(err)
	}
	*/

	svc := dynamodb.New(sess)
	err = updateFleetInstanceState(id, ddbTable, svc)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Ingress %s deleted\n", id)
}

func updateFleetInstanceState(id, table string, svc *dynamodb.DynamoDB) (err error) {
	log.Printf("updating instance: %+v", id)
	// Perform a conditional update to claim the item
	input := &dynamodb.UpdateItemInput{
		TableName: aws.String(table),
		Key: map[string]*dynamodb.AttributeValue{
			"ID": {
				S: aws.String(id),
			},
		},
		UpdateExpression:         aws.String("set #fleet_state = :v2"),
		ExpressionAttributeNames: map[string]*string{"#fleet_state": aws.String("State")},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":v2": {
				S: aws.String("ingress_destroyed"),
			},
		},
	}
	_, err = svc.UpdateItem(input)
	return
}
