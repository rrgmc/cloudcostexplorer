package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/rrgmc/cloudcostexplorer/cmd/cloudcostexplorer/ui"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	http.HandleFunc("/", handlerHome(config))
	for key, value := range config {
		if value.Disabled {
			continue
		}

		cloud, err := CreateCloud(ctx, value)
		if err != nil {
			return fmt.Errorf("failed to create cloud for %s: %w", key, err)
		}
		http.Handle(fmt.Sprintf("/costexplorer/%s", url.PathEscape(key)), handlerCostExplorer(key, cloud))
	}

	fmt.Printf("http server listening at http://localhost:3335\n")
	return http.ListenAndServe(":3335", nil)
}

func handlerHome(config Config) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := ui.NewHTTPOutput(w)
		for key, value := range config {
			if value.Disabled {
				continue
			}
			out.Writef(`<a href="/costexplorer/%s">Cost explorer (%s)</a><br/>`, key, url.PathEscape(key))
		}
	})
}
