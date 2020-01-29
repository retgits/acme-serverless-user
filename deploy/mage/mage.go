//+build mage

package main

import (
	"fmt"
	"os"
	"path"

	"github.com/magefile/mage/sh"
)

var apps = []string{"lambda-catalog-add"}

// Deps resolves and downloads dependencies to the current development module and then builds and installs them.
// Deps will rely on the Go environment variable GOPROXY (go env GOPROXY) to determine from where to obtain the
// sources for the build.
func Deps() error {
	goProxy, _ := sh.Output("go", "env", "GOPROXY")
	fmt.Printf("Getting Go modules from %s", goProxy)
	return sh.Run("go", "get", "./...")
}

// 'Go test' automates testing the packages named by the import paths. go:test compiles and tests each of the
// packages listed on the command line. If a package test passes, go test prints only the final 'ok' summary
// line.
func Test() error {
	return sh.RunV("go", "test", "-cover", "./...")
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
	if !appExists(config.App) {
		return fmt.Errorf("app %s does not exist as valid target", config.App)
	}

	env := make(map[string]string)
	env["GOOS"] = "linux"
	env["GOARCH"] = "amd64"

	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	return sh.RunWith(env, "go", "build", "-o", path.Join(workingDir, "..", "..", "cmd", config.App, "bin", config.App), path.Join(workingDir, "..", "..", "cmd", config.App))
}

// Clean removes object files from package source directories.
func Clean() error {
	workingDir, err := os.Getwd()
	if err != nil {
		return err
	}

	return sh.Rm(path.Join(workingDir, "..", "..", "cmd", config.App, "bin"))
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

	if err := sh.RunV("aws", "cloudformation", "package", "--template-file", path.Join(workingDir, "..", "..", "cmd", config.App, "template.yaml"), "--output-template-file", path.Join(workingDir, "..", "..", "cmd", config.App, "packages.yaml"), "--s3-bucket", config.AWS.Bucket); err != nil {
		return err
	}

	if err := sh.RunV("aws", "cloudformation", "deploy", "--template-file", path.Join(workingDir, "..", "..", "cmd", config.App, "packages.yaml"), "--stack-name", fmt.Sprintf("%s-%s", config.Project, config.Stage), "--capabilities", "CAPABILITY_IAM", "--parameter-overrides", fmt.Sprintf("Version=%s", config.Version), fmt.Sprintf("Author=%s", config.Author), fmt.Sprintf("Team=%s", config.Team)); err != nil {
		return err
	}

	if err := sh.RunV("aws", "cloudformation", "describe-stacks", "--stack-name", fmt.Sprintf("%s-%s", config.Project, config.Stage), "--query", "'Stacks[].Outputs'"); err != nil {
		return err
	}

	return nil
}