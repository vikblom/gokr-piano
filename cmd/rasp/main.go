package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gokrazy/gokrazy"
	piano "github.com/vikblom/gokr-piano"
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

func HasPiano(devices []usbDevice) bool {
	for _, d := range devices {
		if d.Product == pianoProduct {
			return true
		}
	}
	return false
}

func monitorPiano(ctx context.Context, repo *piano.Repo) error {
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
			err := repo.StoreSession(ctx, start, time.Since(start))
			if err != nil {
				log.Printf("store session failed: %s", err)
			}
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

	repo, err := piano.NewDB()
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
