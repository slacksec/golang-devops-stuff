package hm

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/hm9000/apiserver/handlers"
	"github.com/cloudfoundry/hm9000/config"

	"github.com/tedsuo/ifrit/http_server"

	"code.cloudfoundry.org/lager"
)

func ServeAPI(l lager.Logger, conf *config.Config) {
	store := connectToStore(l, conf)

	apiHandler, err := handlers.New(l, store, buildClock(l))
	if err != nil {
		l.Error("initialize-handler.failed", err)
		panic(err)
	}
	handler := handlers.BasicAuthWrap(apiHandler, conf.APIServerUsername, conf.APIServerPassword)

	listenAddr := fmt.Sprintf("%s:%d", conf.APIServerAddress, conf.APIServerPort)

	hs := http_server.New(listenAddr, handler)

	err = ifritize(l, "api", hs, conf)
	if err != nil {
		l.Error("exited", err)
		os.Exit(1)
	}

	l.Info("exited")
	os.Exit(0)
}
