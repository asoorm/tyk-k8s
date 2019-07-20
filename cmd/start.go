package cmd

import (
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/TykTechnologies/tyk-k8s/ingress"
	"github.com/TykTechnologies/tyk-k8s/injector"
	"github.com/TykTechnologies/tyk-k8s/logger"
	"github.com/TykTechnologies/tyk-k8s/webserver"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var log = logger.GetLogger("main")

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "starts the controller",
	Long:  `Starts the controller.`,
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		sConf := &webserver.Config{}
		err = viper.UnmarshalKey("Server", sConf)
		fatalOnErr(err, "no Server entry in config file")

		whConf := &injector.Config{}
		err = viper.UnmarshalKey("Injector", whConf)
		fatalOnErr(err, "couldn't read injector config")

		whs := &injector.WebhookServer{
			SidecarConfig: whConf,
		}

		webserver.Server().Config(sConf)
		webserver.Server().AddRoute(http.MethodPost, "/inject", whs.Serve)

		// Ingress controller
		ingress.NewController()
		err = ingress.Controller().Start()
		fatalOnErr(err, "unable to start ingress controller")

		log.Info("Ingress controller started")

		go webserver.Server().Start()
		log.Info("web server started")

		waitForCtrlC()

		err = webserver.Server().Stop()
		fatalOnErr(err, "unable to stop web server")

		err = ingress.Controller().Stop()
		fatalOnErr(err, "unable to stop ingress controller")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}

// fatalOnErr logs error msg & exits with non 0 exit code when error is not nil
func fatalOnErr(err error, fmt string, fields ...interface{}) {
	if err != nil {
		log.WithError(err).Fatalf(fmt, fields...)
	}
}

// waitForCtrlC blocks until signal to terminate
func waitForCtrlC() {
	var endWaiter sync.WaitGroup
	endWaiter.Add(1)
	var signalCh chan os.Signal
	signalCh = make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	go func() {
		<-signalCh
		endWaiter.Done()
	}()
	endWaiter.Wait()
}
