package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/therealbill/libredis/client"
	"github.com/therealbill/redskull/redskull-controller/common"
	"github.com/zenazn/goji/web"
)

// ShowPods shows the pods view page
func ShowPods(c web.C, w http.ResponseWriter, r *http.Request) {
	context, err := NewPageContext()
	checkContextError(err, &w)
	pods := context.Constellation.GetPods()
	log.Printf("[SHOWPODS] Found %d pods", len(pods))
	title := "Red Skull: Known Pods"
	context.Title = title
	context.ViewTemplate = "show_pods"
	context.CurrentURL = r.URL.Path
	for k, v := range pods {
		v.NeededSentinels = v.Info.Quorum + 1
		if v.NeededSentinels > v.SentinelCount {
			v.MissingSentinels = true
		}
		if v.Master == nil {
			log.Print(v.Name, " has a nil master, probably can't log into it")
			v.HasInfo = false
			pods[k] = v
			continue
		}
		if v.Master.Info.Server.Version == "" {
			log.Print(v.Name, " has a nil master.Info pointer, probably can't log into it")
			v.HasInfo = false
			pods[k] = v
			continue
		}

		pods[k] = v
	}
	context.Data = pods
	render(w, context)
}

//ShowPod shows the view for a specific pod
func ShowPod(c web.C, w http.ResponseWriter, r *http.Request) {
	log.Print("ShowPod called")
	type PodData struct {
		Slaves     []*common.RedisNode
		Conditions map[string]bool
		Metrics    map[string]int
	}
	target := c.URLParams["podName"]
	context, err := NewPageContext()
	checkContextError(err, &w)
	context.Title = fmt.Sprintf("Pod: %s", target)
	context.ViewTemplate = "show_pod"
	pod, err := context.Constellation.GetPod(target)
	if err != nil {
		log.Printf("Unable to c.GetPod(%s) -> Error: %s", target, err)
		context.Error = err
		http.Error(w, "No such Pod", 404)
		return
	}
	sentinels := context.Constellation.GetSentinelsForPod(target)
	var updated_slaves []*common.RedisNode
	if pod == nil {
		// Need to load master here ...
		context.Error = fmt.Errorf("Unable to load master for pod %s FIXME", target)
		render(w, context)
		return
	}

	eligibleSlaves := 0
	for _, slave := range pod.Master.Slaves {
		_, err := slave.UpdateData()
		if err != nil {
			log.Printf("Error on slave.UpdateData() %s", err.Error())
			continue
		}
		if slave.MaxMemory <= pod.Master.MaxMemory {
			slave.HasEnoughMemoryForMaster = true
		} else {
			log.Printf("Slave %s has NOT enough memory", slave.Name)
			log.Printf("Slave %s has %d needs %d", slave.Name, slave.MaxMemory, pod.Master.MaxMemory)
		}
		if slave.Info.Replication.SlavePriority > 0 {
			eligibleSlaves++
		}
		if slave.Name > "" {
			updated_slaves = append(updated_slaves, slave)
		}
	}

	flydata := make(map[string]bool)
	metrics := make(map[string]int)
	flydata["SlavesHaveEnoughMemory"] = pod.SlavesHaveEnoughMemory()
	flydata["CanFailover"] = eligibleSlaves > 0
	flydata["HasQuorum"] = pod.HasQuorum()
	neededSentinels := pod.Info.Quorum + 1

	metrics["ReportedSentinels"] = pod.Info.NumOtherSentinels + 1
	metrics["NeededSentinels"] = neededSentinels
	metrics["LiveSentinels"] = len(sentinels)
	flydata["SentinelConfigMatch"] = metrics["ReportedSentinels"] == metrics["LiveSentinels"]
	flydata["HasFullSentinelComplement"] = neededSentinels <= metrics["LiveSentinels"]

	data := PodData{Slaves: updated_slaves, Conditions: flydata, Metrics: metrics}
	data.Slaves = updated_slaves
	context.Pod = pod
	context.Data = data
	context.Refresh = true
	context.RefreshURL = fmt.Sprintf("/pod/%s", pod.Name)
	context.RefreshTime = 15
	render(w, context)
}

// AddSlaveHTML shows the slave addition form
func AddSlaveHTML(c web.C, w http.ResponseWriter, r *http.Request) {
	target := c.URLParams["podName"]
	title := fmt.Sprintf("Add Slave To Pod: %s", target)
	context, err := NewPageContext()
	checkContextError(err, &w)
	pod, _ := context.Constellation.GetPod(target)
	pod.Master.LastUpdateValid = false
	pod.Master.UpdateData()
	context.Constellation.PodMap[pod.Name] = pod
	context.Constellation.LocalPodMap[pod.Name] = pod
	context.Constellation.RemotePodMap[pod.Name] = pod
	context.Title = title
	context.ViewTemplate = "add-slave-form"
	context.Pod = pod
	render(w, context)
}

// APIAddSlave is the API call handler for adding a slave
func APIAddSlave(c web.C, w http.ResponseWriter, r *http.Request) {
	target := c.URLParams["podName"]
	context, err := NewPageContext()
	checkContextError(err, &w)
	pod, _ := context.Constellation.GetPod(target)
	body, err := ioutil.ReadAll(r.Body)
	var response InfoResponse
	var reqdata common.AddSlaveRequest
	err = json.Unmarshal(body, &reqdata)
	if err != nil {
		retcode, em := throwJSONParseError(r)
		log.Print(em)
		http.Error(w, em, retcode)
	}
	reqdata.Podname = target
	name := fmt.Sprintf("%s:%d", reqdata.SlaveAddress, reqdata.SlavePort)
	slave_target, err := client.DialWithConfig(&client.DialConfig{Address: name, Password: reqdata.SlaveAuth})
	defer slave_target.ClosePool()
	if err != nil {
		log.Print("ERR: Dialing slave -", err)
		response.Status = "ERROR"
		response.StatusMessage = "Unable to connect and command slave"
		http.Error(w, "Unable to contact slave", 400)
		return
	}
	err = slave_target.SlaveOf(pod.Info.IP, fmt.Sprintf("%d", pod.Info.Port))
	if err != nil {
		log.Printf("Err: %v", err)
		if strings.Contains(err.Error(), "Already connected to specified master") {
			response.Status = "NOOP"
			response.StatusMessage = "Already connected to specified master"
			packed, _ := json.Marshal(response)
			w.Write(packed)
			return
		}
	}

	pod.Master.LastUpdateValid = false
	context.Constellation.PodMap[pod.Name] = pod
	slave_target.ConfigSet("masterauth", pod.AuthToken)
	slave_target.ConfigSet("requirepass", pod.AuthToken)
	response.Status = "COMPLETE"
	response.StatusMessage = "Slave added"
	packed, _ := json.Marshal(response)
	w.Write(packed)

}

// BalancePodProcessor calls the constellation's BalancePod function for the pod
func BalancePodProcessor(c web.C, w http.ResponseWriter, r *http.Request) {
	podname := c.URLParams["name"]
	context, err := NewPageContext()
	checkContextError(err, &w)
	context.Title = "Pod Slave Result"
	context.ViewTemplate = "balance-pod"
	context.Refresh = true
	context.RefreshURL = fmt.Sprintf("/pod/%s", podname)
	context.RefreshTime = 5
	pod, err := context.Constellation.GetPod(podname)
	if err != nil {
		log.Print("Unable to obtain entry/data for pod: " + podname + " error returned=" + err.Error())
		context.Error = err
		context.Refresh = false
		render(w, context)
	}
	go context.Constellation.BalancePod(pod)
	context.Pod = pod
	render(w, context)

}

// AddSlaveHTMLProcessor is the action target for the AddSlaveHTML form
func AddSlaveHTMLProcessor(c web.C, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Print("add slave processor called")
	podname := c.URLParams["podName"]
	context, err := NewPageContext()
	checkContextError(err, &w)
	pod, _ := context.Constellation.GetPod(podname)
	context.Title = "Pod Slave Result"
	context.ViewTemplate = "slave-added"
	context.Pod = pod
	context.Refresh = true
	context.RefreshURL = fmt.Sprintf("/pod/%s", pod.Name)
	context.RefreshTime = 5
	address := r.FormValue("host")
	sname := r.FormValue("sname")
	portstr := r.FormValue("port")
	slaveauth := r.FormValue("authtoken")
	port, _ := strconv.Atoi(portstr)

	type results struct {
		PodName      string
		SlaveName    string
		SlaveAddress string
		SlavePort    int
		Error        string
		HasError     bool
		PodURL       string
	}
	res := results{PodName: podname, SlaveName: sname, SlaveAddress: address, SlavePort: port}
	name := fmt.Sprintf("%s:%d", address, port)
	slave_target, err := client.DialWithConfig(&client.DialConfig{Address: name, Password: slaveauth})
	defer slave_target.ClosePool()
	if err != nil {
		log.Print("ERR: Dialing slave -", err)
		context.Data = err
		render(w, context)
		return
	}
	err = slave_target.SlaveOf(pod.Info.IP, fmt.Sprintf("%d", pod.Info.Port))
	if err != nil {
		log.Printf("Err: %v", err)
	} else {
		log.Printf("Slave added success")
		slave_target.ConfigSet("masterauth", pod.AuthToken)
		slave_target.ConfigSet("requirepass", pod.AuthToken)
		slave, err := common.LoadNodeFromHostPort(address, port, pod.AuthToken)
		if err != nil {
			log.Printf("In AddSlaveHTMLProcessor, unable to get new slave node")
		} else {
			pod.Master.Slaves = append(pod.Master.Slaves, slave)
		}
	}
	context.Data = res
	render(w, context)

}

// ResetPodProcessor is called to reset the pod's slave&sentinel configuration
func ResetPodProcessor(c web.C, w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	log.Print("reset pod processor called")
	podname := c.URLParams["name"]
	context, err := NewPageContext()
	checkContextError(err, &w)
	pod, _ := context.Constellation.GetPod(podname)
	context.Title = "Pod Slave Result"
	context.ViewTemplate = "reset-issued"
	context.Refresh = true
	context.RefreshURL = fmt.Sprintf("/pod/%s", pod.Name)
	context.RefreshTime = 10
	context.Pod = pod
	go context.Constellation.ResetPod(podname, false)
	render(w, context)

}
