package main

import (
	// "context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
)

var ec2svc *ec2.EC2
var asgsvc *autoscaling.AutoScaling

var isInitEc2svc = false
var isInitAsgsvc = false

// seconds of graceful period before sending the CompleteLifecycleAction signal
var asgCompleteLifecycleActionGracePeriod time.Duration

// LifecycleHook ...
type LifecycleHook struct {
	LifecycleActionToken string
	AutoScalingGroupName string
	LifecycleHookName    string
	EC2InstanceID        string
	LifecycleTransition  string
	init                 bool
}

func init() {
	asgCompleteLifecycleActionGracePeriod = 10 * time.Second
}

func main() {
	if inLambda() {
		lambda.Start(handler)
	} else {
		handler(events.CloudWatchEvent{})
	}
}

func inLambda() bool {
	if lambdaTaskRoot := os.Getenv("LAMBDA_TASK_ROOT"); lambdaTaskRoot != "" {
		return true
	}
	return false
}

// handler is the Lambda handler function
// func handler(ctx context.Context, cweEvent events.CloudWatchEvent) (string, error) {
func handler(cweEvent events.CloudWatchEvent) (string, error) {
	var err error
	var inputEvent events.CloudWatchEvent
	testJSON := `{
    "version": "0",
    "id": "1af5adec-abad-0254-e9f7-c2373a51599f",
    "detail-type": "EC2 Instance-terminate Lifecycle Action",
    "source": "aws.autoscaling",
    "account": "1234567890",
    "time": "2018-09-29T05:33:52Z",
    "region": "us-west-2",
    "resources": [
        "arn:aws:autoscaling:us-west-2:1234567890:autoScalingGroup:53ffecb4-9996-46c8-b635-9a679d702aef:autoScalingGroupName/eks-demo1-ng0-NodeGroup-1QEXE9U9ENSF7"
    ],
    "detail": {
        "LifecycleActionToken": "c53b152a-496b-4f61-bb57-bb705ba4c7c2",
        "AutoScalingGroupName": "eks-demo1-ng0-NodeGroup-1QEXE9U9ENSF7",
        "LifecycleHookName": "eks-demo1-ng0-ASGTerminateHook2-1IEZV1I4ZDHNS",
        "EC2InstanceId": "i-02ef12f64dfe3da29",
        "LifecycleTransition": "autoscaling:EC2_INSTANCE_TERMINATING"
    }
}`
	testJSONRawByte := json.RawMessage(testJSON)

	if inLambda() {
		inputEvent = cweEvent
	} else {
		log.Info("not in lambda")
		json.Unmarshal(testJSONRawByte, &inputEvent)
	}
	lch := LifecycleHook{
		init: false,
	}
	inputJSON, _ := json.Marshal(inputEvent)
	fmt.Println(string(inputJSON))

	var ec2Detail map[string]interface{}
	json.Unmarshal(inputEvent.Detail, &ec2Detail)
	var instanceID string
	switch detailType := string(inputEvent.DetailType); detailType {
	case "EC2 Spot Instance Interruption Warning":
		log.Info("got detail-type=EC2 Spot Instance Interruption Warning")
		instanceID = ec2Detail["InstanceID"].(string)

	case "EC2 Instance-terminate Lifecycle Action":
		log.Info("got detail-type=EC2 Instance-terminate Lifecycle Action")
		instanceID = ec2Detail["EC2InstanceId"].(string)
		lch.LifecycleActionToken = ec2Detail["LifecycleActionToken"].(string)
		lch.AutoScalingGroupName = ec2Detail["AutoScalingGroupName"].(string)
		lch.LifecycleHookName = ec2Detail["LifecycleHookName"].(string)
		lch.EC2InstanceID = ec2Detail["EC2InstanceId"].(string)
		lch.LifecycleTransition = ec2Detail["LifecycleTransition"].(string)
		lch.init = true

	default:
		log.Infof("unknown detail-type=%v", detailType)
		instanceID = "unknown"
		return fmt.Sprintf("unknown event type=%v", detailType), err

	}

	log.Infof("instanceID=%v", instanceID)
	// outputJSON, err := json.Marshal(inputEvent)
	// log.Infof("outputJSON=%v", string(outputJSON))
	// taintNode("i-0c3fd90fa072e2e47")

	taintNode(instanceID)
	log.Info("checking if it's asgnode")
	if lch.init {
		log.Info("start autoscale complete-lifecycle-actiopn callback")
		asgsvc = initAsgsvc()
		isInitAsgsvc = true
		log.Infof("sleeping for graceful period:%v second(s)", asgCompleteLifecycleActionGracePeriod.Seconds())
		time.Sleep(asgCompleteLifecycleActionGracePeriod)
		asgCompleteLifecycleAction(asgsvc, lch)
		// fmt.Println(asgerr.Error())

	}
	return "OK", err
}

func initEc2svc() *ec2.EC2 {
	if isInitEc2svc {
		log.Infof("already have ec2init,returning the existing one")
		return ec2svc
	}
	log.Infof("init ec2svc")
	currentRegion := os.Getenv("AWS_REGION")
	session := session.Must(session.NewSession(&aws.Config{Region: aws.String(currentRegion)}))
	svc := ec2.New(session)
	return svc
}

func initAsgsvc() *autoscaling.AutoScaling {
	if isInitAsgsvc {
		log.Infof("already have asginit,returning the existing one")
		return asgsvc
	}
	log.Infof("init asgsvc")
	currentRegion := os.Getenv("AWS_REGION")
	session := session.Must(session.NewSession(&aws.Config{Region: aws.String(currentRegion)}))
	svc := autoscaling.New(session)
	return svc
}

func asgCompleteLifecycleAction(asgsvc *autoscaling.AutoScaling, lch LifecycleHook) error {
	var err error
	input := autoscaling.CompleteLifecycleActionInput{
		AutoScalingGroupName:  aws.String(lch.AutoScalingGroupName),
		InstanceId:            aws.String(lch.EC2InstanceID),
		LifecycleActionResult: aws.String("CONTINUE"),
		LifecycleActionToken:  aws.String(lch.LifecycleActionToken),
		LifecycleHookName:     aws.String(lch.LifecycleHookName),
	}
	result, err := asgsvc.CompleteLifecycleAction(&input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeResourceContentionFault:
				fmt.Println(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
	} else {
		log.Info("CompleteLifecycleAction completed with no error")
	}
	fmt.Println(result)
	return err
}

func ec2Info(ec2svc *ec2.EC2, instanceID string) (nodeName string, err error) {
	log.Infof("looking up nodeName of %v", instanceID)

	filters := []*ec2.Filter{
		{
			Name: aws.String("instance-id"),
			Values: []*string{
				aws.String(instanceID),
			},
		},
	}
	input := ec2.DescribeInstancesInput{Filters: filters}
	result, err := ec2svc.DescribeInstances(&input)
	if err != nil {
		log.Errorf("DescribeInstances got error: %v", err)
		return "", err
	}
	log.Info(result.Reservations[0].Instances[0].PrivateDnsName)
	// var ec2DescribeResult map[string]interface{}
	// json.Unmarshal([]byte(result.String()), &ec2DescribeResult)
	// fmt.Println(ec2DescribeResult)
	fmt.Println(result.String())
	if len(result.Reservations) < 1 {
		return "", errors.New("instance not found")
	}
	nodeName = *result.Reservations[0].Instances[0].NetworkInterfaces[0].PrivateDnsName
	log.Info(nodeName)

	return nodeName, err
}

func getClusterNameFromTags(ec2svc *ec2.EC2, instanceID string) (string, error) {
	log.Info("start getClusterNameFromTags")
	var clusterName string
	filters := []*ec2.Filter{
		{
			Name: aws.String("resource-id"),
			Values: []*string{
				aws.String(instanceID),
			},
		},
		{
			Name: aws.String("value"),
			Values: []*string{
				aws.String("owned"),
			},
		},
	}
	input := ec2.DescribeTagsInput{Filters: filters}
	result, err := ec2svc.DescribeTags(&input)
	if err != nil {
		log.Errorf("DescribeTags got err:%v", err)
		return clusterName, err
	}
	for _, k := range result.Tags {
		log.Infof("ec2 tag key=%v value=owned", *k.Key)
		if strings.HasPrefix(*k.Key, "kubernetes.io/cluster/") {
			clusterName = strings.TrimPrefix(*k.Key, "kubernetes.io/cluster/")
			log.Infof("Got cluster name: %v", clusterName)
			return clusterName, err
		}
	}
	log.Info(result)
	err = errors.New("no clusterName found")
	return clusterName, err
}

func taintNode(id string) {
	log.Infof("start processing taintNode on %v", id)
	// nodeName, err := ec2Info("i-0efa14160939310ef")
	var clusterName string
	ec2svc = initEc2svc()
	isInitEc2svc = true
	nodeName, err := ec2Info(ec2svc, id)
	if err != nil {
		log.Errorf("ec2Info got error: %v", err)
		return
	}
	clusterName, err = getClusterNameFromTags(ec2svc, id)
	if err != nil {
		log.Errorf("getClusterNameFromTags got error: %v", err)
		return
	}

	h, err := NewEksHandler(clusterName)
	if err != nil {
		log.Errorf("error creating new EKS handler: %v", err)
		return
	}
	log.Infof("clusterName=%v", h.ClusterName)
	h.TaintNode(&v1.Taint{
		Key:    "SpotTerminating",
		Value:  "true",
		Effect: v1.TaintEffectNoExecute,
		// Effect:    v1.TaintEffectPreferNoSchedule,
		// TimeAdded: &now,
	}, nodeName)
}
