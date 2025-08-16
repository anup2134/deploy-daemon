package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type StatusResponse struct {
	Status string `json:"status"`
}

type BuildRequest struct {
	Event string `json:"event"`
	RepoName string `json:"repoName"`
	CommitHash string `json:"commit"`
}

func main(){
	f, err := os.Open(".env")
	
	if err != nil {
		fmt.Println("Error opening file:", err)
        return
    }
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan(){
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
            continue
        }

		parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
		key := strings.TrimSpace(parts[0])
        val := strings.TrimSpace(parts[1])
		fmt.Printf("%s=%s\n", key, val)
		os.Setenv(key, val)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/build", func(w http.ResponseWriter, r *http.Request) {
		deleteRepoClone := true
		if r.Method != http.MethodPost{
			http.Error(w,"Method not allowed",http.StatusMethodNotAllowed)
			return
		}
		
		var buildRequest BuildRequest
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&buildRequest); err != nil {
			http.Error(w, "Invalid JSON: " + err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Println(buildRequest)

		defer r.Body.Close()
		defer func(){
			if !deleteRepoClone {
				return
			}
			cmd := exec.Command("rm", "-rf", "repo-clone")
			err := cmd.Run()
			if err != nil {
				fmt.Printf("Deleting repo-clone directory failed: %v", err)
			}
		}()

		authHeader := r.Header.Get("Authorization")
		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			http.Error(w, "Invalid auth token", http.StatusUnauthorized)
			return
		}

		authToken := authHeader[7:]
		if authToken != os.Getenv("secretKey") {
			http.Error(w, "Incorrect auth token", http.StatusUnauthorized)
			return
		}
		fmt.Println("Request authorized")

		_, err := os.Stat("repo-clone")
		
		if err == nil {
			deleteRepoClone = false
			http.Error(w, "Service in use", http.StatusLocked)
			return
		}
		fmt.Println(err.Error())
		fmt.Println("service available")
		
		cmd := exec.Command("git", "clone", "https://github.com/" + buildRequest.RepoName, "repo-clone")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("Git clone failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Println("Github repo cloned successfully:", buildRequest.RepoName)
		
		imageName := buildRequest.RepoName + buildRequest.CommitHash
		imageName = strings.ToLower(imageName)
		
		cmd = exec.Command("docker", "build", "-t", imageName, "repo-clone")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			fmt.Printf("docker build failed: %v", err)
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
	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		fmt.Println(err)
	}
}

