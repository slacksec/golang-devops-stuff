package models

import (
	"encoding/json"
	"strconv"

	"code.cloudfoundry.org/lager"
)

type Heartbeat struct {
	DeaGuid            string              `json:"dea"`
	InstanceHeartbeats []InstanceHeartbeat `json:"droplets"`
}

func NewHeartbeatFromJSON(encoded []byte) (*Heartbeat, error) {
	var heartbeat Heartbeat
	err := json.Unmarshal(encoded, &heartbeat)
	if err != nil {
		return nil, err
	}
	for i, instanceHeartbeat := range heartbeat.InstanceHeartbeats {
		instanceHeartbeat.DeaGuid = heartbeat.DeaGuid
		heartbeat.InstanceHeartbeats[i] = instanceHeartbeat
	}
	return &heartbeat, nil
}

func (heartbeat *Heartbeat) ToJSON() []byte {
	encoded, _ := json.Marshal(heartbeat)
	return encoded
}

func (heartbeat *Heartbeat) LogDescription() lager.Data {
	var evacuating, running, crashed, starting int
	for _, instanceHeartbeat := range heartbeat.InstanceHeartbeats {
		switch instanceHeartbeat.State {
		case InstanceStateCrashed:
			crashed += 1
		case InstanceStateEvacuating:
			evacuating += 1
		case InstanceStateRunning:
			running += 1
		case InstanceStateStarting:
			starting += 1
		}
	}
	return lager.Data{
		"DEA":        heartbeat.DeaGuid,
		"Evacuating": strconv.Itoa(evacuating),
		"Crashed":    strconv.Itoa(crashed),
		"Running":    strconv.Itoa(running),
		"Starting":   strconv.Itoa(starting),
	}
}
