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
	"path/filepath"
	"strconv"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/joho/godotenv"
)

var Version = "_development"

type Downloader struct {
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

func authHeader() (http.Header, error) {
	if os.Getenv("CI_JOB_TOKEN") != "" {
		return http.Header{"JOB-TOKEN": {os.Getenv("CI_JOB_TOKEN")}}, nil
	} else if os.Getenv("PRIVATE_TOKEN") != "" {
		fmt.Println(os.Getenv("PRIVATE_TOKEN"))
		return http.Header{"PRIVATE-TOKEN": {os.Getenv("PRIVATE_TOKEN")}}, nil
	} else {
		return http.Header{}, fmt.Errorf("Authentication Token Missing")
	}
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

func (downloader Downloader) downloadFile(secureFile SecureFile) (err error) {
	url := downloader.url + "/projects/" + downloader.projectID + "/secure_files/" + strconv.FormatInt(secureFile.ID, 10) + "/download"

	filePath, err := securejoin.SecureJoin(downloader.downloadPath, secureFile.Name)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	fileLocation, err := securejoin.SecureJoin(downloader.workingDirectory, filePath)
	if err != nil {
		return err
	}

	body, err := downloader.httpGet(url)
	if err != nil {
		return err
	}

	_, err = writeFile(body, fileLocation)
	if err != nil {
		return err
	}

	if err := secureFile.verifyChecksum(fileLocation); err != nil {
		return err
	}

	fmt.Printf("%s downloaded to %s\n", secureFile.Name, filePath)

	return nil
}

func (file SecureFile) verifyChecksum(localFilePath string) error {
	body, err := os.ReadFile(localFilePath)
	if err != nil {
		return err
	}

	sum := sha256.Sum256(body)

	if hex.EncodeToString(sum[:]) == file.Checksum {
		return nil
	}

	return fmt.Errorf("failure validating checksum for %s", localFilePath)
}

func (downloader Downloader) createDownloadLocation() (string, error) {
	downloadLocation, err := securejoin.SecureJoin(downloader.workingDirectory, downloader.downloadPath)
	if err != nil {
		return downloadLocation, err
	}

	if err := os.MkdirAll(downloadLocation, os.ModePerm); err != nil {
		return downloadLocation, err
	}

	return downloadLocation, nil
}

func (downloader Downloader) httpGet(url string) (body []byte, err error) {
	// initialize client
	client := http.Client{}

	// setup new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// add authoriztion header
	req.Header = downloader.authHeader

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

func (downloader Downloader) getFileList() ([]SecureFile, error) {
	var url = downloader.url + "/projects/" + downloader.projectID + "/secure_files?per_page=100"

	body, err := downloader.httpGet(url)
	if err != nil {
		return nil, err
	}

	var secureFiles []SecureFile

	json.Unmarshal([]byte(body), &secureFiles)

	return secureFiles, nil
}

func (downloader Downloader) downloadFiles() error {
	files, err := downloader.getFileList()
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	downloadLocation, err := downloader.createDownloadLocation()
	if err != nil {
		return err
	}

	fmt.Printf("Downloading Secure Files (v%s) to %s\n", Version, downloadLocation)

	for _, file := range files {
		if err := downloader.downloadFile(file); err != nil {
			return err
		}
	}

	return nil
}

func newDownloaderFromEnv() (Downloader, error) {
	apiV4Url := getEnvWithDefault("CI_API_V4_URL", "https://gitlab.com/api/v4")
	downloadPath := getEnvWithDefault("SECURE_FILES_DOWNLOAD_PATH", ".secure_files")
	authHeader, err := authHeader()
	if err != nil {
		return Downloader{}, err
	}

	projectID := url.QueryEscape(os.Getenv("CI_PROJECT_ID"))
	if projectID == "" {
		return Downloader{}, fmt.Errorf("Project ID missing")
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return Downloader{}, err
	}

	return Downloader{
		apiV4Url,
		projectID,
		downloadPath,
		authHeader,
		workingDirectory,
	}, nil
}

func main() {
	godotenv.Load()

	config, err := newDownloaderFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	err = config.downloadFiles()
	if err != nil {
		log.Fatal(err)
	}
}
