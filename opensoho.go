package main

import (
	"log"
	"fmt"
	"net/http"

	"github.com/pocketbase/pocketbase"
	//"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/controller/register/", func(e *core.RequestEvent) error {
			//name := e.Request.PathValue("name")
			//return e.String(http.StatusOK, "Hello "+name)
			data := struct {
				// unexported to prevent binding
				somethingPrivate string

				Backend    string `form:"backend"`
				Key        string `form:"key"`
				Secret     string `form:"secret"`
				Name       string `form:"name"`
				HardwareId string `form:"hardware_id"`
				MacAddress string `form:"mac_address"`
				Tags       string `form:"tags"`
				Model      string `form:"model"`
				Os         string `form:"os"`
				System     string `form:"system"`
			}{}
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			if err := e.BindBody(&data); err != nil {
				return e.BadRequestError("Failed to read request data", err)
			}
			if(data.Secret == "blah"){
				return e.BadRequestError("Not allowed", "")
			}
                        fmt.Print("Hello")
			return e.JSON(http.StatusOK, map[string]bool{"success": true})
		})
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
