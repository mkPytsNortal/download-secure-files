package main

import (
	"net/url"
	"os"
	"testing"
)

func Test_GetApiURL(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantVal string
	}{
		{"Environment variable is set", "http://localhost:3001/api/v4", "http://localhost:3001/api/v4"},
		{"Environment variable is not set", "", "https://gitlab.com/api/v4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CI_API_V4_URL", tt.envVar)

			got := GetApiURL()

			if got != tt.wantVal {
				t.Errorf("GetApiURL() = %v, want %v", got, tt.wantVal)
			}
		})
	}
}

func Test_GetProjectId(t *testing.T) {
	tests := []struct {
		name      string
		envVar    string
		wantVal   string
		wantPanic bool
	}{
		{"Environment variable is set", "username/project", url.QueryEscape("username/project"), false},
		{"Environment variable is not set", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("CI_PROJECT_ID", tt.envVar)

			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("SequenceInt() recover = %v, wantPanic = %v", r, tt.wantPanic)
				}
			}()

			got := GetProjectId()

			if got != tt.wantVal {
				t.Errorf("GetProjectId() = %v, want %v", got, tt.wantVal)
			}
		})
	}
}

func Test_GetDownloadPath(t *testing.T) {
	tests := []struct {
		name    string
		envVar  string
		wantVal string
	}{
		{"Environment variable is set", "my/path", "my/path"},
		{"Environment variable is not set", "", ".secure_files"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("SECURE_FILES_DOWNLOAD_PATH", tt.envVar)

			got := GetDownloadPath()

			if got != tt.wantVal {
				t.Errorf("GetDownloadPath() = %v, want %v", got, tt.wantVal)
			}
		})
	}
}
