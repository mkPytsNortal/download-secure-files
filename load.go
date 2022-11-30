package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type SecureFile struct {
	ID                int64  `json:"id"`
	Name              string `json:"name"`
	Checksum          string `json:"checksum"`
	ChecksumAlgorithm string `json:"checksum_algorithm"`
}

func GetApiURL() string {
	if os.Getenv("CI_API_V4_URL") == "" {
		return "https://gitlab.com/api/v4"
	} else {
		return os.Getenv("CI_API_V4_URL")
	}
}

func GetProjectId() string {
	if os.Getenv("CI_PROJECT_ID") == "" {
		panic("Project ID missing")
	} else {
		return url.QueryEscape(os.Getenv("CI_PROJECT_ID"))
	}
}

func GetDownloadPath() string {
	if os.Getenv("SECURE_FILES_DOWNLOAD_PATH") == "" {
		return ".secure_files"
	} else {
		return os.Getenv("SECURE_FILES_DOWNLOAD_PATH")
	}
}

func AuthHeader() http.Header {
	if os.Getenv("CI_JOB_TOKEN") == "" {
		return http.Header{
			"PRIVATE-TOKEN": {os.Getenv("PRIVATE_TOKEN")},
		}
	} else {
		return http.Header{
			"JOB-TOKEN": {os.Getenv("CI_JOB_TOKEN")},
		}
	}
}

func GetFileList() []SecureFile {
	var url = GetApiURL() + "/projects/" + GetProjectId() + "/secure_files"

	client := http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header = AuthHeader()
	res, err := client.Do(req)
	if err != nil {
		fmt.Printf("Request DO error: %s", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)

	var secureFiles []SecureFile

	json.Unmarshal([]byte(body), &secureFiles)

	return secureFiles
}

func DownloadFile(secureFile SecureFile) (err error) {
	url := GetApiURL() + "/projects/" + GetProjectId() + "/secure_files/" + strconv.FormatInt(secureFile.ID, 10) + "/download"
	filePath := GetDownloadPath() + "/" + secureFile.Name
	cwd, _ := os.Getwd()
	downloadLocation := cwd + "/" + GetDownloadPath()
	fullFilePath := cwd + "/" + GetDownloadPath() + "/" + secureFile.Name

	if err := os.MkdirAll(downloadLocation, os.ModePerm); err != nil {
		return err
	}

	// Create the file
	out, err := os.Create(fullFilePath)
	if err != nil {
		return err
	}
	defer out.Close()

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header = AuthHeader()

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	// Check server response
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", res.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, res.Body)
	if err != nil {
		return err
	}

	fmt.Printf("%s downloaded to %s\n", secureFile.Name, filePath)

	return nil
}

func DownloadFiles() *http.Response {
	files := GetFileList()

	fmt.Printf("Loading Secure Files...\n")

	for _, file := range files {
		DownloadFile(file)
	}

	return nil
}

func main() {
	godotenv.Load()
	DownloadFiles()
}
