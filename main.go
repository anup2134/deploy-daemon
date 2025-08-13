package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const SECRET_KEY string = "your-secret-key"

type StatusResponse struct {
	Status string `json:"status"`
}

type BuildRequest struct {
	RepoName string `json:"repo_name"`
}

func main(){
	mux := http.NewServeMux()

	mux.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost{
			http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
			return
		}
		
		var buildrequest BuildRequest
		if err := json.NewDecoder(r.Body).Decode(&buildrequest); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		auth_header := r.Header.Get("Authorization")
		if len(auth_header) < 7 || auth_header[:7] != "Bearer " {
			http.Error(w, "Invalid auth token", http.StatusUnauthorized)
			return
		}

		auth_token := auth_header[7:]
		if auth_token != SECRET_KEY {
			http.Error(w, "Incorrect auth token", http.StatusUnauthorized)
			return
		}

		resp := StatusResponse{
			Status: "starting build",
		}		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(resp)
	})

	fmt.Println("Server listening on port 8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Println(err)
	}
}

