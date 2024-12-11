package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
)

const (
	snykAPI = "https://api.snyk.io"
)

type SnykAPI struct {
	SnykToken string
	SnykOrgId string
}

type SnykRequest struct {
	Filters Filters `json:"filters"`
}

type Filters struct {
	Projects []string `json:"projects"`
}

type SnykResponse struct {
	Results []struct {
		ID           string `json:"id"`
		Dependencies []struct {
			Name string `json:name`
		} `json:"dependencies"`
	} `json:"results"`
}

var (
	snykResponseAll SnykResponse
)

func main() {

	var sk SnykAPI
	var goModFile string
	goModFile, ok := os.LookupEnv("SNYK_GO_MOD_FILE")
	if !ok {
		log.Fatal("SNYK_GO_MOD_FILE is unset")
	}

	// Fetching token and snyk org id
	sk.SnykToken, ok = os.LookupEnv("SNYK_TOKEN")
	if !ok {
		log.Fatal("SNYK_TOKEN environment variable not set")
	}

	sk.SnykOrgId, ok = os.LookupEnv("SNYK_ORGANIZATION")
	if !ok {
		log.Fatal("SNYK_ORGANIZATION environment variable not set")
	}

	var projectId string
	projectId, ok = os.LookupEnv("SNYK_PROJECT_ID")
	if !ok {
		projectId = "fe17322a-c8ab-442d-96cb-1658da1cd57b"
	}

	req, err := json.Marshal(SnykRequest{
		Filters: Filters{
			Projects: []string{projectId},
		},
	})
	if err != nil {
		fmt.Printf("error creating request body: %v", err)
		return
	}

	url := fmt.Sprintf("%s/v1/org/%s/licenses", snykAPI, sk.SnykOrgId)
	resp, err := sk.callSnykAPI("POST", url, req)
	if err != nil {
		log.Fatal(fmt.Sprintf("error calling api: %v", err))
	}

	snykResponse := SnykResponse{}

	buf := new(strings.Builder)
	_, err = io.Copy(buf, resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal([]byte(buf.String()), &snykResponse)
	if err != nil {
		log.Fatal(err)
	}

	licenseMap, err := parseDirectDependencies(snykResponse, goModFile)
	if err != nil {
		log.Fatal(err)
	}

	licenses, err := json.Marshal(struct {
		Licenses map[string][]string `json:"licenses"`
	}{Licenses: licenseMap})
	if err != nil {
		log.Fatal(err)
	}

	var fileName string
	fileName, ok = os.LookupEnv("SNYK_LICENSE_RESULT_FILE")
	if !ok {
		fileName = "snyk-project-licenses.json"
	}
	err = os.WriteFile(fileName, licenses, 0644)
	if err != nil {
		fmt.Printf("error writing snyk response: %v", err)
		return
	}
}

func parseDirectDependencies(depList SnykResponse, goModFile string) (map[string][]string, error) {
	licenseMap := map[string][]string{}
	goDepsFile, err := os.Open(goModFile)
	if err != nil {
		return licenseMap, err
	}
	defer goDepsFile.Close()

	directDeps := []string{}
	depPattern := regexp.MustCompile("^\t.*[0-9]$")

	scanner := bufio.NewScanner(goDepsFile)
	for scanner.Scan() {
		l := string(scanner.Text())
		if depPattern.Match([]byte(l)) {
			dep := strings.Split(l, " ")[0]
			directDeps = append(directDeps, strings.TrimSpace(dep))
		}
	}
	for _, r := range depList.Results {
		licenseMap[r.ID] = []string{}
		for _, d := range r.Dependencies {
			for _, v := range directDeps {
				if strings.HasPrefix(d.Name, v) {
					if !slices.Contains(licenseMap[r.ID], v) {
						licenseMap[r.ID] = append(licenseMap[r.ID], v)
					}
				}
			}
		}
	}

	return licenseMap, nil
}

func (sn *SnykAPI) callSnykAPI(method string, url string, body []byte) (*http.Response, error) {

	httpReq, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	httpReq.Header.Add("Authorization", "token "+sn.SnykToken)
	httpReq.Header.Add("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(httpReq)

	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error calling snyk API, status code (%d): %v", resp.StatusCode, err)
	}
	return resp, nil
}
