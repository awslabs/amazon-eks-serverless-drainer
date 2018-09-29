

# eks-lambda-drainer

**eks-lambda-drainer** is an Amazon EKS node drainer with AWS Lambda. If you provision spot instances or spotfleet in your Amazon EKS nodegroup, you can listen to the spot termination signal from **CloudWatch Events** 120 seconds in prior to the final termination process. By configuring this Lambda function as the CloudWatch Event target, **eks-lambda-drainer**  will perform the taint-based eviction on the terminating node and all the pods without relative toleration will be evicted and rescheduled to another node - your workload will get very minimal impact on the spot instance termination.



# Installation

Install [SAM CLI](https://github.com/awslabs/aws-sam-cli) and [go dep](https://golang.github.io/dep/docs/installation.html)

1. execute `dep ensure -v`to make sure all packages required can be downloaded to local

2. just type `make` to buiild the `main.zip` for Lambda

3. `sam package` to package the lambda bundle

   ```
   sam package \
     --template-file sam.yaml \
     --output-template-file sam-packaged.yaml \
     --s3-bucket pahud-tmp
   ```

   (change **pahud-tmp** to your temporary S3 bucket name)

4. `sam deploy` to deploy to AWS Lambda 

   ```
   sam deploy \
   > --template-file sam-packaged.yaml \
   > --stack-name eks-lambda-drainer \
   > --capabilities CAPABILITY_IAM
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

By creating your nodegroup with this cloudformation template, your autoscaling group will have a LifecycleHook to a specific SNS topic and eventually invoke **eks-lambda-drainer** to drain the pods from the terminating node. Your node will first enter the **Terminating:Wait** state and after a pre-defined graceful period of time(default: 10 seconds), **eks-lambda-drainer** will put **CompleteLifecycleAction** back to the hook and Autoscaling group then move the **Terminaing:Proceed** phase to execute the real termination. The Pods in the terminating node will be rescheduled just in a few seconds.




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

