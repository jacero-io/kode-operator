/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Configuration constants
const (
	ContainerRestartPolicyAlways corev1.ContainerRestartPolicy = "Always"
	PersistentVolumeClaimName                                  = "kode-pvc"
)

type KodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// logKodeManifest add structured logging for the Kode manifest to improve visibility for debugging
func logKodeManifest(log logr.Logger, kode *kodev1alpha1.Kode) {
	mask := func(s string) string {
		if len(s) > 4 {
			return s[:2] + "****" + s[len(s)-2:]
		}
		return "****"
	}
	log.V(1).Info("Kode Manifest",
		"Name", kode.Name,
		"Namespace", kode.Namespace,
		"TemplateRef", kode.Spec.TemplateRef,
		"User", kode.Spec.User,
		"Password", mask(kode.Spec.Password),
		"HomeDir", kode.Spec.Home,
		"Workspace", kode.Spec.Workspace,
		"Storage", fmt.Sprintf("%v", kode.Spec.Storage),
	)
}

func logKodeTemplateManifest(log logr.Logger, kodeTemplate *kodev1alpha1.KodeTemplate) {
	log.V(1).Info("Kode Template Manifest",
		"Name", kodeTemplate.Name,
		"Namespace", kodeTemplate.Namespace,
		"Image", kodeTemplate.Spec.Image,
		"TZ", kodeTemplate.Spec.TZ,
		"PUID", kodeTemplate.Spec.PUID,
		"PGID", kodeTemplate.Spec.PGID,
		"Port", kodeTemplate.Spec.Port,
		"Envs", fmt.Sprintf("%v", kodeTemplate.Spec.Envs),
		"Args", fmt.Sprintf("%v", kodeTemplate.Spec.Args),
		"Home", kodeTemplate.Spec.Home,
		"DefaultWorkspace", kodeTemplate.Spec.DefaultWorkspace,
	)
}

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithName("Reconcile")
	// Fetch the Kode instance
	kode := &kodev1alpha1.Kode{}
	log.Info("Fetching Kode instance", "Namespace", req.Namespace, "Name", req.Name)
	if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to fetch Kode instance", "Namespace", req.Namespace, "Name", req.Name)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	logKodeManifest(log, kode)

	// Validate references
	if err := r.validateReferences(kode); err != nil {
		log.Error(err, "Invalid references in Kode", "Namespace", kode.Namespace, "Name", kode.Name)
		return ctrl.Result{}, err
	}

	// Fetch the KodeTemplate instance referenced by TemplateRef
	kodeTemplate := &kodev1alpha1.KodeTemplate{}
	kodeTemplateName := types.NamespacedName{Name: kode.Spec.TemplateRef.Name}
	if kode.Spec.TemplateRef.Name != "" {
		log.Info("Fetching KodeTemplate instance", "Name", kodeTemplateName)
		if err := r.Get(ctx, kodeTemplateName, kodeTemplate); err != nil {
			if errors.IsNotFound(err) {
				log.Info("KodeTemplate instance not found, requeuing", "Name", kodeTemplateName)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil // Retry after some time
			}
			log.Error(err, "Failed to fetch KodeTemplate instance", "Name", kode.Spec.TemplateRef.Name)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}
	logKodeTemplateManifest(log, kodeTemplate)

	// Fetch the EnvoyProxyTemplate instance referenced by EnvoyProxyTemplateReference
	envoyProxyTemplate := &kodev1alpha1.EnvoyProxyTemplate{}
	envoyProxyTemplateName := types.NamespacedName{Name: kodeTemplate.Spec.EnvoyProxyTemplateRef.Name}
	if kodeTemplate.Spec.EnvoyProxyTemplateRef.Name != "" {
		log.Info("Fetching EnvoyProxyTemplate instance", "Name", envoyProxyTemplateName)
		if err := r.Get(ctx, envoyProxyTemplateName, envoyProxyTemplate); err != nil {
			if errors.IsNotFound(err) {
				log.Info("EnvoyProxyTemplate instance not found, requeuing", "Name", envoyProxyTemplateName)
				return ctrl.Result{RequeueAfter: 10 * time.Second}, nil // Retry after some time
			}
			log.Error(err, "Failed to fetch EnvoyProxyTemplate instance", "Name", envoyProxyTemplateName)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	// Ensure the Deployment and Service exist
	// log.Info("Ensuring Deployment exists", "Namespace", kode.Namespace, "Name", kode.Name)
	if err := r.ensureDeployment(ctx, kode, kodeTemplate, envoyProxyTemplate); err != nil {
		return ctrl.Result{}, err
	}
	// log.Info("Ensuring Service exists", "Namespace", kode.Namespace, "Name", kode.Name)
	if err := r.ensureService(ctx, kode, kodeTemplate); err != nil {
		return ctrl.Result{}, err
	}

	// Ensure PVC exists
	if !reflect.DeepEqual(kode.Spec.Storage, kodev1alpha1.KodeStorageSpec{}) {
		// log.Info("Ensuring Storage exists", "Namespace", kode.Namespace, "Name", kode.Name)
		if _, err := r.ensurePVC(ctx, kode); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *KodeReconciler) validateReferences(kode *kodev1alpha1.Kode) error {
	if kode.Spec.TemplateRef.Kind != "KodeTemplate" {
		return fmt.Errorf("invalid reference kind for TemplateRef: %s", kode.Spec.TemplateRef.Kind)
	}
	return nil
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ensurePVC ensures that the PersistentVolumeClaim exists for the Kode instance
func (r *KodeReconciler) ensurePVC(ctx context.Context, kode *kodev1alpha1.Kode) (*corev1.PersistentVolumeClaim, error) {
	log := r.Log.WithName("ensurePVC")

	log.Info("Ensuring PVC exists", "Namespace", kode.Namespace, "Name", kode.Name)

	pvc, err := r.getOrCreatePVC(ctx, kode)
	if err != nil {
		return pvc, err
	}

	return pvc, r.updatePVCIfNecessary(ctx, kode, pvc)
}

// getOrCreatePVC gets or creates a PersistentVolumeClaim for the Kode instance
func (r *KodeReconciler) getOrCreatePVC(ctx context.Context, kode *kodev1alpha1.Kode) (*corev1.PersistentVolumeClaim, error) {
	log := r.Log.WithName("getOrCreatePVC")

	pvc := r.constructPVC(kode)
	if err := controllerutil.SetControllerReference(kode, pvc, r.Scheme); err != nil {
		return nil, err
	}

	found := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new PVC", "Namespace", pvc.Namespace, "Name", pvc.Name)
			if err := r.Create(ctx, pvc); err != nil {
				return nil, err
			}
			return pvc, nil
		}
		return nil, err
	}

	return found, nil
}

// updatePVCIfNecessary updates the PVC if the desired state is different from the existing state
func (r *KodeReconciler) updatePVCIfNecessary(ctx context.Context, kode *kodev1alpha1.Kode, existingPVC *corev1.PersistentVolumeClaim) error {
	log := r.Log.WithName("updatePVCIfNecessary")
	desiredPVC := r.constructPVC(kode)

	// Only update mutable fields: Resources.Requests
	if !equality.Semantic.DeepEqual(existingPVC.Spec.Resources.Requests, desiredPVC.Spec.Resources.Requests) {
		existingPVC.Spec.Resources.Requests = desiredPVC.Spec.Resources.Requests
		log.Info("Updating existing PVC resources", "Namespace", existingPVC.Namespace, "Name", existingPVC.Name)
		return r.Update(ctx, existingPVC)
	}

	log.Info("PVC is up-to-date", "Namespace", existingPVC.Namespace, "Name", existingPVC.Name)
	return nil
}

// constructPVC constructs a PersistentVolumeClaim for the Kode instance
func (r *KodeReconciler) constructPVC(kode *kodev1alpha1.Kode) *corev1.PersistentVolumeClaim {
	log := r.Log.WithName("constructPVC")

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PersistentVolumeClaimName,
			Namespace: kode.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: kode.Spec.Storage.AccessModes,
			Resources:   kode.Spec.Storage.Resources,
		},
	}
	if kode.Spec.Storage.StorageClassName != nil {
		pvc.Spec.StorageClassName = kode.Spec.Storage.StorageClassName
	}

	logPVCManifest(log, pvc)

	return pvc
}

// logPVCManifest add structured logging for the PVC manifest to improve visibility for debugging
func logPVCManifest(log logr.Logger, pvc *corev1.PersistentVolumeClaim) {
	log.V(1).Info("PVC Manifest",
		"Name", pvc.Name,
		"Namespace", pvc.Namespace,
		"AccessModes", fmt.Sprintf("%v", pvc.Spec.AccessModes),
		"Resources", fmt.Sprintf("Requests: %v, Limits: %v", pvc.Spec.Resources.Requests, pvc.Spec.Resources.Limits),
		"StorageClassName", pvc.Spec.StorageClassName,
	)
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ensureDeployment ensures that the Deployment exists for the Kode instance
func (r *KodeReconciler) ensureDeployment(ctx context.Context, kode *kodev1alpha1.Kode, kodeTemplate *kodev1alpha1.KodeTemplate, envoyProxyTemplate *kodev1alpha1.EnvoyProxyTemplate) error {
	log := r.Log.WithName("ensureDeployment")

	log.Info("Ensuring Deployment exists", "Namespace", kode.Namespace, "Name", kode.Name)

	deployment, err := r.getOrCreateDeployment(ctx, kode, kodeTemplate, envoyProxyTemplate)
	if err != nil {
		log.Error(err, "Failed to get or create Deployment", "Namespace", kode.Namespace, "Name", kode.Name)
		return err
	}

	if err := r.updateDeploymentIfNecessary(ctx, deployment); err != nil {
		log.Error(err, "Failed to update Deployment if necessary", "Namespace", deployment.Namespace, "Name", deployment.Name)
		return err
	}

	log.Info("Successfully ensured Deployment", "Namespace", kode.Namespace, "Name", kode.Name)

	return nil
}

// getOrCreateDeployment gets or creates a Deployment for the Kode instance
func (r *KodeReconciler) getOrCreateDeployment(ctx context.Context, kode *kodev1alpha1.Kode, kodeTemplate *kodev1alpha1.KodeTemplate, envoyProxyTemplate *kodev1alpha1.EnvoyProxyTemplate) (*appsv1.Deployment, error) {
	log := r.Log.WithName("getOrCreateDeployment")
	deployment := r.constructDeployment(kode, kodeTemplate, envoyProxyTemplate)

	if err := controllerutil.SetControllerReference(kode, deployment, r.Scheme); err != nil {
		return nil, err
	}

	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
			if err := r.Create(ctx, deployment); err != nil {
				log.Error(err, "Failed to create Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
				return nil, err
			}
			log.Info("Deployment created", "Namespace", deployment.Namespace, "Name", deployment.Name)
		} else {
			return nil, err
		}
	}

	return deployment, nil
}

// constructDeployment constructs a Deployment for the Kode instance
func (r *KodeReconciler) constructDeployment(kode *kodev1alpha1.Kode, kodeTemplate *kodev1alpha1.KodeTemplate, envoyProxyTemplate *kodev1alpha1.EnvoyProxyTemplate) *appsv1.Deployment {
	log := r.Log.WithName("constructDeployment")
	replicas := int32(1)

	ContainerName := "kode-" + kode.Name

	workspace := kodeTemplate.Spec.Home + kodeTemplate.Spec.DefaultWorkspace
	if kode.Spec.Workspace != "" {
		workspace = kodeTemplate.Spec.Home + kode.Spec.Workspace
	}

	labels := map[string]string{
		"app":                          ContainerName,
		"kode.jacero.io/name":          kode.Name,
		"kode-template.jacero.io/name": kodeTemplate.Name,
	}

	containers := []corev1.Container{{
		Name:  ContainerName,
		Image: kodeTemplate.Spec.Image,
		Env: []corev1.EnvVar{
			{Name: "PUID", Value: fmt.Sprintf("%d", kodeTemplate.Spec.PUID)},
			{Name: "PGID", Value: fmt.Sprintf("%d", kodeTemplate.Spec.PGID)},
			{Name: "TZ", Value: kodeTemplate.Spec.TZ},
			{Name: "USERNAME", Value: kode.Spec.User},
			{Name: "PASSWORD", Value: kode.Spec.Password},
			{Name: "DEFAULT_WORKSPACE", Value: workspace},
		},
		Ports: []corev1.ContainerPort{{ContainerPort: kodeTemplate.Spec.Port}},
	}}

	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// Add volume and volume mount if storage is defined
	if !reflect.DeepEqual(kode.Spec.Storage, kodev1alpha1.KodeStorageSpec{}) {
		volume := corev1.Volume{
			Name: "kode-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: PersistentVolumeClaimName,
				},
			},
		}

		volumeMount := corev1.VolumeMount{
			Name:      "kode-storage",
			MountPath: kodeTemplate.Spec.Home,
		}

		volumes = append(volumes, volume)
		volumeMounts = append(volumeMounts, volumeMount)
		containers[0].VolumeMounts = volumeMounts
	}

	// Add EnvoyProxy sidecar if specified
	initContainers := []corev1.Container{}
	if kodeTemplate.Spec.EnvoyProxyTemplateRef.Name != "" {
		envoySidecarContainer, err := constructEnvoyProxyContainer(kodeTemplate, envoyProxyTemplate)
		if err != nil {
			log.Error(err, "Failed to construct EnvoyProxy sidecar")
		} else {
			containers = append(containers, envoySidecarContainer)
		}
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					InitContainers: initContainers,
					Containers: containers,
					Volumes:    volumes,
				},
			},
		},
	}

	logDeploymentManifest(log, deployment)
	return deployment
}

// updateDeploymentIfNecessary updates the Deployment if the desired state is different from the existing state
func (r *KodeReconciler) updateDeploymentIfNecessary(ctx context.Context, deployment *appsv1.Deployment) error {
	log := r.Log.WithName("updateDeploymentIfNecessary")

	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
			return err
		}
		log.Info("Deployment not found, skipping update", "Namespace", deployment.Namespace, "Name", deployment.Name)
		return nil
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found); err != nil {
			return err
		}

		if !reflect.DeepEqual(deployment.Spec, found.Spec) {
			found.Spec = deployment.Spec
			log.Info("Updating Deployment due to spec change", "Namespace", found.Namespace, "Name", found.Name)
			return r.Update(ctx, found)
		}
		return nil
	})

	if retryErr != nil {
		log.Error(retryErr, "Failed to update Deployment after retrying", "Namespace", deployment.Namespace, "Name", deployment.Name)
		return retryErr
	}

	log.Info("Deployment is up-to-date", "Namespace", deployment.Namespace, "Name", deployment.Name)

	return nil
}

// logDeploymentManifest add structured logging for the Deployment manifest to improve visibility for debugging
func logDeploymentManifest(log logr.Logger, deployment *appsv1.Deployment) {
	log.V(1).Info("Deployment Manifest",
		"Name", deployment.Name,
		"Namespace", deployment.Namespace,
		"Replicas", *deployment.Spec.Replicas,
		"Image", deployment.Spec.Template.Spec.Containers[0].Image,
		"Ports", fmt.Sprintf("%v", deployment.Spec.Template.Spec.Containers[0].Ports),
		"Env", fmt.Sprintf("%v", deployment.Spec.Template.Spec.Containers[0].Env),
		"VolumeMounts", fmt.Sprintf("%v", deployment.Spec.Template.Spec.Containers[0].VolumeMounts),
		"Volumes", fmt.Sprintf("%v", deployment.Spec.Template.Spec.Volumes),
	)
}

// /////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ensureService ensures that the Service exists for the Kode instance
func (r *KodeReconciler) ensureService(ctx context.Context, kode *kodev1alpha1.Kode, kodeTemplate *kodev1alpha1.KodeTemplate) error {
	log := r.Log.WithName("ensureService")

	log.Info("Ensuring Service exists", "Namespace", kode.Namespace, "Name", kode.Name)

	service, err := r.getOrCreateService(ctx, kode, kodeTemplate)
	if err != nil {
		log.Error(err, "Failed to get or create Service", "Namespace", kode.Namespace, "Name", kode.Name)
		return err
	}

	if err := r.updateServiceIfNecessary(ctx, service); err != nil {
		log.Error(err, "Failed to update Service if necessary", "Namespace", service.Namespace, "Name", service.Name)
		return err
	}

	log.Info("Successfully ensured Service", "Namespace", kode.Namespace, "Name", kode.Name)

	return nil
}

// getOrCreateService gets or creates a Service for the Kode instance
func (r *KodeReconciler) getOrCreateService(ctx context.Context, kode *kodev1alpha1.Kode, kodeTemplate *kodev1alpha1.KodeTemplate) (*corev1.Service, error) {
	log := r.Log.WithName("getOrCreateService")
	service := r.constructService(kode, kodeTemplate)

	if err := controllerutil.SetControllerReference(kode, service, r.Scheme); err != nil {
		return nil, err
	}

	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating Service", "Namespace", service.Namespace, "Name", service.Name)
			if err := r.Create(ctx, service); err != nil {
				log.Error(err, "Failed to create Service", "Namespace", service.Namespace, "Name", service.Name)
				return nil, err
			}
			log.Info("Service created", "Namespace", service.Namespace, "Name", service.Name)
		} else {
			return nil, err
		}
	}

	return service, nil
}

// constructService constructs a Service for the Kode instance
func (r *KodeReconciler) constructService(kode *kodev1alpha1.Kode, kodeTemplate *kodev1alpha1.KodeTemplate) *corev1.Service {
	log := r.Log.WithName("constructService")
	labels := map[string]string{
		"app":                          "kode-" + kode.Name,
		"kode.jacero.io/name":          kode.Name,
		"kode-template.jacero.io/name": kodeTemplate.Name,
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       kodeTemplate.Spec.Port,
				TargetPort: intstr.FromInt(int(kodeTemplate.Spec.Port)),
			}},
		},
	}
	logServiceManifest(log, service)
	return service
}

// updateServiceIfNecessary updates the Service if the desired state is different from the existing state
func (r *KodeReconciler) updateServiceIfNecessary(ctx context.Context, service *corev1.Service) error {
	log := r.Log.WithName("updateServiceIfNecessary")
	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		return nil
	}

	if !reflect.DeepEqual(service.Spec, found.Spec) {
		found.Spec = service.Spec
		log.Info("Updating Service", "Namespace", found.Namespace, "Name", found.Name)
		return r.Update(ctx, found)
	}
	return nil
}

// logServiceManifest add structured logging for the Service manifest to improve visibility for debugging
func logServiceManifest(log logr.Logger, service *corev1.Service) {
	log.V(1).Info("Service Manifest",
		"Name", service.Name,
		"Namespace", service.Namespace,
		"Selector", fmt.Sprintf("%v", service.Spec.Selector),
		"Ports", fmt.Sprintf("%v", service.Spec.Ports),
	)
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log = ctrl.Log.WithName("Kode")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
