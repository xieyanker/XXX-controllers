/*

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

package controllers

import (
	"context"

	"errors"
	"github.com/go-logr/logr"
	testv1alpha1 "iop.inspur.com/XXX-controllers/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	api_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batchv1 "k8s.io/api/batch/v1"
	"strings"
)

// XxxReconciler reconciles a Xxx object
type XxxReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var finalizerName string = "job.finalizers.test.inspur.com"

// +kubebuilder:rbac:groups=test.inspur.com,resources=xxxes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.inspur.com,resources=xxxes/status,verbs=get;update;patch

func (r *XxxReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	_ = r.Log.WithValues("xxx", req.NamespacedName)

	// your logic here
	xxx := &testv1alpha1.Xxx{}
	if err := r.Get(ctx, req.NamespacedName, xxx); err != nil {
		log.Printf("Unable to get Xxx [%v] [%v], Error: [%v] \n", req.Namespace, req.Name, err)
		// if delete xxx remove finalizers, it will trigger the Reconcile
		if api_errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	log.Printf("Get Xxx [%v] [%v], Spec: [%v], Finalizers: [%v] \n", xxx.Namespace, xxx.Name, xxx.Spec, xxx.Finalizers)

	if xxx.ObjectMeta.DeletionTimestamp.IsZero() {
		log.Printf("Xxx [%v] [%v]'s ObjectMeta.DeletionTimestamp is 0 \n", xxx.Namespace, xxx.Name)
		if !containsString(xxx.ObjectMeta.Finalizers, finalizerName) {
			log.Printf("Xxx [%v] [%v] not include finalizer [%v], we will add it \n", xxx.Namespace, xxx.Name, finalizerName)
			xxx.ObjectMeta.Finalizers = append(xxx.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(context.Background(), xxx); err != nil {
				log.Printf("Update Add Xxx [%v] [%v] finalizers Error: [%v] \n", xxx.Namespace, xxx.Name, err)
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(xxx.ObjectMeta.Finalizers, finalizerName) {
			log.Printf("Xxx [%v] [%v]'s ObjectMeta.DeletionTimestamp is not 0 and the finalizerName need to handle \n", xxx.Namespace, xxx.Name)
			deleteJob := newDeleteJob(xxx)
			err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: xxx.Spec.JobName + "-delete"}, deleteJob)
			if err != nil {
				if api_errors.IsNotFound(err) {
					err = r.Create(ctx, deleteJob)
					log.Printf("Create Delete Job [%v] [%v] success \n", req.Namespace, xxx.Spec.JobName + "-delete")
					if err != nil {
						log.Printf("Create Delete Job [%v] [%v], Error: [%v] \n", req.Namespace, xxx.Spec.JobName + "-delete", err)
						return ctrl.Result{}, err
					}
				} else {
					log.Printf("Get Delete Job [%v] [%v], Error: [%v] \n", req.Namespace, xxx.Spec.JobName + "-delete", err)
					return ctrl.Result{}, err
				}
			}

			xxx.ObjectMeta.Finalizers = removeString(xxx.ObjectMeta.Finalizers, finalizerName)
			if err := r.Update(context.Background(), xxx); err != nil {
				log.Printf("Update Remove Xxx [%v] [%v] finalizers Error: [%v] \n", xxx.Namespace, xxx.Name, err)
				return ctrl.Result{}, err
			}
			log.Printf("Remove Xxx [%v] [%v] finalizers success \n", req.Namespace, xxx.Spec.JobName + "-delete")
			return ctrl.Result{}, nil
		}
	}

	if xxx.Spec.GitUrl == "" || xxx.Spec.ClonePath == "" || xxx.Spec.BuildCommand == "" || xxx.Spec.BinaryName == "" || xxx.Spec.JobName == "" {
		err := errors.New("Each spec parameter must be specified ")
		log.Println(err.Error())
		return ctrl.Result{}, err
	}

	buildJob := newBuildJob(xxx)
	err := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: xxx.Spec.JobName}, buildJob)
	if err != nil {
		if api_errors.IsNotFound(err) {
			err = r.Create(ctx, buildJob)
			if err != nil {
				log.Printf("Create Build Job [%v] [%v], Error: [%v] \n", req.Namespace, xxx.Spec.JobName, err)
				return ctrl.Result{}, err
			}
			log.Printf("Create Build Job [%v] [%v] success \n", req.Namespace, xxx.Spec.JobName)
		} else {
			log.Printf("Get Build Job [%v] [%v], Error: [%v] \n", req.Namespace, xxx.Spec.JobName, err)
			return ctrl.Result{}, err
		}
	}

	if !metav1.IsControlledBy(buildJob, xxx) {
		err := errors.New("Build Job " + buildJob.Name + " is not controlled by xxx")
		log.Println(err.Error())
		return ctrl.Result{}, err
	}

	oldCommands := buildJob.Spec.Template.Spec.Containers[0].Command[2]
	log.Printf("The Build Job [%v] [%v]'s old commands are: [%v]", xxx.Namespace, xxx.Name, oldCommands)
	newCommands := getCommands(xxx)
	if oldCommands != newCommands {
		newJob := newBuildJob(xxx)
		log.Printf("The Build Job [%v] [%v]'s new commands are changed: [%v]", xxx.Namespace, xxx.Name, newCommands)
		// job cannot be updated
		//err = r.Update(ctx, newJob)
		//if err != nil {
		//	log.Printf("Update Job [%v] [%v], Error: [%v] \n", req.Namespace, xxx.Spec.JobName, err)
		//	return ctrl.Result{}, err
		//}
		err = r.Delete(ctx, buildJob)
		if err != nil {
			log.Printf("Delete Old Job [%v] [%v], Error: [%v] \n", req.Namespace, xxx.Spec.JobName, err)
			return ctrl.Result{}, err
		}
		log.Printf("Delete Old Job [%v] [%v] success \n", req.Namespace, xxx.Spec.JobName)

		for {
			err = r.Create(ctx, newJob)
			if err != nil {
				if strings.Contains(err.Error(), "object is being deleted") {
					log.Println("The Old Job is still being deleted, we will retry")
					continue
				} else {
					log.Printf("Create New Job [%v] [%v], Error: [%v] \n", req.Namespace, xxx.Spec.JobName, err)
					return ctrl.Result{}, err
				}
			}
			break
		}
		log.Printf("Create New Job [%v] [%v] success \n", req.Namespace, xxx.Spec.JobName)
	}

	return ctrl.Result{}, nil
}

func (r *XxxReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1alpha1.Xxx{}).
		Complete(r)
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func getCommands(xxx *testv1alpha1.Xxx) string {
	return "git clone " + xxx.Spec.GitUrl + " " + xxx.Spec.ClonePath + "; cd " + xxx.Spec.ClonePath + "; " + xxx.Spec.BuildCommand + "; cp " + xxx.Spec.BinaryName + " /opt/"
}

func newBuildJob(xxx *testv1alpha1.Xxx) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      xxx.Spec.JobName,
			Namespace: xxx.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(xxx, schema.GroupVersionKind{Group: testv1alpha1.GroupVersion.Group, Version: testv1alpha1.GroupVersion.Version, Kind: "Xxx"}),
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeName:      "master2",
					HostNetwork:   true,
					RestartPolicy: "Never",
					Containers: []corev1.Container{
						{
							Name:            "golang",
							Image:           "golang:1.13",
							ImagePullPolicy: "IfNotPresent",
							Command: []string{
								"bash",
								"-c",
								getCommands(xxx),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "store-binary",
									MountPath: "/opt/",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "store-binary",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/opt/",
								},
							},
						},
					},
				},
			},
		},
	}
}

func newDeleteJob(xxx *testv1alpha1.Xxx) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      xxx.Spec.JobName + "-delete",
			Namespace: xxx.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeName:      "master2",
					HostNetwork:   true,
					RestartPolicy: "Never",
					Containers: []corev1.Container{
						{
							Name:            "golang",
							Image:           "golang:1.13",
							ImagePullPolicy: "IfNotPresent",
							Command: []string{
								"bash",
								"-c",
								"rm -rf /opt/" + xxx.Spec.BinaryName,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "store-binary",
									MountPath: "/opt/",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "store-binary",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/opt/",
								},
							},
						},
					},
				},
			},
		},
	}
}
