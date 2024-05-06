package provider

import (
	"context"
	"golang.org/x/net/http/httpproxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
	"net/http"
	"net/url"
	"time"
)

const helmReleaseNameAnnotationName string = "meta.helm.sh/release-name"
const helmReleaseNameAnnotationValue string = "coredns"

const helmReleaseNamespaceAnnotationName string = "meta.helm.sh/release-namespace"
const helmReleaseNamespaceAnnotationValue string = "kube-system"

const managedByLabelName string = "app.kubernetes.io/managed-by"
const managedByLabelValue string = "Helm"

const amazonManagedLabelName string = "eks.amazonaws.com/component"

func GetClient(endpoint string, timeout int64, insecure bool, caCertificate string, token string, clientCertificate string, clientKey string) (clientset *kubernetes.Clientset, err error) {
	proxy := func(req *http.Request) (*url.URL, error) {
		return httpproxy.FromEnvironment().ProxyFunc()(req.URL)
	}

	var config *rest.Config

	config = &rest.Config{
		Host: endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: insecure,
		},
		Timeout: time.Duration(timeout) * time.Millisecond,
		Proxy:   proxy,
	}

	if token != "" {
		config.BearerToken = token
	}

	if caCertificate != "" {
		config.TLSClientConfig.CAData = []byte(caCertificate)
	}

	if clientCertificate != "" {
		config.TLSClientConfig.CertData = []byte(clientCertificate)
	}

	if clientKey != "" {
		config.TLSClientConfig.KeyData = []byte(clientKey)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func DaemonsetExist(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	found := false
	for _, daemonset := range daemonsets.Items {
		if daemonset.Name == name {
			found = true
			break
		}
	}

	return found, nil
}

func DeleteDaemonset(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	err = clientset.AppsV1().DaemonSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	} else if errors.IsNotFound(err) {
		return false, nil
	} else {
		return true, nil
	}
}

func DeploymentExist(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	found := false
	for _, deployment := range deployments.Items {
		if deployment.Name == name {
			found = true
			break
		}
	}

	return found, nil
}

func DeleteDeployment(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	err = clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	} else if errors.IsNotFound(err) {
		return false, nil
	} else {
		return true, nil
	}
}

func ServiceExist(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	found := false
	for _, service := range services.Items {
		if service.Name == name {
			found = true
			break
		}
	}

	return found, nil
}

func DeleteService(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (exists bool, err error) {
	err = clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	} else if errors.IsNotFound(err) {
		return false, nil
	} else {
		return true, nil
	}
}

func DeploymentImportedIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (helmReleaseNameAnnotationSet bool, helmReleaseNamespaceAnnotationSet bool, managedByLabelSet bool, amazonManagedLabelRemoved bool, err error) {
	helmReleaseNameAnnotationSet = false
	helmReleaseNamespaceAnnotationSet = false
	managedByLabelSet = false
	amazonManagedLabelRemoved = false

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return false, false, false, false, err
	}

	if deployment.Labels == nil || deployment.Annotations == nil {
		return false, false, false, false, err
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
		return false, false, false, false, err
	}

	if service.Labels == nil || service.Annotations == nil {
		return false, false, false, false, err
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

func ImportDeploymentIntoHelm(ctx context.Context, clientset *kubernetes.Clientset, namespace string, name string) (err error) {
	patchFunc := func(deployment *appsv1.Deployment) (bool, *appsv1.Deployment) {
		updated := false
		value := ""

		if deployment.Labels == nil {
			deployment.Labels = make(map[string]string)
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

		if deployment.Annotations == nil {
			deployment.Annotations = make(map[string]string)
		}

		value, ok = deployment.Labels[managedByLabelName]
		if !ok || value != managedByLabelValue {
			updated = true
			deployment.Labels[managedByLabelName] = managedByLabelValue
		}

		_, ok = deployment.Annotations[amazonManagedLabelName]
		if ok {
			updated = true
			delete(deployment.Labels, amazonManagedLabelName)
		}

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

		if service.Labels == nil {
			service.Labels = make(map[string]string)
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

		if service.Annotations == nil {
			service.Annotations = make(map[string]string)
		}

		value, ok = service.Labels[managedByLabelName]
		if !ok || value != managedByLabelValue {
			updated = true
			service.Labels[managedByLabelName] = managedByLabelValue
		}

		_, ok = service.Annotations[amazonManagedLabelName]
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
