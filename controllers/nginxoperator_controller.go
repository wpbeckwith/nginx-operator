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
	"github.com/example/nginx-operator/assets"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha1 "github.com/example/nginx-operator/api/v1alpha1"
)

// NginxOperatorReconciler reconciles a NginxOperator object
type NginxOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=operator.example.com,resources=nginxoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.example.com,resources=nginxoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.example.com,resources=nginxoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NginxOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *NginxOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Create logger for later use
	logger := log.FromContext(ctx)

	// Create pointer variable to custom resource struct
	operatorCR := &operatorv1alpha1.NginxOperator{}

	// Use the client to look up the custom resource via namespace and name
	err := r.Get(ctx, req.NamespacedName, operatorCR)

	// If we get an error and the error is that the CR is not found then assume it has already been
	// deleted and return normal
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Operator resource object not found.")
		return ctrl.Result{}, nil
	} else if err != nil {
		// Error was something other than Not Found for the CR, so return the error
		logger.Error(err, "Error getting operator resource object")
		return ctrl.Result{}, err
	}

	// We are here, so we found the CR and had no errors

	// Create pointer variable to deployment struct
	deployment := &appsv1.Deployment{}

	// Create flag to control invoking r.Create(...) or r.Update(...)
	create := false

	// Use the client to look up the deploymnet via namespace and name
	err = r.Get(ctx, req.NamespacedName, deployment)

	// If we get an error and the error is that the deployment is not found then assume it needs to be created
	if err != nil && errors.IsNotFound(err) {
		// Set the flag to true to do an update
		create = true

		// Load and set properties on an embedded deployment template from the manifests folder
		deployment = assets.GetDeploymentFromFile("manifests/nginx_deployment.yaml")

		// Use same namespace and name as the CR to make look up easier
		deployment.Namespace = req.Namespace
		deployment.Name = req.Name

	} else if err != nil {
		// Error was something other than Not Found for the deployment, so return the error
		logger.Error(err, "Error getting existing Nginx deployment.")
		return ctrl.Result{}, err
	}

	// If the CR's attribute is not nil then override the default value and update it in the deployment
	if operatorCR.Spec.Replicas != nil {
		deployment.Spec.Replicas = operatorCR.Spec.Replicas
	}

	// If the CR's attribute is not nil then override the default value and update it in the deployment
	if operatorCR.Spec.Port != nil {
		deployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort = *operatorCR.Spec.Port
	}

	// Set the CR as the owner of the deployment
	ctrl.SetControllerReference(operatorCR, deployment, r.Scheme)

	// So are we creating a new deployment or updating a current one?
	if create == true {
		// Use the client to create the deployment
		err = r.Create(ctx, deployment)

	} else {
		err = r.Update(ctx, deployment)
	}

	// If we make it to here, then it's all good OR we got an error from trying to create/update the deployment
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NginxOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.NginxOperator{}).
		// Add owned deployments to the list of resources to monitor
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
