package sender

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/yagnats"
	"code.cloudfoundry.org/clock"
)

//go:generate counterfeiter -o fakesender/fake_sender.go . Sender
type Sender interface {
	Send(clock.Clock, map[string]*models.App, []models.PendingStartMessage, []models.PendingStopMessage) error
}

type sender struct {
	store  store.Store
	conf   *config.Config
	logger lager.Logger

	apps        map[string]*models.App
	messageBus  yagnats.NATSConn
	currentTime time.Time

	numberOfStartMessagesSent int
	sentStartMessages         []models.PendingStartMessage
	startMessagesToSave       []models.PendingStartMessage
	startMessagesToDelete     []models.PendingStartMessage
	sentStopMessages          []models.PendingStopMessage
	stopMessagesToSave        []models.PendingStopMessage
	stopMessagesToDelete      []models.PendingStopMessage
	metricsAccountant         metricsaccountant.MetricsAccountant
	clock                     clock.Clock
	didSucceed                bool
}

func New(store store.Store, metricsAccountant metricsaccountant.MetricsAccountant, conf *config.Config, messageBus yagnats.NATSConn, logger lager.Logger, clock clock.Clock) *sender {
	return &sender{
		store:                 store,
		conf:                  conf,
		logger:                logger,
		messageBus:            messageBus,
		sentStartMessages:     []models.PendingStartMessage{},
		startMessagesToSave:   []models.PendingStartMessage{},
		startMessagesToDelete: []models.PendingStartMessage{},
		sentStopMessages:      []models.PendingStopMessage{},
		stopMessagesToSave:    []models.PendingStopMessage{},
		stopMessagesToDelete:  []models.PendingStopMessage{},
		metricsAccountant:     metricsAccountant,
		didSucceed:            true,
		clock:                 clock,
	}
}

func (sender *sender) Send(clock clock.Clock, apps map[string]*models.App, pendingStartMessages []models.PendingStartMessage, pendingStopMessages []models.PendingStopMessage) error {
	sender.currentTime = clock.Now()
	sender.apps = apps

	err := sender.store.VerifyFreshness(sender.currentTime)
	if err != nil {
		sender.logger.Error("Store is not fresh", err)
		return err
	}

	sender.sendStartMessages(pendingStartMessages)
	sender.sendStopMessages(pendingStopMessages)

	err = sender.metricsAccountant.IncrementSentMessageMetrics(sender.sentStartMessages, sender.sentStopMessages)
	if err != nil {
		sender.logger.Error("Failed to increment metrics", err)
		sender.didSucceed = false
	}

	err = sender.store.SavePendingStartMessages(sender.startMessagesToSave...)
	if err != nil {
		sender.logger.Error("Failed to save start messages", err)
		sender.didSucceed = false
	}

	err = sender.store.DeletePendingStartMessages(sender.startMessagesToDelete...)
	if err != nil {
		sender.logger.Error("Failed to delete start messages", err)
		sender.didSucceed = false
	}

	err = sender.store.SavePendingStopMessages(sender.stopMessagesToSave...)
	if err != nil {
		sender.logger.Error("Failed to save stop messages", err)
		sender.didSucceed = false
	}

	err = sender.store.DeletePendingStopMessages(sender.stopMessagesToDelete...)
	if err != nil {
		sender.logger.Error("Failed to delete stop messages", err)
		sender.didSucceed = false
	}

	if !sender.didSucceed {
		return errors.New("Sender failed. See logs for details.")
	}

	return nil
}

func (sender *sender) sendStartMessages(startMessages []models.PendingStartMessage) {
	sortedStartMessages := models.SortStartMessagesByPriority(startMessages)

	for _, startMessage := range sortedStartMessages {
		if startMessage.IsTimeToSend(sender.currentTime) {
			sender.sendStartMessageHttp(startMessage)
		} else if startMessage.IsExpired(sender.currentTime) {
			sender.queueStartMessageForDeletion(startMessage, "expired start message")
		}
	}
}

func (sender *sender) sendStopMessages(stopMessages []models.PendingStopMessage) {
	for _, stopMessage := range stopMessages {
		if stopMessage.IsTimeToSend(sender.currentTime) {
			sender.sendStopMessageHttp(stopMessage)
		} else if stopMessage.IsExpired(sender.currentTime) {
			sender.queueStopMessageForDeletion(stopMessage, "expired stop message")
		}
	}
}

func (sender *sender) sendStartMessage(startMessage models.PendingStartMessage) {
	messageToSend, shouldSend := sender.startMessageToSend(startMessage)
	if shouldSend {
		if sender.numberOfStartMessagesSent < sender.conf.SenderMessageLimit {
			sender.logger.Info("Sending message", startMessage.LogDescription())
			err := sender.messageBus.Publish(sender.conf.SenderNatsStartSubject, messageToSend.ToJSON())

			if err != nil {
				sender.logger.Error("Failed to send start message", err, startMessage.LogDescription())
				sender.didSucceed = false
				return
			}

			sender.sentStartMessages = append(sender.sentStartMessages, startMessage)

			if startMessage.KeepAlive == 0 {
				sender.queueStartMessageForDeletion(startMessage, "a sent start message with no keep alive")
			} else {
				sender.markStartMessageSent(startMessage)
			}

			sender.numberOfStartMessagesSent += 1
		}
	} else {
		sender.queueStartMessageForDeletion(startMessage, "start message that will not be sent")
	}
}

func (sender *sender) sendStartMessageHttp(startMessage models.PendingStartMessage) {
	messageToSend, shouldSend := sender.startMessageToSend(startMessage)

	if shouldSend {
		if sender.numberOfStartMessagesSent < sender.conf.SenderMessageLimit {
			sender.logger.Info("Sending message", startMessage.LogDescription())

			client := http.Client{}
			targetURL := fmt.Sprintf("%s/internal/dea/hm9000/start/%s", sender.conf.CCInternalURL, startMessage.AppGuid)
			req, err := http.NewRequest("POST", targetURL, bytes.NewReader(messageToSend.ToJSON()))

			authorization := models.BasicAuthInfo{
				User:     sender.conf.APIServerUsername,
				Password: sender.conf.APIServerPassword,
			}.Encode()

			req.Header.Add("Authorization", authorization)
			req.Header.Set("Content-Type", "application/json")
			resp, err := client.Do(req)

			if err != nil {
				sender.logger.Error("Failed to send start message", err, startMessage.LogDescription())
				sender.didSucceed = false
				return
			}

			if resp.StatusCode != 200 {
				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)
				respBody := buf.String()
				err = errors.New(respBody)

				sender.logger.Error("Cloud Controller did not accept start message", err, startMessage.LogDescription())
				sender.didSucceed = false
				return
			}

			sender.sentStartMessages = append(sender.sentStartMessages, startMessage)
			sender.numberOfStartMessagesSent += 1

			if startMessage.KeepAlive == 0 {
				sender.queueStartMessageForDeletion(startMessage, "sent start message with no keep alive")
			} else {
				sender.markStartMessageSent(startMessage)
			}
		}
	} else {
		sender.queueStartMessageForDeletion(startMessage, "start message that will not be sent")
	}
}

func (sender *sender) sendStopMessageHttp(stopMessage models.PendingStopMessage) {
	messageToSend, shouldSend := sender.stopMessageToSend(stopMessage)

	if shouldSend {
		client := http.Client{}
		targetURL := fmt.Sprintf("%s/internal/dea/hm9000/stop/%s", sender.conf.CCInternalURL, stopMessage.AppGuid)
		req, err := http.NewRequest("POST", targetURL, bytes.NewReader(messageToSend.ToJSON()))

		authorization := models.BasicAuthInfo{
			User:     sender.conf.APIServerUsername,
			Password: sender.conf.APIServerPassword,
		}.Encode()

		req.Header.Add("Authorization", authorization)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)

		if err != nil {
			sender.logger.Error("Failed to send stop message", err, stopMessage.LogDescription())
			sender.didSucceed = false
			return
		}

		if resp.StatusCode != 200 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			respBody := buf.String()
			err = errors.New(respBody)

			sender.logger.Error("Cloud Controller did not accept stop message", err, stopMessage.LogDescription())
			sender.didSucceed = false
			return
		}

		sender.sentStopMessages = append(sender.sentStopMessages, stopMessage)

		if stopMessage.KeepAlive == 0 {
			sender.queueStopMessageForDeletion(stopMessage, "sent stop message with no keep alive")
		} else {
			sender.markStopMessageSent(stopMessage)
		}
	} else {
		sender.queueStopMessageForDeletion(stopMessage, "stop message that will not be sent")
	}
}

func (sender *sender) markStartMessageSent(startMessage models.PendingStartMessage) {
	startMessage.SentOn = sender.currentTime.Unix()
	sender.startMessagesToSave = append(sender.startMessagesToSave, startMessage)
}

func (sender *sender) markStopMessageSent(stopMessage models.PendingStopMessage) {
	stopMessage.SentOn = sender.currentTime.Unix()
	sender.stopMessagesToSave = append(sender.stopMessagesToSave, stopMessage)
}

func (sender *sender) queueStartMessageForDeletion(startMessage models.PendingStartMessage, reason string) {
	sender.logger.Info(fmt.Sprintf("Deleting %s", reason), startMessage.LogDescription())
	sender.startMessagesToDelete = append(sender.startMessagesToDelete, startMessage)
}

func (sender *sender) queueStopMessageForDeletion(stopMessage models.PendingStopMessage, reason string) {
	sender.logger.Info(fmt.Sprintf("Deleting %s", reason), stopMessage.LogDescription())
	sender.stopMessagesToDelete = append(sender.stopMessagesToDelete, stopMessage)
}

func (sender *sender) startMessageToSend(message models.PendingStartMessage) (models.StartMessage, bool) {
	messageToSend := models.StartMessage{
		MessageId:     message.MessageId,
		AppGuid:       message.AppGuid,
		AppVersion:    message.AppVersion,
		InstanceIndex: message.IndexToStart,
	}

	if message.SkipVerification {
		sender.logger.Info("Sending start message: message is marked with SkipVerification", message.LogDescription())
		return messageToSend, true
	}

	appKey := sender.store.AppKey(message.AppGuid, message.AppVersion)
	app, found := sender.apps[appKey]

	if !found {
		sender.logger.Info("Skipping sending start message: app is no longer desired", message.LogDescription())
		return models.StartMessage{}, false
	}

	if !app.IsDesired() {
		sender.logger.Info("Skipping sending start message: app is no longer desired", message.LogDescription(), app.LogDescription())
		return models.StartMessage{}, false
	}

	if !app.IsIndexDesired(message.IndexToStart) {
		sender.logger.Info("Skipping sending start message: instance index is beyond the desired # of instances", message.LogDescription(), app.LogDescription())
		return models.StartMessage{}, false
	}

	if app.HasStartingOrRunningInstanceAtIndex(message.IndexToStart) {
		sender.logger.Info("Skipping sending start message: instance is already running", message.LogDescription(), app.LogDescription())
		return models.StartMessage{}, false
	}

	sender.logger.Info("Sending start message: instance is not running at desired index", message.LogDescription(), app.LogDescription())
	return messageToSend, true
}

func (sender *sender) stopMessageToSend(message models.PendingStopMessage) (models.StopMessage, bool) {
	appKey := sender.store.AppKey(message.AppGuid, message.AppVersion)
	app, found := sender.apps[appKey]

	if !found {
		sender.logger.Info("Skipping sending stop message: instance is no longer running", message.LogDescription())
		return models.StopMessage{}, false
	}

	instanceToStop := app.InstanceWithGuid(message.InstanceGuid)
	messageToSend := models.StopMessage{
		AppGuid:       message.AppGuid,
		AppVersion:    message.AppVersion,
		InstanceGuid:  message.InstanceGuid,
		InstanceIndex: instanceToStop.InstanceIndex,
		MessageId:     message.MessageId,
	}

	if !app.IsDesired() {
		sender.logger.Info("Sending stop message: instance is running, app is no longer desired", message.LogDescription(), app.LogDescription())
		messageToSend.IsDuplicate = false
		return messageToSend, true
	}

	if !app.IsIndexDesired(instanceToStop.InstanceIndex) {
		sender.logger.Info("Sending stop message: index of instance to stop is beyond desired # of instances", message.LogDescription(), app.LogDescription())
		messageToSend.IsDuplicate = false
		return messageToSend, true
	}

	if instanceToStop.State == models.InstanceStateEvacuating {
		sender.logger.Info("Sending stop message for evacuating app", message.LogDescription(), app.LogDescription())
		messageToSend.IsDuplicate = true
		return messageToSend, true
	}

	if len(app.StartingOrRunningInstancesAtIndex(instanceToStop.InstanceIndex)) > 1 {
		sender.logger.Info("Sending stop message: instance is a duplicate running at a desired index", message.LogDescription(), app.LogDescription())
		messageToSend.IsDuplicate = true
		return messageToSend, true
	}

	sender.logger.Info("Skipping sending stop message: instance is running on a desired index (and there are no other instances running at that index)", message.LogDescription(), app.LogDescription())
	return models.StopMessage{}, false
}
