package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	//"net/http"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase"
	//"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func generateLedConfig(led *core.Record)(string){
	return fmt.Sprintf(`
config led
        option name '%s'
        option sysfs '%s'
        option trigger '%s'
`, led.GetString("name"), led.GetString("led_name"), led.GetString("trigger"))
}

func createConfigTar(files map[string]string) ([]byte, string, error) {
	var buf bytes.Buffer

	gzWriter := gzip.NewWriter(&buf)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	for filePath, content := range files {
		fileBytes := []byte(content)

		header := &tar.Header{
			Name: filePath,
			Size: int64(len(fileBytes)),
			Mode: 0644,
		}

		// Write header and file content to tar archive
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, "", err
		}
		if _, err := tarWriter.Write(fileBytes); err != nil {
			return nil, "", err
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, "", err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, "", err
	}

	// Compress
	tarGzData := buf.Bytes()

	// Compute MD5 checksum
	md5Checksum := md5.Sum(tarGzData)
	md5Hex := hex.EncodeToString(md5Checksum[:])

	return tarGzData, md5Hex, nil
}

func generateLedsConfig(leds []*core.Record)(string){
	output:=""
	for _, led := range leds{
		fmt.Println(led)
		output+=generateLedConfig(led);
	}

	return output
}

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
	return bigInt.Text(36)[0:15], nil
}

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/controller/register/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
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
			if err := e.BindBody(&data); err != nil {
				return e.BadRequestError("Missing fields", err)
			}
			if data.Secret != "blah" {
				return e.BadRequestError("Registration failed!", "unrecognized secret")
			}
			pbID, err := hexToPocketBaseID(data.Key)
			fmt.Println(pbID)
			record, err := app.FindRecordById("devices", pbID)

			isNew := 1
			var device_uuid string = uuid.New().String()
			fmt.Println(device_uuid)

			if err == nil {
				isNew = 0
				fmt.Print("Hello back")
				device_uuid = record.GetString("uuid")
			} else {
				// Register new device
				if data.Backend != "netjsonconfig.OpenWrt" {
					return e.BadRequestError("Registration failed!", "wrong backend")
				}
				fmt.Print("Hello")
				collection, err := app.FindCollectionByNameOrId("devices")

				if err != nil {
					return e.BadRequestError("Registration failed!", err)
				}
				fmt.Print(pbID)

				record := core.NewRecord(collection)
				record.Set("id", pbID)
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
			}
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
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			return e.Blob(200, "text/plain", []byte(response))
		})

		se.Router.GET("/controller/checksum/{device_uuid}/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			key := e.Request.URL.Query().Get("key")
			if len(key) != 32 {
				return e.ForbiddenError("Not allowed", "Key not valid")
			}
			pbID, err := hexToPocketBaseID(key)
			if err != nil {
				return e.ForbiddenError("Not allowed", "Key not hex")
			}
			record, err := app.FindRecordById("devices", pbID)
			if ! security.Equal(record.GetString("key"), key) {
				return e.ForbiddenError("Not allowed", "Key not allowed")
			}
			fmt.Println("OK")
			configfiles :=map [string]string{}
			leds := record.Get("leds").([]string)
			fmt.Println(leds)
			ledrecords, err := app.FindRecordsByIds("leds", leds)
			if err != nil {
				    return e.InternalServerError("Internal error", err)
			}
			ledconfigs := generateLedsConfig(ledrecords)
			if len(ledconfigs)>0{
				configfiles["/etc/config/system"] = ledconfigs
			}
			fmt.Println(ledconfigs)

			_, checksum, err := createConfigTar(configfiles)
			if err != nil {
				    return e.InternalServerError("Internal error", err)
			}

			response := checksum
			return e.Blob(200, "text/plain", []byte(response))
		})

		se.Router.GET("/controller/download-config/{device_uuid}/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			response := ""
			return e.Blob(200, "text/plain", []byte(response))
		})
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
