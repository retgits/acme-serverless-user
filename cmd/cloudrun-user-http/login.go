package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// Login ...
func Login(ctx *fasthttp.RequestCtx) {
	usr, err := acmeserverless.UnmarshalUser(string(ctx.Request.Body()))
	if err != nil {
		ErrorHandler(ctx, "Login", "UnmarshalUser", err)
		return
	}

	usr, err = db.FindUser(usr.Username)
	if err != nil {
		ErrorHandler(ctx, "Login", "FindUser", err)
		return
	}

	accessToken, refreshToken, err := GenerateTokenPair(usr.Username, usr.ID)
	if err != nil {
		ErrorHandler(ctx, "Login", "GenerateTokenPair", err)
		return
	}

	res := acmeserverless.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Status:       http.StatusOK,
	}

	payload, err := res.Marshal()
	if err != nil {
		ErrorHandler(ctx, "Login", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
