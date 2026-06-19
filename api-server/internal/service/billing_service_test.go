package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/internal/opencost"
)

func newTestBillingService(resourceConfigs []ResourceDefinition) *BillingService {
	data, _ := json.Marshal(resourceConfigs)
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceConfigName,
			Namespace: ResourceConfigNamespace,
		},
		Data: map[string]string{ResourceConfigDataKey: string(data)},
	}
	cs := k8sfake.NewSimpleClientset(cm)
	client := k8s.NewClientWithInterfaces(cs, nil)
	rcSvc := NewResourceConfigService(client)
	balSvc := NewBalanceService(client)
	return NewBillingService(client, nil, balSvc, nil, nil, rcSvc)
}

func TestCalculateCostWithConfiguredPrices(t *testing.T) {
	svc := newTestBillingService([]ResourceDefinition{
		{Name: "cpu", Enabled: true, Price: 0.1, Category: CategoryCompute},
		{Name: "memory", Enabled: true, Price: 0.05, Category: CategoryMemory},
		{Name: "nvidia.com/gpu", Enabled: true, Price: 8, Category: CategoryAccelerator},
	})

	config := &BillingConfig{Enabled: true}
	alloc := &opencost.Allocation{
		CPUCoreHours: 10,
		RAMGBHours:   20,
		GPUHours:     2,
		// Fallback costs that must be ignored when a configured price exists.
		CPUCost: 999,
		RAMCost: 999,
		GPUCost: 999,
	}

	got := svc.calculateCost(context.Background(), config, alloc)
	want := 10*0.1 + 20*0.05 + 2*8.0 // 1 + 1 + 16 = 18
	if got != want {
		t.Fatalf("calculateCost = %v, want %v", got, want)
	}
}

func TestCalculateCostFallsBackToAllocationCost(t *testing.T) {
	// No enabled/priced resources configured -> use OpenCost's own cost numbers.
	svc := newTestBillingService([]ResourceDefinition{})

	config := &BillingConfig{Enabled: true}
	alloc := &opencost.Allocation{
		CPUCost: 1.5,
		RAMCost: 2.5,
		GPUCost: 6.0,
	}

	got := svc.calculateCost(context.Background(), config, alloc)
	want := 1.5 + 2.5 + 6.0
	if got != want {
		t.Fatalf("calculateCost fallback = %v, want %v", got, want)
	}
}

func TestCalculateCostDisabledReturnsTotalCost(t *testing.T) {
	svc := newTestBillingService([]ResourceDefinition{})

	alloc := &opencost.Allocation{TotalCost: 42}
	if got := svc.calculateCost(context.Background(), nil, alloc); got != 42 {
		t.Fatalf("calculateCost(nil config) = %v, want 42", got)
	}
	if got := svc.calculateCost(context.Background(), &BillingConfig{Enabled: false}, alloc); got != 42 {
		t.Fatalf("calculateCost(disabled) = %v, want 42", got)
	}
}

func TestIsGracePeriodExpired(t *testing.T) {
	s := &BillingService{}
	recent := time.Now().Add(-1 * time.Hour)
	old := time.Now().Add(-72 * time.Hour)

	cases := []struct {
		name      string
		cfg       *BillingConfig
		overdueAt *time.Time
		want      bool
	}{
		{"nil overdue", &BillingConfig{GracePeriodValue: 1, GracePeriodUnit: "days"}, nil, false},
		{"within hours", &BillingConfig{GracePeriodValue: 24, GracePeriodUnit: "hours"}, &recent, false},
		{"expired hours", &BillingConfig{GracePeriodValue: 1, GracePeriodUnit: "hours"}, &recent, true},
		{"within days", &BillingConfig{GracePeriodValue: 7, GracePeriodUnit: "days"}, &old, false},
		{"expired days", &BillingConfig{GracePeriodValue: 1, GracePeriodUnit: "days"}, &old, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.isGracePeriodExpired(tc.cfg, tc.overdueAt); got != tc.want {
				t.Fatalf("isGracePeriodExpired = %v, want %v", got, tc.want)
			}
		})
	}
}
