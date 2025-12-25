package http

import (
	"fmt"
	"net/http"

	"github.com/404LifeFound/es-snapshot-restore/internal/elastic"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type Handler struct {
	ESClient *elastic.ES
	DBClient *gorm.DB
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

func RegisterHandler(e *gin.Engine, es_client *elastic.ES, db_client *gorm.DB) error {
	handler := &Handler{
		ESClient: es_client,
		DBClient: db_client,
	}

	restore_snaphost_handler := &RestoreSnapshotHandler{Handler: handler}

	e.GET("/indices", handler.QueryIndex)
	e.POST("/restore", restore_snaphost_handler.RestoreSnapshot)
	return nil
}
