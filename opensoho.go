package main

import (
	"encoding/hex"
	"fmt"
	"log"
		"math/big"
	//"net/http"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase"
	//"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func hexToPocketBaseID(hexStr string) (string, error) {
	// Convert hex string to bytes
	bytes, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	// Ensure the byte array is exactly 16 bytes (PocketBase requirement)
	if len(bytes) != 16 {
		return "", fmt.Errorf("invalid length: keys must be 16 bytes")
	}
	bigInt := new(big.Int).SetBytes(bytes)

	// Convert to Base36
	return bigInt.Text(36), nil
}

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

			pbID, err := hexToPocketBaseID(data.Key)
			if err != nil {
				return e.BadRequestError("Registration failed!", err)
			}
			fmt.Print(pbID)
			device_uuid := uuid.New()

			record := core.NewRecord(collection)
			record.Set("id", pbID[0:15])
			record.Set("backend", data.Backend)
			record.Set("key", data.Key)
			record.Set("name", data.Name)
			record.Set("hardware_id", data.HardwareId)
			record.Set("mac_address", data.MacAddress)
			record.Set("uuid", device_uuid)
			record.Set("tags", data.Tags)
			record.Set("model", data.Model)
			record.Set("os", data.Os)
			record.Set("system", data.System)
			record.Set("last_ip_address", e.RealIP())
			record.Set("health_status", "unknown")
			record.Set("config_status", "applied")
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
`, "success", device_uuid, data.Key, data.Name, isNew)

			return e.Blob(201, "text/plain", []byte(response))
		})
		se.Router.GET("/controller/report-status/{device_uuid}/", func(e *core.RequestEvent) error {
		response := ""
		return e.Blob(200, "text/plain", []byte(response))
		})

		se.Router.GET("/controller/checksum/{device_uuid}/", func(e *core.RequestEvent) error {
		response := ""
		return e.Blob(200, "text/plain", []byte(response))
		})

		se.Router.GET("/controller/download-config/{device_uuid}/", func(e *core.RequestEvent) error {
		response := ""
		return e.Blob(200, "text/plain", []byte(response))
		})
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
