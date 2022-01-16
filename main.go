package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"

	dockermanifest "github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/pkg/docker/config"
)

const (
	contentType = "application/vnd.docker.distribution.manifest.v2+json"
)

type Taglist struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func listTags(regURL, user, pass string) ([]string, error) {
	req, err := http.NewRequest("GET", regURL, nil)
	if err != nil {
		return nil, err
	}

	if user != "" {
		req.Header.Set("Authorization", "Basic "+basicAuth(user, pass))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	manifest := string(res)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("getting manifest failed: %s", manifest)
	}

	var t Taglist

	err = json.Unmarshal(res, &t)
	if err != nil {
		return nil, err
	}

	return t.Tags, nil
}

func getManifest(regURL, user, pass string) (string, error) {
	req, err := http.NewRequest("GET", regURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", contentType)

	if user != "" {
		req.Header.Set("Authorization", "Basic "+basicAuth(user, pass))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	manifest := string(res)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("getting manifest failed: %s", manifest)
	}

	return manifest, nil
}

func addTag(regURL, user, pass, manifest string) error {
	body := strings.NewReader(manifest)

	req, err := http.NewRequest("PUT", regURL, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", contentType)

	if user != "" {
		req.Header.Set("Authorization", "Basic "+basicAuth(user, pass))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return fmt.Errorf("tagging failed, got status %v", resp.StatusCode)
	}

	return nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func printTags(myURL *url.URL, registry, repo, baseURL, baseTag, user, pass string) {
	tags, err := listTags(myURL.Scheme+"://"+registry+"/v2"+repo+"/tags/list", user, pass)
	if err != nil {
		log.Fatalf("listTags failed %s", err)
	}

	manifest, err := getManifest(baseURL+"/"+baseTag, user, pass)
	if err != nil {
		log.Fatalf("failed to get manifest on %s: %s", baseURL+"/"+baseTag, err)
	}

	var m dockermanifest.Schema2

	err = json.Unmarshal([]byte(manifest), &m)
	if err != nil {
		log.Fatalf("unmarshal failed %s", err)
	}

	baseDigest := m.ConfigDescriptor.Digest.String()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\n", baseTag, baseDigest)

	for _, tag := range tags {
		if tag == baseTag {
			continue
		}
		manifest, err = getManifest(baseURL+"/"+tag, user, pass)
		err = json.Unmarshal([]byte(manifest), &m)
		if err != nil {
			log.Fatalf("unmarshal failed %s", err)
		}

		if baseDigest == m.ConfigDescriptor.Digest.String() {
			fmt.Fprintf(w, "%s\t%s\n", tag, m.ConfigDescriptor.Digest.String())
		}
	}

	w.Flush()
}

func printHelp() {
	fmt.Printf("Usage of %s:\n", os.Args[0])
	fmt.Println("\tlist alternative tags: " + os.Args[0] + " registry/image:tag (uses docker login credentials by default)")
	fmt.Println("\tadd a tag            : " + os.Args[0] + " registry/image:tag extratag (uses docker login credentials by default)")
	fmt.Println()
	flag.PrintDefaults()
}

func main() {
	var user, pass, creds string

	flag.StringVar(&creds, "creds", "", "use [username[:password]] for accessing the registry")
	flag.Parse()

	if len(os.Args) == 1 {
		printHelp()
		return
	}

	if creds != "" {
		res := strings.Split(creds, ":")
		user = res[0]

		if len(res) > 1 {
			pass = res[1]
		}
	}

	if len(flag.Args()) == 0 {
		printHelp()
		return
	}

	imageTag := flag.Arg(0)

	if !strings.Contains(imageTag, "://") {
		imageTag = "https://" + imageTag
	}

	myURL, err := url.Parse(imageTag)
	if err != nil {
		log.Fatalf("parsing failed: %s", err)
	}

	registry := myURL.Host
	if registry == "" {
		log.Fatalf("parsing failed, registry url not found in %s", imageTag)
	}

	res := strings.Split(myURL.Path, ":")
	baseTag := "latest"
	repo := ""

	if len(res) == 2 {
		repo = res[0]
		baseTag = res[1]
	}

	if repo == "" {
		log.Fatalf("parsing failed, registry url not found in %s", imageTag)
	}

	if creds == "" {
		user, pass, err = config.GetAuthentication(nil, registry)
		if err != nil {
			log.Fatalf("parsing docker authentication failed: %s", err)
		}
	}

	baseURL := myURL.Scheme + "://" + registry + "/v2" + repo + "/manifests"

	if len(flag.Args()) == 1 {
		printTags(myURL, registry, repo, baseURL, baseTag, user, pass)
		os.Exit(0)
	}

	newTag := flag.Arg(1)
	manifest, err := getManifest(baseURL+"/"+baseTag, user, pass)
	if err != nil {
		log.Fatalf("failed to get manifest on %s: %s", baseURL+"/"+baseTag, err)
	}

	if err = addTag(baseURL+"/"+newTag, user, pass, manifest); err != nil {
		log.Fatalf("failed to set tag on %s: %s", baseURL+"/"+newTag, err)
	}

	fmt.Printf("%s added to %s\n", newTag, imageTag)
}
