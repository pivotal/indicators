package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/k8s/client/clientset/versioned"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/k8s/indicator_status"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/prometheus_oauth_client"
	"k8s.io/client-go/rest"
	"k8s.io/klog"

	"code.cloudfoundry.org/go-envstruct"
	informers "github.com/pivotal/monitoring-indicator-protocol/pkg/k8s/client/informers/externalversions"
)

type config struct {
	Namespace          string `env:"NAMESPACE,required,report"`
	PrometheusURL      string `env:"PROMETHEUS_URL,required,report"`
	PrometheusApiToken string `env:"PROMETHEUS_API_TOKEN,report"`
}

func init() {
	klog.SetOutput(os.Stderr)
}

func main() {
	flag.Parse()
	ctx := setupSignalHandler()

	var conf config
	err := envstruct.Load(&conf)
	if err != nil {
		log.Fatal("failed to load env variables: NAMESPACE is required")
	}
	err = envstruct.WriteReport(&conf)
	if err != nil {
		log.Fatal("failed to write report using env variables")
	}

	cfg, err := rest.InClusterConfig()
	cfg.Timeout = 5 * time.Second
	if err != nil {
		log.Fatal("failed to configure kubernetes cluster; make sure kubernetes is running")
	}

	client, err := versioned.NewForConfig(cfg)
	if err != nil {
		log.Fatal("failed to create clientSet for the given config")
	}

	tokenFetcher := func() (string, error) { return conf.PrometheusApiToken, nil }
	prometheusClient, err := prometheus_oauth_client.Build(conf.PrometheusURL, tokenFetcher, false)
	if err != nil {
		log.Fatal("error building prometheus client")
	}

	controller := indicator_status.NewController(
		client.IndicatorprotocolV1(),
		prometheusClient,
		30*time.Second,
		clock.New(),
		conf.Namespace,
		indicator_status.NewIndicatorStore(),
	)

	informerFactory := informers.NewSharedInformerFactory(
		client,
		time.Second*30,
	)

	indicatorInformer := informerFactory.Indicatorprotocol().
		V1().
		Indicators().
		Informer()
	indicatorInformer.AddEventHandler(controller)

	go controller.Start()

	log.Println("running informer...")
	indicatorInformer.Run(ctx.Done())
}

var onlyOneSignalHandler = make(chan struct{})

// setupSignalHandler registers SIGTERM and SIGINT. A context is returned
// which is canceled on one of these signals. If a second signal is caught,
// the program is terminated with exit code 1.
func setupSignalHandler() context.Context {
	close(onlyOneSignalHandler) // only call once, panic on calls > 1

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()

	return ctx
}
