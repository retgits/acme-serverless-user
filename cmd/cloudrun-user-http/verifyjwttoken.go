package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// VerifyJWTToken ...
func VerifyJWTToken(ctx *fasthttp.RequestCtx) {
	login, err := acmeserverless.UnmarshalLoginResponse(string(ctx.Request.Body()))
	if err != nil {
		ErrorHandler(ctx, "VerifyJWTToken", "UnmarshalLoginResponse", err)
		return
	}

	valid, _, key, err := ValidateToken(login.AccessToken)

	res := acmeserverless.VerifyTokenResponse{
		Message: "Token Valid. User Authorized",
		Status:  http.StatusOK,
	}

	if !valid || key != "signin_1" {
		res.Message = "Invalid Key. User Not Authorized"
		res.Status = http.StatusForbidden
	}

	payload, err := res.Marshal()
	if err != nil {
		ErrorHandler(ctx, "VerifyJWTToken", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
