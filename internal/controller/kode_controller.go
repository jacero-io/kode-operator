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

	kodev1alpha1 "github.com/emil-jacero/kode-operator/api/v1alpha1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Fetch the Kode instance
	kode := &kodev1alpha1.Kode{}
	r.Log.Info("Fetching Kode instance", "Namespace", req.Namespace, "Name", req.Name)
	if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
		if client.IgnoreNotFound(err) != nil {
			r.Log.Error(err, "Failed to fetch Kode instance", "Namespace", req.Namespace, "Name", req.Name)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	r.Log.Info("Successfully fetched Kode instance", "Namespace", req.Namespace, "Name", req.Name)

	// Ensure the Deployment and Service exist
	r.Log.Info("Ensuring Deployment exists", "Namespace", kode.Namespace, "Name", kode.Name)
	if err := r.ensureDeployment(ctx, kode); err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("Deployment ensured", "Namespace", kode.Namespace, "Name", kode.Name)
	r.Log.Info("Ensuring Service exists", "Namespace", kode.Namespace, "Name", kode.Name)
	if err := r.ensureService(ctx, kode); err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("Service ensured", "Namespace", kode.Namespace, "Name", kode.Name)

	// Ensure PVC exists
	if !reflect.DeepEqual(kode.Spec.Storage, kodev1alpha1.KodeStorageSpec{}) {
		r.Log.Info("Ensuring Storage exists", "Namespace", kode.Namespace, "Name", kode.Name)
		if _, err := r.ensurePVC(ctx, kode); err != nil {
			return ctrl.Result{}, err
		}
		r.Log.Info("Storage ensured", "Namespace", kode.Namespace, "Name", kode.Name)
	}

	// Update the status
	r.Log.Info("Updating Kode status", "Namespace", kode.Namespace, "Name", kode.Name)
	kode.Status.AvailableReplicas = r.getAvailableReplicas(ctx, kode)
	if err := r.Status().Update(ctx, kode); err != nil {
		return ctrl.Result{}, err
	}
	r.Log.Info("Kode status updated", "Namespace", kode.Namespace, "Name", kode.Name)

	return ctrl.Result{}, nil
}

func (r *KodeReconciler) ensurePVC(ctx context.Context, kode *kodev1alpha1.Kode) (*v1.PersistentVolumeClaim, error) {
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name + "-pvc",
			Namespace: kode.Namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: kode.Spec.Storage.AccessModes,
			Resources:   kode.Spec.Storage.Resources,
		},
	}
	if kode.Spec.Storage.StorageClassName != nil {
		pvc.Spec.StorageClassName = kode.Spec.Storage.StorageClassName
	}

	// Set the Kode instance as the owner and controller
	if err := controllerutil.SetControllerReference(kode, pvc, r.Scheme); err != nil {
		return nil, err
	}

	// Check if this PVC already exists
	found := &v1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		r.Log.Info("Creating a new PVC", "Namespace", pvc.Namespace, "Name", pvc.Name)
		err = r.Create(ctx, pvc)
		if err != nil {
			return nil, err
		}
		return pvc, nil
	} else if err != nil {
		return nil, err
	}

	// Update the PVC if it exists and is different
	if !equality.Semantic.DeepEqual(found.Spec, pvc.Spec) {
		found.Spec = pvc.Spec
		r.Log.Info("Updating existing PVC", "Namespace", pvc.Namespace, "Name", pvc.Name)
		err = r.Update(ctx, found)
		if err != nil {
			return nil, err
		}
	}

	return found, nil
}

func (r *KodeReconciler) ensureDeployment(ctx context.Context, kode *kodev1alpha1.Kode) error {
	replicas := int32(1)
	defaultWorkspace := kode.Spec.DefaultWorkspace
	if defaultWorkspace == "" {
		defaultWorkspace = kode.Spec.ConfigPath + "/workspace"
	}

	// Define the PVC volume
	volume := corev1.Volume{
		Name: "kode-storage",
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: kode.Name + "-pvc",
			},
		},
	}

	// Define the volume mount
	volumeMount := corev1.VolumeMount{
		Name:      "kode-storage",
		MountPath: kode.Spec.ConfigPath,
	}

	// Check if the PVC exists
	pvcExists := false
	pvc := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: kode.Name + "-pvc", Namespace: kode.Namespace}, pvc)
	if err == nil {
		pvcExists = true
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "code-server"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "code-server"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "code-server",
						Image: kode.Spec.Image,
						Env: []corev1.EnvVar{
							{Name: "PUID", Value: fmt.Sprintf("%d", kode.Spec.PUID)},
							{Name: "PGID", Value: fmt.Sprintf("%d", kode.Spec.PGID)},
							{Name: "TZ", Value: kode.Spec.TZ},
							{Name: "PASSWORD", Value: kode.Spec.Password},
							{Name: "HASHED_PASSWORD", Value: kode.Spec.HashedPassword},
							{Name: "SUDO_PASSWORD", Value: kode.Spec.SudoPassword},
							{Name: "SUDO_PASSWORD_HASH", Value: kode.Spec.SudoPasswordHash},
							{Name: "DEFAULT_WORKSPACE", Value: defaultWorkspace},
						},
						Ports: []corev1.ContainerPort{{ContainerPort: kode.Spec.ServicePort}},
						VolumeMounts: []corev1.VolumeMount{
							volumeMount,
						},
					}},
					Volumes: []corev1.Volume{
						volume,
					},
				},
			},
		},
	}

	// Add volume and volume mount if PVC exists
	if pvcExists {
		deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{volumeMount}
		deployment.Spec.Template.Spec.Volumes = []corev1.Volume{volume}
	}

	if err := controllerutil.SetControllerReference(kode, deployment, r.Scheme); err != nil {
		return err
	}

	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, found)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return err
	}

	if err != nil && client.IgnoreNotFound(err) == nil {
		r.Log.Info("Creating Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
		r.Log.V(1).Info("Deployment Spec", "Spec", deployment.Spec)
		if err := r.Create(ctx, deployment); err != nil {
			r.Log.Error(err, "Failed to create Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
			return err
		}
		r.Log.Info("Deployment created", "Namespace", deployment.Namespace, "Name", deployment.Name)
	} else if !reflect.DeepEqual(deployment.Spec, found.Spec) {
		found.Spec = deployment.Spec
		r.Log.Info("Updating Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
		r.Log.V(1).Info("Deployment Spec", "Spec", found.Spec)
		if err := r.Update(ctx, found); err != nil {
			r.Log.Error(err, "Failed to update Deployment", "Namespace", deployment.Namespace, "Name", deployment.Name)
			return err
		}
		r.Log.Info("Deployment updated", "Namespace", deployment.Namespace, "Name", deployment.Name)
	}

	return nil
}

func (r *KodeReconciler) ensureService(ctx context.Context, kode *kodev1alpha1.Kode) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Name,
			Namespace: kode.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "code-server"},
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       kode.Spec.ServicePort,
				TargetPort: intstr.FromInt(int(kode.Spec.ServicePort)),
			}},
		},
	}

	if err := controllerutil.SetControllerReference(kode, service, r.Scheme); err != nil {
		return err
	}

	found := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, found)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return err
	}

	if err != nil && client.IgnoreNotFound(err) == nil {
		r.Log.Info("Creating Service", "Namespace", service.Namespace, "Name", service.Name)
		if err := r.Create(ctx, service); err != nil {
			r.Log.Error(err, "Failed to create Service", "Namespace", service.Namespace, "Name", service.Name)
			return err
		}
		r.Log.Info("Service created", "Namespace", service.Namespace, "Name", service.Name)
	} else if !reflect.DeepEqual(service.Spec, found.Spec) {
		found.Spec = service.Spec
		r.Log.Info("Updating Service", "Namespace", service.Namespace, "Name", service.Name)
		if err := r.Update(ctx, found); err != nil {
			r.Log.Error(err, "Failed to update Service", "Namespace", service.Namespace, "Name", service.Name)
			return err
		}
		r.Log.Info("Service updated", "Namespace", service.Namespace, "Name", service.Name)
	}

	return nil
}

func (r *KodeReconciler) getAvailableReplicas(ctx context.Context, kode *kodev1alpha1.Kode) int32 {
	deployment := &appsv1.Deployment{}
	_ = r.Get(ctx, types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace}, deployment)
	return deployment.Status.AvailableReplicas
}

func (r *KodeReconciler) ensureResource(ctx context.Context, kode *kodev1alpha1.Kode, ensureFunc func(context.Context, *kodev1alpha1.Kode) error) error {
	return ensureFunc(ctx, kode)
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Log = ctrl.Log.WithName("controllers").WithName("Kode")
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
