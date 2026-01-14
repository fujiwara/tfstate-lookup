package tfstate

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/hashicorp/go-tfe"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
)

const (
	tfeServiceID = "tfe.v2"
)

func readTFEState(ctx context.Context, config map[string]any, ws string) (io.ReadCloser, error) {
	hostname, organization, token := *strpe(config["hostname"]), *strp(config["organization"]), *strpe(config["token"])
	if token == "" {
		token = os.Getenv("TFE_TOKEN")
	}

	workspaces, ok := config["workspaces"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("failed to parse workspaces")
	}

	name, prefix := *strpe(workspaces["name"]), *strpe(workspaces["prefix"])
	if name != "" {
		return readTFE(ctx, hostname, organization, name, token)
	}

	if prefix != "" {
		return readTFE(ctx, hostname, organization, prefix+ws, token)
	}

	return nil, fmt.Errorf("workspaces requires either name or prefix")
}

func readTFE(ctx context.Context, hostname string, organization string, ws string, token string) (io.ReadCloser, error) {
	if hostname == "" {
		u, err := url.Parse(tfe.DefaultAddress)
		if err != nil {
			return nil, err
		}

		hostname = u.Hostname()
	}

	host, err := svchost.ForComparison(hostname)
	if err != nil {
		return nil, err
	}

	serviceDiscovery := disco.New()
	var service *url.URL
	fn := func() error {
		var err error
		service, err = serviceDiscovery.DiscoverServiceURL(host, tfeServiceID)
		// Return the error, unless its a disco.ErrVersionNotSupported error.
		if _, ok := err.(*disco.ErrVersionNotSupported); !ok && err != nil {
			return err
		}
		return nil
	}
	if err := hideStderr(fn); err != nil {
		return nil, fmt.Errorf("failed to discover TFE service URL for host %q: %w", host, err)
	}
	client, err := tfe.NewClient(&tfe.Config{
		Address:  service.String(),
		BasePath: service.Path,
		Token:    token,
	})
	if err != nil {
		return nil, err
	}

	workspace, err := client.Workspaces.Read(ctx, organization, ws)
	if err != nil {
		return nil, err
	}
	state, err := client.StateVersions.ReadCurrent(ctx, workspace.ID)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, state.DownloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token)
	return readHTTPWithRequest(ctx, req)
}
