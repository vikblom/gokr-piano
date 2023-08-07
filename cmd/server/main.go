package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/nikolaydubina/calendarheatmap/charts"
	piano "github.com/vikblom/gokr-piano"
)

const index string = `
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <meta name="description" content="Visualize piano practice from this year.">
    <title>Piano practice</title>
    <link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Open+Sans">
    <style>
      body {
          font-family: 'Open Sans', serif;
      }
    </style>
  </head>
  <body>
    <h1 class="text-center">Piano Practice</h1>
    <div id="header">
      <style>
        img {
            width: 90%;
            height: auto;
            margin: 5%;
        }
      </style>
      <img class="img" src="/chart.png">
    </div>
  </body>
</html>
`

func handleIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, index)
}

func handleChart(repo *piano.Repo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		counts, err := repo.Sessions(r.Context())
		if err != nil {
			log.Printf("Sessions: %s", err)
		}

		// TODO: Query within current year.
		// now := time.Now()
		// from := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		// to := time.Date(now.Year(), 12, 31, 0, 0, 0, 0, now.Location())

		cfg := piano.DefaultConfig
		cfg.Counts = counts
		err = charts.WriteHeatmap(cfg, w)
		if err != nil {
			log.Printf("WriteHeatmap: %v", err)
		}
	}
}

func runMain() error {
	repo, err := piano.NewDB()
	if err != nil {
		log.Fatalf("Connect to DB: %s", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/chart.png", handleChart(repo))
	return http.ListenAndServe(":"+port, nil)
}

func main() {
	err := runMain()
	if err != nil {
		log.Fatal(err)
	}
}
