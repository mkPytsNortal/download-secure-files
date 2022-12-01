package main

import (
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

	response := HttpGet(url)

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	response.Body.Close()

	var secureFiles []SecureFile

	json.Unmarshal([]byte(body), &secureFiles)

	return secureFiles
}

func HttpGet(url string) *http.Response {
	// initialize client
	client := http.Client{}

	// setup new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}

	// add authoriztion header
	req.Header = AuthHeader()

	// make request
	res, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	// check response
	if res.StatusCode != http.StatusOK {
		log.Fatalf("bad status: %s", res.Status)
	}

	// return response
	return res
}

func WriteFile(fileData io.Reader, path string) bool {
	// Create the file
	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
		return false
	}
	defer out.Close()

	// Writer the body to file
	_, err = io.Copy(out, fileData)
	if err != nil {
		log.Fatal(err)
		return false
	}

	return true
}

func DownloadFile(secureFile SecureFile) (err error) {
	url := GetApiURL() + "/projects/" + GetProjectId() + "/secure_files/" + strconv.FormatInt(secureFile.ID, 10) + "/download"
	cwd, _ := os.Getwd()

	filePath, err := securejoin.SecureJoin(GetDownloadPath(), secureFile.Name)

	fileLocation, err := securejoin.SecureJoin(cwd, filePath)
	if err != nil {
		log.Fatal(err)
	}

	response := HttpGet(url)
	WriteFile(response.Body, fileLocation)
	response.Body.Close()

	if VerifyChecksum(secureFile, fileLocation) {
		fmt.Printf("%s downloaded to %s\n", secureFile.Name, filePath)
	} else {
		os.Remove(fileLocation)
		return fmt.Errorf("Checksum validation failed for %s", secureFile.Name)
	}

	return nil
}

func VerifyChecksum(file SecureFile, localFilePath string) bool {
	body, _ := os.ReadFile(localFilePath)

	sum := sha256.Sum256([]byte(body))

	if hex.EncodeToString(sum[:]) == file.Checksum {
		return true
	} else {
		return false
	}
}

func CreateDownloadLocation() bool {
	cwd, _ := os.Getwd()
	downloadLocation := cwd + "/" + GetDownloadPath()

	if err := os.MkdirAll(downloadLocation, os.ModePerm); err != nil {
		log.Fatal(err)
	}

	return true
}

func DownloadFiles() *http.Response {
	files := GetFileList()

	if len(files) > 0 {
		CreateDownloadLocation()

		fmt.Printf("Loading Secure Files...\n")

		for _, file := range files {
			downloadErr := DownloadFile(file)
			if downloadErr != nil {
				log.Fatal(downloadErr)
			}
		}
	}

	return nil
}

func main() {
	godotenv.Load()
	DownloadFiles()
}
