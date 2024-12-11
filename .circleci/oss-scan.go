package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
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

	if len(os.Args) > 1 {
		helpText := strings.Builder{}
		if os.Args[1] == "--help" || os.Args[1] == "-h" {
			helpText.WriteString("OSS-Scan\n")
			helpText.WriteString("	Generate OSS License dossier for direct project dependencies from Snyk\n")
			helpText.WriteString("	Usage:\n")
			helpText.WriteString("		go run ./oss-scan.go\n")
			helpText.WriteString("	Configuration:\n")
			helpText.WriteString("		Environment Variables:\n")
			helpText.WriteString("			SNYK_TOKEN: A Snyk API token\n")
			helpText.WriteString("				Default: \"go.mod\"\n")
			helpText.WriteString("			SNYK_ORGANIZATION: The Snyk Organization ID the intended project belongs to\n")
			helpText.WriteString("				Defult: The CircleCI Orgnaization ID\n")
			helpText.WriteString("			SNYK_PROJECT_ID: The Snyk Project ID for the project to scan\n")
			helpText.WriteString("				Defult: The runner-init Project ID\n")
			helpText.WriteString("			SNYK_LICENSE_RESULT_FILE: The name of the file to write the license scan results to\n")
			helpText.WriteString("				Defult: \"snyk-project-licences.csv\"\n")
			fmt.Print(helpText.String())
			return
		}
	}
	var sk SnykAPI
	var goModFile string
	goModFile, ok := os.LookupEnv("SNYK_GO_MOD_FILE")
	if !ok {
		goModFile = "go.mod"
	}

	sk.SnykToken, ok = os.LookupEnv("SNYK_TOKEN")
	if !ok {
		log.Fatal("SNYK_TOKEN environment variable not set")
	}

	sk.SnykOrgId, ok = os.LookupEnv("SNYK_ORGANIZATION")
	if !ok {
		// the CircleCI Snyk Org ID
		sk.SnykOrgId = "844e0371-ef50-48c1-a0d1-1dbd652b2982"
	}

	var projectId string
	projectId, ok = os.LookupEnv("SNYK_PROJECT_ID")
	if !ok {
		// the runner-init project ID
		projectId = "fe17322a-c8ab-442d-96cb-1658da1cd57b"
	}

	var fileName string
	fileName, ok = os.LookupEnv("SNYK_LICENSE_RESULT_FILE")
	if !ok {
		fileName = "snyk-project-licenses.csv"
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

	licenseRows, err := parseDirectDependencies(snykResponse, goModFile)
	if err != nil {
		log.Fatal(err)
	}

	outFile, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer outFile.Close()

	w := csv.NewWriter(outFile)
	w.Write([]string{"Name", "License"})
	for _, row := range licenseRows {
		if err := w.Write(row); err != nil {
			log.Fatal(err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}

}

func parseDirectDependencies(depList SnykResponse, goModFile string) ([][]string, error) {
	licenses := []string{}
	goDepsFile, err := os.Open(goModFile)
	if err != nil {
		return [][]string{}, err
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
		for _, d := range r.Dependencies {
			for _, v := range directDeps {
				if strings.HasPrefix(d.Name, v) {
					entry := fmt.Sprintf("%s::%s", v, r.ID)
					if !slices.Contains(licenses, entry) {
						licenses = append(licenses, entry)
					}
				}
			}
		}
	}
	licenseRows := [][]string{}
	for _, l := range licenses {
		licenseRow := strings.Split(l, "::")
		licenseRows = append(licenseRows, licenseRow)
	}

	return licenseRows, nil
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
