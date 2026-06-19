package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/bison/api-server/internal/k8s"
)

func newTestBalanceService() *BalanceService {
	cs := k8sfake.NewSimpleClientset()
	client := k8s.NewClientWithInterfaces(cs, nil)
	return NewBalanceService(client)
}

// installOptimisticConcurrency makes the fake clientset enforce resourceVersion on
// ConfigMap update so it behaves like etcd. Without this the default fake silently
// accepts stale writes and the RetryOnConflict path is never exercised. The tracker
// is the single source of truth for the current resourceVersion.
func installOptimisticConcurrency(cs *k8sfake.Clientset) {
	var mu sync.Mutex

	cs.PrependReactor("update", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		cm := action.(k8stesting.UpdateAction).GetObject().(*corev1.ConfigMap).DeepCopy()
		mu.Lock()
		defer mu.Unlock()

		existing, err := cs.Tracker().Get(action.GetResource(), action.GetNamespace(), cm.Name)
		if err != nil {
			return true, nil, err
		}
		existingCM := existing.(*corev1.ConfigMap)
		if cm.ResourceVersion != existingCM.ResourceVersion {
			return true, nil, apierrors.NewConflict(
				schema.GroupResource{Resource: "configmaps"}, cm.Name,
				fmt.Errorf("resourceVersion conflict"))
		}

		rv, _ := strconv.Atoi(existingCM.ResourceVersion)
		cm.ResourceVersion = strconv.Itoa(rv + 1)
		if err := cs.Tracker().Update(action.GetResource(), cm, action.GetNamespace()); err != nil {
			return true, nil, err
		}
		return true, cm, nil
	})
}

func seedConfigMap(name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: BisonNamespace, ResourceVersion: "1"},
		Data:       map[string]string{},
	}
}

func TestRechargeThenDeduct(t *testing.T) {
	svc := newTestBalanceService()
	ctx := context.Background()

	if err := svc.Recharge(ctx, "team-a", 100, "admin", "init"); err != nil {
		t.Fatalf("Recharge: %v", err)
	}

	bal, err := svc.GetBalance(ctx, "team-a")
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if bal.Amount != 100 {
		t.Fatalf("after recharge: got %v, want 100", bal.Amount)
	}

	newBal, err := svc.Deduct(ctx, "team-a", 30, "usage")
	if err != nil {
		t.Fatalf("Deduct: %v", err)
	}
	if newBal != 70 {
		t.Fatalf("Deduct returned %v, want 70", newBal)
	}

	bal, _ = svc.GetBalance(ctx, "team-a")
	if bal.Amount != 70 {
		t.Fatalf("stored balance %v, want 70", bal.Amount)
	}

	recs, err := svc.GetRechargeHistory(ctx, "team-a", 0)
	if err != nil {
		t.Fatalf("GetRechargeHistory: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("history records = %d, want 2", len(recs))
	}
}

func TestDeductAllowsNegativeBalance(t *testing.T) {
	svc := newTestBalanceService()
	ctx := context.Background()

	if err := svc.Recharge(ctx, "team-a", 10, "admin", ""); err != nil {
		t.Fatal(err)
	}
	newBal, err := svc.Deduct(ctx, "team-a", 25, "usage")
	if err != nil {
		t.Fatalf("Deduct: %v", err)
	}
	if newBal != -15 {
		t.Fatalf("Deduct returned %v, want -15", newBal)
	}
}

func TestRechargeRejectsNonPositive(t *testing.T) {
	svc := newTestBalanceService()
	ctx := context.Background()

	if err := svc.Recharge(ctx, "team-a", 0, "admin", ""); err == nil {
		t.Fatal("Recharge(0) should error")
	}
	if err := svc.Recharge(ctx, "team-a", -5, "admin", ""); err == nil {
		t.Fatal("Recharge(-5) should error")
	}
	if _, err := svc.Deduct(ctx, "team-a", 0, "usage"); err == nil {
		t.Fatal("Deduct(0) should error")
	}
}

// TestDeductPreservesOverdueAt guards the regression where a deduction overwrote
// the whole Balance object and silently wiped OverdueAt, which would reset the
// grace-period clock on every billing cycle and prevent suspension.
func TestDeductPreservesOverdueAt(t *testing.T) {
	svc := newTestBalanceService()
	ctx := context.Background()

	if err := svc.Recharge(ctx, "team-a", 10, "admin", ""); err != nil {
		t.Fatal(err)
	}
	overdue := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	if err := svc.SetOverdueAt(ctx, "team-a", &overdue); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Deduct(ctx, "team-a", 50, "usage"); err != nil {
		t.Fatal(err)
	}

	bal, _ := svc.GetBalance(ctx, "team-a")
	if bal.OverdueAt == nil {
		t.Fatal("OverdueAt was wiped by Deduct")
	}
	if !bal.OverdueAt.Equal(overdue) {
		t.Fatalf("OverdueAt = %v, want %v", bal.OverdueAt, overdue)
	}
	if bal.Amount != -40 {
		t.Fatalf("amount = %v, want -40", bal.Amount)
	}
}

func addDeduction(t *testing.T, svc *BalanceService, team string, amount float64, ts time.Time) {
	t.Helper()
	if err := svc.addRechargeRecord(context.Background(), team, &RechargeRecord{
		ID:        ts.Format(time.RFC3339Nano),
		Timestamp: ts,
		Type:      "deduction",
		Amount:    -amount,
		Operator:  "system",
	}); err != nil {
		t.Fatalf("addRechargeRecord: %v", err)
	}
}

func TestCalculateDailyConsumptionUsesDeductionSpan(t *testing.T) {
	svc := newTestBalanceService()
	ctx := context.Background()
	now := time.Now()

	// Two deductions spanning ~4 days, total 200 -> ~50/day.
	addDeduction(t, svc, "team-a", 100, now.Add(-4*24*time.Hour))
	addDeduction(t, svc, "team-a", 100, now)

	rate, err := svc.CalculateDailyConsumption(ctx, "team-a")
	if err != nil {
		t.Fatal(err)
	}
	if rate < 45 || rate > 55 {
		t.Fatalf("expected ~50/day, got %.2f", rate)
	}
}

func TestCalculateDailyConsumptionFloorsSpan(t *testing.T) {
	svc := newTestBalanceService()
	ctx := context.Background()

	// Single very recent deduction: span floored to 0.5 day -> 10 / 0.5 = 20.
	addDeduction(t, svc, "team-a", 10, time.Now())
	rate, err := svc.CalculateDailyConsumption(ctx, "team-a")
	if err != nil {
		t.Fatal(err)
	}
	if rate < 18 || rate > 22 {
		t.Fatalf("expected ~20/day (floored span), got %.2f", rate)
	}
}

func TestCalculateDailyConsumptionIgnoresOldAndNonDeductions(t *testing.T) {
	svc := newTestBalanceService()
	ctx := context.Background()
	now := time.Now()

	addDeduction(t, svc, "team-a", 1000, now.Add(-30*24*time.Hour)) // outside 7d window
	// a recharge inside the window must not count as consumption
	if err := svc.addRechargeRecord(ctx, "team-a", &RechargeRecord{
		ID: "r1", Timestamp: now, Type: "recharge", Amount: 500,
	}); err != nil {
		t.Fatal(err)
	}
	rate, err := svc.CalculateDailyConsumption(ctx, "team-a")
	if err != nil {
		t.Fatal(err)
	}
	if rate != 0 {
		t.Fatalf("expected 0 (no in-window deductions), got %.2f", rate)
	}
}

// TestConcurrentRecharge exercises the optimistic-concurrency retry path against a
// fake that enforces resourceVersion. Each Recharge re-reads, recomputes and
// re-writes under retry.RetryOnConflict, so the final balance must equal the sum of
// all operations with no lost updates. Contention is kept within DefaultRetry's
// budget (5 attempts) by using a small number of writers.
func TestConcurrentRecharge(t *testing.T) {
	cs := k8sfake.NewSimpleClientset(seedConfigMap(BalancesConfigMap), seedConfigMap(RechargeHistoryConfigMap))
	installOptimisticConcurrency(cs)
	svc := NewBalanceService(k8s.NewClientWithInterfaces(cs, nil))
	ctx := context.Background()

	const n = 4
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if err := svc.Recharge(ctx, "team-a", 5, "admin", "concurrent"); err != nil {
				t.Errorf("Recharge: %v", err)
			}
		}()
	}
	wg.Wait()

	bal, err := svc.GetBalance(ctx, "team-a")
	if err != nil {
		t.Fatal(err)
	}
	if bal.Amount != float64(n*5) {
		t.Fatalf("concurrent recharge total = %v, want %v (lost update)", bal.Amount, n*5)
	}
}
