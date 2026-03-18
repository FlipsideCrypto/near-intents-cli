package main

import (
	"context"
	"fmt"
	"os"

	oneclick "github.com/defuse-protocol/one-click-sdk-go"
)

type Client struct {
	api   *oneclick.APIClient
	token string
	cfg   *Config
}

func newClient(cfg *Config) *Client {
	token := resolveToken(flagToken, cfg)

	apiCfg := oneclick.NewConfiguration()
	if cfg.APIEndpoint != DefaultAPIEndpoint {
		apiCfg.Servers = oneclick.ServerConfigurations{
			{URL: cfg.APIEndpoint},
		}
	}
	if flagVerbose {
		apiCfg.Debug = true
	}

	return &Client{
		api:   oneclick.NewAPIClient(apiCfg),
		token: token,
		cfg:   cfg,
	}
}

func (c *Client) ctx() context.Context {
	ctx := context.Background()
	if c.token != "" {
		ctx = context.WithValue(ctx, oneclick.ContextAccessToken, c.token)
	}
	return ctx
}

func verbose(format string, args ...interface{}) {
	if flagVerbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
	}
}
