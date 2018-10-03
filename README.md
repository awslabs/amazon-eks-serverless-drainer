[![Build Status](https://travis-ci.org/pahud/eks-lambda-drainer.svg?branch=master)](https://travis-ci.org/pahud/eks-lambda-drainer)
[![Go Report Card](https://goreportcard.com/badge/github.com/pahud/eks-lambda-drainer)](https://goreportcard.com/report/github.com/pahud/eks-lambda-drainer)

# eks-lambda-drainer

**eks-lambda-drainer** is an Amazon EKS node drainer with AWS Lambda. If you provision spot instances or spotfleet in your Amazon EKS nodegroup, you can listen to the spot termination signal from **CloudWatch Events** 120 seconds in prior to the final termination process. By configuring this Lambda function as the CloudWatch Event target, **eks-lambda-drainer**  will perform the taint-based eviction on the terminating node and all the pods without relative toleration will be evicted and rescheduled to another node - your workload will get very minimal impact on the spot instance termination.



# Installation

1. `git clone` to check out the repository to local and `cd` to the directory

2. run `dep ensure -v` to install required go packages - you might need to install [go dep](https://golang.github.io/dep/docs/installation.html) first.

3. edit `Makefile` and update **S3TMPBUCKET** variable:

modify this to your private S3 bucket you have read/write access to
```
S3TMPBUCKET ?= pahud-temp
```

4. type `make world` to build, pack, package and deploy to Lambda
```
pahud:~/go/src/eks-lambda-drainer (master) $ make world
Checking dependencies...
Building...
Packing binary...
updating: main (deflated 73%)
sam packaging...
Uploading to a33bb95c227378e21102db1274f5dffd  8423458 / 8423458.0  (100.00%)
Successfully packaged artifacts and wrote output template to file sam-packaged.yaml.
Execute the following command to deploy the packaged template
aws cloudformation deploy --template-file /home/ec2-user/go/src/eks-lambda-drainer/sam-packaged.yaml --stack-name <YOUR STACK NAME>
sam deploying...

Waiting for changeset to be created..
Waiting for stack create/update to complete
Successfully created/updated stack - eks-lambda-drainer
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

By creating your nodegroup with this cloudformation template, your autoscaling group will have a LifecycleHook to a specific SNS topic and eventually invoke **eks-lambda-drainer** to drain the pods from the terminating node. Your node will first enter the **Terminating:Wait** state and after a pre-defined graceful period of time(default: 10 seconds), **eks-lambda-drainer** will put **CompleteLifecycleAction** back to the hook and Autoscaling group therefore move on to the **Terminaing:Proceed** phase to execute the real termination process. The Pods in the terminating node will be rescheduled to other node(s) just in a few seconds. Your service will have almost zero impact.




# In Actions

![](images/01.png)



try `kubectl describe` this node and see the `Taints` on it

![](images/03.png)





# TODO

- [x] package the Lambda function in [AWS SAM](https://docs.aws.amazon.com/lambda/latest/dg/serverless_app.html) format
- [ ] publish to [AWS Serverless Applicaton Repository](https://aws.amazon.com/tw/serverless/serverlessrepo/)
- [x] ASG/LifeCycle integration [#2](https://github.com/pahud/eks-lambda-drainer/issues/2)
- [ ] add more samples



# FAQ



### Do I need to specify the Amazon EKS cluster name in Lambda?

**ANS:** No, **eks-lambda-drainer** will determine the Amazon EKS cluster name from the EC2 Tags(key=*kubernetes.io/cluster/{CLUSTER_NAME}* with value=*owned*). You just need single Lambda function to handle all spot instances from different nodegroups from different Amazon EKS clusters.

