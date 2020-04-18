package main

import (
	"net/http"

	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// GetUserDetails ...
func GetUserDetails(ctx *fasthttp.RequestCtx) {
	// Create the key attributes
	userID := ctx.UserValue("id").(string)

	usr, err := db.GetUser(userID)
	if err != nil {
		ErrorHandler(ctx, "GetUserDetails", "GetUser", err)
		return
	}

	res := acmeserverless.UserDetailsResponse{
		User:   usr,
		Status: http.StatusOK,
	}

	payload, err := res.Marshal()
	if err != nil {
		ErrorHandler(ctx, "GetUserDetails", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
