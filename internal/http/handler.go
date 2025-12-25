package http

import (
	"fmt"
	"net/http"

	"github.com/404LifeFound/es-snapshot-restore/config"
	"github.com/404LifeFound/es-snapshot-restore/internal/elastic"
	"github.com/404LifeFound/es-snapshot-restore/internal/k8s"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	ESClient  *elastic.ES
	DBClient  *gorm.DB
	K8Sclient runtimeclient.Client
}

type RestoreSnapshotHandler struct {
	*Handler
}

type QueryIndexParam struct {
	Name    []string `form:"name" binding:"required" json:"name"`
	StartAt string   `form:"start_at" json:"start_at"`
	EndAt   string   `form:"end_at" json:"end_at"`
}

func (h *Handler) QueryIndex(c *gin.Context) {
	var p QueryIndexParam
	if err := c.ShouldBindQuery(&p); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("failed to bind query param to QueryIndexParam: %s", err.Error()),
		})
		return
	}

	all_result, err := h.QueryIndexResultViaTime(p.Name, p.StartAt, p.EndAt)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to query index to meet the condition: %s", err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"all_index": all_result,
	})
}

type RestoreSnapshotRequest struct {
	Name    []string `form:"name" binding:"required" json:"name"`
	StartAt string   `form:"start_at" json:"start_at"`
	EndAt   string   `form:"end_at" json:"end_at"`
}

func (r *RestoreSnapshotHandler) RestoreSnapshot(c *gin.Context) {
	if c.ContentType() != "application/json" {
		err := fmt.Errorf("content type should be application/json")
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("invalid content type %s, please use application/json", c.ContentType()),
		})
		return
	}

	var restore_snapshot_request RestoreSnapshotRequest
	if err := c.ShouldBindJSON(&restore_snapshot_request); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("invalid post data: %s", err.Error()),
		})
	}

	matched_indices, err := r.QueryIndexResultViaTime(restore_snapshot_request.Name, restore_snapshot_request.StartAt, restore_snapshot_request.EndAt)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to query index to meet the condition: %s", err.Error()),
		})
		return
	}

	log.Info().Msgf("total index size is: %.6f", r.GetIndexGBSize(matched_indices))

	map_index_snapshot, err := r.QueryLatestSnapshotsViaIndex(matched_indices)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to query snapshot to meet the indices condition: %s", err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"index_snapshot": map_index_snapshot,
	})
}

func (h *Handler) DebugHandler(c *gin.Context) {
	err := h.DeleteRestoreESNode(c.Request.Context(), "restore-xxxx")
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("can't get elasticsearch %s from namespace %s: %s", config.GlobalConfig.ES.Name, config.GlobalConfig.ES.Namespace, err.Error()),
		})

	}
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
	})
}
func RegisterHandler(e *gin.Engine, es_client *elastic.ES, db_client *gorm.DB, k8s_client *k8s.Client) error {
	handler := &Handler{
		ESClient:  es_client,
		DBClient:  db_client,
		K8Sclient: k8s_client,
	}

	restore_snaphost_handler := &RestoreSnapshotHandler{Handler: handler}

	e.GET("/indices", handler.QueryIndex)
	e.POST("/restore", restore_snaphost_handler.RestoreSnapshot)
	e.GET("/debug", handler.DebugHandler)
	return nil
}
