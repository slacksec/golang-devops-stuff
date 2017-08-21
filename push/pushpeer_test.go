/*
 * Copyright 2011 Nan Deng
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package push

import (
	"fmt"
	"testing"
)

type testPushServiceType struct {
	name string
}

func newTestPushServiceType() *testPushServiceType {
	ret := new(testPushServiceType)
	ret.name = "testService"
	return ret
}

func (t *testPushServiceType) Name() string {
	return t.name
}

func (t *testPushServiceType) Finalize() {
	return
}

func (t *testPushServiceType) BuildPushServiceProviderFromMap(kv map[string]string, psp *PushServiceProvider) error {
	for k, v := range kv {
		psp.FixedData[k] = v
	}
	return nil
}

func (t *testPushServiceType) BuildDeliveryPointFromMap(kv map[string]string, dp *DeliveryPoint) error {
	for k, v := range kv {
		dp.FixedData[k] = v
	}
	return nil
}

func (t *testPushServiceType) Push(*PushServiceProvider, <-chan *DeliveryPoint, chan<- *PushResult, *Notification) {
	fmt.Print("Push!\n")
}

func (t *testPushServiceType) Preview(*Notification) ([]byte, PushError) {
	fmt.Print("Preview!\n")
	return []byte("{}"), nil
}

func (t *testPushServiceType) SetErrorReportChan(chan<- PushError) {}

func TestPushPeer(t *testing.T) {
	pp := new(PushPeer)
	tpst := newTestPushServiceType()
	pp.pushServiceType = tpst
	pp.FixedData = make(map[string]string, 2)
	pp.FixedData["senderid"] = "uniqush.go@gmail.com"
	pp.FixedData["authtoken"] = "fasdf"
	pp.FixedData["service"] = "servicenameHavingTestService"

	pp.VolatileData = make(map[string]string, 1)
	pp.VolatileData["realauthtoken"] = "fsfad"

	fmt.Printf("Name: %s\n", pp.Name())

	str := pp.Marshal()
	fmt.Printf("Marshal: %s\n", string(str))

	psm := GetPushServiceManager()

	psm.RegisterPushServiceType(tpst)

	psp, err := psm.BuildPushServiceProviderFromBytes(str)
	if err != nil {
		t.Errorf("BuildPushServiceProviderFromBytes failed: %v\n", err)
		return
	}
	fmt.Printf("Push Service: %s", psp.String())
	value := psp.Marshal()
	fmt.Printf("PSP Name: %v\n", psp.Name())
	fmt.Printf("PSP Marshal: %s\n", string(value))
}

func TestCompatability(t *testing.T) {
	pspm := make(map[string]string, 2)
	pspm["pushservicetype"] = "testService"
	pspm["senderid"] = "uniqush.go@gmail.com"
	pspm["authtoken"] = "fsafds"
	pspm["service"] = "servicenameHavingTestService"

	dpm := make(map[string]string, 2)
	dpm["pushservicetype"] = "testService"
	dpm["regid"] = "fdsafas"
	dpm["subscriber"] = "subscriber.1234"

	tpst := newTestPushServiceType()
	psm := GetPushServiceManager()
	psm.RegisterPushServiceType(tpst)

	psp, err := psm.BuildPushServiceProviderFromMap(pspm)
	if err != nil {
		t.Fatalf("BuildPushServiceProviderFromMap failed: %v", err)
	}
	dp, err := psm.BuildDeliveryPointFromMap(dpm)
	if err != nil {
		t.Fatalf("BuildDeliveryPointFromMap failed: %v", err)
	}

	serviceNamePSP := psp.PushServiceName()
	serviceNameDP := dp.PushServiceName()
	if serviceNamePSP != serviceNameDP {
		t.Errorf("Should be compatible, but %q != %q\n", serviceNamePSP, serviceNameDP)
	}
}
