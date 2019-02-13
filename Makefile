# customize the value below
# your temporary private S3 bucket 
S3BUCKET	?= pahud-tmp-ap-northeast-1
LAMBDA_REGION ?= ap-northeast-1
LAMBDA_FUNC_NAME ?= eks-lambda-drainer
STACKNAME	?= eks-lambda-drainer
# Your Amazon EKS cluster name
CLUSTER_NAME ?= eksdemo


build:
ifeq ($(GOOS),darwin)
	@docker run -ti --rm -v $(shell pwd):/go/src/myapp.github.com -w /go/src/myapp.github.com  golang:1.10 /bin/sh -c "make build-darwin"
else
	@docker run -ti --rm -v $(shell pwd):/go/src/myapp.github.com -w /go/src/myapp.github.com  golang:1.10 /bin/sh -c "make build-linux"
endif
	

.PHONY: build-linux	
build-linux:
	@go get -u github.com/golang/dep/cmd/dep
	@[ ! -f ./Gopkg.toml ] && dep init || true
	@dep ensure
	@GOOS=linux GOARCH=amd64 go build -o ./func.d/main *.go 

build-darwin:
	@go get -u github.com/golang/dep/cmd/dep
	@[ ! -f ./Gopkg.toml ] && dep init || true
	@dep ensure
	@GOOS=darwin GOARCH=amd64 go build -o ./func.d/main *.go 


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

# pack:
# 	@echo "Packing binary..."
# 	@zip $(PACKAGE).zip $(HANDLER)

# clean:
# 	@echo "Cleaning up..."
# 	@rm -rf $(HANDLER) $(PACKAGE).zip

# package:
# 	@echo "sam packaging..."
# 	@aws cloudformation package --template-file sam.yaml --s3-bucket $(S3BUCKET) --output-template-file sam-packaged.yaml

# deploy:
# 	@echo "sam deploying..."
# 	@aws cloudformation deploy --template-file sam-packaged.yaml --stack-name $(STACKNAME) --capabilities CAPABILITY_IAM

# world: all deploy
