package main

import (
	"net/http"

	"github.com/gofrs/uuid"
	acmeserverless "github.com/retgits/acme-serverless"
	"github.com/valyala/fasthttp"
)

// RegisterUser ...
func RegisterUser(ctx *fasthttp.RequestCtx) {
	// Update the user with an ID
	usr, err := acmeserverless.UnmarshalUser(string(ctx.Request.Body()))
	if err != nil {
		ErrorHandler(ctx, "RegisterUser", "UnmarshalUser", err)
		return
	}
	usr.ID = uuid.Must(uuid.NewV4()).String()

	err = db.AddUser(usr)
	if err != nil {
		ErrorHandler(ctx, "RegisterUser", "AddUser", err)
		return
	}

	status := acmeserverless.RegisterUserResponse{
		Message:    "User created successfully!",
		ResourceID: usr.ID,
		Status:     http.StatusCreated,
	}

	payload, err := status.Marshal()
	if err != nil {
		ErrorHandler(ctx, "RegisterUser", "Marshal", err)
		return
	}

	ctx.SetStatusCode(http.StatusOK)
	ctx.Write(payload)
}
