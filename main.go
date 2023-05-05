package main

import (
	"flag"
	"github.com/semirm-dev/iopipe/controllers"
	"os"

	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(corev1.AddToScheme(scheme))
}

func main() {
	//Parse flags

	opts := &zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)

	var webhookPort int
	var metricsPort int

	flag.IntVar(&webhookPort, "webhook-port", 8888, "Port webhook endpoint binds to.")
	flag.IntVar(&metricsPort, "metrics-port", 1025, "Port metric endpoint binds to.")
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(opts)))
	setupLog.Info("Initializing...")

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		//set manager options
		Scheme:             scheme,
		MetricsBindAddress: fmt.Sprintf(":%d", metricsPort),
		Port:               webhookPort,
	})
	if err != nil {
		setupLog.Error(err, "unable to create manager")
		os.Exit(1)
	}

	//initialise controllers

	if err = (&controllers.IOPipeReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Namespace: "semirmahovkic",
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IOPipeReconciler")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
