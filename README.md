# download-secure-files


## Background

The download-secure-files project exists to create a simple way to integrate a GitLab Runner with the [Project-level Secure Files](https://docs.gitlab.com/ee/ci/secure_files/) feature in GitLab. Please note: this feature is still under active development. Please report any bugs or problems by creating an issue in the [download-secure-files project](https://gitlab.com/gitlab-org/incubation-engineering/mobile-devops/download-secure-files/-/issues).


## Getting started

The quickest way to get started is to add the following line at the start of your CI script in your `.gitlab-ci.yml` file. 

```
curl -s https://gitlab.com/gitlab-org/incubation-engineering/mobile-devops/load-secure-files/-/raw/main/installer | bash
```

This command will run this program which will download all of the Secure Files for the project into the `.secure_files` directory in the CI job.

If you'd like to change the location where the files are downloaded, simply set the `SECURE_FILES_DOWNLOAD_PATH` environment variable. This can be done in either the `.gitlab-ci.yml` file, or as a CI variable.

## Configuration

`download-secure-files` will load all the necessary configuration from the CI job. The configuration can be overridden by setting the environment variables defined below:

| Environment Variable       | Description                                                                                 | Automatically added in CI | Default Value             |
|----------------------------|---------------------------------------------------------------------------------------------|---------------------------|---------------------------|
| CI_API_V4_URL              | The API URL for the GitLab Instance                                                         | yes                       | https://gitlab.com/api/v4 |
| CI_PROJECT_ID              | The Project ID where the Secure Files are stored                                            | yes                       |                           |
| SECURE_FILES_DOWNLOAD_PATH | The location to download the files to in the CI job                                         | no                        | .secure_files             |
| CI_JOB_TOKEN               | Authentication token when running in CI                                                     | yes                       |                           |
| PRIVATE_TOKEN              | Authentication token when running outside of CI (can be a personal or project access token) | n/a                       |                           |

## Sample config

Below is a simple `.gitlab-ci.yml` file that can be used to test the feature

```
stages:
  - test

test:
  script: 
    - curl -s https://gitlab.com/gitlab-org/incubation-engineering/mobile-devops/load-secure-files/-/raw/main/installer | bash
    - ls -lah .secure_files
```

## How it works

This project builds a self contained distribution that can be run on Linux, OS X, or Windows without having to install additional dependencies. 

The `installer` script will attempt to detect the architecture of the machine and download the appropriate distribution. Once downloaded it will automatically run the program to download the Secure Files

If for some reason the `installer` script doesn't work on your architecture, please create an issue in [this project](https://gitlab.com/gitlab-org/incubation-engineering/mobile-devops/download-secure-files/-/issues), or feel free to contribute a merge request with suppport added for your architecture.


## Development

A few notes to get started:

* Ensure you're running golang 1.19 or later
* Install the dependencies by running `go get`
* Run the tests by running `go test -v`
* To run the program locally, add a `.env` file in the project root to set the environment variables, then run `go run .`
