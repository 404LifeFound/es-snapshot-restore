package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/404LifeFound/es-snapshot-restore/config"
	"github.com/404LifeFound/es-snapshot-restore/internal/db"
	"github.com/404LifeFound/es-snapshot-restore/internal/elastic"
	"github.com/404LifeFound/es-snapshot-restore/internal/k8s"
	"github.com/404LifeFound/es-snapshot-restore/internal/utils"
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
	Name    []string `form:"name" binding:"required,min=1" json:"name"`
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

	storage_size := h.GetIndexGBSize(all_result)
	if storage_size < config.GlobalConfig.ES.DiskMinSize {
		storage_size = config.GlobalConfig.ES.DiskMinSize
	}

	c.JSON(http.StatusOK, gin.H{
		"index":      h.GetIndexNames(all_result),
		"store_size": fmt.Sprintf("%fGi", storage_size),
	})
}

type RestoreSnapshotRequest struct {
	Name    []string `form:"name" binding:"required,min=1" json:"name"`
	Node    string   `form:"node" json:"node"`
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
			"message": fmt.Sprintf("invalid post data: %s,can't bind post data to RestoreSnapshotRequest", err.Error()),
		})
		return
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

	storage_size := r.GetIndexGBSize(matched_indices)
	if storage_size < config.GlobalConfig.ES.DiskMinSize {
		storage_size = config.GlobalConfig.ES.DiskMinSize
	}

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
		"store_size":     fmt.Sprintf("%fGi", storage_size),
	})
}

type CreateRestoreNodeRequest struct {
	TaskID string `json:"task_id" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Size   string `json:"size" binding:"required"`
}

func (h *Handler) CreateRestoreNode(c *gin.Context) {
	var create_restore_node_req CreateRestoreNodeRequest
	if c.ContentType() != "application/json" {
		err := fmt.Errorf("content type should be application/json")
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("invalid content type %s, please use application/json", c.ContentType()),
		})
		return
	}
	err := c.ShouldBindJSON(&create_restore_node_req)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("faild to parse delete_restore_node_req: %s", err.Error()),
		})
		return
	}

	task, err := db.QueryAll[db.Task](h.DBClient, "", 0, "task_id = ?", create_restore_node_req.TaskID)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("failed to get task id %s: %s", create_restore_node_req.TaskID, err.Error()),
		})
		return
	}

	if len(task) != 1 || len(task) == 0 {
		err := fmt.Errorf("The task_id of %s have %d record which not equal 1", create_restore_node_req.TaskID, len(task))
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("%s", err.Error()),
		})
		return

	}

	t := task[0]
	log.Info().Msgf("update task stage from %s to %s and status from %s to %s for %s...",
		t.CurrentStage,
		string(utils.StagCreateESNode),
		t.Status,
		string(utils.TaskRunning),
		create_restore_node_req.TaskID,
	)
	if err := h.DBClient.Model(&t).Updates(map[string]any{
		"CurrentStage": string(utils.StagCreateESNode),
		"Status":       string(utils.TaskRunning),
		"UpdatedAt":    time.Now(),
	}).Error; err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to update task of %s: %s", create_restore_node_req.TaskID, err.Error()),
		})
		return
	}
	log.Debug().Msgf("updated task stage from %s to %s and status from %s to %s for %s",
		t.CurrentStage,
		string(utils.StagCreateESNode),
		t.Status,
		string(utils.TaskRunning),
		create_restore_node_req.TaskID,
	)

	//t.CurrentStage = string(utils.StagCreateESNode)
	//t.Status = string(utils.TaskRunning)
	//h.DBClient.Save(t)

	err = h.NewRestoreESNode(c.Request.Context(), create_restore_node_req.Name, create_restore_node_req.Size)
	if err != nil {
		c.Error(err)
		if dberr := h.DBClient.Model(&t).Updates(map[string]any{
			"Status":    string(utils.TaskFailed),
			"UpdatedAt": time.Now(),
		}).Error; dberr != nil {
			c.Error(dberr)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message": fmt.Sprintf("failed to update task status of %s: %s", create_restore_node_req.TaskID, dberr.Error()),
			})
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to create restore node %s with store size %s: %s", create_restore_node_req.Name, create_restore_node_req.Size, err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("success to create restore node %s with store size %s", create_restore_node_req.Name, create_restore_node_req.Size),
		"task_id": create_restore_node_req.TaskID,
		"name":    create_restore_node_req.Name,
		"size":    create_restore_node_req.Size,
	})
}

func (h *Handler) NewTask(c *gin.Context) {
	task := db.Task{
		TaskID:       utils.TaskID(),
		Status:       string(utils.TaskPending),
		CurrentStage: string(utils.StagInit),
		StartedAt:    utils.PtrToAny(time.Now()),
	}

	if err := db.CreateRecords(h.DBClient, &[]db.Task{task}); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("faild to create task %s", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success to create restore",
		"task_id": task.TaskID,
	})
}

type DeleteRestoreNodeRequest struct {
	Name string `json:"name" binding:"required"`
}

func (h *Handler) DeleteRestoreNode(c *gin.Context) {
	var delete_restore_node_req DeleteRestoreNodeRequest
	err := c.ShouldBindJSON(&delete_restore_node_req)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("faild to parse delete_restore_node_req: %s", err.Error()),
		})
		return
	}

	err = h.DeleteRestoreESNode(c.Request.Context(), delete_restore_node_req.Name)
	if err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"message": fmt.Sprintf("failed to delete restore node %s: %s", delete_restore_node_req.Name, err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("success to delete restore node %s", delete_restore_node_req.Name),
	})
}

// TODO
type RestoreSnapshotOneStepRequest struct {
	TaskID string   `json:"task_id" binding:"required"`
	Name   []string `form:"name" binding:"required,min=1" json:"name"`
	Node   string   `form:"node" json:"node"`
}

func (h *Handler) RestoreSnapshotOneStep(c *gin.Context) {
	if c.ContentType() != "application/json" {
		err := fmt.Errorf("content type should be application/json")
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("invalid content type %s, please use application/json", c.ContentType()),
		})
		return
	}

	var r RestoreSnapshotOneStepRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		c.Error(err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"message": fmt.Sprintf("invalid post data: %s,can't bind post data to RestoreSnapshotRequest", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "success",
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
	e.PUT("/task", handler.NewTask)
	e.PUT("/node", handler.CreateRestoreNode)
	e.DELETE("/node", handler.DeleteRestoreNode)
	e.GET("/debug", handler.DebugHandler)
	return nil
}
