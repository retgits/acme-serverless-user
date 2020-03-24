# User

> A user service, because what is a shop without users to buy our awesome red pants?

The User service is part of the [ACME Fitness Serverless Shop](https://github.com/retgits/acme-serverless). The goal of this specific service is to register and authenticate users using JWT tokens.

## Prerequisites

* [Go (at least Go 1.12)](https://golang.org/dl/)
* [An AWS account](https://portal.aws.amazon.com/billing/signup)
* [A Pulumi account](https://app.pulumi.com/signup)
* [A Sentry.io account](https://sentry.io) if you want to enable tracing and error reporting

## Deploying

To deploy the User Service you'll need a [Pulumi account](https://app.pulumi.com/signup). Once you have your Pulumi account and configured the [Pulumi CLI](https://www.pulumi.com/docs/get-started/aws/install-pulumi/), you can initialize a new stack using the Pulumi templates in the [pulumi](./pulumi) folder.

```bash
cd pulumi
pulumi stack init <your pulumi org>/acmeserverless-user/dev
```

Pulumi is configured using a file called `Pulumi.dev.yaml`. A sample configuration is available in the Pulumi directory. You can rename [`Pulumi.dev.yaml.sample`](./pulumi/Pulumi.dev.yaml.sample) to `Pulumi.dev.yaml` and update the variables accordingly. Alternatively, you can change variables directly in the [main.go](./pulumi/main.go) file in the pulumi directory. The configuration contains:

```yaml
config:
  aws:region: us-west-2 ## The region you want to deploy to
  awsconfig:generic:
    sentrydsn: ## The DSN to connect to Sentry
    accountid: ## Your AWS Account ID
  awsconfig:tags:
    author: retgits ## The author, you...
    feature: acmeserverless
    team: vcs ## The team you're on
    version: 0.2.0 ## The version
```

To create the Pulumi stack, and create the User service, run `pulumi up`.

If you want to keep track of the resources in Pulumi, you can add tags to your stack as well.

```bash
pulumi stack tag set app:name acmeserverless
pulumi stack tag set app:feature acmeserverless-user
pulumi stack tag set app:domain user
```

## API

### `GET /users`

Returns the list of all users

```bash
curl --request GET \
  --url https://<api>.execute-api.us-west-2.amazonaws.com/Prod/users
```

```json
{
"data": [
    {
        "username": "walter",
        "email": "walter@acmefitness.com",
        "firstname": "Walter",
        "lastname": "White",
        "id": "5c61ed848d891bd9e8016898"
    },
    {
        "username": "dwight",
        "email": "dwight@acmefitness.com",
        "firstname": "Dwight",
        "lastname": "Schrute",
        "id": "5c61ed848d891bd9e8016899"
    }
]}
```

### `GET /users/:id`

Returns details about a specific user id

```bash
curl --request GET \
  --url https://<api>.execute-api.us-west-2.amazonaws.com/Prod/users/5c61ed848d891bd9e8016899
```

```json
{
    "data": {
        "username": "dwight",
        "email": "dwight@acmefitness.com",
        "firstname": "Dwight",
        "lastname": "Schrute",
        "id": "5c61ed848d891bd9e8016899"
    },
    "status": 200
}
```

### `POST /login/`

Authenticate and Login user

```bash
curl --request POST \
  --url https://<api>.execute-api.us-west-2.amazonaws.com/Prod/login \
  --header 'content-type: application/json' \
  --data '{ 
    "username": "username",
    "password": "password"
}'
```

The request to login needs to have a username and password

```json
{ 
    "username": "username",
    "password": "password"
}
```

When the login succeeds, an access token is returned

```json
{
    "access_token":    "eyJhbGciOiJIUzI1NiIsImtpZCI6InNpZ25pbl8xIiwidHlwIjoiSldUIn0.eyJVc2VybmFtZSI6ImVyaWMiLCJleHAiOjE1NzA3NjI5NzksInN1YiI6IjVkOTNlMTFjNmY4Zjk4YzlmYjI0ZGU0NiJ9.n70EAaiY6rbH1QzpoUJhx3hER4odW8FuN2wYG1sgH7g",
"refresh_token": "eyJhbGciOiJIUzI1NiIsImtpZCI6InNpZ25pbl8yIiwidHlwIjoiSldUIn0.eyJleHAiOjE1NzA3NjM1NzksInN1YiI6IjVkOTNlMTFjNmY4Zjk4YzlmYjI0ZGU0NiJ9.zwGB1340IVMLjMf_UnFC_rEeNdD131OGPcg_S0ea8DE",
"status": 200
    }
```

The access_token is used to make requests to other services to get data. The refresh_token is used to request new access_token. If both refresh_token and access_token expire, then the user needs to log back in again.

### `POST /refresh-token`

Request new access_token by using the `refresh_token`

```bash
curl --request POST \
  --url https://<api>.execute-api.us-west-2.amazonaws.com/Prod/refresh-token \
  --header 'content-type: application/json' \
  --data '{
    "refresh_token" : "eyJhbGciOiJIUzI1NiIsImtpZCI6InNpZ25pbl8yIiwidHlwIjoiSldUIn0.eyJleHAiOjE1NzA3NjM1NzksInN1YiI6IjVkOTNlMTFjNmY4Zjk4YzlmYjI0ZGU0NiJ9.zwGB1340IVMLjMf_UnFC_rEeNdD131OGPcg_S0ea8DE"
}'
```

The request to the refresh-token service, needs a valid refresh_token

```json
{
    "refresh_token" : "eyJhbGciOiJIUzI1NiIsImtpZCI6InNpZ25pbl8yIiwidHlwIjoiSldUIn0.eyJleHAiOjE1NzA3NjM1NzksInN1YiI6IjVkOTNlMTFjNmY4Zjk4YzlmYjI0ZGU0NiJ9.zwGB1340IVMLjMf_UnFC_rEeNdD131OGPcg_S0ea8DE"
}
```

When the token is valid, a new access_token is returned

```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIsImtpZCI6InNpZ25pbl8xIiwidHlwIjoiSldUIn0.eyJVc2VybmFtZSI6ImVyaWMiLCJleHAiOjE1NzA3NjMyMjksInN1YiI6IjVkOTNlMTFjNmY4Zjk4YzlmYjI0ZGU0NiJ9.wrWsDNor28aWv6huKUHAuVyROGAXqjO5luPfa5K5NQI",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsImtpZCI6InNpZ25pbl8yIiwidHlwIjoiSldUIn0.eyJleHAiOjE1NzA3NjM1NzksInN1YiI6IjVkOTNlMTFjNmY4Zjk4YzlmYjI0ZGU0NiJ9.zwGB1340IVMLjMf_UnFC_rEeNdD131OGPcg_S0ea8DE",
    "status": 200
}
```

### `POST /verify-token`

Verify access_token

```bash
curl --request POST \
  --url https://<api>.execute-api.us-west-2.amazonaws.com/Prod/verify-token \
  --header 'content-type: application/json' \
  --data '{
    "access_token": "eyJhbGciOiJIUzI1NiIsImtpZCI6InNpZ25pbl8xIiwidHlwIjoiSldUIn0.eyJVc2VybmFtZSI6ImVyaWMiLCJleHAiOjE1NzA3NjMyMjksInN1YiI6IjVkOTNlMTFjNmY4Zjk4YzlmYjI0ZGU0NiJ9.wrWsDNor28aWv6huKUHAuVyROGAXqjO5luPfa5K5NQI"
}'
```

The request to verify-token needs a valid access_token

```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIsImtpZCI6InNpZ25pbl8xIiwidHlwIjoiSldUIn0.eyJVc2VybmFtZSI6ImVyaWMiLCJleHAiOjE1NzA3NjMyMjksInN1YiI6IjVkOTNlMTFjNmY4Zjk4YzlmYjI0ZGU0NiJ9.wrWsDNor28aWv6huKUHAuVyROGAXqjO5luPfa5K5NQI"
}
```

If the the JWT is valid and user is authorized, an HTTP/200 message is returned

```json
{
   "message": "Token Valid. User Authorized",
   "status": 200
}
```

If the JWT is not valid (either expired or invalid signature) then the user is NOT authorized and an HTTP/401 message is returned

```json
{
    "message": "Invalid Key. User Not Authorized",
    "status": 401
}
```

### `POST /register`

Register/Create new user

```bash
curl --request POST \
  --url https://<api>.execute-api.us-west-2.amazonaws.com/Prod/register \
  --header 'content-type: application/json' \
  --data '{
    "username":"peterp",
    "password":"vmware1!",
    "firstname":"amazing",
    "lastname":"spiderman",
    "email":"peterp@acmefitness.com"
}'
```

To create a new user, a valid user object needs to be provided

```json
{
    "username":"peterp",
    "password":"vmware1!",
    "firstname":"amazing",
    "lastname":"spiderman",
    "email":"peterp@acmefitness.com"
}
```

When the user is successfully created, an HTTP/201 message is returned

```json
{
    "message": "User created successfully!",
    "resourceId": "5c61ef891d41c8de20281dd2",
    "status": 201
}
```

## Contributing

[Pull requests](https://github.com/retgits/acme-serverless-user/pulls) are welcome. For major changes, please open [an issue](https://github.com/retgits/acme-serverless-user/issues) first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License

See the [LICENSE](./LICENSE) file in the repository