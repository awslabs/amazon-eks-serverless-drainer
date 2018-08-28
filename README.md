# eks-lambda-drainer
**eks-lambda-drainer** is an Amazon EKS node drainer with AWS Lambda. If you provision spot instances or spotfleet in your Amazon EKS nodegroup, you can listen to the spot termination signal from **CloudWatch Events** 120 seconds in prior to the final termination process. By configuring this Lambda function as the CloudWatch Event target, **eks-lambda-drainer**  will perform the taint-based eviction on the terminating node and all the pods without relative toleration will be evicted and rescheduled to another node - your workload will get very minimal impact on the spot instance termination.



# In Actions

![](images/01.png)



![](images/02.png)



![](images/03.png)





# TODO

- [ ] package the Lambda function in [AWS SAM](https://docs.aws.amazon.com/lambda/latest/dg/serverless_app.html) format
- [ ] publish to [AWS Serverless Applicaton Repository](https://aws.amazon.com/tw/serverless/serverlessrepo/)
- [ ] add more samples