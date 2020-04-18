package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// RefreshJWTToken ...
func RefreshJWTToken(ctx *fasthttp.RequestCtx) {
	login, err := acmeserverless.UnmarshalLoginResponse(string(ctx.Request.Body()))
	if err != nil {
		ErrorHandler(ctx, "RefreshJWTToken", "UnmarshalLoginResponse", err)
		return
	}

	valid, id, _, err := ValidateToken(login.RefreshToken)

	if !valid || id == "" {
		res := acmeserverless.VerifyTokenResponse{
			Message: "Invalid Key. User Not Authorized",
			Status:  http.StatusForbidden,
		}
		payload, err := res.Marshal()
		if err != nil {
			ErrorHandler(ctx, "RefreshJWTToken", "Marshal", err)
			return
		}

		ctx.SetStatusCode(http.StatusOK)
		ctx.Write(payload)

		return
	}

	newToken, err := GenerateAccessToken("eric", id)
	if err != nil {
		ErrorHandler(ctx, "RefreshJWTToken", "GenerateAccessToken", err)
		return
	}

	res := acmeserverless.LoginResponse{
		AccessToken:  newToken,
		RefreshToken: login.RefreshToken,
		Status:       http.StatusOK,
	}

	payload, err := res.Marshal()
	if err != nil {
		ErrorHandler(ctx, "RefreshJWTToken", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
