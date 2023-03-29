package tfstate

import (
	"context"
	"fmt"
	"io"

	tfe "github.com/hashicorp/go-tfe"
)

func readTFEState(ctx context.Context, config map[string]interface{}, ws string) (io.ReadCloser, error) {
	hostname, organization, token := *strpe(config["hostname"]), *strp(config["organization"]), *strpe(config["token"])

	workspaces, ok := config["workspaces"].(map[string]interface{})
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
	var address string
	address = tfe.DefaultAddress
	if hostname != "" {
		address = "https://" + hostname
	}

	var err error
	var client *tfe.Client
	if token != "" {
		client, err = tfe.NewClient(&tfe.Config{
			Address: address,
			Token:   token,
		})
	} else {
		client, err = tfe.NewClient(&tfe.Config{
			Address: address,
		})
	}
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

	return readHTTP(ctx, state.DownloadURL)
}
