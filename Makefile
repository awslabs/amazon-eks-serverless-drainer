# modify this as your own S3 temp bucket. Make sure your locak IAM user have read/write access
S3BUCKET	?= pahud-tmp-ap-northeast-1
LAMBDA_REGION ?= ap-northeast-1
LAMBDA_FUNC_NAME ?= eks-lambda-drainer
STACKNAME	?= eks-lambda-drainer
# Your Amazon EKS cluster name
CLUSTER_NAME ?= eksdemo

	
.PHONY: all
all: func-prep sam-package sam-deploy

.PHONY: func-prep
func-prep:
	@rm -rf ./func.d; mkdir ./func.d
	@cp main.sh bootstrap libs.sh func.d/ && chmod +x ./func.d/bootstrap ./func.d/main.sh

.PHONY: sam-package
sam-package:
	@docker run -ti \
	-v $(PWD):/home/samcli/workdir \
	-v $(HOME)/.aws:/home/samcli/.aws \
	-w /home/samcli/workdir \
	-e AWS_DEFAULT_REGION=$(LAMBDA_REGION) \
	pahud/aws-sam-cli:latest sam package --template-file sam.yaml --s3-bucket $(S3BUCKET) --output-template-file packaged.yaml

.PHONY: sam-deploy	
sam-deploy:
	@aws --region $(LAMBDA_REGION)  cloudformation deploy \
	--parameter-overrides FunctionName=$(LAMBDA_FUNC_NAME) ClusterName=$(CLUSTER_NAME) \
	--template-file ./packaged.yaml --stack-name "$(LAMBDA_FUNC_NAME)" --capabilities CAPABILITY_IAM
	# print the cloudformation stack outputs
	@aws --region $(LAMBDA_REGION) cloudformation describe-stacks --stack-name "$(LAMBDA_FUNC_NAME)" --query 'Stacks[0].Outputs'

.PHONY: sam-logs-tail
sam-logs-tail:
	@docker run -ti \
	-v $(PWD):/home/samcli/workdir \
	-v $(HOME)/.aws:/home/samcli/.aws \
	-w /home/samcli/workdir \
	-e AWS_DEFAULT_REGION=$(LAMBDA_REGION) \
	pahud/aws-sam-cli:latest sam logs --name $(LAMBDA_FUNC_NAME) --tail

.PHONY: sam-destroy
sam-destroy:
	# destroy the stack now
	@aws --region $(LAMBDA_REGION) cloudformation delete-stack --stack-name "$(LAMBDA_FUNC_NAME)"
	# deleting the stack. check your cloudformaion console to make sure stack is completely deleted

