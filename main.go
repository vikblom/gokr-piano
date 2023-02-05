package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gokrazy/gokrazy"
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

func hello(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(sysUSB)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "read dir: %s", err)
		return
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "usb") || strings.Contains(e.Name(), ":") {
			continue
		}

		vendorFile := path.Join(sysUSB, e.Name(), "idVendor")
		vendor := readFile(vendorFile)

		productFile := path.Join(sysUSB, e.Name(), "idProduct")
		product := readFile(productFile)

		manufacturerFile := path.Join(sysUSB, e.Name(), "manufacturer")
		manufacturer := readFile(manufacturerFile)

		prodFile := path.Join(sysUSB, e.Name(), "product")
		prod := readFile(prodFile)

		fmt.Fprintf(w, "%s vendor:  %s\n", e.Name(), vendor)
		fmt.Fprintf(w, "%s product: %s\n", e.Name(), product)
		fmt.Fprintf(w, "%s mfct:    %s\n", e.Name(), manufacturer)
		fmt.Fprintf(w, "%s name:    %s\n", e.Name(), prod)
		fmt.Fprintf(w, "\n")
	}
}

func main() {

	// Wait until network interfaces have a chance to work.
	gokrazy.WaitForClock()

	server := &http.Server{Addr: ":8080", Handler: http.HandlerFunc(hello)}
	err := server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Println(err)
	}
	os.Exit(0) // Let gokrazy restart this service.
}
