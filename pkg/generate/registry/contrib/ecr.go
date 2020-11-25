// Package contrib provides functionality for getting digests from
// registries supported by docker-lock's community.
package contrib

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

// ECRWrapper is a registry wrapper for Elastic Container Registry (AWS)
type ECRWrapper struct {
	client   *registry.HTTPClient
	endpoint string
	region   string // TODO: remove if unnecessary in the end
}

// NewECRWrapper creates an ECRWrapper.
func NewECRWrapper(client *registry.HTTPClient, configPath string) (*ECRWrapper, error) {
	w := &ECRWrapper{}
	endpoint, region, err := loadECREndpoint(configPath)
	if err != nil {
		return nil, err
	}
	w.endpoint = endpoint
	w.region = region

	if client == nil {
		w.client = &registry.HTTPClient{
			Client:      &http.Client{},
			RegistryURL: fmt.Sprintf("https://%sv2", w.Prefix()),
		}
	}

	return w, nil
}

func loadECREndpoint(configPath string) (string, string, error) {
	f, err := os.Open(configPath)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	var config struct {
		Auths map[string]interface{} `json:"auths"`
	}
	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return "", "", err
	}

	for k, _ := range config.Auths {
		// TODO: can I be authed into multiple accounts?
		if strings.HasSuffix(k, ".amazonaws.com") {
			// parse region out of "0123456789.dkr.ecr.us-east-1.amazonaws.com"
			parts := strings.SplitN(k, ".", 4)
			if len(parts) < 4 {
				return "", "", fmt.Errorf("invalid ECR endpoint: %v", k)
			}
			return k, parts[3], nil
		}
	}
	// TODO: is this properly handled?
	return "", "", errors.New("no Amazon ECR login found")
}

// init registers ECRWrapper for use by docker-lock.
func init() { // nolint: gochecknoinits
	constructor := func(
		client *registry.HTTPClient,
		configPath string,
	) (registry.Wrapper, error) {
		return NewECRWrapper(client, configPath)
	}

	constructors = append(constructors, constructor)
}

// Digest queries the container registry for the digest given a repo and ref.
func (w *ECRWrapper) Digest(repo string, ref string) (string, error) {
	repo = strings.Replace(repo, w.Prefix(), "", 1)

	r, err := registry.NewV2(w.client)
	if err != nil {
		return "", err
	}
	// TODO: how will we retrieve the token?
	// https://docs.aws.amazon.com/AmazonECR/latest/APIReference/API_GetAuthorizationToken.html
	// https://docs.aws.amazon.com/cli/latest/reference/ecr/get-login-password.html
	//  the cli just calls GetAuthorizationToken and extracts it
	// But I already passed in this token when using `docker login` - so how can I retrieve it
	// without shelling out to `awscli`?

	// this won't work, because I don't think AWS supports basic auth
	// tokenURL := fmt.Sprintf("https://ecr.%s.amazonaws.com", w.region)
	// // the following would need the AWS user/pass, but I'm not sure that would work
	// // I think we need https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html
	// token, err := r.Token(tokenURL, "", "", &registry.DefaultTokenExtractor{})
	// if err != nil {
	// 	return "", err
	// }

	// this is ugly, but it still doesn't work for some reason
	rawToken, err := exec.Command("aws", "ecr", "get-login-password").Output()
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(rawToken))

	return r.Digest(repo, ref, token)
}

// Prefix returns the registry prefix that identifies MCR.
func (w *ECRWrapper) Prefix() string {
	return w.endpoint + "/"
}
