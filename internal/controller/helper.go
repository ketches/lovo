package controller

import (
	"context"
	"os"

	"github.com/ketches/lovo/internal/consts"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsLovoStorageClassPVC(pvc *corev1.PersistentVolumeClaim) bool {
	return pvc.Spec.StorageClassName != nil && *pvc.Spec.StorageClassName == consts.LovoStorageClassName
}

func IsLovoStorageClassPV(pv *corev1.PersistentVolume) bool {
	return pv.Spec.StorageClassName == consts.LovoStorageClassName
}

func CreateDirectory(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func DeleteDirectory(path string) error {
	return os.RemoveAll(path)
}

func LovoOnCurrentNode(pv *corev1.PersistentVolume) bool {
	return pv.Annotations[consts.LovoPVNodeAnnotation] == os.Getenv("NODE_NAME")
}

func ReclaimPV(ctx context.Context, client client.Client, pv *corev1.PersistentVolume) error {
	// PersistentVolumeClaim is not found, delete the lovo PersistentVolume
	if err := client.Delete(ctx, pv); err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	// Remove lovo path
	if v, ok := pv.Annotations[consts.LovoPVPathAnnotation]; ok && len(v) > 0 {
		return DeleteDirectory(v)
	}
	return nil
}
