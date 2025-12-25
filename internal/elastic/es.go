package elastic

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/404LifeFound/es-snapshot-restore/config"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/rs/zerolog/log"
)

type ESConfigOption func(*ESConfig)

type ESConfig struct {
	config *elasticsearch.Config
}

func NewESConfig(opts ...ESConfigOption) *ESConfig {
	c := &ESConfig{
		config: &elasticsearch.Config{},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func WithAddr(addr []string) ESConfigOption {
	return func(c *ESConfig) {
		c.config.Addresses = addr
	}
}

func WithUsername(username string) ESConfigOption {
	return func(c *ESConfig) {
		c.config.Username = username
	}
}

func WithPassword(password string) ESConfigOption {
	return func(c *ESConfig) {
		c.config.Password = password
	}
}

func WithSkipTlsVerify(skip bool) ESConfigOption {
	return func(c *ESConfig) {
		c.config.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skip,
			},
		}
	}
}

type ES struct {
	Client *elasticsearch.Client
}

func NewES(config *ESConfig) (*ES, error) {
	client, err := elasticsearch.NewClient(*config.config)
	if err != nil {
		log.Error().Err(err).Send()
		return nil, err
	}

	return &ES{
		Client: client,
	}, nil
}

func NewDefaultESConfig() *ESConfig {
	c := NewESConfig(
		WithAddr([]string{fmt.Sprintf("%s://%s:%d", config.GlobalConfig.ES.Protocol, config.GlobalConfig.ES.Host, config.GlobalConfig.ES.Port)}),
		WithUsername(config.GlobalConfig.ES.Username),
		WithPassword(config.GlobalConfig.ES.Password),
		WithSkipTlsVerify(true),
	)

	return c
}

type Indexs struct {
	Indexs []Index
}

type Index struct {
	Name      string `json:"index"`
	CreateAt  string `json:"creation.date.string"`
	StoreSize string `json:"store.size"`
}

func (es *ES) CatAllIndexRequest() esapi.CatIndicesRequest {
	return esapi.CatIndicesRequest{
		Format:          "json",
		H:               []string{"index", "creation.date.string", "store.size"},
		S:               []string{"creation.date.string"},
		ExpandWildcards: "all",
	}
}

func (es *ES) GetAllIndex(ctx context.Context) ([]Index, error) {
	var all_index []Index
	resp, err := es.CatAllIndexRequest().Do(ctx, es.Client)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&all_index); err != nil {
		return nil, err
	}

	return all_index, nil
}

type Repo struct {
	ID string `json:"id"`
}

type CatSnapshot struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Status     string `json:"status"`
}

type Snapshot struct {
	Snapshot   string   `json:"snapshot"`
	Repository string   `json:"repository"`
	State      string   `json:"state"`
	StartTime  string   `json:"start_time"`
	Indices    []string `json:"indices"`
}

type Snapshots struct {
	Snapshots []Snapshot `json:"snapshots"`
}

type CatSnapshots struct {
	CatSnapshots []CatSnapshot
}

func (s *CatSnapshots) GetAllSnapshotName() []string {
	var all_snapshot []string
	for _, s := range s.CatSnapshots {
		all_snapshot = append(all_snapshot, s.ID)
	}

	return all_snapshot
}

func (es *ES) CatRepoRequest() esapi.CatRepositoriesRequest {
	return esapi.CatRepositoriesRequest{
		Format: "json",
		H:      []string{"id"},
	}
}

func (es *ES) CatRepoSnapshotRequest(repo string) esapi.CatSnapshotsRequest {
	ignore_unavailable := true
	return esapi.CatSnapshotsRequest{
		Repository:        []string{repo},
		Format:            "json",
		IgnoreUnavailable: &ignore_unavailable,
		S:                 []string{"id"},
		H:                 []string{"id", "repository", "status"},
	}
}

func (es *ES) GetSnapshotRequest(repo string, snapshot []string) esapi.SnapshotGetRequest {
	index_names := true
	return esapi.SnapshotGetRequest{
		Repository: repo,
		Snapshot:   snapshot,
		IndexNames: &index_names,
	}
}

func (es *ES) RestoreSnapshotRequest(repo, snapshot, prefix, restore_attr_key, restore_attr_value string, restore_index []string) esapi.SnapshotRestoreRequest {
	indices := strings.Join(restore_index[:], ",")
	json_body := fmt.Sprintf(`{
		"indices": "%s",
		"rename_pattern": "(.+)",
		"rename_replacement": "%s_%s_$1",
		"ignore_index_settings": ["index.lifecycle.name"],
		"index_settings": {
			"index.hidden": false,
			"index.routing.allocation.include._tier_preference": null,
			"index.routing.allocation.exclude.%s": null,
			"index.routing.allocation.require.%s": "%s"
		}
	}`, indices, prefix, restore_attr_value, restore_attr_key, restore_attr_key, restore_attr_value)

	//wait_for_completion := true
	return esapi.SnapshotRestoreRequest{
		Repository: repo,
		Snapshot:   snapshot,
		Body:       strings.NewReader(json_body),
		//WaitForCompletion: &wait_for_completion,
	}
}

func (es *ES) GetAllRepo(ctx context.Context) ([]Repo, error) {
	var repos []Repo

	resp, err := es.CatRepoRequest().Do(ctx, es.Client)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func (es *ES) GetAllSnaphost(ctx context.Context, repo string) ([]CatSnapshot, error) {
	var cat_snapshots []CatSnapshot

	resp, err := es.CatRepoSnapshotRequest(repo).Do(ctx, es.Client)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&cat_snapshots); err != nil {
		return nil, err
	}

	return cat_snapshots, nil
}

func (es *ES) GetSnapshotDetail(ctx context.Context, repo string, snapshot []string) (Snapshots, error) {
	var snapshots Snapshots
	resp, err := es.GetSnapshotRequest(repo, snapshot).Do(ctx, es.Client)
	if err != nil {
		return Snapshots{}, err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&snapshots); err != nil {
		return Snapshots{}, err
	}

	return snapshots, nil
}

func (es *ES) Restore(ctx context.Context, repo, snapshot, prefix, restore_attr_key, restore_attr_value string, restore_index []string) error {
	resp, err := es.RestoreSnapshotRequest(
		repo,
		snapshot,
		prefix,
		restore_attr_key,
		restore_attr_value,
		restore_index,
	).Do(ctx, es.Client)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to restore snapshot of %s from %s: %v", restore_index, snapshot, string(body))
	}

	return nil
}

func (es *ES) GetAllSnapshotDetails(ctx context.Context) ([]Snapshot, error) {
	var all_snapshots []Snapshot
	all_repo, err := es.GetAllRepo(ctx)
	if err != nil {
		log.Error().Err(err).Msg("faild to get all repo")
		return nil, err
	}

	for _, r := range all_repo {
		snapshot_of_repo, err := es.GetAllSnaphost(ctx, r.ID)
		if err != nil {
			log.Error().Err(err).Msgf("failed to get all snapshots from repo %s", r.ID)
			continue
		}

		if len(snapshot_of_repo) > 0 {
			all_snapshot_of_repo_name := &CatSnapshots{
				CatSnapshots: snapshot_of_repo,
			}
			snapshots, err := es.GetSnapshotDetail(ctx, r.ID, all_snapshot_of_repo_name.GetAllSnapshotName())
			all_snapshots = append(all_snapshots, snapshots.Snapshots...)
			if err != nil {
				log.Error().Err(err).Msgf("faild to get snapshot detail of snapshot %s from repo %s", all_snapshot_of_repo_name.GetAllSnapshotName(), r.ID)
			}
		}
	}

	return all_snapshots, nil
}
