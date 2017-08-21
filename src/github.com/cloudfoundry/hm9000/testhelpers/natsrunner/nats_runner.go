package natsrunner

import (
	"fmt"
	"os"

	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/yagnats"

	"os/exec"
	"strconv"
)

var natsCommand *exec.Cmd

type NATSRunner struct {
	natsPort           int
	natsMonitoringPort int
	natsCommand        *exec.Cmd
	MessageBus         yagnats.NATSConn
}

func NewNATSRunner(natsPort, natsMonitoringPort int) *NATSRunner {
	return &NATSRunner{
		natsPort:           natsPort,
		natsMonitoringPort: natsMonitoringPort,
	}
}

func (runner *NATSRunner) Start() {
	_, err := exec.LookPath("gnatsd")
	if err != nil {
		fmt.Println("You need gnatsd installed!")
		os.Exit(1)
	}

	runner.natsCommand = exec.Command("gnatsd", "-p", strconv.Itoa(runner.natsPort), "-m", strconv.Itoa(runner.natsMonitoringPort))
	err = runner.natsCommand.Start()
	Ω(err).ShouldNot(HaveOccurred(), "Make sure to have gnatsd on your path")
	var messageBus yagnats.NATSConn
	Eventually(func() error {
		messageBus, err = yagnats.Connect([]string{fmt.Sprintf("nats://127.0.0.1:%d", runner.natsPort)})
		return err
	}, 5, 0.1).ShouldNot(HaveOccurred())

	runner.MessageBus = messageBus
}

func (runner *NATSRunner) Stop() {
	if runner.natsCommand != nil {
		runner.natsCommand.Process.Kill()
		runner.MessageBus = nil
		runner.natsCommand = nil
	}
}
