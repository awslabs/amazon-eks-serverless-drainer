[![Build Status](https://travis-ci.org/pahud/eks-lambda-drainer.svg?branch=master)](https://travis-ci.org/pahud/eks-lambda-drainer)
[![Go Report Card](https://goreportcard.com/badge/github.com/pahud/eks-lambda-drainer)](https://goreportcard.com/report/github.com/pahud/eks-lambda-drainer)

# eks-lambda-drainer

**eks-lambda-drainer** is an Amazon EKS node drainer with AWS Lambda. If you provision spot instances or spotfleet in your Amazon EKS nodegroup, you can listen to the spot termination signal from **CloudWatch Events** 120 seconds in prior to the final termination process. By configuring this Lambda function as the CloudWatch Event target, **eks-lambda-drainer**  will perform the taint-based eviction on the terminating node and all the pods without relative toleration will be evicted and rescheduled to another node - your workload will get very minimal impact on the spot instance termination.

## Implementations

- `golang` implementation(golang branch)
- `bash` implementation(current master branch)

Previously this project has a native `golang` implementation with `go-client`(see `golang` [branch](https://github.com/pahud/eks-lambda-drainer/tree/golang)).
However, as AWS announced AWS Lambda layer and Lambda custom runtime and thanks to `pahud/lambda-layer-kubectl`([link](https://github.com/pahud/lambda-layer-kubectl)) project,
it's very easy to implement this with a few lines of bash script in Lambda([tweet](https://twitter.com/pahudnet/status/1095369690556162049)) whilst the code size being reduced from `11MB` to just `2.4KB`.
So the current implementation would be simply `bash`. We believe this will eliminate the complexity to develop similar projects in the future.


# Prepare your Layer

Follow the [instructions](https://github.com/pahud/lambda-layer-kubectl) to build and publish your `lambda-layer-kubectl` Lambda Layer.
Copy the layer ARN(e.g. `arn:aws:lambda:ap-northeast-1:${AWS::AccountId}:layer:layer-eks-kubectl-layer-stack:2`)

# Edit the sam.yaml

2. edit `Makefile` and update **S3BUCKET** variable:

modify this to your private S3 bucket you have read/write access to
```
S3BUCKET ?= pahud-temp-ap-northeast-1
```

set the AWS region you are deploying to
```
LAMBDA_REGION ?= ap-northeast-1
```


4. type `make build` to build the binary and store in `./func.d/main`
```
$ make build

```
(This may take 1-2 minutes. Be patient.)

```
$ ls -alR ./func.d/                                                                                                                                
./func.d/:
total 49104
drwxr-xr-x 2 root     root         4096 Feb 13 05:55 .
drwxrwxr-x 6 ec2-user ec2-user     4096 Feb 13 05:55 ..
-rwxr-xr-x 1 root     root     50270699 Feb 13 05:55 main
```


4. package and deploy with `SAM`
```
$ make sam-package sam-deploy
```
(`SAM` will deplly a cloudformation stack for you in your `{LAMBDA_REGION}` and register cloudwatch events as the Lambda source event)
```
Uploading to 032ea7f22f8fedab0d016ed22f2bdea4  11594869 / 11594869.0  (100.00%)
Successfully packaged artifacts and wrote output template to file packaged.yaml.
Execute the following command to deploy the packaged template
aws cloudformation deploy --template-file /home/samcli/workdir/packaged.yaml --stack-name <YOUR STACK NAME>

Waiting for changeset to be created..
Waiting for stack create/update to complete
Successfully created/updated stack - eks-lambda-drainer
# print the cloudformation stack outputs
aws --region ap-northeast-1 cloudformation describe-stacks --stack-name "eks-lambda-drainer" --query 'Stacks[0].Outputs'
[
    {
        "Description": "Lambda function Arn", 
        "OutputKey": "Func", 
        "OutputValue": "arn:aws:lambda:ap-northeast-1:xxxxxxxx:function:eks-lambda-drainer-Func-1P5RHJ50KEVND"
    }, 
    {
        "Description": "Lambda function IAM role Arn", 
        "OutputKey": "FuncIamRole", 
        "OutputValue": "arn:aws:iam::xxxxxxxx:role/eks-lambda-drainer-FuncRole-TCZVVLEG1HKD"
    }
]
```




# Add Lambda Role into ConfigMap

Read Amazon EKS [document](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html) about how to add an IAM Role to the `aws-auth` ConfigMap. 

Edit the `aws-auth` ConfigMap by 

```
kubectl edit -n kube-system configmap/aws-auth
```

And insert `rolearn`, `groups` and `username` into the `mapRoles`, make sure the groups contain `system:masters`

![](images/04.png)



You can get the `rolearn` from the output tab of cloudformation console.

![](images/05.png)



# Autoscaling Group LifecycleHook Support

By creating your nodegroup with this cloudformation template, your autoscaling group will create a `LifecycleHook` and 
`CloudWatch Event` will receive this event and eventually invoke **eks-lambda-drainer** to drain the pods from the terminating node. 
Your node will first enter the **Terminating:Wait** state and after a pre-defined graceful period of time(default: 10 seconds), 
**eks-lambda-drainer** will put **CompleteLifecycleAction** back to the hook and Autoscaling group therefore move on to the 
**Terminaing:Proceed** phase to execute the real termination process. The Pods in the terminating node will be rescheduled to other node(s) 
just in a few seconds. Your service will have almost zero impact.




# In Actions

![](images/01.png)



try `kubectl describe` this node and see the `Taints` on it

![](images/03.png)


# clean up

```
$ make sam-destroy
```
(this will destroy the cloudformation stack and all resources in it)


# TODO

- [x] package the Lambda function in [AWS SAM](https://docs.aws.amazon.com/lambda/latest/dg/serverless_app.html) format
- [ ] publish to [AWS Serverless Applicaton Repository](https://aws.amazon.com/tw/serverless/serverlessrepo/)
- [x] ASG/LifeCycle integration [#2](https://github.com/pahud/eks-lambda-drainer/issues/2)
- [ ] add more samples



# FAQ



### Do I need to specify the Amazon EKS cluster name in Lambda?

**ANS:** No, **eks-lambda-drainer** will determine the Amazon EKS cluster name from the EC2 Tags(key=*kubernetes.io/cluster/{CLUSTER_NAME}* with value=*owned*). You just need single Lambda function to handle all spot instances from different nodegroups from different Amazon EKS clusters.

