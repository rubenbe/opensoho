package main

import (
	"fmt"
	"log"
	//"net/http"

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
				return e.BadRequestError("Missing fields", err)
			}
			if data.Secret != "blah" {
				return e.BadRequestError("Registration failed!", "unrecognized secret")
			}
			if data.Backend != "netjsonconfig.OpenWrt" {
				return e.BadRequestError("Registration failed!", "wrong backend")
			}
			fmt.Print("Hello")
			collection, err := app.FindCollectionByNameOrId("devices")
			if err != nil {
				return e.BadRequestError("Registration failed!", err)
			}
			record := core.NewRecord(collection)
			record.Set("backend", data.Backend)
			record.Set("key", data.Key)
			record.Set("name", data.Name)
			record.Set("hardware_id", data.HardwareId)
			record.Set("mac_address", data.MacAddress)
			record.Set("tags", data.Tags)
			record.Set("model", data.Model)
			record.Set("os", data.Os)
			record.Set("system", data.System)
			record.Set("last_ip_address", e.RealIP())
			record.Set("health_status", "unknown")
			record.Set("enabled", "true")
			err = app.Save(record)
			if err != nil {
				return e.BadRequestError("Registration failed!", err)
			}
			isNew := 1
			response := fmt.Sprintf(`registration-result: %s
uuid: %s
key: %s
hostname: %s
is-new: %d
`, "success", "9f9d293c-be7c-4352-841c-240410f7e3c9", data.Key, data.Name, isNew)

			return e.Blob(201, "text/plain", []byte(response))
		})
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
