package multiclusterruntime

import (
	"math/rand/v2"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/platform-mesh/subroutines/spread"

	"github.com/platform-mesh/extension-manager-operator/api/v1alpha1"
)

// contentConfigurationSpread preserves golang-commons spread behavior for
// ContentConfiguration (per-type max from GenerateNextReconcileTime).
type contentConfigurationSpread struct{}

const legacyDefaultMaxReconcileDuration = 24 * time.Hour

// legacyNextReconcileDelay returns a random duration between max/2 and max
// (same algorithm as golang-commons spread.getNextReconcileTime).
func legacyNextReconcileDelay(maxReconcileTime time.Duration) time.Duration {
	minMinutes := maxReconcileTime.Minutes() / 2
	jitter := rand.Int64N(int64(minMinutes))
	return time.Duration(jitter+int64(minMinutes)) * time.Minute
}

func (contentConfigurationSpread) ReconcileRequired(obj client.Object) bool {
	cc, ok := obj.(*v1alpha1.ContentConfiguration)
	if !ok {
		return true
	}
	if obj.GetGeneration() != cc.Status.ObservedGeneration {
		return true
	}
	labels := obj.GetLabels()
	if labels != nil {
		if _, has := labels[spread.RefreshLabel]; has {
			return true
		}
	}
	nrt := cc.Status.NextReconcileTime
	if nrt.IsZero() {
		return true
	}
	return time.Now().UTC().After(nrt.UTC())
}

func (contentConfigurationSpread) RequeueDelay(obj client.Object) time.Duration {
	cc, ok := obj.(*v1alpha1.ContentConfiguration)
	if !ok {
		return 0
	}
	nrt := cc.Status.NextReconcileTime
	if nrt.IsZero() {
		return 0
	}
	remaining := time.Until(nrt.UTC())
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (contentConfigurationSpread) SetNextReconcileTime(obj client.Object) {
	cc, ok := obj.(*v1alpha1.ContentConfiguration)
	if !ok {
		return
	}
	border := legacyDefaultMaxReconcileDuration
	if g := cc.GenerateNextReconcileTime(); g > 0 {
		border = g
	}
	delay := legacyNextReconcileDelay(border)
	cc.Status.NextReconcileTime = metav1.NewTime(time.Now().Add(delay))
}

func (contentConfigurationSpread) UpdateObservedGeneration(obj client.Object) {
	cc, ok := obj.(*v1alpha1.ContentConfiguration)
	if !ok {
		return
	}
	cc.Status.ObservedGeneration = obj.GetGeneration()
}

func (contentConfigurationSpread) RemoveRefreshLabel(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}
	if _, ok := labels[spread.RefreshLabel]; !ok {
		return false
	}
	delete(labels, spread.RefreshLabel)
	obj.SetLabels(labels)
	return true
}
