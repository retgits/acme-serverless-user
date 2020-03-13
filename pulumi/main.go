package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/pulumi/pulumi-aws/sdk/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/go/pulumi"
	"github.com/pulumi/pulumi/sdk/go/pulumi/config"
)

const (
	// The shell to use
	shell = "sh"

	// The flag for the shell to read commands from a string
	shellFlag = "-c"
)

// Tags are key-value pairs to apply to the resources created by this stack
type Tags struct {
	// Author is the person who created the code, or performed the deployment
	Author pulumi.String

	// Feature is the project that this resource belongs to
	Feature pulumi.String

	// Team is the team that is responsible to manage this resource
	Team pulumi.String

	// Version is the version of the code for this resource
	Version pulumi.String
}

// LambdaConfig contains the key-value pairs for the configuration of AWS Lambda in this stack
type LambdaConfig struct {
	// The DSN used to connect to Sentry
	SentryDSN string `json:"sentrydsn"`

	// The ARN for the DynamoDB table
	DynamoARN string `json:"dynamoarn"`

	// The AWS region used
	Region string `json:"region"`

	// The AWS AccountID used
	AccountID string `json:"accountid"`
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Read the configuration data from Pulumi.<stack>.yaml
		conf := config.New(ctx, "awsconfig")

		// Create a new Tags object with the data from the configuration
		var tags Tags
		conf.RequireObject("tags", &tags)

		// Create a new DynamoConfig object with the data from the configuration
		var lambdaConfig LambdaConfig
		conf.RequireObject("lambda", &lambdaConfig)

		// Create a map[string]pulumi.Input of the tags
		// the first four tags come from the configuration file
		// the last two are derived from this deployment
		tagMap := make(map[string]pulumi.Input)
		tagMap["Author"] = tags.Author
		tagMap["Feature"] = tags.Feature
		tagMap["Team"] = tags.Team
		tagMap["Version"] = tags.Version
		tagMap["ManagedBy"] = pulumi.String("Pulumi")
		tagMap["Stage"] = pulumi.String(ctx.Stack())

		// functions are the functions that need to be deployed
		functions := []string{
			"lambda-user-all",
			"lambda-user-get",
			"lambda-user-login",
			"lambda-user-refreshtoken",
			"lambda-user-register",
			"lambda-user-verifytoken",
		}

		// Compile and zip the AWS Lambda functions
		wd, err := os.Getwd()
		if err != nil {
			return err
		}

		for _, fnName := range functions {
			// Find the working folder
			fnFolder := path.Join(wd, "..", "cmd", fnName)

			// Run go build
			if err := run(fnFolder, "GOOS=linux GOARCH=amd64 go build"); err != nil {
				fmt.Printf("Error building code: %s", err.Error())
				os.Exit(1)
			}

			// Zip up the binary
			if err := run(fnFolder, fmt.Sprintf("zip ./%s.zip ./%s", fnName, fnName)); err != nil {
				fmt.Printf("Error creating zipfile: %s", err.Error())
				os.Exit(1)
			}
		}

		// Create an API Gateway
		gateway, err := apigateway.NewRestApi(ctx, "UserService", &apigateway.RestApiArgs{
			Name:        pulumi.String("UserService"),
			Description: pulumi.String("ACME Serverless Fitness Shop - User"),
			Tags:        pulumi.Map(tagMap),
			Policy:      pulumi.String(`{ "Version": "2012-10-17", "Statement": [ { "Action": "sts:AssumeRole", "Principal": { "Service": "lambda.amazonaws.com" }, "Effect": "Allow", "Sid": "" },{ "Action": "execute-api:Invoke", "Resource":"execute-api:/*", "Principal": "*", "Effect": "Allow", "Sid": "" } ] }`),
		})
		if err != nil {
			return err
		}

		// Create the parent resources in the API Gateway
		usersResource, err := apigateway.NewResource(ctx, "UsersAPIResource", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("users"),
			ParentId: gateway.RootResourceId,
		})
		if err != nil {
			return err
		}

		// dynamoCRUDPolicyString is a policy template, derived from AWS SAM, to allow apps
		// to connect to and execute command on Amazon DynamoDB
		dynamoCRUDPolicyString := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Action": [
						"dynamodb:GetItem",
						"dynamodb:DeleteItem",
						"dynamodb:PutItem",
						"dynamodb:Scan",
						"dynamodb:Query",
						"dynamodb:UpdateItem",
						"dynamodb:BatchWriteItem",
						"dynamodb:BatchGetItem",
						"dynamodb:DescribeTable",
						"dynamodb:ConditionCheckItem"
					],
					"Effect": "Allow",
					"Resource": "%s"
				}
			]
		}`, lambdaConfig.DynamoARN)

		roles := make(map[string]*iam.Role)

		// Create a new IAM role for each Lambda function
		for _, function := range functions {
			// Give the role the ability to run on AWS Lambda
			roleArgs := &iam.RoleArgs{
				AssumeRolePolicy: pulumi.String(`{
					"Version": "2012-10-17",
					"Statement": [
					{
						"Action": "sts:AssumeRole",
						"Principal": {
							"Service": "lambda.amazonaws.com"
						},
						"Effect": "Allow",
						"Sid": ""
					}
					]
				}`),
				Description: pulumi.String(fmt.Sprintf("Role for the User Service (%s) of the ACME Serverless Fitness Shop", function)),
				Tags:        pulumi.Map(tagMap),
			}

			role, err := iam.NewRole(ctx, fmt.Sprintf("ACMEServerlessUserRole-%s", function), roleArgs)
			if err != nil {
				return err
			}

			// Attach the AWSLambdaBasicExecutionRole so the function can create Log groups in CloudWatch
			_, err = iam.NewRolePolicyAttachment(ctx, fmt.Sprintf("AWSLambdaBasicExecutionRole-%s", function), &iam.RolePolicyAttachmentArgs{
				PolicyArn: pulumi.String("arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"),
				Role:      role.Name,
			})
			if err != nil {
				return err
			}

			// Add the DynamoDB policy
			_, err = iam.NewRolePolicy(ctx, fmt.Sprintf("ACMEServerlessUserPolicy-%s", function), &iam.RolePolicyArgs{
				Name:   pulumi.String(fmt.Sprintf("ACMEServerlessUserPolicy-%s", function)),
				Role:   role.Name,
				Policy: pulumi.String(dynamoCRUDPolicyString),
			})
			if err != nil {
				return err
			}

			ctx.Export(fmt.Sprintf("%s-role::Arn", function), role.Arn)
			roles[function] = role
		}

		// All functions will have the same environment variables
		variables := make(map[string]pulumi.StringInput)
		variables["REGION"] = pulumi.String(lambdaConfig.Region)
		variables["SENTRY_DSN"] = pulumi.String(lambdaConfig.SentryDSN)
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-payment", ctx.Stack()))
		variables["VERSION"] = tags.Version
		variables["STAGE"] = pulumi.String(ctx.Stack())
		parts := strings.Split(lambdaConfig.DynamoARN, "/")
		variables["TABLE"] = pulumi.String(parts[1])

		environment := lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

		// Create the All function
		functionArgs := &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to get all users from DynamoDB"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-user-all", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-user-all"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-user-all/lambda-user-all.zip"),
			Role:        roles["lambda-user-all"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-all", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "AllUsersAPIGetMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("GET"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    usersResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, usersResource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "AllUsersAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("GET"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            usersResource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, usersResource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "AllUsersAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/users", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, usersResource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-all::Arn", function.Arn)

		// Create the Get function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to get a user from DynamoDB"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-user-get", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-user-get"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-user-get/lambda-user-get.zip"),
			Role:        roles["lambda-user-get"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-get", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err := apigateway.NewResource(ctx, "GetUserAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("{id}"),
			ParentId: usersResource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "GetUserAPIGetMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("GET"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "GetUserAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("GET"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "GetUserAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/users/*", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-get::Arn", function.Arn)

		// Create the Login function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to login"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-user-login", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-user-login"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-user-login/lambda-user-login.zip"),
			Role:        roles["lambda-user-login"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-login", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "LoginUserAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("login"),
			ParentId: gateway.RootResourceId,
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "LoginUserAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("POST"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "LoginUserAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("POST"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "LoginUserAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/login", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-login::Arn", function.Arn)

		// Create the RefreshToken function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to refresh a JWT token"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-user-refreshtoken", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-user-refreshtoken"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-user-refreshtoken/lambda-user-refreshtoken.zip"),
			Role:        roles["lambda-user-refreshtoken"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-refreshtoken", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "RefreshTokenAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("refresh-token"),
			ParentId: gateway.RootResourceId,
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "RefreshTokenAPIAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("POST"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "RefreshTokenAPIAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("POST"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "RefreshTokenAPIAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/refresh-token", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-refreshtoken::Arn", function.Arn)

		// Create the Register function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to register new users in DynamoDB"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-user-register", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-user-register"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-user-register/lambda-user-register.zip"),
			Role:        roles["lambda-user-register"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-register", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "RegisterUserAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("register"),
			ParentId: gateway.RootResourceId,
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "RegisterUserAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("POST"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "RegisterUserAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("POST"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "RegisterUserAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/register", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-register::Arn", function.Arn)

		// Create the VerifyToken function
		functionArgs = &lambda.FunctionArgs{
			Description: pulumi.String("A Lambda function to verify a JWT token"),
			Runtime:     pulumi.String("go1.x"),
			Name:        pulumi.String(fmt.Sprintf("%s-lambda-user-verifytoken", ctx.Stack())),
			MemorySize:  pulumi.Int(256),
			Timeout:     pulumi.Int(10),
			Handler:     pulumi.String("lambda-user-verifytoken"),
			Environment: environment,
			Code:        pulumi.NewFileArchive("../cmd/lambda-user-verifytoken/lambda-user-verifytoken.zip"),
			Role:        roles["lambda-user-verifytoken"].Arn,
			Tags:        pulumi.Map(tagMap),
		}

		function, err = lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-verifytoken", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		resource, err = apigateway.NewResource(ctx, "VerifyTokenAPI", &apigateway.ResourceArgs{
			RestApi:  gateway.ID(),
			PathPart: pulumi.String("verify-token"),
			ParentId: gateway.RootResourceId,
		}, pulumi.DependsOn([]pulumi.Resource{gateway}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewMethod(ctx, "VerifyTokenAPIPostMethod", &apigateway.MethodArgs{
			HttpMethod:    pulumi.String("POST"),
			Authorization: pulumi.String("NONE"),
			RestApi:       gateway.ID(),
			ResourceId:    resource.ID(),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource}))
		if err != nil {
			return err
		}

		_, err = apigateway.NewIntegration(ctx, "VerifyTokenAPIIntegration", &apigateway.IntegrationArgs{
			HttpMethod:            pulumi.String("POST"),
			IntegrationHttpMethod: pulumi.String("POST"),
			ResourceId:            resource.ID(),
			RestApi:               gateway.ID(),
			Type:                  pulumi.String("AWS_PROXY"),
			Uri:                   function.InvokeArn,
		}, pulumi.DependsOn([]pulumi.Resource{gateway, resource, function}))
		if err != nil {
			return err
		}

		_, err = lambda.NewPermission(ctx, "VerifyTokenAPIPermission", &lambda.PermissionArgs{
			Action:    pulumi.String("lambda:InvokeFunction"),
			Function:  function.Name,
			Principal: pulumi.String("apigateway.amazonaws.com"),
			SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/verify-token", lambdaConfig.Region, lambdaConfig.AccountID, gateway.ID()),
		}, pulumi.DependsOn([]pulumi.Resource{gateway, function}))
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-verifytoken::Arn", function.Arn)

		return nil
	})
}

// run creates a Cmd struct to execute the named program with the given arguments.
// After that, it starts the specified command and waits for it to complete.
func run(folder string, args string) error {
	cmd := exec.Command(shell, shellFlag, args)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = folder
	return cmd.Run()
}
