/*
Copyright 2025 404LifeFound.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/404LifeFound/es-snapshot-restore/config"
	restorev1 "github.com/404LifeFound/es-snapshot-restore/internal/controller/api/v1"
	"github.com/404LifeFound/es-snapshot-restore/internal/db"
	"github.com/404LifeFound/es-snapshot-restore/internal/elastic"
	"github.com/404LifeFound/es-snapshot-restore/internal/k8s"
	"github.com/404LifeFound/es-snapshot-restore/internal/utils"
	esv1 "github.com/elastic/cloud-on-k8s/v3/pkg/apis/elasticsearch/v1"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

type RestoreTask struct {
	Namespace string
	Name      string
	TaskID    string
	Index     []string
}

// RestoreTaskReconciler reconciles a RestoreTask object
type RestoreTaskReconciler struct {
	client.Client
	ESClient  *elastic.ES
	DBClient  *gorm.DB
	Scheme    *runtime.Scheme
	taskQueue chan *RestoreTask // taskID queue
	sem       chan struct{}     // concurrent queue
}

func (r *RestoreTaskReconciler) StartWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case task := <-r.taskQueue:
				r.sem <- struct{}{}
				go func(task *RestoreTask) {
					defer func() { <-r.sem }()
					err := r.restoreIndices(task)
					if err != nil {
						r.updateTaskStatus(ctx, task, RestoreStatusFailed)
					} else {
						r.updateTaskStatus(ctx, task, RestoreStatusDone)
					}
				}(task)
			}
		}
	}()
}

func (r *RestoreTaskReconciler) updateTaskStatus(ctx context.Context, task *RestoreTask, status string) {
	var restore_task restorev1.RestoreTask
	if err := r.Get(ctx, client.ObjectKey{Namespace: task.Namespace, Name: task.Name}, &restore_task); err != nil {
		log.Error().Err(err).Msgf("Failed to get RestoreTask: %s", task.Name)
		return
	}

	restore_task.Status.FinishedAt = utils.PtrToAny(metav1.Now())
	restore_task.Status.Status = status
	if err := r.Client.Update(ctx, &restore_task); err != nil {
		log.Error().Err(err).Msgf("failed to update RestoreTask %s", restore_task.Name)
		// TODO retry
	}
}

func (r *RestoreTaskReconciler) restoreIndices(task *RestoreTask) error {
	t, err := db.QueryAll[db.Task](r.DBClient, "", 0, "task_id = ? and index = ?", task.TaskID, task.Index[0])
	if err != nil {
		log.Error().Err(err).Msgf("failed to query task id %s for index %s", task.TaskID, task.Index[0])
		return err
	}

	if len(t) != 1 {
		log.Error().Err(err).Msgf("records for task_id %s of index %s not equal 1 but %d", task.TaskID, task.Index[0], len(t))
		return err
	}

	task_one := t[0]

	log.Info().Msgf("restoring index %s from snapshot %s", task_one.Index, task_one.Snapshot)
	if err := r.ESClient.Restore(
		context.Background(),
		task_one.Repository,
		task_one.Snapshot,
		config.GlobalConfig.ES.RestoreKey,
		config.GlobalConfig.ES.RestoreKey,
		task_one.Repository,
		[]string{task_one.Index},
	); err != nil {
		log.Error().Err(err).Msgf("failed to restore index %s from snapshot %s", task_one.Index, task_one.Snapshot)
		if err := r.DBClient.Model(&t).Updates(map[string]any{
			"Status":       string(utils.TaskFailed),
			"ErrorMessage": utils.PtrToAny(fmt.Sprintf("failed to restore index %s from snapshot %s", task_one.Index, task_one.Snapshot)),
		}).Error; err != nil {
			log.Error().Err(err).Msgf("failed to update status and error_message for task id %s of index %s", task_one.TaskID, task_one.Index)
		}
		return err
	}

	restoreTimeout := time.Duration(config.GlobalConfig.ES.Timeout) * time.Minute
	pollInterval := time.Duration(config.GlobalConfig.ES.Interval) * time.Second

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	timeout := time.After(restoreTimeout)

	for {
		select {
		case <-ticker.C:
			res, err := r.ESClient.GetRestoreIndexProcess([]string{task_one.Index})
			if err != nil {
				log.Error().Err(err).Msgf("failed to check the recovery process of restore index %s from snapshot %s", task_one.Index, task_one.Snapshot)
				continue
			}

			if len(res) == 0 {
				log.Warn().Msgf("no recovery info found for index %s, retrying...", task_one.Index)
				continue
			}

			log.Info().Msgf("restore progress of index %s: %s", task_one.Index, res[0].RecoveredPercent)

			if res[0].RecoveredPercent == "100%" {
				log.Info().Msgf("restore of index %s completed successfully", task_one.Index)
				if err := r.DBClient.Model(&t).Updates(map[string]any{
					"Status": string(utils.TaskSuccess),
				}).Error; err != nil {
					log.Error().Err(err).Msgf("failed to update status for task id %s of index %s when task success", task_one.TaskID, task_one.Index)
					continue
				}
				return nil
			}

		case <-timeout:
			if err := r.DBClient.Model(&t).Updates(map[string]any{
				"Status": string(utils.TaskTimeout),
			}).Error; err != nil {
				log.Error().Err(err).Msgf("failed to update status for task id %s of index %s when task timeout", task_one.TaskID, task_one.Index)
			}
			return fmt.Errorf("restore of index %s timed out after %s", task_one.Index, restoreTimeout)
		}
	}
}

// +kubebuilder:rbac:groups=restore.restore.elastic.co,resources=restoretasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=restore.restore.elastic.co,resources=restoretasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=restore.restore.elastic.co,resources=restoretasks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RestoreTask object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *RestoreTaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var restore_task restorev1.RestoreTask
	if err := r.Get(ctx, req.NamespacedName, &restore_task); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if restore_task.Status.StartAt == nil {
		restore_task.Status.StartAt = utils.PtrToAny(metav1.Now())
		if err := r.Update(ctx, &restore_task); err != nil {
			return ctrl.Result{}, err
		}
	}

	var es esv1.Elasticsearch

	es_ns := restore_task.Spec.ElasticsearchRef.Namespace
	es_name := restore_task.Spec.ElasticsearchRef.Name
	if es_ns == "" {
		es_ns = restore_task.Namespace
	}

	if err := r.Get(ctx, client.ObjectKey{Namespace: es_ns, Name: es_name}, &es); err != nil {
		log.Error().Err(err).Msgf("Elasticsearch of %s in %s Namespace not found: %v", es_name, es_ns, err)
		// requeue
		return ctrl.Result{}, err
	}

	// es nodesets
	var exist_node esv1.NodeSet
	var node_exist bool
	var exist_node_index int
	for i, n := range es.Spec.NodeSets {
		if n.Name == restore_task.Spec.NodeName {
			node_exist = true
			exist_node = n
			exist_node_index = i
		}
	}

	if !node_exist {
		log.Info().Msgf("node % not exists, so create it", restore_task.Spec.NodeName)
		restore_node := k8s.NewESNodeSet(restore_task.Spec.NodeName, restore_task.Spec.StoreSize)
		original_es := es.DeepCopy()
		es.Spec.NodeSets = append(es.Spec.NodeSets, *restore_node.NodeSet)
		if err := r.Patch(ctx, &es, client.MergeFrom(original_es)); err != nil {
			log.Error().Err(err).Msgf("failed to patch Elasticsearch of %s in %s Namespace to add new node: %s", es_name, es_ns, restore_task.Spec.NodeName)
			return ctrl.Result{}, err
		}
	} else {
		store_size, err := utils.ToGB(restore_task.Spec.StoreSize)
		storage_quanlity := exist_node.VolumeClaimTemplates[0].Spec.Resources.Requests[corev1.ResourceStorage]
		exist_node_storage := float64(storage_quanlity.Value()) / (1024 * 1024 * 1024)
		if err != nil {
			log.Error().Err(err).Msgf("failed to transfer %s to float of RestoreTask %s", restore_task.Spec.StoreSize, restore_task.Name)
			return ctrl.Result{}, err
		}
		if store_size > exist_node_storage {
			original_es := es.DeepCopy()
			es.Spec.NodeSets[exist_node_index].VolumeClaimTemplates[0].Spec.Resources = corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%sGi", store_size)),
				},
			}
			if err := r.Patch(ctx, &es, client.MergeFrom(original_es)); err != nil {
				log.Error().Err(err).Msgf("failed to patch Elasticsearch of %s in %s Namespace to add new node: %s", es_name, es_ns, restore_task.Spec.NodeName)
				return ctrl.Result{}, err
			}
		}
	}

	// check sts status
	var sts appsv1.StatefulSet
	sts_name := fmt.Sprintf("%s-es-%s", es_name, restore_task.Spec.NodeName)
	if err := r.Get(ctx, client.ObjectKey{Namespace: es_ns, Name: sts_name}, &sts); err != nil {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, err
	}

	// ensure the sts owned by Elasticsearch
	for _, owner := range sts.OwnerReferences {
		if owner.Kind == "Elasticsearch" &&
			owner.APIVersion == "elasticsearch.k8s.elastic.co/v1" {
			return ctrl.Result{}, fmt.Errorf("statefulset %s not owned by Elasticsearch %s", sts_name, es.Name)
		}
	}

	if sts.Status.ReadyReplicas != *sts.Spec.Replicas {
		log.Info().Str("sts", sts.Name).Msg("StatefulSet not ready yet")
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	r.taskQueue <- &RestoreTask{
		TaskID: restore_task.Spec.TaskId,
		Index:  restore_task.Spec.Indices,
	}

	return ctrl.Result{}, nil
}

func (r *RestoreTaskReconciler) filterCreate(e event.CreateEvent) bool {
	restore_task, ok := e.Object.(*restorev1.RestoreTask)
	if !ok {
		return false
	}

	return r.match(restore_task)
}

func (r *RestoreTaskReconciler) filterUpdate(e event.UpdateEvent) bool {
	old_restore_task, ok1 := e.ObjectOld.(*restorev1.RestoreTask)
	new_restore_task, ok2 := e.ObjectNew.(*restorev1.RestoreTask)
	if !ok1 || !ok2 {
		return false
	}

	oldAnn := old_restore_task.GetAnnotations()
	newAnn := new_restore_task.GetAnnotations()
	if oldAnn == nil || newAnn == nil {
		return false
	}
	return r.match(new_restore_task)
}

func (r *RestoreTaskReconciler) match(restore_task *restorev1.RestoreTask) bool {
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestoreTaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&restorev1.RestoreTask{}).
		//WithEventFilter(predicate.Funcs{
		//CreateFunc:  r.filterCreate,
		//UpdateFunc:  r.filterUpdate,
		//DeleteFunc:  func(e event.DeleteEvent) bool { return false },
		//GenericFunc: func(e event.GenericEvent) bool { return false },
		//}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("restoretask").
		Complete(r)
}

func NewRestoreTaskReconciler(c client.Client, s *runtime.Scheme, es *elastic.ES, db *gorm.DB) *RestoreTaskReconciler {
	return &RestoreTaskReconciler{
		Client:    c,
		Scheme:    s,
		ESClient:  es,
		DBClient:  db,
		taskQueue: make(chan *RestoreTask, config.GlobalConfig.ES.MaxTasks),
		sem:       make(chan struct{}, config.GlobalConfig.ES.Concurrency),
	}
}

func NewManager() (*ctrl.Manager, error) {
	scheme := runtime.NewScheme()
	//setupLog := ctrl.Log.WithName("setup")
	ctrl.SetLogger(zap.New())
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(esv1.AddToScheme(scheme))
	utilruntime.Must(restorev1.AddToScheme(scheme))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache:  cache.Options{},
	})
	if err != nil {
		return nil, err
	}
	return &mgr, nil
}

func NewRestoreReconcilerCtrl(lc fx.Lifecycle, mgr *ctrl.Manager, es_client *elastic.ES, db_client *gorm.DB) *RestoreTaskReconciler {
	r := NewRestoreTaskReconciler(
		(*mgr).GetClient(),
		(*mgr).GetScheme(),
		es_client,
		db_client,
	)

	r.StartWorker(context.Background())
	r.SetupWithManager(*mgr)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info().Msg("restart worker start")
			r.StartWorker(ctx)
			return nil
		},
		OnStop: func(context.Context) error {
			log.Info().Msg("restart worker stop")
			return nil
		},
	})
	return r
}

func RunManager(lc fx.Lifecycle, mgr *ctrl.Manager, _ *RestoreTaskReconciler) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Info().Msg("controller start")
			go (*mgr).Start(ctrl.SetupSignalHandler())
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Info().Msg("controller stop")
			return nil
		},
	})
}
