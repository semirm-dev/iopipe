/***************************************************************
 * Copyright (c) 2021 Cisco Systems, Inc.
 * All rights reserved.
 * Unauthorized redistribution prohibited.
 **************************************************************/

package controllers

import (
	"context"
	"errors"
	"github.com/semirm-dev/iopipe/syncer"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	ioPipeLabel = "iopipesync"
)

// IOPipeReconciler reconciles a ConfigMap object.
type IOPipeReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Namespace string
}

// Reconcile will read from input using read, filter data and write to output using writer
func (r *IOPipeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	l.Info("iopipe reconcile started")

	var ioPipeConfigMap corev1.ConfigMap
	if err := r.Client.Get(ctx, req.NamespacedName, &ioPipeConfigMap); err != nil {
		if kerrors.IsNotFound(err) {
			l.Info("iopipe configMap object does not exist")
			return ctrl.Result{}, nil
		}
		l.Error(err, "failed to get object")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	l.WithValues("configMap", ioPipeConfigMap).Info("config received")

	stepsData, ok := ioPipeConfigMap.Data["steps"]
	if !ok {
		err := errors.New("invalid (missing) map key from the configMap: <steps>")
		l.Error(err, "failed to get config data")
		return ctrl.Result{}, err
	}

	var steps []syncer.Step
	if err := yaml.Unmarshal([]byte(stepsData), &steps); err != nil {
		l.Error(err, "failed to unmarshal config data into steps")
		return ctrl.Result{}, err
	}

	if err := syncer.Sync(ctx, steps); err != nil {
		return ctrl.Result{}, err
	}

	l.Info("iopipe syncer writing config finished")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IOPipeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.ConfigMap{}).
		WithEventFilter(ioPipeConfigMapFilter(r.Namespace)).
		Complete(r)
}

func ioPipeConfigMapFilter(namespace string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return isIOPipeConfig(e.Object, namespace)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isIOPipeConfig(e.ObjectNew, namespace)
		},
	}
}

func isIOPipeConfig(obj client.Object, namespace string) bool {
	configMap, validConfigMapObj := obj.(*corev1.ConfigMap)
	marker, isLabeled := configMap.Labels[ioPipeLabel]

	return validConfigMapObj && isLabeled && marker == "yes" && configMap.Namespace == namespace
}
