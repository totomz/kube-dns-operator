/*
Copyright 2022 Tommaso Doninelli.

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
	"fmt"
	netv1alpha1 "github.com/totomz/kube-dns-operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

// DnsRecordReconciler reconciles a DnsRecord object
type DnsRecordReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const dnsRecordFinalizer = "dnsrecord.net.beekube.cloud/finalizer"

// +kubebuilder:rbac:groups=net.beekube.cloud,resources=dnsrecords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=net.beekube.cloud,resources=dnsrecords/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=net.beekube.cloud,resources=dnsrecords/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=get;list;create

func (r *DnsRecordReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info(fmt.Sprintf("Reconciliation for: %s", req.NamespacedName.String()))

	crd := &netv1alpha1.DnsRecord{}
	errGetCrd := r.Get(ctx, req.NamespacedName, crd)
	if errGetCrd != nil {
		if errors.IsNotFound(errGetCrd) {
			logger.Info("DnsRecord resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		logger.Error(errGetCrd, "Failed to get DnsRecord")
		return ctrl.Result{}, errGetCrd
	}

	var errApi error
	r.LogEvent(ctx, "Normal", "InitReconciliation", "Upserting dns record", req, crd)

	// Resource deletion
	if crd.GetDeletionTimestamp() != nil {
		logger.Info("Cleaning up records")
		if controllerutil.ContainsFinalizer(crd, dnsRecordFinalizer) {
			logger.Info("Found a finalizer")

			if crd.Spec.Route53Records.Name != "" {
				if err := r.FinalizeAwsRoute53(ctx, req.Namespace, crd.Spec.Route53Records); err != nil {
					// Run finalization logic for memcachedFinalizer. If the
					// finalization logic fails, don't remove the finalizer so
					// that we can retry during the next reconciliation.
					logger.Error(err, "can't cleanup - retry later")
					r.LogEvent(ctx, "Warning", "ErrorDnsApiFinalize", err.Error(), req, crd)
					return RequeueAfter(120 * time.Second)
				}
			}

			// Remove the finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(crd, dnsRecordFinalizer)
			err := r.Update(ctx, crd)
			if err != nil {
				logger.Error(err, "can't remove finalizer - won't retry")
				r.LogEvent(ctx, "Warning", "ErrorDnsApiFinalize", err.Error(), req, crd)
			}
		}

		return DoNotRequeue()
	}

	// Resource Upsert
	if crd.Spec.Route53Records.Name != "" {
		// It's an aws record!
		// Resource creation
		errApi = r.ReconcileRoute53(ctx, req.Namespace, crd.Spec.Route53Records)
	}

	if errApi != nil {
		r.LogEvent(ctx, "Warning", "ErrorDnsApi", errApi.Error(), req, crd)
		return DoNotRequeue()
	}

	r.LogEvent(ctx, "Normal", "DnsApiUpdated", "DNS Record updated", req, crd)

	if !controllerutil.ContainsFinalizer(crd, dnsRecordFinalizer) {
		controllerutil.AddFinalizer(crd, dnsRecordFinalizer)
		r.LogEvent(ctx, "Normal", "InternalApi", "Adding record finalizer - the dns will probably be updated again", req, crd)
		errUpdate := r.Update(ctx, crd)
		if errUpdate != nil {
			logger.Error(errUpdate, "can't add finalizer - won't retry")
			r.LogEvent(ctx, "Warning", "ErrorDnsApiFinalize", errUpdate.Error(), req, crd)
		}
	}

	return DoNotRequeue()
}

// SetupWithManager sets up the controller with the Manager.
func (r *DnsRecordReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netv1alpha1.DnsRecord{}).
		Complete(r)
}

func (r *DnsRecordReconciler) GetSecret(ctx context.Context, ns string, secretName string, secretKey string) (string, error) {
	awsSecret := v1.Secret{}
	errGetSecret := r.Get(ctx, client.ObjectKey{
		Namespace: ns,
		Name:      secretName,
	}, &awsSecret)

	if errGetSecret != nil {
		return "", errGetSecret
	}

	data, hasData := awsSecret.Data[secretKey]
	if !hasData {
		return "", fmt.Errorf("secret key %s not found", secretKey)
	}

	return string(data), nil
}

// LogEvent Creates an Event resource. Level must be either Normal or Warning
func (r *DnsRecordReconciler) LogEvent(ctx context.Context, level, reason, message string, req ctrl.Request, crd *netv1alpha1.DnsRecord) {
	logger := log.FromContext(ctx)
	inst := metav1.Time{Time: time.Now()}
	event := v1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.Namespace,
			Name:      fmt.Sprintf("edns-%v", time.Now().UnixMilli()),
		},
		InvolvedObject: v1.ObjectReference{
			Kind:            "DnsRecord",
			Namespace:       req.Namespace,
			Name:            req.Name,
			UID:             crd.UID,
			APIVersion:      crd.APIVersion,
			ResourceVersion: crd.ResourceVersion,
		},
		Reason:         reason,
		Message:        message,
		Type:           level,
		Source:         v1.EventSource{Component: "DnsRecordOperator"},
		FirstTimestamp: inst,
		LastTimestamp:  inst,
		Count:          1,
	}

	e := r.Client.Create(ctx, &event)
	if e != nil {
		logger.Error(e, "error saving Event")
	}
}

func DoNotRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func RequeueWithError(e error) (ctrl.Result, error) {
	return ctrl.Result{}, e
}

func RequeueAfter(t time.Duration) (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: t}, nil
}
