package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/pulumi/pulumi-aws/sdk/go/aws/apigateway"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/dynamodb"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/go/aws/lambda"
	"github.com/pulumi/pulumi/sdk/go/pulumi"
	"github.com/pulumi/pulumi/sdk/go/pulumi/config"
	"github.com/retgits/pulumi-helpers/builder"
	gw "github.com/retgits/pulumi-helpers/gateway"
	"github.com/retgits/pulumi-helpers/sampolicies"
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

// GenericConfig contains the key-value pairs for the configuration of AWS in this stack
type GenericConfig struct {
	// The AWS region used
	Region string

	// The DSN used to connect to Sentry
	SentryDSN string `json:"sentrydsn"`

	// The AWS AccountID to use
	AccountID string `json:"accountid"`
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get the region
		region, found := ctx.GetConfig("aws:region")
		if !found {
			return fmt.Errorf("region not found")
		}

		// Read the configuration data from Pulumi.<stack>.yaml
		conf := config.New(ctx, "awsconfig")

		// Create a new Tags object with the data from the configuration
		var tags Tags
		conf.RequireObject("tags", &tags)

		// Create a new GenericConfig object with the data from the configuration
		var genericConfig GenericConfig
		conf.RequireObject("generic", &genericConfig)
		genericConfig.Region = region

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
			buildFactory := builder.NewFactory().WithFolder(fnFolder)
			buildFactory.MustBuild()
			buildFactory.MustZip()
		}

		// Create a factory to get policies from
		iamFactory := sampolicies.NewFactory().WithAccountID(genericConfig.AccountID).WithPartition("aws").WithRegion(genericConfig.Region)

		// Lookup the DynamoDB table
		dynamoTable, err := dynamodb.LookupTable(ctx, &dynamodb.LookupTableArgs{
			Name: fmt.Sprintf("%s-acmeserverless-dynamodb", ctx.Stack()),
		})

		// dynamoPolicy is a policy template, derived from AWS SAM, to allow apps
		// to connect to and execute command on Amazon DynamoDB
		iamFactory.ClearPolicies()
		iamFactory.AddDynamoDBCrudPolicy(dynamoTable.Name)
		dynamoPolicy, err := iamFactory.GetPolicyStatement()
		if err != nil {
			return err
		}

		roles := make(map[string]*iam.Role)

		// Create a new IAM role for each Lambda function
		for _, function := range functions {
			// Give the role the ability to run on AWS Lambda
			roleArgs := &iam.RoleArgs{
				AssumeRolePolicy: pulumi.String(sampolicies.AssumeRoleLambda()),
				Description:      pulumi.String(fmt.Sprintf("Role for the User Service (%s) of the ACME Serverless Fitness Shop", function)),
				Tags:             pulumi.Map(tagMap),
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
				Policy: pulumi.String(dynamoPolicy),
			})
			if err != nil {
				return err
			}

			ctx.Export(fmt.Sprintf("%s-role::Arn", function), role.Arn)
			roles[function] = role
		}

		// All functions will have the same environment variables, with the exception
		// of the function name
		variables := make(map[string]pulumi.StringInput)
		variables["REGION"] = pulumi.String(genericConfig.Region)
		variables["SENTRY_DSN"] = pulumi.String(genericConfig.SentryDSN)
		variables["VERSION"] = tags.Version
		variables["STAGE"] = pulumi.String(ctx.Stack())
		variables["TABLE"] = pulumi.String(dynamoTable.Name)

		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-user-all", ctx.Stack()))
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

		userAllFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-all", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-all::Arn", userAllFunction.Arn)

		// Create the Get function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-user-get", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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

		userGetFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-get", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-get::Arn", userGetFunction.Arn)

		// Create the Login function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-user-login", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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

		userLoginFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-login", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-login::Arn", userLoginFunction.Arn)

		// Create the RefreshToken function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-user-refreshtoken", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-user-refreshtoken", ctx.Stack()))

		userRefreshTokenFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-refreshtoken", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-refreshtoken::Arn", userRefreshTokenFunction.Arn)

		// Create the Register function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-user-register", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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

		userRegisterFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-register", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-register::Arn", userRegisterFunction.Arn)

		// Create the VerifyToken function
		variables["FUNCTION_NAME"] = pulumi.String(fmt.Sprintf("%s-lambda-user-verifytoken", ctx.Stack()))
		environment = lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap(variables),
		}

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

		userVerifyTokenFunction, err := lambda.NewFunction(ctx, fmt.Sprintf("%s-lambda-user-verifytoken", ctx.Stack()), functionArgs)
		if err != nil {
			return err
		}

		ctx.Export("lambda-user-verifytoken::Arn", userVerifyTokenFunction.Arn)

		// Create the API Gateway Policy
		iamFactory.ClearPolicies()
		iamFactory.AddAssumeRoleLambda()
		iamFactory.AddExecuteAPI()
		policies, err := iamFactory.GetPolicyStatement()
		if err != nil {
			return err
		}

		// Read the OpenAPI specification
		bytes, err := ioutil.ReadFile("../api/openapi.json")
		if err != nil {
			return err
		}

		// Create an API Gateway
		gateway, err := apigateway.NewRestApi(ctx, "UserService", &apigateway.RestApiArgs{
			Name:        pulumi.String("UserService"),
			Description: pulumi.String("ACME Serverless Fitness Shop - User"),
			Tags:        pulumi.Map(tagMap),
			Policy:      pulumi.String(policies),
			Body:        pulumi.StringPtr(string(bytes)),
		})
		if err != nil {
			return err
		}

		gatewayURL := gateway.ID().ToStringOutput().ApplyString(func(id string) string {
			resource := gw.MustGetGatewayResource(ctx, id, "/users")

			_, err = apigateway.NewIntegration(ctx, "AllUsersAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("GET"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   userAllFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "AllUsersAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  userAllFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/users", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/users/{id}")

			_, err = apigateway.NewIntegration(ctx, "GetUserAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("GET"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   userGetFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "GetUserAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  userGetFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/GET/users/*", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/login")

			_, err = apigateway.NewIntegration(ctx, "LoginUserAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("POST"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   userLoginFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "LoginUserAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  userLoginFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/login", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/refresh-token")

			_, err = apigateway.NewIntegration(ctx, "RefreshTokenAPIAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("POST"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   userRefreshTokenFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "RefreshTokenAPIAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  userRefreshTokenFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/refresh-token", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/register")

			_, err = apigateway.NewIntegration(ctx, "RegisterUserAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("POST"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   userRegisterFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "RegisterUserAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  userRegisterFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/register", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			resource = gw.MustGetGatewayResource(ctx, id, "/verify-token")

			_, err = apigateway.NewIntegration(ctx, "VerifyTokenAPIIntegration", &apigateway.IntegrationArgs{
				HttpMethod:            pulumi.String("POST"),
				IntegrationHttpMethod: pulumi.String("POST"),
				ResourceId:            pulumi.String(resource.Id),
				RestApi:               gateway.ID(),
				Type:                  pulumi.String("AWS_PROXY"),
				Uri:                   userVerifyTokenFunction.InvokeArn,
			})
			if err != nil {
				fmt.Println(err)
			}

			_, err = lambda.NewPermission(ctx, "VerifyTokenAPIPermission", &lambda.PermissionArgs{
				Action:    pulumi.String("lambda:InvokeFunction"),
				Function:  userVerifyTokenFunction.Name,
				Principal: pulumi.String("apigateway.amazonaws.com"),
				SourceArn: pulumi.Sprintf("arn:aws:execute-api:%s:%s:%s/*/POST/verify-token", genericConfig.Region, genericConfig.AccountID, gateway.ID()),
			})
			if err != nil {
				fmt.Println(err)
			}

			return fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/prod/", id, genericConfig.Region)
		})

		ctx.Export("Gateway::URL", gatewayURL)

		return nil
	})
}
