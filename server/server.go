// This file contains the implementation of the HTTP server that provides the `/tiers`, `/my-llm`, and `/my-llm/reset` endpoints.
package main

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

// counter is a global variable used to keep track of the number of requests made to the `/my-llm` endpoint
var counter uint64

// tiers is a map of tier names to their respective rate limits
var tiers = map[string]int{
	"Bronze":   100,
	"Silver":   200,
	"Gold":     300,
	"Platinum": 400,
}

// expectedUsername and expectedPassword are the expected username and password for basic authentication
const expectedUsername = "eventing"
const expectedPassword = "eventing123"

// tiersHandler handles requests to the `/tiers` endpoint
// This endpoint allows users to view and update the rate limits for different tiers
func tiersHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the request is authorized with the expected username and password
	username, password, ok := r.BasicAuth()
	if !ok || username != expectedUsername || password != expectedPassword {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(tiers)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case http.MethodPost:
		var newTierLimits map[string]int
		err := json.NewDecoder(r.Body).Decode(&newTierLimits)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for tier, newLimit := range newTierLimits {
			if _, ok := tiers[tier]; !ok {
				http.Error(w, "Invalid tier", http.StatusBadRequest)
				return
			}
			tiers[tier] = newLimit
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

}

// myLLMEndpointHandler handles requests to the `/my-llm` endpoint
// This endpoint allows users to make requests and view the number of requests made
func myLLMEndpointHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		username, password, ok := r.BasicAuth()
		if !ok || username != expectedUsername || password != expectedPassword {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		atomic.AddUint64(&counter, 1)
		w.WriteHeader(http.StatusOK)
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]uint64{"counter": atomic.LoadUint64(&counter)})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// myLLMResetEndpointHandler handles requests to the `/my-llm/reset` endpoint
// This endpoint allows users to reset the counter that keeps track of the number of requests made to the `/my-llm` endpoint
func myLLMResetEndpointHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username, password, ok := r.BasicAuth()
		if !ok || username != expectedUsername || password != expectedPassword {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		atomic.StoreUint64(&counter, 0)
		w.WriteHeader(http.StatusOK)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/tiers", tiersHandler)
	http.HandleFunc("/my-llm", myLLMEndpointHandler)
	http.HandleFunc("/my-llm/reset", myLLMResetEndpointHandler)
	http.ListenAndServe(":3054", nil)
}
