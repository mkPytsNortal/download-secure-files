package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/joho/godotenv"
)

type Config struct {
	url              string
	projectID        string
	downloadPath     string
	authHeader       http.Header
	workingDirectory string
}

type SecureFile struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Checksum          string `json:"checksum"`
	ChecksumAlgorithm string `json:"checksum_algorithm"`
}

func getEnvWithDefault(envVar, defaultValue string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}

	return value
}

func authHeader() http.Header {
	if os.Getenv("CI_JOB_TOKEN") == "" {
		return http.Header{"PRIVATE-TOKEN": {os.Getenv("PRIVATE_TOKEN")}}
	}

	return http.Header{"JOB-TOKEN": {os.Getenv("CI_JOB_TOKEN")}}
}

func writeFile(fileData []byte, path string) (bool, error) {
	// Create the file
	out, err := os.Create(path)
	if err != nil {
		return false, err
	}
	defer out.Close()

	// Writer the body to file
	_, err = io.Copy(out, bytes.NewReader(fileData))
	if err != nil {
		return false, err
	}

	return true, nil
}

func downloadFile(config Config, secureFile SecureFile) (err error) {
	url := config.url + "/projects/" + config.projectID + "/secure_files/" + strconv.FormatInt(secureFile.ID, 10) + "/download"

	filePath, err := securejoin.SecureJoin(config.downloadPath, secureFile.Name)
	if err != nil {
		return err
	}

	fileLocation, err := securejoin.SecureJoin(config.workingDirectory, filePath)
	if err != nil {
		return err
	}

	body, err := httpGet(config, url)
	if err != nil {
		return err
	}

	_, err = writeFile(body, fileLocation)
	if err != nil {
		return err
	}

	if err := verifyChecksum(secureFile, fileLocation); err != nil {
		return err
	}

	fmt.Printf("%s downloaded to %s\n", secureFile.Name, filePath)

	return nil
}

func verifyChecksum(file SecureFile, localFilePath string) error {
	body, err := os.ReadFile(localFilePath)
	if err != nil {
		return err
	}

	sum := sha256.Sum256(body)

	if hex.EncodeToString(sum[:]) == file.Checksum {
		return nil
	}

	return fmt.Errorf("validating checksum for %s", file.Name)
}

func createDownloadLocation(config Config) error {
	downloadLocation, err := securejoin.SecureJoin(config.workingDirectory, config.downloadPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(downloadLocation, os.ModePerm); err != nil {
		return err
	}

	return nil
}

func httpGet(config Config, url string) (body []byte, err error) {
	// initialize client
	client := http.Client{}

	// setup new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// add authoriztion header
	req.Header = config.authHeader

	// make request
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// check response
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %s", res.Status)
	}

	body, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func getFileList(config Config) ([]SecureFile, error) {
	var url = config.url + "/projects/" + config.projectID + "/secure_files"

	body, err := httpGet(config, url)
	if err != nil {
		return nil, err
	}

	var secureFiles []SecureFile

	json.Unmarshal([]byte(body), &secureFiles)

	return secureFiles, nil
}

func downloadFiles(config Config) error {
	files, err := getFileList(config)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	if err := createDownloadLocation(config); err != nil {
		return err
	}

	fmt.Printf("Loading Secure Files...\n")

	for _, file := range files {
		if err := downloadFile(config, file); err != nil {
			return err
		}
	}

	return nil
}

func loadConfig() (Config, error) {
	apiV4Url := getEnvWithDefault("CI_API_V4_URL", "https://gitlab.com/api/v4")
	projectID := url.QueryEscape(os.Getenv("CI_PROJECT_ID"))
	downloadPath := getEnvWithDefault("SECURE_FILES_DOWNLOAD_PATH", ".secure_files")
	authHeader := authHeader()

	workingDirectory, err := os.Getwd()
	if err != nil {
		return Config{}, err
	}

	return Config{
		apiV4Url,
		projectID,
		downloadPath,
		authHeader,
		workingDirectory,
	}, nil
}

func main() {
	godotenv.Load()

	config, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	err = downloadFiles(config)
	if err != nil {
		log.Fatal(err)
	}
}
