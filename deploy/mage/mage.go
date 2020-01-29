//+build mage

package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/magefile/mage/sh"
)

var (
	stage    = "dev"
	project  = "serverless-user"
	author   = "retgits"
	team     = "vcs"
	s3bucket = "MyS3Bucket"
)

var lambdas = []string{"lambda-user-all", "lambda-user-get", "lambda-user-login", "lambda-user-login", "lambda-user-refreshtoken", "lambda-user-register", "lambda-user-verifytoken"}

func gitVersion() string {
	v, _ := sh.Output("git", "describe", "--tags", "--always", "--dirty=-dev")
	if len(v) == 0 {
		v = "dev"
	}
	return v
}

// Deps resolves and downloads dependencies to the current development module and then builds and installs them.
// Deps will rely on the Go environment variable GOPROXY (go env GOPROXY) to determine from where to obtain the
// sources for the build.
func Deps() error {
	goProxy, _ := sh.Output("go", "env", "GOPROXY")
	fmt.Printf("Getting Go modules from %s", goProxy)
	return sh.Run("go", "get", "../.././...")
}

// 'Go test' automates testing the packages named by the import paths. go:test compiles and tests each of the
// packages listed on the command line. If a package test passes, go test prints only the final 'ok' summary
// line.
func Test() error {
	return sh.RunV("go", "test", "-cover", "../.././...")
}

// Vuln uses Snyk to test for any known vulnerabilities in go.mod. The command relies on access to the Snyk.io
// vulnerability database, so it cannot be used without Internet access.
func Vuln() error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	return sh.RunV("snyk", "test", fmt.Sprintf("--file=%s", path.Join(workingDir, "..", "..", "go.mod")))
}

// Build compiles the individual commands in the cmd folder, along with their dependencies. All built executables
// are stored in the 'bin' folder. Specifically for deployment to AWS Lambda, GOOS is set to linux and GOARCH is
// set to amd64.
func Build() error {
	return buildLambda()
}

func buildLambda() error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	env := make(map[string]string)
	env["GOOS"] = "linux"
	env["GOARCH"] = "amd64"

	for _, lambda := range lambdas {
		err := sh.RunWith(env, "go", "build", "-o", path.Join(workingDir, "..", "cloudformation", "bin", lambda), path.Join(workingDir, "..", "..", "cmd", lambda))
		if err != nil {
			log.Printf("error building %s: %s", lambda, err.Error())
		}
	}

	return nil
}

// Clean removes object files from package source directories.
func Clean() error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	return sh.Rm(path.Join(workingDir, "..", "cloudformation", "bin"))
}

// Deploy packages, deploys, and returns all outputs of your stack. Packages the local artifacts (local paths) that your
// AWS CloudFormation template references and uploads  local  artifacts to an S3 bucket. The command returns a copy of your
// template, replacing references to local artifacts with the S3 location where the command uploaded the artifacts. Deploys
// the specified AWS CloudFormation template by creating and then executing a change set. The command terminates after AWS
// CloudFormation executes  the change set. Returns the description for the specified stack.
func Deploy() error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	version := gitVersion()

	if err := sh.RunV("aws", "cloudformation", "package", "--template-file", path.Join(workingDir, "..", "cloudformation", "lambda-template.yaml"), "--output-template-file", path.Join(workingDir, "..", "cloudformation", "lambda-packaged.yaml"), "--s3-bucket", s3bucket); err != nil {
		return err
	}

	if err := sh.RunV("aws", "cloudformation", "deploy", "--template-file", path.Join(workingDir, "..", "cloudformation", "lambda-packaged.yaml"), "--stack-name", fmt.Sprintf("%s-%s", project, stage), "--capabilities", "CAPABILITY_IAM", "--parameter-overrides", fmt.Sprintf("Version=%s", version), fmt.Sprintf("Author=%s", author), fmt.Sprintf("Team=%s", team)); err != nil {
		return err
	}

	if err := sh.RunV("aws", "cloudformation", "describe-stacks", "--stack-name", fmt.Sprintf("%s-%s", project, stage), "--query", "'Stacks[].Outputs'"); err != nil {
		return err
	}

	return nil
}
