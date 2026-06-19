// Package leader provides Kubernetes lease-based leader election so that
// singleton background work (the billing/auto-recharge/alert scheduler) runs on
// exactly one api-server replica at a time, even when scaled horizontally.
package leader

import (
	"context"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/bison/api-server/pkg/logger"
)

// LeaseName is the coordination.k8s.io Lease used to elect the scheduler leader.
const LeaseName = "bison-scheduler"

// identity returns a per-process identity. In Kubernetes the pod name is the
// hostname, which is unique per replica; POD_NAME overrides it when set.
func identity() string {
	if v := os.Getenv("POD_NAME"); v != "" {
		return v
	}
	if h, err := os.Hostname(); err == nil && h != "" {
		return h
	}
	return "bison-api"
}

func namespace(def string) string {
	if v := os.Getenv("POD_NAMESPACE"); v != "" {
		return v
	}
	return def
}

// Run blocks running leader election until ctx is cancelled.
//
// onStarted is invoked (in its own goroutine) with a context that is cancelled
// when leadership is lost or ctx is cancelled; it should start the leader-only
// work and return promptly when its context is done. onStopped is invoked when
// leadership is lost. The scheduler must be re-startable, since leadership can be
// re-acquired after a transient loss.
func Run(ctx context.Context, clientset kubernetes.Interface, ns string, onStarted func(context.Context), onStopped func()) {
	id := identity()
	leaseNS := namespace(ns)

	lock := &resourcelock.LeaseLock{
		LeaseMeta:  metav1.ObjectMeta{Name: LeaseName, Namespace: leaseNS},
		Client:     clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{Identity: id},
	}

	logger.Info("Starting leader election", "identity", id, "namespace", leaseNS, "lease", LeaseName)

	config := leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(leaderCtx context.Context) {
				logger.Info("Acquired scheduler leadership", "identity", id)
				onStarted(leaderCtx)
			},
			OnStoppedLeading: func() {
				logger.Warn("Lost scheduler leadership", "identity", id)
				onStopped()
			},
			OnNewLeader: func(current string) {
				if current != id {
					logger.Info("Observed scheduler leader", "leader", current)
				}
			},
		},
	}

	// RunOrDie returns when ctx is cancelled or leadership is lost. Loop so a
	// transient loss leads to re-election rather than permanently idle.
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		leaderelection.RunOrDie(ctx, config)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}
