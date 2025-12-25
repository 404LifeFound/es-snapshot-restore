package http

import (
	"context"
	"fmt"
	"strings"

	"github.com/404LifeFound/es-snapshot-restore/config"
	"github.com/404LifeFound/es-snapshot-restore/internal/db"
	"github.com/404LifeFound/es-snapshot-restore/internal/k8s"
	elasticsearchv1 "github.com/elastic/cloud-on-k8s/v3/pkg/apis/elasticsearch/v1"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (h *Handler) QueryIndexResultViaTime(name []string, startAt, endAt string) ([]db.ESIndex, error) {
	log.Info().Msgf("name is %v", name)
	log.Info().Msgf("startAt is %v", startAt)
	log.Info().Msgf("endAt is %v", endAt)
	var all_result []db.ESIndex
	var before_start_time_first_result []db.ESIndex
	var after_start_time_result []db.ESIndex
	var after_start_time_end_before_end_time_result []db.ESIndex
	var before_end_time_result []db.ESIndex
	var err error

	var name_conds []string
	var param []any

	for _, n := range name {
		name_conds = append(name_conds, "name LIKE ?")
		param = append(param, fmt.Sprintf("%%%s%%", n))
	}

	//nameQuery := "(" + strings.Join(name_conds, " OR ") + ")"
	nameQuery := strings.Join(name_conds, " OR ")
	log.Info().Msgf("nameQuery is: %s", nameQuery)
	//nameQuery := fmt.Sprintf("%s%s%s", "(", strings.Join(name_conds, " OR "), ")")

	if startAt != "" && endAt == "" {
		before_start_time_first_query := fmt.Sprintf("%s AND index_create_at <= ?", nameQuery)
		before_start_time_first_query_param := param
		before_start_time_first_query_param = append(before_start_time_first_query_param, startAt)
		before_start_time_first_query_conds := []any{}
		before_start_time_first_query_conds = append(append(before_start_time_first_query_conds, before_start_time_first_query), before_start_time_first_query_param...)

		log.Info().Msgf("before_start_time_first_query_conds is %s", before_start_time_first_query_conds)

		if before_start_time_first_result, err = db.QueryAll[db.ESIndex](
			h.DBClient,
			"index_create_at DESC",
			1,
			before_start_time_first_query_conds...,
		); err != nil {
			return nil, err
		}
		if len(before_start_time_first_result) > 0 {
			all_result = append(all_result, before_start_time_first_result...)
		}

		after_start_time_query := fmt.Sprintf("%s AND index_create_at >= ?", nameQuery)
		after_start_time_query_param := param
		after_start_time_query_param = append(after_start_time_query_param, startAt)
		after_start_time_query_conds := []any{}
		after_start_time_query_conds = append(append(after_start_time_query_conds, after_start_time_query), after_start_time_query_param...)

		if after_start_time_result, err = db.QueryAll[db.ESIndex](
			h.DBClient,
			"index_create_at DESC",
			0,
			after_start_time_query_conds...,
		); err != nil {
			return nil, err
		}
		if len(after_start_time_result) > 0 {
			all_result = append(all_result, after_start_time_result...)
		}

	} else if startAt != "" && endAt != "" {
		before_start_time_first_query := fmt.Sprintf("%s AND index_create_at <= ?", nameQuery)
		before_start_time_first_query_param := param
		before_start_time_first_query_param = append(before_start_time_first_query_param, startAt)
		before_start_time_first_query_conds := []any{}
		before_start_time_first_query_conds = append(append(before_start_time_first_query_conds, before_start_time_first_query), before_start_time_first_query_param...)
		if before_start_time_first_result, err = db.QueryAll[db.ESIndex](
			h.DBClient,
			"index_create_at DESC",
			1,
			before_start_time_first_query_conds...,
		); err != nil {
			return nil, err
		}
		if len(before_start_time_first_result) > 0 {
			all_result = append(all_result, before_start_time_first_result...)
		}

		after_start_time_end_before_end_time_query := fmt.Sprintf("%s AND index_create_at >= ? AND index_create_at <= ?", nameQuery)
		after_start_time_end_before_end_time_query_param := param
		after_start_time_end_before_end_time_query_param = append(append(after_start_time_end_before_end_time_query_param, startAt), endAt)
		after_start_time_end_before_end_time_query_conds := []any{}
		after_start_time_end_before_end_time_query_conds = append(append(after_start_time_end_before_end_time_query_conds, after_start_time_end_before_end_time_query), after_start_time_end_before_end_time_query_param...)
		if after_start_time_end_before_end_time_result, err = db.QueryAll[db.ESIndex](
			h.DBClient,
			"index_create_at DESC",
			0,
			after_start_time_end_before_end_time_query_conds...,
		); err != nil {
			return nil, err
		}
		if len(after_start_time_end_before_end_time_result) > 0 {
			all_result = append(all_result, after_start_time_end_before_end_time_result...)
		}

	} else if startAt == "" && endAt != "" {
		before_end_time_query := fmt.Sprintf("%s AND index_create_at <= ?", nameQuery)
		before_end_time_query_param := param
		before_end_time_query_param = append(before_end_time_query_param, endAt)
		before_end_time_query_conds := []any{}
		before_end_time_query_conds = append(append(before_end_time_query_conds, before_end_time_query), before_end_time_query_param...)
		if before_end_time_result, err = db.QueryAll[db.ESIndex](
			h.DBClient,
			"index_create_at DESC",
			0,
			before_end_time_query_conds...,
		); err != nil {
			return nil, err
		}
		if len(before_end_time_result) > 0 {
			all_result = append(all_result, before_end_time_result...)
		}

	} else {
		default_query := nameQuery
		default_query_param := param
		default_query_conds := []any{}
		default_query_conds = append(append(default_query_conds, default_query), default_query_param...)
		if all_result, err = db.QueryAll[db.ESIndex](
			h.DBClient,
			"index_create_at DESC",
			0,
			default_query_conds...,
		); err != nil {
			return nil, err
		}
	}

	return all_result, err
}

func (h *Handler) QueryLatestSnapshotsViaIndex(index []db.ESIndex) (map[string]db.ESSnapshot, error) {
	all_matched_snapshots := make(map[string]db.ESSnapshot)
	for _, i := range index {
		snapshot, err := db.QueryAll[db.ESSnapshot](h.DBClient, "start_time DESC", 1, "state = 'SUCCESS' AND JSON_CONTAINS(indices,JSON_QUOTE(?))", i.Name)
		if err != nil {
			log.Error().Err(err).Msgf("failed to get snapshot for index %s", i.Name)
			continue
		}

		if len(snapshot) == 1 {
			all_matched_snapshots[i.Name] = snapshot[0]
		}
	}

	return all_matched_snapshots, nil
}

func (h *Handler) GetIndexGBSize(index []db.ESIndex) float64 {
	indices := db.ESIndexs(index)
	return indices.StoreSize()
}

func (h *Handler) GetElasticsearch(ctx context.Context) (*elasticsearchv1.Elasticsearch, error) {
	es := &elasticsearchv1.Elasticsearch{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Elasticsearch",
			APIVersion: "elasticsearch.k8s.elastic.co/v1",
		},
	}

	err := h.K8Sclient.Get(ctx,
		runtimeclient.ObjectKey{Namespace: config.GlobalConfig.ES.Namespace, Name: config.GlobalConfig.ES.Name},
		es,
	)

	if err != nil {
		log.Error().Err(err).Msgf("faild to get Elasticsearch %s from %s namespace", config.GlobalConfig.ES.Name, config.GlobalConfig.ES.Namespace)
		return nil, err
	}

	return es, nil
}

func (h *Handler) MergeElasticsearch(ctx context.Context, name, size string) (*elasticsearchv1.Elasticsearch, error) {
	node_set := k8s.NewESNodeSet(name, size)
	es, err := h.GetElasticsearch(ctx)
	if err != nil {
		return nil, err
	}

	es.Spec.NodeSets = append(es.Spec.NodeSets, *node_set.NodeSet)
	return es, nil
}

func (h *Handler) NewRestoreESNode(ctx context.Context, name, size string) error {
	node_set := k8s.NewESNodeSet(name, size)
	es, err := h.GetElasticsearch(ctx)
	if err != nil {
		return err
	}

	patch := runtimeclient.MergeFrom(es.DeepCopy())
	es.Spec.NodeSets = append(es.Spec.NodeSets, *node_set.NodeSet)

	err = h.K8Sclient.Patch(ctx, es, patch)
	if err != nil {
		log.Error().Err(err).Msgf("failed to patch Elasticsearch of %s in %s nanespace", config.GlobalConfig.ES.Name, config.GlobalConfig.ES.Namespace)
		return err
	}

	log.Info().Msgf("success to patch Elasticsearch of %s in %s nanespace", config.GlobalConfig.ES.Name, config.GlobalConfig.ES.Namespace)
	return nil
}

func (h *Handler) DeleteRestoreESNode(ctx context.Context, name string) error {
	es, err := h.GetElasticsearch(ctx)
	if err != nil {
		return err
	}

	patch := runtimeclient.MergeFrom(es.DeepCopy())
	var node_sets []elasticsearchv1.NodeSet
	for _, node := range es.Spec.NodeSets {
		if node.Name != name {
			node_sets = append(node_sets, node)
		}
	}
	es.Spec.NodeSets = node_sets

	err = h.K8Sclient.Patch(ctx, es, patch)
	if err != nil {
		log.Error().Err(err).Msgf("failed to patch Elasticsearch of %s in %s nanespace", config.GlobalConfig.ES.Name, config.GlobalConfig.ES.Namespace)
		return err
	}

	log.Info().Msgf("success to patch Elasticsearch of %s in %s nanespace", config.GlobalConfig.ES.Name, config.GlobalConfig.ES.Namespace)
	return nil
}
