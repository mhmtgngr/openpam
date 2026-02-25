package rbac

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// HandlerManager manages Kubernetes RBAC operations
type HandlerManager struct {
	k8sClient *kubernetes.Clientset
	logger    zerolog.Logger
}

// NewHandlerManager creates a new RBAC handler manager
func NewHandlerManager(k8sClient *kubernetes.Clientset, logger zerolog.Logger) (*HandlerManager, error) {
	return &HandlerManager{
		k8sClient: k8sClient,
		logger:    logger,
	}, nil
}

// CreateRoleBinding creates a role binding for JIT access
func (m *HandlerManager) CreateRoleBinding(ctx context.Context, namespace, name, role, serviceAccount, user string) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"openpam-jit":     "true",
				"managed-by":      "openpam-operator",
				"temporary-access": "true",
			},
			Annotations: map[string]string{
				"openpam.org/user":            user,
				"openpam.org/creation-time":   metav1.Now().String(),
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     role,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err := m.k8sClient.RbacV1().RoleBindings(namespace).Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("rbac.CreateRoleBinding: %w", err)
	}

	m.logger.Info().
		Str("name", name).
		Str("namespace", namespace).
		Str("role", role).
		Str("user", user).
		Msg("Role binding created")

	return nil
}

// CreateClusterRoleBinding creates a cluster role binding for cluster-wide JIT access
func (m *HandlerManager) CreateClusterRoleBinding(ctx context.Context, name, clusterRole, serviceAccount, user string) error {
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"openpam-jit":     "true",
				"managed-by":      "openpam-operator",
				"temporary-access": "true",
			},
			Annotations: map[string]string{
				"openpam.org/user": user,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: "openpam-system",
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterRole,
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err := m.k8sClient.RbacV1().ClusterRoleBindings().Create(ctx, roleBinding, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("rbac.CreateClusterRoleBinding: %w", err)
	}

	return nil
}

// DeleteRoleBinding deletes a role binding
func (m *HandlerManager) DeleteRoleBinding(ctx context.Context, namespace, name string) error {
	err := m.k8sClient.RbacV1().RoleBindings(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("rbac.DeleteRoleBinding: %w", err)
	}

	m.logger.Info().
		Str("name", name).
		Str("namespace", namespace).
		Msg("Role binding deleted")

	return nil
}

// DeleteRoleBindingsByPrefix deletes role bindings with a given prefix
func (m *HandlerManager) DeleteRoleBindingsByPrefix(ctx context.Context, prefix string) error {
	// List all namespaces
	namespaces, err := m.k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		// List role bindings with our label
		bindings, err := m.k8sClient.RbacV1().RoleBindings(ns.Name).List(ctx, metav1.ListOptions{
			LabelSelector: "openpam-jit=true",
		})
		if err != nil {
			continue
		}

		for _, binding := range bindings.Items {
			if len(binding.Name) >= len(prefix) && binding.Name[:len(prefix)] == prefix {
				_ = m.DeleteRoleBinding(ctx, ns.Name, binding.Name)
			}
		}
	}

	return nil
}

// DeleteClusterRoleBinding deletes a cluster role binding
func (m *HandlerManager) DeleteClusterRoleBinding(ctx context.Context, name string) error {
	err := m.k8sClient.RbacV1().ClusterRoleBindings().Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("rbac.DeleteClusterRoleBinding: %w", err)
	}

	return nil
}

// CreateClusterRole creates the cluster role for JIT access
func (m *HandlerManager) CreateClusterRole(ctx context.Context, name string) error {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "openpam-operator",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/log", "pods/exec"},
				Verbs:     []string{"get", "list", "create", "delete", "deletecollection"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "statefulsets", "daemonsets"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs", "cronjobs"},
				Verbs:     []string{"get", "list"},
			},
		},
	}

	_, err := m.k8sClient.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("rbac.CreateClusterRole: %w", err)
	}

	m.logger.Info().Str("name", name).Msg("Cluster role created")

	return nil
}

// CreateServiceAccount creates a service account
func (m *HandlerManager) CreateServiceAccount(ctx context.Context, namespace, name string) error {
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "openpam-operator",
			},
		},
	}

	_, err := m.k8sClient.CoreV1().ServiceAccounts(namespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("rbac.CreateServiceAccount: %w", err)
	}

	m.logger.Info().
		Str("name", name).
		Str("namespace", namespace).
		Msg("Service account created")

	return nil
}

// GetUserAccess returns the RBAC bindings for a user
func (m *HandlerManager) GetUserAccess(ctx context.Context, user string) ([]RBACAccess, error) {
	access := make([]RBACAccess, 0)

	// List all role bindings cluster-wide
	bindings, err := m.k8sClient.RbacV1().RoleBindings("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, binding := range bindings.Items {
		for _, subject := range binding.Subjects {
			if subject.Kind == "User" && subject.Name == user {
				access = append(access, RBACAccess{
					Namespace:    binding.Namespace,
					RoleBinding:  binding.Name,
					Role:         binding.RoleRef.Name,
					RoleKind:     binding.RoleRef.Kind,
				})
			}
		}
	}

	return access, nil
}

// CleanupExpiredBindings removes expired role bindings
func (m *HandlerManager) CleanupExpiredBindings(ctx context.Context, olderThan time.Time) error {
	namespaces, err := m.k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, ns := range namespaces.Items {
		bindings, err := m.k8sClient.RbacV1().RoleBindings(ns.Name).List(ctx, metav1.ListOptions{
			LabelSelector: "temporary-access=true",
		})
		if err != nil {
			continue
		}

		for _, binding := range bindings.Items {
			creationTime := binding.CreationTimestamp.Time
			if creationTime.Before(olderThan) {
				_ = m.DeleteRoleBinding(ctx, ns.Name, binding.Name)
			}
		}
	}

	return nil
}

// RBACAccess represents a user's RBAC access
type RBACAccess struct {
	Namespace   string
	RoleBinding string
	Role        string
	RoleKind    string
}
