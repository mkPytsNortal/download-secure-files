package main

import (
	"net/http"
	"os"
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func Test_getEnvWithDefault(t *testing.T) {
	t.Run("when environment variable is set", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "envVarValue")

		actual := getEnvWithDefault("TEST_ENV_VAR", "envVarValue")

		assert.Equal(t, "envVarValue", actual)
	})

	t.Run("when environment variable is set", func(t *testing.T) {
		os.Unsetenv("TEST_ENV_VAR")

		actual := getEnvWithDefault("TEST_ENV_VAR", "defaultValue")

		assert.Equal(t, "defaultValue", actual)
	})
}

func Test_authHeader(t *testing.T) {
	t.Run("when the CI_JOB_TOKEN environment variable is set", func(t *testing.T) {
		os.Unsetenv("PRIVATE_TOKEN")
		os.Setenv("CI_JOB_TOKEN", "jobToken")

		actualHeader, actualError := authHeader()

		assert.Equal(t, nil, actualError)
		assert.Equal(t, http.Header{"JOB-TOKEN": {"jobToken"}}, actualHeader)
	})

	t.Run("when the PRIVATE_TOKEN environment variable is set", func(t *testing.T) {
		os.Unsetenv("CI_JOB_TOKEN")
		os.Setenv("PRIVATE_TOKEN", "privateToken")

		actualHeader, actualError := authHeader()

		assert.Equal(t, nil, actualError)
		assert.Equal(t, http.Header{"PRIVATE-TOKEN": {"privateToken"}}, actualHeader)
	})

	t.Run("when the no authentication environment variable is set", func(t *testing.T) {
		os.Unsetenv("CI_JOB_TOKEN")
		os.Unsetenv("PRIVATE_TOKEN")

		actualHeader, actualError := authHeader()

		assert.Equal(t, "Authentication Token Missing", actualError.Error())
		assert.Equal(t, http.Header{}, actualHeader)
	})
}

func Test_writeFile(t *testing.T) {
	t.Run("when successfully writing a file", func(t *testing.T) {
		fileData := "filedata"
		filePath := "file.txt"

		actualWriteStatus, actualError := writeFile([]byte(fileData), filePath)

		assert.Equal(t, nil, actualError)
		assert.Equal(t, true, actualWriteStatus)
		os.Remove(filePath)
	})

	t.Run("when failing to writing a file", func(t *testing.T) {
		fileData := "filedata"
		filePath := "/foo/bar/file.txt"

		actualWriteStatus, actualError := writeFile([]byte(fileData), filePath)

		assert.Equal(t, "open /foo/bar/file.txt: no such file or directory", actualError.Error())
		assert.Equal(t, false, actualWriteStatus)
	})
}

func Test_newDownloaderFromEnv_Url(t *testing.T) {
	t.Run("when the CI_API_V4_URL environment variable is set", func(t *testing.T) {
		os.Setenv("CI_PROJECT_ID", "foo")
		os.Setenv("PRIVATE_TOKEN", "privateToken")
		os.Setenv("CI_API_V4_URL", "http://example.com/api/v4")

		actualDownloader, actualError := newDownloaderFromEnv()

		assert.Equal(t, nil, actualError)
		assert.Equal(t, "http://example.com/api/v4", actualDownloader.url)
	})

	t.Run("when the CI_API_V4_URL environment variable is not set", func(t *testing.T) {
		os.Setenv("CI_PROJECT_ID", "foo")
		os.Setenv("PRIVATE_TOKEN", "privateToken")
		os.Unsetenv("CI_API_V4_URL")

		actualDownloader, actualError := newDownloaderFromEnv()

		assert.Equal(t, nil, actualError)
		assert.Equal(t, "https://gitlab.com/api/v4", actualDownloader.url)
	})
}

func Test_newDownloaderFromEnv_Path(t *testing.T) {
	t.Run("when the SECURE_FILES_DOWNLOAD_PATH environment variable is set", func(t *testing.T) {
		os.Setenv("CI_PROJECT_ID", "foo")
		os.Setenv("PRIVATE_TOKEN", "privateToken")
		os.Setenv("SECURE_FILES_DOWNLOAD_PATH", "my/file/path")

		actualDownloader, actualError := newDownloaderFromEnv()

		assert.Equal(t, nil, actualError)
		assert.Equal(t, "my/file/path", actualDownloader.downloadPath)
	})

	t.Run("when the SECURE_FILES_DOWNLOAD_PATH environment variable is set", func(t *testing.T) {
		os.Setenv("CI_PROJECT_ID", "foo")
		os.Setenv("PRIVATE_TOKEN", "privateToken")
		os.Unsetenv("SECURE_FILES_DOWNLOAD_PATH")

		actualDownloader, actualError := newDownloaderFromEnv()

		assert.Equal(t, nil, actualError)
		assert.Equal(t, ".secure_files", actualDownloader.downloadPath)
	})
}

func Test_newDownloaderFromEnv_ProjectID(t *testing.T) {
	t.Run("when the CI_PROJECT_ID environment variable is set", func(t *testing.T) {
		os.Setenv("CI_PROJECT_ID", "foo")
		os.Setenv("PRIVATE_TOKEN", "privateToken")

		actualDownloader, actualError := newDownloaderFromEnv()

		assert.Equal(t, nil, actualError)
		assert.Equal(t, "foo", actualDownloader.projectID)
	})

	t.Run("when the CI_PROJECT_ID environment variable is not set", func(t *testing.T) {
		os.Unsetenv("CI_PROJECT_ID")
		os.Setenv("PRIVATE_TOKEN", "privateToken")
		actualDownloader, actualError := newDownloaderFromEnv()

		assert.Equal(t, "Project ID missing", actualError.Error())
		assert.Equal(t, "", actualDownloader.projectID)
	})
}

func Test_verifyChecksum(t *testing.T) {
	t.Run("when the checksums match", func(t *testing.T) {
		secureFile := SecureFile{Checksum: "aec070645fe53ee3b3763059376134f058cc337247c978add178b6ccdfb0019f"}
		actualError := secureFile.verifyChecksum("fixtures/test_file.txt")
		assert.Equal(t, nil, actualError)
	})

	t.Run("when the checksums do not match", func(t *testing.T) {
		secureFile := SecureFile{Checksum: "foo", Name: "foo.txt"}
		actualError := secureFile.verifyChecksum("fixtures/test_file.txt")
		assert.Equal(t, "failure validating checksum for fixtures/test_file.txt", actualError.Error())
	})

	t.Run("when the file could not be found", func(t *testing.T) {
		secureFile := SecureFile{Checksum: "123"}
		actualError := secureFile.verifyChecksum("fixtures/test_file2.txt")
		assert.Equal(t, "open fixtures/test_file2.txt: no such file or directory", actualError.Error())
	})
}

func Test_createDownloadLocation(t *testing.T) {
	t.Run("when the download folder is created successfully", func(t *testing.T) {
		workingDirectory, _ := os.Getwd()
		downloader := Downloader{workingDirectory: workingDirectory, downloadPath: "fixtures/a"}

		gotDownloadLocation, gotErr := downloader.createDownloadLocation()

		assert.Equal(t, nil, gotErr)
		assert.Equal(t, workingDirectory+"/fixtures/a", gotDownloadLocation)
		os.Remove(gotDownloadLocation)
	})

	t.Run("removes unallowed values from the download path", func(t *testing.T) {
		workingDirectory, _ := os.Getwd()
		downloader := Downloader{workingDirectory: workingDirectory, downloadPath: "../../../"}

		gotDownloadLocation, gotErr := downloader.createDownloadLocation()

		assert.Equal(t, nil, gotErr)
		assert.Equal(t, workingDirectory, gotDownloadLocation)
		os.Remove(gotDownloadLocation)
	})
}

func Test_downloadFile(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", `=~^http://localhost:3001\/api\/v4\/projects\/123\/secure_files\/1\/download`,
		httpmock.NewStringResponder(200, `foobar`))

	t.Run("a subfolder is created successfully when provided", func(t *testing.T) {
		workingDirectory, err := os.Getwd()
		assert.NoError(t, err)

		downloadPath := "fixtures/a"

		testDownloader := Downloader{
			url:              "http://localhost:3001/api/v4",
			workingDirectory: workingDirectory,
			downloadPath:     downloadPath,
			projectID:        "123",
		}

		assert.Equal(t, "fixtures/a", testDownloader.downloadPath)

		secureFile := SecureFile{
			ID:                1,
			Name:              "foo/bar/mockfile.txt",
			Checksum:          "c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2",
			ChecksumAlgorithm: "sha256",
		}

		gotErr := testDownloader.downloadFile(secureFile)
		assert.NoError(t, gotErr)

		_, pathErr := os.Stat("fixtures/a/foo/bar")
		assert.NoError(t, pathErr)

		os.RemoveAll(downloadPath)
	})

	t.Run("a subfolder is not created when the filename does not contain a subfolder", func(t *testing.T) {
		workingDirectory, err := os.Getwd()
		assert.NoError(t, err)

		downloadPath := "fixtures/a"

		testDownloader := Downloader{
			url:              "http://localhost:3001/api/v4",
			workingDirectory: workingDirectory,
			downloadPath:     downloadPath,
			projectID:        "123",
		}

		assert.Equal(t, "fixtures/a", testDownloader.downloadPath)

		secureFile := SecureFile{
			ID:                1,
			Name:              "mockfile.txt",
			Checksum:          "c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2",
			ChecksumAlgorithm: "sha256",
		}

		gotErr := testDownloader.downloadFile(secureFile)
		assert.NoError(t, gotErr)

		_, pathErr := os.Stat("fixtures/a")
		assert.NoError(t, pathErr)

		os.RemoveAll(downloadPath)
	})
}
