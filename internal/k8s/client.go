package k8s

import (
	"context"

	"github.com/404LifeFound/es-snapshot-restore/config"
	esv1 "github.com/elastic/cloud-on-k8s/v3/pkg/apis/elasticsearch/v1"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	runtimeclient.Client
}

func NewClient() (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", config.GlobalConfig.Kube.Config)
	if err != nil {
		log.Error().Err(err).Msg("failed to build kubernetes config")
		return nil, err
	}

	scheme := runtime.NewScheme()
	esv1.AddToScheme(scheme)

	c, err := runtimeclient.New(
		config,
		runtimeclient.Options{
			Scheme: scheme,
		},
	)

	if err != nil {
		log.Error().Err(err).Msg("faild to create controller runtime client")
		return nil, err
	}

	return &Client{Client: c}, nil
}

func (c *Client) Update(ctx context.Context) {
	c.Patch(ctx, obj runtimeclient.Object, patch runtimeclient.Patch, opts ...runtimeclient.PatchOption)
}
