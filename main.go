package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pahud/eks/pkg/eksutils"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

func main() {
	lambda.Start(handler)
}

// handler is the Lambda handler function
func handler(ctx context.Context, cweEvent events.CloudWatchEvent) (string, error) {
	var err error
	// inputJSON := []byte(
	// 	"{\"version\":\"0\",\"id\":\"890abcde-f123-4567-890a-bcdef1234567\"," +
	// 		"\"detail-type\":\"EC2 Spot Instance Interruption Warning\",\"source\":\"aws.ec2\"," +
	// 		"\"account\":\"123456789012\",\"time\":\"2016-12-30T18:44:49Z\"," +
	// 		"\"region\":\"us-west-2\"," +
	// 		"\"resources\":[\"arn:aws:ec2:us-west-2b:instance/i-0efa14160939310ef\"]," +
	// 		"\"detail\":{\"instance-id\":\"i-0e05ad95febabe07e\", \"instance-action\":\"terminate\"}}")
	// var inputEvent events.CloudWatchEvent
	var inputEvent = cweEvent
	// err := json.Unmarshal(inputJSON, &inputEvent)
	// if err != nil {
	// 	log.Errorf("Could not unmarshal cloudwatch event: %v", err)
	// }
	type Ec2Detail struct {
		InstanceID     string `json:"instance-id,omitempty"`
		InstanceAction string `json:"instance-action,omitempty"`
	}
	var ec2Detail Ec2Detail
	log.Infof("detail=%v", string(inputEvent.Detail))
	json.Unmarshal(inputEvent.Detail, &ec2Detail)
	log.Infof("ec2Detail=%v", ec2Detail)
	instanceID := ec2Detail.InstanceID
	log.Infof("instanceID=%v", instanceID)
	// outputJSON, err := json.Marshal(inputEvent)
	// log.Infof("outputJSON=%v", string(outputJSON))
	// taintNode("i-0c3fd90fa072e2e47")
	taintNode(instanceID)
	return "OK", err
}

func ec2Info(instanceID string) (nodeName string, err error) {
	log.Infof("looking up nodeName of %v", instanceID)
	session := session.Must(session.NewSession(&aws.Config{Region: aws.String("us-west-2")}))
	svc := ec2.New(session)
	filters := []*ec2.Filter{
		&ec2.Filter{
			Name: aws.String("instance-id"),
			Values: []*string{
				aws.String(instanceID),
			},
		},
	}
	input := ec2.DescribeInstancesInput{Filters: filters}
	result, err := svc.DescribeInstances(&input)
	// log.Info(aws.String(result.Reservations[0].Instances[0].PrivateDnsName))
	log.Info(*result)
	if len(result.Reservations) < 1 {
		return "", errors.New("instance not found")
	}
	nodeName = *result.Reservations[0].Instances[0].NetworkInterfaces[0].PrivateDnsName
	log.Info(nodeName)

	return nodeName, err
}

func taintNode(id string) {
	// nodeName, err := ec2Info("i-0efa14160939310ef")
	nodeName, err := ec2Info(id)
	if err != nil {
		log.Errorf("got error: %v", err)
	}
	h := eksutils.EksHandler{}
	if h.ClusterName = os.Getenv("CLUSTER_NAME"); h.ClusterName == "" {
		h.ClusterName = "myeks"
	}
	log.Infof("clusterName=%v", h.ClusterName)
	h.GetClientSet()
	// h.GetNodes()
	// h.GetPods()
	// nodeName := "ip-192-168-112-39.us-west-2.compute.internal"
	// now := metav1.Now()
	h.TaintNode(&v1.Taint{
		Key:    "SpotTerminating",
		Value:  "true",
		Effect: v1.TaintEffectNoExecute,
		// Effect:    v1.TaintEffectPreferNoSchedule,
		// TimeAdded: &now,
	}, nodeName)
}
