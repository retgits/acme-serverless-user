# -----------------------------------------------------------------------------
# Description: Makefile
# Author(s): retgits <https://github.com/retgits/>
# Last updated: 2020-01-22
# 
# This software may be modified and distributed under the terms of the
# MIT license. See the LICENSE file for details.
# -----------------------------------------------------------------------------

## The stage to deploy to
stage         	:= dev

## The name of the project
project_name  	:= user-verifytoken

## The author of the project
author          := retgits

## The name of the team
team			:= vcs

## The version of the project, either uses the current commit hash, or will default to "dev"
version       	:= $(strip $(if $(shell git describe --tags --always --dirty="-dev"),$(shell git describe --tags --always --dirty="-dev"),dev))

## The Amazon S3 bucket to upload files to
aws_bucket    	?= $$AWS_S3_BUCKET

# Suppress checking files and all Make output
.PHONY: help build vuln clean deploy destroy
.SILENT: help build vuln clean deploy destroy

# Targets
help: ## Displays the help for each target (this message)
	echo
	echo Usage: make [TARGET]
	echo
	echo Makefile targets
	grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
	echo

build: ## Build the executable for Lambda
	echo
	GOOS=linux GOARCH=amd64 go build -o ./bin/lambda-user-verifytoken .
	echo

vuln: ## Scans the Go.mod file for known vulnerabilities using Snyk.
	echo
	snyk test --file=../../go.mod
	echo

clean: ## Remove all generated files
	echo
	-rm -rf bin
	-rm packaged.yaml
	echo

deploy: ## Deploy the app to AWS Lambda
	echo
	aws cloudformation package --template-file template.yaml --output-template-file packaged.yaml --s3-bucket $(aws_bucket)
	aws cloudformation deploy \
		--template-file packaged.yaml \
		--stack-name $(project_name)-$(stage) \
		--capabilities CAPABILITY_IAM \
		--parameter-overrides Version=$(version) \
		Author=$(author) \
		Team=$(team) \
		Version=$(version)
	aws cloudformation describe-stacks --stack-name $(project_name)-$(stage) --query 'Stacks[].Outputs'
	echo

destroy: ## Deletes the CloudFormation stack and all created resources
	echo
	aws cloudformation delete-stack --stack-name $(project_name)-$(stage)
	echo