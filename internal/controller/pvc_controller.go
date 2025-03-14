package controller

import (
	"context"
	"math/rand"
	"path"
	"time"

	"github.com/go-logr/logr"
	"github.com/ketches/lovo/internal/consts"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PersistentVolumeClaimReconciler reconciles a corev1.PersistentVolumeClaim object
type PersistentVolumeClaimReconciler struct {
	log logr.Logger
	client.Client
	Scheme *runtime.Scheme
}

func (r *PersistentVolumeClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx)

	var pvc corev1.PersistentVolumeClaim
	if err := r.Get(ctx, req.NamespacedName, &pvc); err != nil && k8serrors.IsNotFound(err) {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !IsLovoStorageClassPVC(&pvc) {
		return ctrl.Result{}, nil
	}

	r.log.Info("Reconciling lovo PersistentVolumeClaim", "Namespace", pvc.Namespace, "Name", pvc.Name)

	if pvc.DeletionTimestamp != nil {
		// PersistentVolumeClaim is deleted, reclaim the lovo PersistentVolume
		if err := r.reclaimPersistentVolume(ctx, &pvc); err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, err
		}
	}

	if err := r.provisionPersistentVolume(ctx, &pvc); err != nil {
		r.log.Error(err, "unable to provision lovo PersistentVolume", "Namespace", pvc.Namespace, "Name", pvc.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PersistentVolumeClaimReconciler) provisionPersistentVolume(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	pvName := targetPersistentVolumeName(pvc)
	lovoPath := targetPersistentVolumeLocalPath(pvc)
	targetNode := r.getTargetNode()

	// Check if the lovo PersistentVolume already exists, if not, create it
	var pv corev1.PersistentVolume
	if err := r.Get(ctx, client.ObjectKey{Namespace: pvc.Namespace, Name: pvName}, &pv); err != nil && k8serrors.IsNotFound(err) {
		// Create a new PersistentVolume
		pv = corev1.PersistentVolume{
			ObjectMeta: metav1.ObjectMeta{
				Name: pvName,
				Labels: map[string]string{
					consts.LovoPVCNamespaceLabel: pvc.GetNamespace(),
					consts.LovoPVCNameLabel:      pvc.GetName(),
				},
				Annotations: map[string]string{
					consts.LovoPVNodeAnnotation: targetNode.GetName(),
				},
			},
			Spec: corev1.PersistentVolumeSpec{
				StorageClassName: consts.LovoStorageClassName,
				Capacity: corev1.ResourceList{
					corev1.ResourceStorage: pvc.Spec.Resources.Requests[corev1.ResourceStorage],
				},
				AccessModes:                   pvc.Spec.AccessModes,
				VolumeMode:                    pvc.Spec.VolumeMode,
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
				PersistentVolumeSource: corev1.PersistentVolumeSource{
					Local: &corev1.LocalVolumeSource{
						Path: lovoPath,
					},
				},
				ClaimRef: &corev1.ObjectReference{
					Namespace: pvc.Namespace,
					Name:      pvc.Name,
				},
				NodeAffinity: &corev1.VolumeNodeAffinity{
					Required: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      consts.HostnameSelectorKey,
										Operator: corev1.NodeSelectorOpIn,
										Values: []string{
											targetNode.GetName(),
										},
									},
								},
							},
						},
					},
				},
			},
		}

		if err := r.Create(ctx, &pv); err != nil {
			return err
		}

		r.log.Info("Created lovo PersistentVolume", "Namespace", pvc.Namespace, "Name", pvName)
	}

	return nil
}

func (r *PersistentVolumeClaimReconciler) reclaimPersistentVolume(ctx context.Context, pvc *corev1.PersistentVolumeClaim) error {
	pvName := targetPersistentVolumeName(pvc)

	var pv corev1.PersistentVolume
	if err := r.Get(ctx, types.NamespacedName{Name: pvName}, &pv); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	if !LovoOnCurrentNode(&pv) {
		return nil
	}

	ReclaimPV(ctx, r.Client, &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: pvName,
		},
	})

	return nil
}

func targetPersistentVolumeName(pvc *corev1.PersistentVolumeClaim) string {
	return "pvc-" + string(pvc.GetUID())
}

func targetPersistentVolumeLocalPath(pvc *corev1.PersistentVolumeClaim) string {
	return path.Join(consts.LovoPersistentVolumePathPrefix, pvc.GetNamespace(), string(pvc.GetUID()))
}

func (r *PersistentVolumeClaimReconciler) getTargetNode() *corev1.Node {
	var nodes corev1.NodeList
	if err := r.Client.List(context.Background(), &nodes); err != nil {
		r.log.Error(err, "unable to list nodes")
		return nil
	}

	no := rand.Intn(len(nodes.Items))
	res := nodes.Items[no]
	// Find the node with the most available ephemeral storage
	for i, node := range nodes.Items {
		if i == no {
			continue
		}

		resStorage := res.Status.Allocatable[corev1.ResourceEphemeralStorage]
		nodeStorage := node.Status.Allocatable[corev1.ResourceEphemeralStorage]

		if resStorage.Cmp(nodeStorage) < 0 {
			res = node
		}
	}

	return &res
}

// SetupWithManager sets up the controller with the Manager.
func (r *PersistentVolumeClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
