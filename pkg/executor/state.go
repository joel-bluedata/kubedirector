// Copyright 2021 Hewlett Packard Enterprise Development LP

package executor

import (
	"context"

	"github.com/bluek8s/kubedirector/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// labelsForStateNamespace generates a set of resource labels appropriate for
// the kdcluster state namespace.
func labelsForStateNamespace() map[string]string {

	labels := make(map[string]string)
	labels[StateNamespaceLabel] = ""

	return labels
}

// CreateNamespace creates a namespace object in k8s, for use in kdcluster
// state storage.
func CreateNamespace(
	namespaceName string,
) error (*corev1.Namespace, error) {

	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: labelsForStateNamespace(),
		},
		// XXX TODO check why we were doing this
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceActive,
		},
	}
	return shared.Create(context.TODO(), namespace)
}

// UpdateNamespace examines a current namespace in k8s and may take steps to
// reconcile it to the desired spec.
func UpdateNamespace(
	reqLogger logr.Logger,
	cr *kdv1.KubeDirectorConfig,
	namespace *corev1.Namespace,
) error {

	labelsModified := false
	if namespace.Labels == nil {
		labelsModified = true
		namespace.Labels = labelsForStateNamespace()
	} else {
		for k, v := range labelsForStateNamespace() {
			currentValue, hasLabel := namespace.Labels[k]
			if (!hasLabel) || (hasLabel && (currentValue != v)) {
				labelsModified = true
				namespace.Labels[k] = v
			}
		}
	}

	if !labelsModified {
		return nil
	}

	shared.LogInfo(
		reqLogger,
		cr,
		shared.EventReasonConfig,
		"updating the state namespace labels",
	)

	err := shared.Update(context.TODO(), namespace)
	if err == nil {
		return nil
	}

	// See https://github.com/bluek8s/kubedirector/issues/194
	// Migrate Client().Update() calls back to Patch() calls.
	if !errors.IsConflict(err) {
		shared.LogError(
			reqLogger,
			err,
			cr,
			shared.EventReasonConfig,
			"failed to update the state namespace",
		)
		return err
	}

	// If there was a resourceVersion conflict then fetch a more
	// recent version of the object and attempt to update that.
	currentNamespace := &corev1.Namespace{}
	err = shared.Get(
		context.TODO(),
		types.NamespacedName{Name: namespace.Name},
		currentNamespace,
	)
	if err != nil {
		shared.LogError(
			reqLogger,
			err,
			cr,
			shared.EventReasonConfig,
			"failed to retrieve the state namespace",
		)
		return err
	}

	currentNamespace.Labels = namespace.Labels
	currentNamespace.OwnerReferences = namespace.OwnerReferences
	err = shared.Update(context.TODO(), currentNamespace)
	if err != nil {
		shared.LogError(
			reqLogger,
			err,
			cr,
			shared.EventReasonConfig,
			"failed to update the state namespace",
		)
	}
	return err
}

// DeleteNamespace deletes a namespace resource from k8s.
func DeleteNamespace(
	namespaceName string,
) error {

	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}
	return shared.Delete(context.TODO(), namespace)
}
