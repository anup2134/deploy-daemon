package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)	

type StatusResponse struct {
	Status string `json:"status"`
}
type BuildRequest struct {
	Owner string `json:"owner"`
	Event string `json:"event"`
	RepoName string `json:"repoName"`
	CommitHash string `json:"commit"`
}

	
func GetSecret(){
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
}	


func DeleteRepo(){
	cmd := exec.Command("rm", "-rf", "repo-clone")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Deleting repo-clone directory failed: %v", err)
	}	
}	

func Authorization(authHeader string) (error) {
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return errors.New("invalid auth header")
	}	

	authToken := authHeader[7:]
	if authToken != os.Getenv("secretKey") {
		return errors.New("incorrect auth token")
	}	
	return nil
}	


func CloneRepo(repoName string) error{
	cmd := exec.Command("git", "clone", "https://github.com/" + repoName, "repo-clone")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
		
	if err := cmd.Run(); err != nil {
		fmt.Printf("Git clone failed: %v", err)
		DeleteRepo()
		return err
	}	
	return nil
}	


func BuildDockerImage(imageName string) error {	
	cmd := exec.Command("docker", "build", "-t", imageName, "repo-clone")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("docker build failed: %v", err)
		return err
	}	
	return nil
}	


func DecodeRequestJSON(requestBody io.ReadCloser) (BuildRequest, error){
	var buildRequest BuildRequest
	dec := json.NewDecoder(requestBody)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&buildRequest); err != nil {
		return BuildRequest {RepoName: "", Event: "", CommitHash: ""}, errors.New("Invalid JSON: " + err.Error())
	}
	if buildRequest.Event == "" || buildRequest.RepoName == "" || buildRequest.CommitHash == "" {
		return BuildRequest {RepoName: "", Event: "", CommitHash: ""}, errors.New("missing required fields")
	}

	return buildRequest, nil
}