HANDLER ?= main
# modify this as your own S3 temp bucket. Make sure your locak IAM user have read/write access
S3BUCKET	?= pahud-tmp-ap-northeast-1
LAMBDA_REGION ?= ap-northeast-1
LAMBDA_FUNC_NAME ?= eks-lambda-drainer
STACKNAME	?= eks-lambda-drainer
WORKDIR = $(CURDIR:$(GOPATH)%=/go%)
ifeq ($(WORKDIR),$(CURDIR))
	WORKDIR = /tmp
endif


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



all: build pack package

# dep:
# 	@echo "Checking dependencies..."
# 	@dep ensure

# build:
# 	@echo "Building..."
# 	@GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags='-w -s' -o $(HANDLER)

# devbuild:
# 	@echo "Building..."
# 	@GOOS=$(GOOSDEV) GOARCH=$(GOARCH) go build -ldflags='-w -s' -o $(HANDLER)

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
	--template-file ./packaged.yaml --stack-name "$(LAMBDA_FUNC_NAME)" --capabilities CAPABILITY_IAM
	# print the cloudformation stack outputs
	aws --region $(LAMBDA_REGION) cloudformation describe-stacks --stack-name "$(LAMBDA_FUNC_NAME)" --query 'Stacks[0].Outputs'

.PHONY: sam-destroy
sam-destroy:
	# destroy the stack now
	@aws --region $(LAMBDA_REGION) cloudformation delete-stack --stack-name "$(LAMBDA_FUNC_NAME)"
	# deleting the stack. check your cloudformaion console to make sure stack is completely deleted

pack:
	@echo "Packing binary..."
	@zip $(PACKAGE).zip $(HANDLER)

clean:
	@echo "Cleaning up..."
	@rm -rf $(HANDLER) $(PACKAGE).zip

package:
	@echo "sam packaging..."
	@aws cloudformation package --template-file sam.yaml --s3-bucket $(S3BUCKET) --output-template-file sam-packaged.yaml

deploy:
	@echo "sam deploying..."
	@aws cloudformation deploy --template-file sam-packaged.yaml --stack-name $(STACKNAME) --capabilities CAPABILITY_IAM

world: all deploy

.PHONY: all dep build pack clean
