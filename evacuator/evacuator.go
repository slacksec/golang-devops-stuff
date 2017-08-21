package evacuator

import (
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/sender"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/yagnats"
	"github.com/nats-io/nats"
	"code.cloudfoundry.org/clock"
)

type Evacuator struct {
	messageBus        yagnats.NATSConn
	store             store.Store
	clock             clock.Clock
	metricsAccountant metricsaccountant.MetricsAccountant
	config            *config.Config
	logger            lager.Logger
	sub               *nats.Subscription
}

func New(messageBus yagnats.NATSConn, store store.Store, clock clock.Clock, metricsAccountant metricsaccountant.MetricsAccountant, config *config.Config, logger lager.Logger) *Evacuator {
	return &Evacuator{
		messageBus:        messageBus,
		store:             store,
		clock:             clock,
		metricsAccountant: metricsAccountant,
		config:            config,
		logger:            logger,
	}
}

func (e *Evacuator) listen() error {
	var err error
	e.sub, err = e.messageBus.Subscribe("droplet.exited", func(message *nats.Msg) {
		dropletExited, err := models.NewDropletExitedFromJSON([]byte(message.Data))
		if err != nil {
			e.logger.Error("Failed to parse droplet exited message", err)
			return
		}

		e.handleExited(dropletExited)
	})

	return err
}

func (e *Evacuator) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	e.logger.Info("Listening for DEA Evacuations")
	err := e.listen()
	if err != nil {
		return err
	}

	close(ready)
	select {
	case <-signals:
		if e.sub != nil {
			e.messageBus.Unsubscribe(e.sub)
		}
		return nil
	}
}

func (e *Evacuator) handleExited(exited models.DropletExited) {
	switch exited.Reason {
	case models.DropletExitedReasonDEAShutdown, models.DropletExitedReasonDEAEvacuation:
		startMessage := models.NewPendingStartMessage(
			e.clock.Now(),
			0,
			e.config.GracePeriod(),
			exited.AppGuid,
			exited.AppVersion,
			exited.InstanceIndex,
			2.0,
			models.PendingStartMessageReasonEvacuating,
		)
		startMessage.SkipVerification = true

		e.logger.Info("Scheduling start message for droplet.exited message", startMessage.LogDescription(), exited.LogDescription())

		e.store.SavePendingStartMessages(startMessage)

		evacuatorSender := sender.New(e.store, e.metricsAccountant, e.config, e.messageBus, e.logger, e.clock)
		evacuatorSender.Send(e.clock, nil, []models.PendingStartMessage{startMessage}, nil)
	}
}
