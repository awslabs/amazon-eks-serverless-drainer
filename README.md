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



