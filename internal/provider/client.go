package provider

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

const helmReleaseNameAnnotationName string = "meta.helm.sh/release-name"
const helmReleaseNameAnnotationValue string = "coredns"

const helmReleaseNamespaceAnnotationName string = "meta.helm.sh/release-namespace"
const helmReleaseNamespaceAnnotationValue string = "kube-system"

const managedByLabelName string = "app.kubernetes.io/managed-by"
const managedByLabelValue string = "Helm"

const amazonManagedLabelName string = "eks.amazonaws.com/component"

func DaemonsetExist(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	_, err = clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func DeleteDaemonset(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	err = clientset.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func DeploymentExist(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	_, err = clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func DeleteDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	err = clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func ServiceExist(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	_, err = clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func DeleteService(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	err = clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func DeleteServiceAccount(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	err = clientset.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func DeleteConfigMap(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	err = clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func ConfigMapExist(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	_, err = clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func DeletePodDisruptionBudget(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	err = clientset.PolicyV1().PodDisruptionBudgets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		return true, nil
	}
}

func DeploymentExistsAndIsAwsOne(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	switch {
	case err != nil && !errors.IsNotFound(err):
		return false, err
	case errors.IsNotFound(err):
		return false, nil
	default:
		if deployment.Labels == nil {
			return false, nil
		}

		_, ok := deployment.Annotations[amazonManagedLabelName]
		return ok, nil
	}
}

func DeploymentImportedIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (helmReleaseNameAnnotationSet bool, helmReleaseNamespaceAnnotationSet bool, managedByLabelSet bool, amazonManagedLabelRemoved bool, err error) {
	helmReleaseNameAnnotationSet = false
	helmReleaseNamespaceAnnotationSet = false
	managedByLabelSet = false
	amazonManagedLabelRemoved = false

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return true, true, true, true, nil
		} else {
			return false, false, false, false, err
		}
	}

	if deployment.Labels == nil {
		deployment.Labels = map[string]string{}
	}

	if deployment.Annotations == nil {
		deployment.Annotations = map[string]string{}
	}

	value, ok := deployment.Annotations[helmReleaseNameAnnotationName]
	if ok && value == helmReleaseNameAnnotationValue {
		helmReleaseNameAnnotationSet = true
	}

	value, ok = deployment.Annotations[helmReleaseNamespaceAnnotationName]
	if ok && value == helmReleaseNamespaceAnnotationValue {
		helmReleaseNamespaceAnnotationSet = true
	}

	value, ok = deployment.Labels[managedByLabelName]
	if ok && value == managedByLabelValue {
		managedByLabelSet = true
	}

	_, ok = deployment.Annotations[amazonManagedLabelName]
	if !ok {
		amazonManagedLabelRemoved = true
	}

	return helmReleaseNameAnnotationSet, helmReleaseNamespaceAnnotationSet, managedByLabelSet, amazonManagedLabelRemoved, nil
}

func ServiceImportedIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (helmReleaseNameAnnotationSet bool, helmReleaseNamespaceAnnotationSet bool, managedByLabelSet bool, amazonManagedLabelRemoved bool, err error) {
	helmReleaseNameAnnotationSet = false
	helmReleaseNamespaceAnnotationSet = false
	managedByLabelSet = false
	amazonManagedLabelRemoved = false

	service, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return true, true, true, true, nil
		} else {
			return false, false, false, false, err
		}
	}

	if service.Labels == nil {
		service.Labels = map[string]string{}
	}

	if service.Annotations == nil {
		service.Annotations = map[string]string{}
	}

	value, ok := service.Annotations[helmReleaseNameAnnotationName]
	if ok && value == helmReleaseNameAnnotationValue {
		helmReleaseNameAnnotationSet = true
	}

	value, ok = service.Annotations[helmReleaseNamespaceAnnotationName]
	if ok && value == helmReleaseNamespaceAnnotationValue {
		helmReleaseNamespaceAnnotationSet = true
	}

	value, ok = service.Labels[managedByLabelName]
	if ok && value == managedByLabelValue {
		managedByLabelSet = true
	}

	_, ok = service.Annotations[amazonManagedLabelName]
	if !ok {
		amazonManagedLabelRemoved = true
	}

	return helmReleaseNameAnnotationSet, helmReleaseNamespaceAnnotationSet, managedByLabelSet, amazonManagedLabelRemoved, nil
}

func ServiceAccountImportedIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (helmReleaseNameAnnotationSet bool, helmReleaseNamespaceAnnotationSet bool, managedByLabelSet bool, amazonManagedLabelRemoved bool, err error) {
	helmReleaseNameAnnotationSet = false
	helmReleaseNamespaceAnnotationSet = false
	managedByLabelSet = false
	amazonManagedLabelRemoved = false

	serviceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return true, true, true, true, nil
		} else {
			return false, false, false, false, err
		}
	}

	if serviceAccount.Labels == nil {
		serviceAccount.Labels = map[string]string{}
	}

	if serviceAccount.Annotations == nil {
		serviceAccount.Annotations = map[string]string{}
	}

	value, ok := serviceAccount.Annotations[helmReleaseNameAnnotationName]
	if ok && value == helmReleaseNameAnnotationValue {
		helmReleaseNameAnnotationSet = true
	}

	value, ok = serviceAccount.Annotations[helmReleaseNamespaceAnnotationName]
	if ok && value == helmReleaseNamespaceAnnotationValue {
		helmReleaseNamespaceAnnotationSet = true
	}

	value, ok = serviceAccount.Labels[managedByLabelName]
	if ok && value == managedByLabelValue {
		managedByLabelSet = true
	}

	_, ok = serviceAccount.Annotations[amazonManagedLabelName]
	if !ok {
		amazonManagedLabelRemoved = true
	}

	return helmReleaseNameAnnotationSet, helmReleaseNamespaceAnnotationSet, managedByLabelSet, amazonManagedLabelRemoved, nil
}

func PodDisruptionBudgetImportedIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (helmReleaseNameAnnotationSet bool, helmReleaseNamespaceAnnotationSet bool, managedByLabelSet bool, amazonManagedLabelRemoved bool, err error) {
	helmReleaseNameAnnotationSet = false
	helmReleaseNamespaceAnnotationSet = false
	managedByLabelSet = false
	amazonManagedLabelRemoved = false

	podDisruptionBudget, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return true, true, true, true, nil
		} else {
			return false, false, false, false, err
		}
	}

	if podDisruptionBudget.Labels == nil {
		podDisruptionBudget.Labels = map[string]string{}
	}

	if podDisruptionBudget.Annotations == nil {
		podDisruptionBudget.Annotations = map[string]string{}
	}

	value, ok := podDisruptionBudget.Annotations[helmReleaseNameAnnotationName]
	if ok && value == helmReleaseNameAnnotationValue {
		helmReleaseNameAnnotationSet = true
	}

	value, ok = podDisruptionBudget.Annotations[helmReleaseNamespaceAnnotationName]
	if ok && value == helmReleaseNamespaceAnnotationValue {
		helmReleaseNamespaceAnnotationSet = true
	}

	value, ok = podDisruptionBudget.Labels[managedByLabelName]
	if ok && value == managedByLabelValue {
		managedByLabelSet = true
	}

	_, ok = podDisruptionBudget.Annotations[amazonManagedLabelName]
	if !ok {
		amazonManagedLabelRemoved = true
	}

	return helmReleaseNameAnnotationSet, helmReleaseNamespaceAnnotationSet, managedByLabelSet, amazonManagedLabelRemoved, nil
}

func ConfigMapImportedIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (helmReleaseNameAnnotationSet bool, helmReleaseNamespaceAnnotationSet bool, managedByLabelSet bool, amazonManagedLabelRemoved bool, err error) {
	helmReleaseNameAnnotationSet = false
	helmReleaseNamespaceAnnotationSet = false
	managedByLabelSet = false
	amazonManagedLabelRemoved = false

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return true, true, true, true, nil
		} else {
			return false, false, false, false, err
		}
	}

	if configMap.Labels == nil {
		configMap.Labels = map[string]string{}
	}

	if configMap.Annotations == nil {
		configMap.Annotations = map[string]string{}
	}

	value, ok := configMap.Annotations[helmReleaseNameAnnotationName]
	if ok && value == helmReleaseNameAnnotationValue {
		helmReleaseNameAnnotationSet = true
	}

	value, ok = configMap.Annotations[helmReleaseNamespaceAnnotationName]
	if ok && value == helmReleaseNamespaceAnnotationValue {
		helmReleaseNamespaceAnnotationSet = true
	}

	value, ok = configMap.Labels[managedByLabelName]
	if ok && value == managedByLabelValue {
		managedByLabelSet = true
	}

	_, ok = configMap.Labels[amazonManagedLabelName]
	if !ok {
		amazonManagedLabelRemoved = true
	}

	return helmReleaseNameAnnotationSet, helmReleaseNamespaceAnnotationSet, managedByLabelSet, amazonManagedLabelRemoved, nil
}

func ImportDeploymentIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (err error) {
	patchFunc := func(deployment *appsv1.Deployment) (bool, *appsv1.Deployment) {
		updated := false
		value := ""

		if deployment.Annotations == nil {
			deployment.Annotations = make(map[string]string)
		}

		value, ok := deployment.Annotations[helmReleaseNameAnnotationName]
		if !ok || value != helmReleaseNameAnnotationValue {
			updated = true
			deployment.Annotations[helmReleaseNameAnnotationName] = helmReleaseNameAnnotationValue
		}

		value, ok = deployment.Annotations[helmReleaseNamespaceAnnotationName]
		if !ok || value != helmReleaseNamespaceAnnotationValue {
			updated = true
			deployment.Annotations[helmReleaseNamespaceAnnotationName] = helmReleaseNamespaceAnnotationValue
		}

		if deployment.Labels == nil {
			deployment.Labels = make(map[string]string)
		}

		value, ok = deployment.Labels[managedByLabelName]
		if !ok || value != managedByLabelValue {
			updated = true
			deployment.Labels[managedByLabelName] = managedByLabelValue
		}

		_, ok = deployment.Labels[amazonManagedLabelName]
		if ok {
			updated = true
			delete(deployment.Labels, amazonManagedLabelName)
		}

		// Create a copy of the existing template
		_, ok = deployment.Spec.Template.ObjectMeta.Labels[amazonManagedLabelName]
		if ok {
			newTemplate := deployment.Spec.Template.DeepCopy()
			if newTemplate.ObjectMeta.Labels == nil {
				newTemplate.ObjectMeta.Labels = make(map[string]string)
			}

			_, ok = newTemplate.ObjectMeta.Labels[amazonManagedLabelName]
			if ok {
				updated = true
				delete(newTemplate.ObjectMeta.Labels, amazonManagedLabelName)
			}

			deployment.Spec.Template = *newTemplate
		}

		/*
			if deployment.Spec.Selector.MatchLabels == nil {
				deployment.Spec.Selector.MatchLabels = make(map[string]string)
			}

			// We can only remove the label if it is not part of the selector.MatchLabels. selector.MatchLabels are immutable.
			_, ok = deployment.Spec.Selector.MatchLabels[amazonManagedLabelName]
			if !ok {
				if deployment.Spec.Template.ObjectMeta.Labels == nil {
					deployment.Spec.Template.ObjectMeta.Labels = make(map[string]string)
				}

				// Update template spec so that pods will get the correct labels
				_, ok = deployment.Spec.Template.ObjectMeta.Labels[amazonManagedLabelName]
				if ok {
					updated = true
					delete(deployment.Spec.Template.ObjectMeta.Labels, amazonManagedLabelName)
				}
			}
		*/

		return updated, deployment
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		updated, updatedDeployment := patchFunc(deployment.DeepCopy())
		if !updated {
			return nil
		}

		_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, updatedDeployment, metav1.UpdateOptions{})
		return err
	})
	return err
}

func ImportServiceIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (err error) {
	patchFunc := func(service *corev1.Service) (bool, *corev1.Service) {
		updated := false
		value := ""

		if service.Annotations == nil {
			service.Annotations = make(map[string]string)
		}

		value, ok := service.Annotations[helmReleaseNameAnnotationName]
		if !ok || value != helmReleaseNameAnnotationValue {
			updated = true
			service.Annotations[helmReleaseNameAnnotationName] = helmReleaseNameAnnotationValue
		}

		value, ok = service.Annotations[helmReleaseNamespaceAnnotationName]
		if !ok || value != helmReleaseNamespaceAnnotationValue {
			updated = true
			service.Annotations[helmReleaseNamespaceAnnotationName] = helmReleaseNamespaceAnnotationValue
		}

		if service.Labels == nil {
			service.Labels = make(map[string]string)
		}

		value, ok = service.Labels[managedByLabelName]
		if !ok || value != managedByLabelValue {
			updated = true
			service.Labels[managedByLabelName] = managedByLabelValue
		}

		_, ok = service.Labels[amazonManagedLabelName]
		if ok {
			updated = true
			delete(service.Labels, amazonManagedLabelName)
		}

		return updated, service
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		service, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		updated, updatedService := patchFunc(service.DeepCopy())
		if !updated {
			return nil
		}

		_, err = clientset.CoreV1().Services(namespace).Update(ctx, updatedService, metav1.UpdateOptions{})
		return err
	})
	return err
}

func ImportServiceAccountIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (err error) {
	patchFunc := func(serviceAccount *corev1.ServiceAccount) (bool, *corev1.ServiceAccount) {
		updated := false
		value := ""

		if serviceAccount.Annotations == nil {
			serviceAccount.Annotations = make(map[string]string)
		}

		value, ok := serviceAccount.Annotations[helmReleaseNameAnnotationName]
		if !ok || value != helmReleaseNameAnnotationValue {
			updated = true
			serviceAccount.Annotations[helmReleaseNameAnnotationName] = helmReleaseNameAnnotationValue
		}

		value, ok = serviceAccount.Annotations[helmReleaseNamespaceAnnotationName]
		if !ok || value != helmReleaseNamespaceAnnotationValue {
			updated = true
			serviceAccount.Annotations[helmReleaseNamespaceAnnotationName] = helmReleaseNamespaceAnnotationValue
		}

		if serviceAccount.Labels == nil {
			serviceAccount.Labels = make(map[string]string)
		}

		value, ok = serviceAccount.Labels[managedByLabelName]
		if !ok || value != managedByLabelValue {
			updated = true
			serviceAccount.Labels[managedByLabelName] = managedByLabelValue
		}

		_, ok = serviceAccount.Labels[amazonManagedLabelName]
		if ok {
			updated = true
			delete(serviceAccount.Labels, amazonManagedLabelName)
		}

		return updated, serviceAccount
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		serviceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		updated, updatedServiceAccount := patchFunc(serviceAccount.DeepCopy())
		if !updated {
			return nil
		}

		_, err = clientset.CoreV1().ServiceAccounts(namespace).Update(ctx, updatedServiceAccount, metav1.UpdateOptions{})
		return err
	})
	return err
}

func ImportPodDisruptionBudgetIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (err error) {
	patchFunc := func(serviceAccount *policyv1.PodDisruptionBudget) (bool, *policyv1.PodDisruptionBudget) {
		updated := false
		value := ""

		if serviceAccount.Annotations == nil {
			serviceAccount.Annotations = make(map[string]string)
		}

		value, ok := serviceAccount.Annotations[helmReleaseNameAnnotationName]
		if !ok || value != helmReleaseNameAnnotationValue {
			updated = true
			serviceAccount.Annotations[helmReleaseNameAnnotationName] = helmReleaseNameAnnotationValue
		}

		value, ok = serviceAccount.Annotations[helmReleaseNamespaceAnnotationName]
		if !ok || value != helmReleaseNamespaceAnnotationValue {
			updated = true
			serviceAccount.Annotations[helmReleaseNamespaceAnnotationName] = helmReleaseNamespaceAnnotationValue
		}

		if serviceAccount.Labels == nil {
			serviceAccount.Labels = make(map[string]string)
		}

		value, ok = serviceAccount.Labels[managedByLabelName]
		if !ok || value != managedByLabelValue {
			updated = true
			serviceAccount.Labels[managedByLabelName] = managedByLabelValue
		}

		_, ok = serviceAccount.Labels[amazonManagedLabelName]
		if ok {
			updated = true
			delete(serviceAccount.Labels, amazonManagedLabelName)
		}

		return updated, serviceAccount
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		podDisruptionBudget, err := clientset.PolicyV1().PodDisruptionBudgets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		updated, updatedPodDisruptionBudget := patchFunc(podDisruptionBudget.DeepCopy())
		if !updated {
			return nil
		}

		_, err = clientset.PolicyV1().PodDisruptionBudgets(namespace).Update(ctx, updatedPodDisruptionBudget, metav1.UpdateOptions{})
		return err
	})
	return err
}

func ImportConfigMapAccountIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (err error) {
	patchFunc := func(configMap *corev1.ConfigMap) (bool, *corev1.ConfigMap) {
		updated := false
		value := ""

		if configMap.Annotations == nil {
			configMap.Annotations = make(map[string]string)
		}

		value, ok := configMap.Annotations[helmReleaseNameAnnotationName]
		if !ok || value != helmReleaseNameAnnotationValue {
			updated = true
			configMap.Annotations[helmReleaseNameAnnotationName] = helmReleaseNameAnnotationValue
		}

		value, ok = configMap.Annotations[helmReleaseNamespaceAnnotationName]
		if !ok || value != helmReleaseNamespaceAnnotationValue {
			updated = true
			configMap.Annotations[helmReleaseNamespaceAnnotationName] = helmReleaseNamespaceAnnotationValue
		}

		if configMap.Labels == nil {
			configMap.Labels = make(map[string]string)
		}

		value, ok = configMap.Labels[managedByLabelName]
		if !ok || value != managedByLabelValue {
			updated = true
			configMap.Labels[managedByLabelName] = managedByLabelValue
		}

		_, ok = configMap.Labels[amazonManagedLabelName]
		if ok {
			updated = true
			delete(configMap.Labels, amazonManagedLabelName)
		}

		return updated, configMap
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		updated, updatedConfigMap := patchFunc(configMap.DeepCopy())
		if !updated {
			return nil
		}

		_, err = clientset.CoreV1().ConfigMaps(namespace).Update(ctx, updatedConfigMap, metav1.UpdateOptions{})
		return err
	})
	return err
}
