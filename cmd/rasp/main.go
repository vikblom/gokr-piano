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
	"time"

	"github.com/gokrazy/gokrazy"

	// libsql (Turso) DB driver.
	_ "github.com/libsql/libsql-client-go/libsql"
)

const (
	// https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-bus-usb
	sysUSB = "/sys/bus/usb/devices"

	yamahaVendorID = "0499"
	pianoProduct   = "Digital Piano"

	// pollInterval between querying USB devices.
	pollInterval = time.Second
)

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

type Repo struct {
	db *sql.DB
}

func NewDB() (*Repo, error) {
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
		return nil, fmt.Errorf("open DB: %w", err)
	}

	return &Repo{db: db}, nil
}

func (r *Repo) StoreSession(ctx context.Context, at time.Time, length time.Duration) error {
	_, err := r.db.ExecContext(
		ctx,
		`insert into piano_sessions(at, seconds) values (?, ?)`,
		at.Format(time.RFC3339),
		int(length.Seconds()))
	if err != nil {
		log.Fatalf("insert: %s\n", err)
	}
	return nil
}

func HasPiano(devices []usbDevice) bool {
	for _, d := range devices {
		if d.Product == pianoProduct {
			return true
		}
	}
	return false
}

func monitorPiano(ctx context.Context, repo *Repo) error {
	for ctx.Err() == nil {
		devices, err := readUSBs()
		if err != nil {
			log.Printf("read USBs: %s", err)
		}

		if HasPiano(devices) {
			log.Printf("piano connected")
			start := time.Now()

			for HasPiano(devices) {
				time.Sleep(pollInterval)
				devices, err = readUSBs()
				if err != nil {
					log.Printf("read USBs: %s", err)
					break
				}
			}

			log.Printf("piano disconnected")
			repo.StoreSession(ctx, start, time.Since(start))
		}

		time.Sleep(pollInterval)
	}
	return nil
}

func main() {
	// Wait until network interfaces have a chance to work.
	gokrazy.WaitForClock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo, err := NewDB()
	if err != nil {
		log.Fatalf("Connect to DB: %s", err)
	}

	go monitorPiano(ctx, repo)

	server := &http.Server{Addr: ":8080", Handler: http.HandlerFunc(handleGetDevices)}
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Println(err)
	}
	os.Exit(0)
}
