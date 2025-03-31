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
	"strings"

	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase"
	//"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/security"
)

func generateLedConfig(led *core.Record) string {
	name := led.GetString("name")
	return fmt.Sprintf(`
config led 'led_%s'
        option name '%s'
        option sysfs '%s'
        option trigger '%s'
`, strings.ToLower(name), name, led.GetString("led_name"), led.GetString("trigger"))
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

func generateLedsConfig(leds []*core.Record) string {
	output := ""
	for _, led := range leds {
		fmt.Println(led)
		output += generateLedConfig(led)
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
func getDeviceRecord(app *pocketbase.PocketBase, key string) (*core.Record, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("Key not valid")
	}
	pbID, err := hexToPocketBaseID(key)
	if err != nil {
		return nil, fmt.Errorf("Key not hex")
	}
	record, err := app.FindRecordById("devices", pbID)
	if !security.Equal(record.GetString("key"), key) {
		return nil, fmt.Errorf("Key not allowed")
	}
	return record, nil
}

func generateDeviceConfig(app *pocketbase.PocketBase, record *core.Record) ([]byte, string, error) {
	configfiles := map[string]string{}
	leds := record.Get("leds").([]string)
	fmt.Println(leds)
	ledrecords, err := app.FindRecordsByIds("leds", leds)
	if err != nil {
		return nil, "", err
	}
	ledconfigs := generateLedsConfig(ledrecords)
	if len(ledconfigs) > 0 {
		configfiles["/etc/config/system"] = ledconfigs
	}
	fmt.Println(ledconfigs)

	blob, checksum, err := createConfigTar(configfiles)
	if err != nil {
	}
	return blob, checksum, err
}

func main() {
	app := pocketbase.New()

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		se.Router.POST("/controller/register/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
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
		se.Router.POST("/controller/report-status/{device_uuid}/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			data := struct {
				// unexported to prevent binding
				somethingPrivate string

				Status string `form:"status"`
				Key    string `form:"key"`
			}{}
			if err := e.BindBody(&data); err != nil {
				return e.BadRequestError("Missing fields", err)
			}
			record, err := getDeviceRecord(app, data.Key)
			if err != nil {
				return e.ForbiddenError("Not allowed", err)
			}
			record.Set("config_status", data.Status)
			err = app.Save(record)
			if err != nil {
				return e.InternalServerError("Status update failed", err)
			}
			response := fmt.Sprintf("report-result: success\ncurrent-status: %s\n", data.Status)
			fmt.Println(response)

			return e.Blob(200, "text/plain", []byte(response))
		})

		se.Router.GET("/controller/checksum/{device_uuid}/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			key := e.Request.URL.Query().Get("key")
			record, err := getDeviceRecord(app, key)
			if err != nil {
				return e.ForbiddenError("Not allowed", err)
			}

			fmt.Println("OK")
			_, response, err := generateDeviceConfig(app, record)
			if err != nil {
				return e.InternalServerError("Internal error", err)
			}

			return e.Blob(200, "text/plain", []byte(response))
		})

		se.Router.GET("/controller/download-config/{device_uuid}/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			key := e.Request.URL.Query().Get("key")
			record, err := getDeviceRecord(app, key)
			if err != nil {
				return e.ForbiddenError("Not allowed", err)
			}

			fmt.Println("OK")
			response, _, err := generateDeviceConfig(app, record)
			if err != nil {
				return e.InternalServerError("Internal error", err)
			}

			return e.Blob(200, "application/octet-stream", []byte(response))
		})

		se.Router.POST("/api/v1/monitoring/device/", func(e *core.RequestEvent) error {
			e.Response.Header().Set("X-Openwisp-Controller", "true")
			key := e.Request.URL.Query().Get("key")
			_, err := getDeviceRecord(app, key)
			if err != nil {
				return e.ForbiddenError("Not allowed", err)
			}
			time := e.Request.URL.Query().Get("time")
			//current := e.Request.URL.Query().Get("current")
			fmt.Println("monitoring %s", time)
			return e.Blob(200, "text/plain", []byte(""))
		})
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
