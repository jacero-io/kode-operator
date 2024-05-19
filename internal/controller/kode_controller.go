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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type KodeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=kode.jacero.io,resources=kodes/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *KodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Kode instance
	kode := &kodev1alpha1.Kode{}
	if err := r.Get(ctx, req.NamespacedName, kode); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to fetch Kode")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Ensure the PVC exists if storage is specified
	if kode.Spec.Storage != nil {
		if err := r.ensurePVC(ctx, kode); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Ensure the Deployment exists
	if err := r.ensureDeployment(ctx, kode); err != nil {
		return ctrl.Result{}, err
	}

	// Ensure the Service exists
	if err := r.ensureService(ctx, kode); err != nil {
		return ctrl.Result{}, err
	}

	// Update the status
	kode.Status.AvailableReplicas = r.getAvailableReplicas(ctx, kode)
	if err := r.Status().Update(ctx, kode); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *KodeReconciler) ensurePVC(ctx context.Context, kode *kodev1alpha1.Kode) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      kode.Spec.Storage.Name,
			Namespace: kode.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(kode.Spec.Storage.Size),
				},
			},
		},
	}
	if pvc.ObjectMeta.Name == "" {
		pvc.ObjectMeta.Name = kode.Name
	}
	if kode.Spec.Storage.StorageClassName != "" {
		pvc.Spec.StorageClassName = &kode.Spec.Storage.StorageClassName
	}

	if err := controllerutil.SetControllerReference(kode, pvc, r.Scheme); err != nil {
		return err
	}

	foundPVC := &corev1.PersistentVolumeClaim{}
	err := r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, foundPVC)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return err
	}

	if err != nil && client.IgnoreNotFound(err) == nil {
		log := log.FromContext(ctx)
		log.Info("Creating PVC", "Namespace", pvc.Namespace, "Name", pvc.Name)
		return r.Create(ctx, pvc)
	}

	return nil
}

func (r *KodeReconciler) ensureDeployment(ctx context.Context, kode *kodev1alpha1.Kode) error {
	replicas := int32(1)
	deploy := &appsv1.Deployment{
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
							{
								Name:  "PUID",
								Value: fmt.Sprintf("%d", kode.Spec.PUID),
							},
							{
								Name:  "PGID",
								Value: fmt.Sprintf("%d", kode.Spec.PGID),
							},
							{
								Name:  "TZ",
								Value: kode.Spec.TZ,
							},
							{
								Name:  "PASSWORD",
								Value: kode.Spec.Password,
							},
							{
								Name:  "HASHED_PASSWORD",
								Value: kode.Spec.HashedPassword,
							},
							{
								Name:  "SUDO_PASSWORD",
								Value: kode.Spec.SudoPassword,
							},
							{
								Name:  "SUDO_PASSWORD_HASH",
								Value: kode.Spec.SudoPasswordHash,
							},
							{
								Name:  "PROXY_DOMAIN",
								Value: kode.Spec.ProxyDomain,
							},
							{
								Name:  "DEFAULT_WORKSPACE",
								Value: kode.Spec.DefaultWorkspace,
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: kode.Spec.ServicePort,
						}},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "config-volume",
								MountPath: "/config",
							},
						},
					}},
					Volumes: []corev1.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: kode.Spec.ConfigPath,
								},
							},
						},
					},
				},
			},
		},
	}

	if kode.Spec.Storage != nil {
		deploy.Spec.Template.Spec.Containers[0].VolumeMounts = append(deploy.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{
			Name:      "pvc-volume",
			MountPath: kode.Spec.ConfigPath,
		})
		deploy.Spec.Template.Spec.Volumes = append(deploy.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: "pvc-volume",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: kode.Spec.Storage.Name,
				},
			},
		})
	}

	if err := controllerutil.SetControllerReference(kode, deploy, r.Scheme); err != nil {
		return err
	}

	found := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return err
	}

	if err != nil && client.IgnoreNotFound(err) == nil {
		log := log.FromContext(ctx)
		log.Info("Creating Deployment", "Namespace", deploy.Namespace, "Name", deploy.Name)
		return r.Create(ctx, deploy)
	} else if !reflect.DeepEqual(deploy.Spec, found.Spec) {
		found.Spec = deploy.Spec
		log := log.FromContext(ctx)
		log.Info("Updating Deployment", "Namespace", deploy.Namespace, "Name", deploy.Name)
		return r.Update(ctx, found)
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
		log := log.FromContext(ctx)
		log.Info("Creating Service", "Namespace", service.Namespace, "Name", service.Name)
		return r.Create(ctx, service)
	} else if !reflect.DeepEqual(service.Spec, found.Spec) {
		found.Spec = service.Spec
		log := log.FromContext(ctx)
		log.Info("Updating Service", "Namespace", service.Namespace, "Name", service.Name)
		return r.Update(ctx, found)
	}

	return nil
}

func (r *KodeReconciler) getAvailableReplicas(ctx context.Context, kode *kodev1alpha1.Kode) int32 {
	deploy := &appsv1.Deployment{}
	_ = r.Get(ctx, types.NamespacedName{Name: kode.Name, Namespace: kode.Namespace}, deploy)
	return deploy.Status.AvailableReplicas
}

func (r *KodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kodev1alpha1.Kode{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
