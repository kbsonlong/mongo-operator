/*
Copyright 2022.

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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	mongov1 "github.com/kbsonlong/mongo-operator/api/v1"
	k8sappsv1 "k8s.io/api/apps/v1"
	k8scorev1 "k8s.io/api/core/v1"
)

// MongoReconciler reconciles a Mongo object
type MongoReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=mongo.alongparty.cn,resources=mongoes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mongo.alongparty.cn,resources=mongoes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=mongo.alongparty.cn,resources=mongoes/finalizers,verbs=update
//+kubebuilder:rbac:groups=mongo.alongparty.cn,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mongo.alongparty.cn,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mongo.alongparty.cn,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Mongo object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *MongoReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	// 实例化
	mongo := &mongov1.Mongo{}

	err := r.Get(ctx, req.NamespacedName, mongo)
	if err != nil {
		return ctrl.Result{}, nil
	}

	deployment := &k8sappsv1.Deployment{}
	err = r.Get(ctx, req.NamespacedName, deployment)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Deployment Not Found")
			err = r.CreateDeployment(ctx, mongo)
			if err != nil {
				r.Recorder.Event(mongo, k8scorev1.EventTypeWarning, "FailedCreateDeployment", err.Error())
				return ctrl.Result{}, err
			}
			log.Info("Deployment Created")
			mongo.Status.Replica = 1
			err = r.Update(ctx, mongo)
			if err != nil {
				r.Recorder.Event(mongo, k8scorev1.EventTypeWarning, "FailedUpdateStatus", err.Error())
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}
	//binding deployment to mongo
	if err = ctrl.SetControllerReference(mongo, deployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// 设置 status
	// mongo.Status.Replica = *mongo.Spec.Replica
	// err = r.Status().Update(ctx, mongo)
	r.Recorder.Event(mongo, k8scorev1.EventTypeNormal, "Info", "test event")

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&mongov1.Mongo{}).
		Complete(r)
}

func (r *MongoReconciler) CreateDeployment(ctx context.Context, mongo *mongov1.Mongo) error {
	deployment := &k8sappsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mongo.Namespace,
			Name:      mongo.Name,
		},
		Spec: k8sappsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(*mongo.Spec.Replica),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": mongo.Name,
				},
			},

			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": mongo.Name,
					},
				},
				Spec: k8scorev1.PodSpec{
					Containers: []k8scorev1.Container{
						{
							Name:            mongo.Name,
							Image:           mongo.Spec.Image,
							ImagePullPolicy: "IfNotPresent",
							Ports: []k8scorev1.ContainerPort{
								{
									Name:          mongo.Name,
									Protocol:      k8scorev1.ProtocolSCTP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	err := r.Create(ctx, deployment)
	if err != nil {
		return err
	}
	return nil
}

func (r *MongoReconciler) CreateStatefulset(ctx context.Context, mongo *mongov1.Mongo) error {
	statefulset := &k8sappsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: mongo.Namespace,
			Name:      mongo.Name,
		},
		Spec: k8sappsv1.StatefulSetSpec{
			Replicas: pointer.Int32Ptr(*mongo.Spec.Replica),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": mongo.Name,
				},
			},

			Template: k8scorev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": mongo.Name,
					},
				},
				Spec: k8scorev1.PodSpec{
					Containers: []k8scorev1.Container{
						{
							Name:            mongo.Name,
							Image:           mongo.Spec.Image,
							ImagePullPolicy: "IfNotPresent",
							Ports: []k8scorev1.ContainerPort{
								{
									Name:          mongo.Name,
									Protocol:      k8scorev1.ProtocolSCTP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	err := r.Create(ctx, statefulset)
	if err != nil {
		return err
	}
	return nil
}
