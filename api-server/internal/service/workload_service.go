package service

import (
	"context"
	"time"

	"github.com/bison/api-server/internal/k8s"
	"github.com/bison/api-server/pkg/logger"
)

// WorkloadSummary represents a summary of workloads in a namespace
type WorkloadSummary struct {
	Deployments  int `json:"deployments"`
	StatefulSets int `json:"statefulSets"`
	Pods         int `json:"pods"`       // Orphan pods (not managed by controllers)
	Jobs         int `json:"jobs"`
	CronJobs     int `json:"cronJobs"`
	TotalPods    int `json:"totalPods"`  // Total pods including controller-managed ones
}

// Workload represents a single workload resource
type Workload struct {
	Kind      string    `json:"kind"`      // Deployment, StatefulSet, Pod, Job, CronJob
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Replicas  int32     `json:"replicas"`  // Desired replicas (for scalable resources)
	Ready     int32     `json:"ready"`     // Ready replicas
	Status    string    `json:"status"`    // Running, Pending, Failed, Succeeded, etc.
	Image     string    `json:"image,omitempty"` // Main container image
	CreatedAt time.Time `json:"createdAt"`
}

// WorkloadService handles workload-related operations
type WorkloadService struct {
	k8sClient *k8s.Client
}

// NewWorkloadService creates a new WorkloadService
func NewWorkloadService(k8sClient *k8s.Client) *WorkloadService {
	return &WorkloadService{
		k8sClient: k8sClient,
	}
}

// GetWorkloadSummary returns a summary of workloads in a namespace
func (s *WorkloadService) GetWorkloadSummary(ctx context.Context, namespace string) (*WorkloadSummary, error) {
	logger.Debug("Getting workload summary", "namespace", namespace)

	summary := &WorkloadSummary{}

	// Count deployments
	deployments, err := s.k8sClient.ListDeployments(ctx, namespace)
	if err != nil {
		logger.Warn("Failed to list deployments", "namespace", namespace, "error", err)
	} else {
		summary.Deployments = len(deployments.Items)
	}

	// Count statefulsets
	statefulSets, err := s.k8sClient.ListStatefulSets(ctx, namespace)
	if err != nil {
		logger.Warn("Failed to list statefulsets", "namespace", namespace, "error", err)
	} else {
		summary.StatefulSets = len(statefulSets.Items)
	}

	// Count jobs
	jobs, err := s.k8sClient.ListJobs(ctx, namespace, "")
	if err != nil {
		logger.Warn("Failed to list jobs", "namespace", namespace, "error", err)
	} else {
		summary.Jobs = len(jobs.Items)
	}

	// Count cronjobs
	cronJobs, err := s.k8sClient.ListCronJobs(ctx, namespace)
	if err != nil {
		logger.Warn("Failed to list cronjobs", "namespace", namespace, "error", err)
	} else {
		summary.CronJobs = len(cronJobs.Items)
	}

	// Count pods
	pods, err := s.k8sClient.ListPods(ctx, namespace, "")
	if err != nil {
		logger.Warn("Failed to list pods", "namespace", namespace, "error", err)
	} else {
		summary.TotalPods = len(pods.Items)
		// Count orphan pods (not managed by any controller)
		for _, pod := range pods.Items {
			if len(pod.OwnerReferences) == 0 {
				summary.Pods++
			}
		}
	}

	return summary, nil
}

// ListWorkloads returns all workloads in a namespace
func (s *WorkloadService) ListWorkloads(ctx context.Context, namespace string) ([]*Workload, error) {
	logger.Debug("Listing workloads", "namespace", namespace)

	var workloads []*Workload

	// List deployments
	deployments, err := s.k8sClient.ListDeployments(ctx, namespace)
	if err != nil {
		logger.Warn("Failed to list deployments", "namespace", namespace, "error", err)
	} else {
		for _, deploy := range deployments.Items {
			image := ""
			if len(deploy.Spec.Template.Spec.Containers) > 0 {
				image = deploy.Spec.Template.Spec.Containers[0].Image
			}

			status := "Running"
			if deploy.Status.AvailableReplicas == 0 && *deploy.Spec.Replicas > 0 {
				status = "Pending"
			} else if deploy.Status.AvailableReplicas < *deploy.Spec.Replicas {
				status = "Progressing"
			}

			workloads = append(workloads, &Workload{
				Kind:      "Deployment",
				Name:      deploy.Name,
				Namespace: deploy.Namespace,
				Replicas:  *deploy.Spec.Replicas,
				Ready:     deploy.Status.ReadyReplicas,
				Status:    status,
				Image:     image,
				CreatedAt: deploy.CreationTimestamp.Time,
			})
		}
	}

	// List statefulsets
	statefulSets, err := s.k8sClient.ListStatefulSets(ctx, namespace)
	if err != nil {
		logger.Warn("Failed to list statefulsets", "namespace", namespace, "error", err)
	} else {
		for _, sts := range statefulSets.Items {
			image := ""
			if len(sts.Spec.Template.Spec.Containers) > 0 {
				image = sts.Spec.Template.Spec.Containers[0].Image
			}

			status := "Running"
			if sts.Status.ReadyReplicas == 0 && *sts.Spec.Replicas > 0 {
				status = "Pending"
			} else if sts.Status.ReadyReplicas < *sts.Spec.Replicas {
				status = "Progressing"
			}

			workloads = append(workloads, &Workload{
				Kind:      "StatefulSet",
				Name:      sts.Name,
				Namespace: sts.Namespace,
				Replicas:  *sts.Spec.Replicas,
				Ready:     sts.Status.ReadyReplicas,
				Status:    status,
				Image:     image,
				CreatedAt: sts.CreationTimestamp.Time,
			})
		}
	}

	// List jobs
	jobs, err := s.k8sClient.ListJobs(ctx, namespace, "")
	if err != nil {
		logger.Warn("Failed to list jobs", "namespace", namespace, "error", err)
	} else {
		for _, job := range jobs.Items {
			image := ""
			if len(job.Spec.Template.Spec.Containers) > 0 {
				image = job.Spec.Template.Spec.Containers[0].Image
			}

			status := "Running"
			if job.Status.Succeeded > 0 {
				status = "Succeeded"
			} else if job.Status.Failed > 0 {
				status = "Failed"
			} else if job.Status.Active > 0 {
				status = "Running"
			} else {
				status = "Pending"
			}

			workloads = append(workloads, &Workload{
				Kind:      "Job",
				Name:      job.Name,
				Namespace: job.Namespace,
				Replicas:  1,
				Ready:     job.Status.Succeeded,
				Status:    status,
				Image:     image,
				CreatedAt: job.CreationTimestamp.Time,
			})
		}
	}

	// List cronjobs
	cronJobs, err := s.k8sClient.ListCronJobs(ctx, namespace)
	if err != nil {
		logger.Warn("Failed to list cronjobs", "namespace", namespace, "error", err)
	} else {
		for _, cj := range cronJobs.Items {
			image := ""
			if len(cj.Spec.JobTemplate.Spec.Template.Spec.Containers) > 0 {
				image = cj.Spec.JobTemplate.Spec.Template.Spec.Containers[0].Image
			}

			status := "Active"
			if cj.Spec.Suspend != nil && *cj.Spec.Suspend {
				status = "Suspended"
			}

			workloads = append(workloads, &Workload{
				Kind:      "CronJob",
				Name:      cj.Name,
				Namespace: cj.Namespace,
				Replicas:  int32(len(cj.Status.Active)),
				Ready:     int32(len(cj.Status.Active)),
				Status:    status,
				Image:     image,
				CreatedAt: cj.CreationTimestamp.Time,
			})
		}
	}

	// List orphan pods (not managed by any controller)
	pods, err := s.k8sClient.ListPods(ctx, namespace, "")
	if err != nil {
		logger.Warn("Failed to list pods", "namespace", namespace, "error", err)
	} else {
		for _, pod := range pods.Items {
			if len(pod.OwnerReferences) == 0 {
				image := ""
				if len(pod.Spec.Containers) > 0 {
					image = pod.Spec.Containers[0].Image
				}

				status := string(pod.Status.Phase)

				workloads = append(workloads, &Workload{
					Kind:      "Pod",
					Name:      pod.Name,
					Namespace: pod.Namespace,
					Replicas:  1,
					Ready:     boolToInt32(pod.Status.Phase == "Running"),
					Status:    status,
					Image:     image,
					CreatedAt: pod.CreationTimestamp.Time,
				})
			}
		}
	}

	return workloads, nil
}

func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

