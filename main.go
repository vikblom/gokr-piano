package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gokrazy/gokrazy"
)

func hello(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir("/dev")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "read dir: %s", err)
		return
	}
	for _, e := range entries {
		fmt.Fprintln(w, e.Name())
	}
}

func main() {
	// ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	// defer cancel()

	// Wait until network interfaces have a chance to work.
	gokrazy.WaitForClock()

	err := http.ListenAndServe(":8080", http.HandlerFunc(hello))
	if err != nil {
		log.Fatal(err)
	}
}
