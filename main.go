package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

//go:embed info.py
var infoScript []byte

// important: this cannot be named build.py
//
//go:embed generate.py
var generateScript []byte

func infoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", "0")
		fmt.Fprintf(w, "")
		return
	} else if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fileName := strings.TrimPrefix(r.URL.Path, "/info/")
	if fileName == "" {
		http.Error(w, "File name is required", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "info-temp-*")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create temp dir: %v", err), http.StatusInternalServerError)
		return
	}
	outputFile := tmpDir + "/output.json"
	inputFileNameCopy := tmpDir + "/" + fileName
	infoScriptFile := tmpDir + "/info.py"

	inputFile, err := os.Open(fileName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open file: %v", err), http.StatusInternalServerError)
		return
	}
	defer inputFile.Close()

	inputFileCopy, err := os.Create(inputFileNameCopy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open file: %v", err), http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(inputFileCopy, inputFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to copy file: %v", err), http.StatusInternalServerError)
		return
	}
	inputFileCopy.Close()

	err = os.WriteFile(infoScriptFile, infoScript, 0755)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write info script file: %v", err), http.StatusInternalServerError)
		return
	}
	cmd := exec.Command("FreeCADCmd", "-c", infoScriptFile, "--pass", inputFileNameCopy, outputFile)
	fmt.Printf("Running: %v", cmd.Args)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = cmd.Run()
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	fmt.Printf("stdout: %s", stdout.String())
	fmt.Printf("stderr: %s", stderr.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute command: %v", err), http.StatusInternalServerError)
		return
	}
	outputBytes, err := os.ReadFile(outputFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read output file: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(outputBytes)
}

type ListResponse struct {
	Files []string `json:"files"`
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Length", "0")
		fmt.Fprintf(w, "")
		return
	} else if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dir := "."
	files, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "Unable to read directory", http.StatusInternalServerError)
		return
	}
	var txtFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".FCStd") {
			txtFiles = append(txtFiles, file.Name())
		}
	}

	listResp := ListResponse{Files: txtFiles}

	respJSON, err := json.Marshal(listResp)
	if err != nil {
		http.Error(w, "Unable to marshal JSON", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

func buildHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	fileName := strings.TrimPrefix(r.URL.Path, "/build/")
	if fileName == "" {
		http.Error(w, "File name is required", http.StatusBadRequest)
		return
	}
	tmpDir, err := os.MkdirTemp("", "build-temp-*")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create temp dir: %v", err), http.StatusInternalServerError)
		return
	}

	inputFileNameCopy := tmpDir + "/input.FCStd"
	outputFile := tmpDir + "/output.stl"
	generateScriptFile := tmpDir + "/generate.py"
	configFileName := tmpDir + "/config.json"

	inputFile, err := os.Open(fileName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open file: %v", err), http.StatusInternalServerError)
		return
	}
	defer inputFile.Close()

	inputFileCopy, err := os.Create(inputFileNameCopy)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open file: %v", err), http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(inputFileCopy, inputFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to copy file: %v", err), http.StatusInternalServerError)
		return
	}
	inputFileCopy.Close()

	configFile, err := os.Create(configFileName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create config file: %v", err), http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(configFile, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write config file: %v", err), http.StatusInternalServerError)
		configFile.Close()
		return
	}
	configFile.Close()

	err = os.WriteFile(generateScriptFile, generateScript, 0755)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to write build script file: %v", err), http.StatusInternalServerError)
		return
	}
	cmd := exec.Command("FreeCADCmd", "-c", generateScriptFile, "--pass", inputFileNameCopy, configFileName, outputFile)
	fmt.Printf("Running: %v", cmd.Args)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = cmd.Run()
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	fmt.Printf("stdout: %s", stdout.String())
	fmt.Printf("stderr: %s", stderr.String())
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute command: %v", err), http.StatusInternalServerError)
		return
	}
	outputBytes, err := os.ReadFile(outputFile)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read output file: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(outputBytes)))
	w.Header().Set("Content-Disposition", "attachment; filename=\"output.stl\"")
	w.Write(outputBytes)

}

func main() {
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/info/", infoHandler)
	http.HandleFunc("/build/", buildHandler)
	fmt.Println("Starting server at port 8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}
