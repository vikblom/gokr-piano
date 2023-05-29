package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/gokrazy/gokrazy"

	// libsql (Turso) DB driver.
	_ "github.com/libsql/libsql-client-go/libsql"
)

// https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-bus-usb
const sysUSB = "/sys/bus/usb/devices"

// readFile content, w/o whitespace, or "" if not possible.
func readFile(p string) string {
	bs, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(bs))
}

type usbDevice struct {
	VendorID     string `json:"vendor-id,omitempty"`
	ProductID    string `json:"product-id,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	Product      string `json:"product,omitempty"`
}

func readUSBs() ([]usbDevice, error) {
	entries, err := os.ReadDir(sysUSB)
	if err != nil {
		return nil, fmt.Errorf("read dir: %s", err)
	}

	var usbs []usbDevice
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "usb") || strings.Contains(e.Name(), ":") {
			continue
		}
		usb := usbDevice{
			VendorID:     readFile(path.Join(sysUSB, e.Name(), "idVendor")),
			ProductID:    readFile(path.Join(sysUSB, e.Name(), "idProduct")),
			Manufacturer: readFile(path.Join(sysUSB, e.Name(), "manufacturer")),
			Product:      readFile(path.Join(sysUSB, e.Name(), "product")),
		}
		usbs = append(usbs, usb)

	}
	return usbs, nil
}

func handleGetDevices(w http.ResponseWriter, r *http.Request) {
	entries, err := readUSBs()
	if err != nil {
		http.Error(w, fmt.Errorf("read usb devices: %s", err).Error(), http.StatusInternalServerError)
		return
	}

	bs, err := json.MarshalIndent(entries, "", "    ")
	if err != nil {
		http.Error(w, fmt.Errorf("json marshal: %s", err).Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(bs)
}

func main() {
	// Wait until network interfaces have a chance to work.
	gokrazy.WaitForClock()

	// Database configuration from deployment.
	scheme := os.Getenv("DB_SCHEME")
	host := os.Getenv("DB_HOST")
	token := os.Getenv("DB_TOKEN")
	u := url.URL{
		Scheme:   scheme,
		Host:     host,
		RawQuery: url.Values{"jwt": {token}}.Encode(),
	}

	db, err := sql.Open("libsql", u.String())
	if err != nil {
		log.Fatalf("open DB: %s\n", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := db.QueryContext(ctx, `select * from people`)
	if err != nil {
		log.Fatalf("query: %s\n", err)
	}
	for rows.Next() {
		var name string
		var age int
		if err := rows.Scan(&name, &age); err != nil {
			log.Fatalf("row scan: %s\n", err)
		}
		log.Printf("%q %d", name, age)
	}

	server := &http.Server{Addr: ":8080", Handler: http.HandlerFunc(handleGetDevices)}
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Println(err)
	}
	os.Exit(0)
}
