package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// GetAllUsers ...
func GetAllUsers(ctx *fasthttp.RequestCtx) {
	users, err := db.AllUsers()
	if err != nil {
		ErrorHandler(ctx, "GetAllUsers", "AllUsers", err)
		return
	}

	res := acmeserverless.AllUsers{
		Data: users,
	}

	payload, err := res.Marshal()
	if err != nil {
		ErrorHandler(ctx, "GetAllUsers", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
