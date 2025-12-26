package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/fx"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/404LifeFound/es-snapshot-restore/config"
	"github.com/rs/zerolog/log"
)

type RestoreTask struct {
	TaskID    string
	Namespace string
	Name      string
}

type RestoreReconciler struct {
	client.Client
	taskQueue chan *RestoreTask // taskID queue
}

func NewRestoreReconciler(c client.Client) *RestoreReconciler {
	return &RestoreReconciler{
		Client:    c,
		taskQueue: make(chan *RestoreTask, 100),
	}
}

func (r *RestoreReconciler) StartWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case RestoreTask := <-r.taskQueue:
				log.Info().Msgf("Executing restore task %s", RestoreTask)
				err := restoreIndices(RestoreTask)
				if err != nil {
					r.updateTaskStatus(ctx, RestoreTask, RestoreStatusFailed)
				} else {
					r.updateTaskStatus(ctx, RestoreTask, RestoreStatusDone)
				}
			}
		}
	}()
}
func (r *RestoreReconciler) updateTaskStatus(ctx context.Context, task *RestoreTask, status string) {
	var sts appsv1.StatefulSet
	if err := r.Get(ctx, client.ObjectKey{Namespace: task.Namespace, Name: task.Name}, &sts); err != nil {
		log.Error().Err(err).Msg("Failed to get StatefulSet")
		return
	}

	ann := sts.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}

	if ann[AnnotationRestoreTaskID] != task.TaskID {
		log.Warn().Str("sts", task.Name).Str("taskID", task.TaskID).Msg("taskID mismatch, skip update")
		return
	}

	original := sts.DeepCopy()
	ann[AnnotationRestoreTaskStatus] = status
	sts.SetAnnotations(ann)
	if err := r.Patch(ctx, &sts, client.MergeFrom(original)); err != nil {
		log.Error().Err(err).Str("sts", task.Name).Msg("Failed to patch StatefulSet")
	}
}

func (r *RestoreReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	var sts appsv1.StatefulSet
	if err := r.Get(ctx, req.NamespacedName, &sts); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if sts.Status.ReadyReplicas != *sts.Spec.Replicas {
		log.Info().
			Str("sts", sts.Name).
			Msg("StatefulSet not ready yet")
		return ctrl.Result{}, nil
	}

	ann := sts.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}

	taskID := ann[AnnotationRestoreTaskID]
	state := ann[AnnotationRestoreTaskStatus]

	if state != RestoreStatusPending {
		log.Info().
			Str("sts", sts.Name).
			Str("taskID", taskID).
			Str("state", state).
			Msg("Restore task already running or done, skip")
		return ctrl.Result{}, nil
	}

	original := sts.DeepCopy()
	ann[AnnotationRestoreTaskStatus] = RestoreStatusRunning
	sts.SetAnnotations(ann)
	if err := r.Patch(ctx, &sts, client.MergeFrom(original)); err != nil {
		return ctrl.Result{}, err
	}

	r.taskQueue <- &RestoreTask{
		TaskID:    taskID,
		Namespace: req.Namespace,
		Name:      req.Name,
	}

	return ctrl.Result{}, nil
}

func (r *RestoreReconciler) filterCreate(e event.CreateEvent) bool {
	sts, ok := e.Object.(*appsv1.StatefulSet)
	if !ok {
		return false
	}

	return r.match(sts)
}

func (r *RestoreReconciler) filterUpdate(e event.UpdateEvent) bool {
	oldSts, ok1 := e.ObjectOld.(*appsv1.StatefulSet)
	newSts, ok2 := e.ObjectNew.(*appsv1.StatefulSet)
	if !ok1 || !ok2 {
		return false
	}

	oldAnn := oldSts.GetAnnotations()
	newAnn := newSts.GetAnnotations()
	if oldAnn == nil || newAnn == nil {
		return false
	}

	if oldAnn[AnnotationRestoreTaskID] != newAnn[AnnotationRestoreTaskID] ||
		oldAnn[AnnotationRestoreTaskStatus] != newAnn[AnnotationRestoreTaskStatus] {
		return r.match(newSts)
	}

	return false
}

func (r *RestoreReconciler) match(sts *appsv1.StatefulSet) bool {

	if !strings.Contains(sts.Name, fmt.Sprintf("%s-", config.GlobalConfig.ES.RestoreKey)) {
		return false
	}

	// ensure restore.elastic.co/task-id and  restore.elastic.co/state anntation exist
	annotaions := sts.GetAnnotations()
	if annotaions == nil || annotaions[AnnotationRestoreTaskID] == "" || annotaions[AnnotationRestoreTaskStatus] == "" {
		return false
	}

	// ensure the sts owned by Elasticsearch
	for _, owner := range sts.OwnerReferences {
		if owner.Kind == ElasticsearchKind &&
			owner.APIVersion == ElasticsearchAPIVersion {
			return true
		}
	}

	return false
}

func restoreIndices(task *RestoreTask) error {
	// TODO: 调用 Elasticsearch restore API
	log.Info().Msgf("Restoring indices for task %s", task.TaskID)
	time.Sleep(5 * time.Second) // 模拟耗时
	log.Info().Msgf("Restore task %s completed", task.TaskID)
	return nil
}

func (r *RestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc:  r.filterCreate,
			UpdateFunc:  r.filterUpdate,
			DeleteFunc:  func(e event.DeleteEvent) bool { return false },
			GenericFunc: func(e event.GenericEvent) bool { return false },
		}).
		Complete(r)
}

func NewManager() (*ctrl.Manager, error) {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Cache: cache.Options{
			DefaultNamespaces: map[string]cache.Config{
				config.GlobalConfig.ES.Namespace: {},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return &mgr, nil
}

func NewRestoreReconcilerCtrl(mgr *ctrl.Manager) *RestoreReconciler {
	r := NewRestoreReconciler((*mgr).GetClient())
	log.Info().Msg("restore worker start")
	r.StartWorker(context.Background())
	r.SetupWithManager(*mgr)
}

func RunManager(lc fx.Lifecycle, mgr *ctrl.Manager) {
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
