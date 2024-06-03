package controller

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/ketches/lovo/internal/consts"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PersistentVolumeReconciler reconciles a corev1.PersistentVolume object
type PersistentVolumeReconciler struct {
	log logr.Logger
	client.Client
	Scheme *runtime.Scheme
}

func (r *PersistentVolumeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx)

	var pv corev1.PersistentVolume
	if err := r.Get(ctx, req.NamespacedName, &pv); err != nil && k8serrors.IsNotFound(err) {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !IsLovoStorageClassPV(&pv) {
		return ctrl.Result{}, nil
	}

	if !LovoOnCurrentNode(&pv) {
		return ctrl.Result{}, nil
	}

	r.log.Info("Reconciling lovo PersistentVolume", "Namespace", pv.Namespace, "Name", pv.Name)

	if pv.Status.Phase != corev1.VolumeBound {
		// bound the lovo PersistentVolumeClaim
		if err := r.boundPersistentVolume(ctx, &pv); err != nil {
			r.log.Error(err, "unable to provision lovo PersistentVolume", "Namespace", pv.Namespace, "Name", pv.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *PersistentVolumeReconciler) boundPersistentVolume(ctx context.Context, pv *corev1.PersistentVolume) error {
	if pv.Status.Phase == corev1.VolumeFailed {
		return ReclaimPV(ctx, r.Client, pv)
	}

	var pvc corev1.PersistentVolumeClaim
	if err := r.Get(ctx, types.NamespacedName{Name: pv.Spec.ClaimRef.Name, Namespace: pv.Spec.ClaimRef.Namespace}, &pvc); err != nil {
		if k8serrors.IsNotFound(err) {
			return ReclaimPV(ctx, r.Client, pv)
		}
	}

	if v, ok := pv.Annotations[consts.LovoPVPathAnnotation]; ok && len(v) > 0 {
		return nil
	}

	if err := CreateDirectory(pv.Spec.Local.Path); err != nil {
		return err
	}

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.Get(ctx, types.NamespacedName{Name: pv.Name}, pv); err != nil {
			return err
		}
		pv.Annotations[consts.LovoPVPathAnnotation] = pv.Spec.Local.Path
		return r.Update(ctx, pv)
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *PersistentVolumeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.PersistentVolume{}).
		Complete(r)
}
