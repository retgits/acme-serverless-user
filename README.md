# User

> A user service, because what is a shop without users to buy our awesome red pants?

The User service is part of the [ACME Fitness Serverless Shop](https://github.com/vmwarecloudadvocacy/acme_fitness_demo). The goal of this specific service is to register and authenticate users using JWT tokens.

## Prerequisites

* [Go (at least Go 1.12)](https://golang.org/dl/)
* [An AWS Account](https://portal.aws.amazon.com/billing/signup)
* The _vuln_ targets for Make and Mage rely on the [Snyk](http://snyk.io/) CLI

## Eventing Options

The catalog service has Lambdas triggered by [Amazon API Gateway](https://aws.amazon.com/api-gateway/)

## Data Stores

The order service supports the following data stores:

* [Amazon DynamoDB](https://aws.amazon.com/dynamodb/). The table can be created using the makefile in [create-dynamodb](./cmd/create-dynamodb).

## Using Amazon API Gateway

### Prerequisites for Amazon API Gateway

* [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-install.html) installed and configured

### Build and deploy for Amazon API Gateway

Clone this repository

```bash
git clone https://github.com/retgits/acme-serverless-user
cd acme-serverless-user
```

Get the Go Module dependencies

```bash
go get ./...
```

Switch directories to any of the Lambda folders

```bash
cd ./cmd/lambda-catalog-<name>
```

Use make to deploy

```bash
make build
make deploy
```

### Testing Amazon API Gateway

After the deployment you'll see the URL to which you can send the below mentioned API requests

## API

### `GET /users`

Returns the list of all users

```bash
curl --request GET \
  --url http://localhost:8081/users
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
  --url http://localhost:8081/users/5c61ed848d891bd9e8016899
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
  --url http://localhost:8081/login \
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
  --url http://localhost:8081/refresh-token \
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
  --url http://localhost:8081/verify-token \
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
  --url http://localhost:8081/register \
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