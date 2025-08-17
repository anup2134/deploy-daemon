package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func main(){
	GetSecret()
	mux := http.NewServeMux()

	mux.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost{
			http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
			return
		}
		
		defer r.Body.Close()
		buildRequest, err := DecodeRequestJSON(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		fmt.Printf("Payload:\n{\n\tevent: %s,\n\trepoName: %s,\n\tcommit: %s\n}", buildRequest.Event, buildRequest.RepoName, buildRequest.CommitHash)

		authHeader := r.Header.Get("Authorization")
		err = Authorization(authHeader)
		if err != nil {
			http.Error(w, err.Error(),http.StatusUnauthorized)
		}
		fmt.Println("Request authorized")

		_, err = os.Stat("repo-clone")
		if err == nil {
			http.Error(w, "Service in use", http.StatusLocked)
			return
		}
		fmt.Println("service available")
		
		err = CloneRepo(buildRequest.Owner + "/" + buildRequest.RepoName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println("Github repo cloned successfully:", buildRequest.RepoName)
		defer DeleteRepo()
		
		imageName := "owner:" + buildRequest.Owner + "::name:" + buildRequest.RepoName + "::sha:" + buildRequest.CommitHash
		imageName = strings.ToLower(imageName)
		err = BuildDockerImage(imageName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println("Image built successfully:", imageName)
		
		resp := StatusResponse{
			Status: "success",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(resp)
	})

	fmt.Println("Running server on port 8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Println(err)
	}
}

