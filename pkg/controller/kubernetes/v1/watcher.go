package v1

import (
	"context"
	"warptail/pkg/router"

	"github.com/gosimple/slug"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type WarpTailServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Router *router.Router
}

const myFinalizerName = "warptail.exceptionerror.io/finalizer"

func (r *WarpTailServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var wtservice WarpTailService
	if err := r.Get(ctx, req.NamespacedName, &wtservice); err != nil {
		return ctrl.Result{}, nil
	}

	if err := r.UpdateService(ctx, wtservice); err != nil {
		return ctrl.Result{}, err
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if wtservice.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&wtservice, myFinalizerName) {
			controllerutil.AddFinalizer(&wtservice, myFinalizerName)
			if err := r.Update(ctx, &wtservice); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(&wtservice, myFinalizerName) {
			if err := r.RemoveService(ctx, wtservice); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&wtservice, myFinalizerName)
			if err := r.Update(ctx, &wtservice); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WarpTailServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&WarpTailService{}).
		Named("warptailservice").
		Complete(r)
}

func (r *WarpTailServiceReconciler) UpdateService(ctx context.Context, svc WarpTailService) *router.RouterError {
	logger := log.FromContext(ctx)
	defer r.Router.Save()
	id := slug.Make(svc.Name)
	if _, err := r.Router.Get(id); err != nil {

		logger.Info("creating", "service", svc.Name)
		_, err := r.Router.Create(svc.ToServiceConfig())
		return err
	}
	logger.Info("updating", "service", svc.Name)
	_, err := r.Router.Update(id, svc.ToServiceConfig())
	return err
}

func (r *WarpTailServiceReconciler) RemoveService(ctx context.Context, svc WarpTailService) *router.RouterError {
	logger := log.FromContext(ctx)
	defer r.Router.Save()
	logger.Info("Removing", "service", svc.Name)
	id := slug.Make(svc.Name)
	return r.Router.Remove(id)
}
