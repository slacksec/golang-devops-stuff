// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"
	"time"

	"code.cloudfoundry.org/gorouter/metrics"
	"code.cloudfoundry.org/gorouter/route"
)

type FakeVarzReporter struct {
	CaptureBadRequestStub            func()
	captureBadRequestMutex           sync.RWMutex
	captureBadRequestArgsForCall     []struct{}
	CaptureBadGatewayStub            func()
	captureBadGatewayMutex           sync.RWMutex
	captureBadGatewayArgsForCall     []struct{}
	CaptureRoutingRequestStub        func(b *route.Endpoint)
	captureRoutingRequestMutex       sync.RWMutex
	captureRoutingRequestArgsForCall []struct {
		b *route.Endpoint
	}
	CaptureRoutingResponseLatencyStub        func(b *route.Endpoint, statusCode int, t time.Time, d time.Duration)
	captureRoutingResponseLatencyMutex       sync.RWMutex
	captureRoutingResponseLatencyArgsForCall []struct {
		b          *route.Endpoint
		statusCode int
		t          time.Time
		d          time.Duration
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeVarzReporter) CaptureBadRequest() {
	fake.captureBadRequestMutex.Lock()
	fake.captureBadRequestArgsForCall = append(fake.captureBadRequestArgsForCall, struct{}{})
	fake.recordInvocation("CaptureBadRequest", []interface{}{})
	fake.captureBadRequestMutex.Unlock()
	if fake.CaptureBadRequestStub != nil {
		fake.CaptureBadRequestStub()
	}
}

func (fake *FakeVarzReporter) CaptureBadRequestCallCount() int {
	fake.captureBadRequestMutex.RLock()
	defer fake.captureBadRequestMutex.RUnlock()
	return len(fake.captureBadRequestArgsForCall)
}

func (fake *FakeVarzReporter) CaptureBadGateway() {
	fake.captureBadGatewayMutex.Lock()
	fake.captureBadGatewayArgsForCall = append(fake.captureBadGatewayArgsForCall, struct{}{})
	fake.recordInvocation("CaptureBadGateway", []interface{}{})
	fake.captureBadGatewayMutex.Unlock()
	if fake.CaptureBadGatewayStub != nil {
		fake.CaptureBadGatewayStub()
	}
}

func (fake *FakeVarzReporter) CaptureBadGatewayCallCount() int {
	fake.captureBadGatewayMutex.RLock()
	defer fake.captureBadGatewayMutex.RUnlock()
	return len(fake.captureBadGatewayArgsForCall)
}

func (fake *FakeVarzReporter) CaptureRoutingRequest(b *route.Endpoint) {
	fake.captureRoutingRequestMutex.Lock()
	fake.captureRoutingRequestArgsForCall = append(fake.captureRoutingRequestArgsForCall, struct {
		b *route.Endpoint
	}{b})
	fake.recordInvocation("CaptureRoutingRequest", []interface{}{b})
	fake.captureRoutingRequestMutex.Unlock()
	if fake.CaptureRoutingRequestStub != nil {
		fake.CaptureRoutingRequestStub(b)
	}
}

func (fake *FakeVarzReporter) CaptureRoutingRequestCallCount() int {
	fake.captureRoutingRequestMutex.RLock()
	defer fake.captureRoutingRequestMutex.RUnlock()
	return len(fake.captureRoutingRequestArgsForCall)
}

func (fake *FakeVarzReporter) CaptureRoutingRequestArgsForCall(i int) *route.Endpoint {
	fake.captureRoutingRequestMutex.RLock()
	defer fake.captureRoutingRequestMutex.RUnlock()
	return fake.captureRoutingRequestArgsForCall[i].b
}

func (fake *FakeVarzReporter) CaptureRoutingResponseLatency(b *route.Endpoint, statusCode int, t time.Time, d time.Duration) {
	fake.captureRoutingResponseLatencyMutex.Lock()
	fake.captureRoutingResponseLatencyArgsForCall = append(fake.captureRoutingResponseLatencyArgsForCall, struct {
		b          *route.Endpoint
		statusCode int
		t          time.Time
		d          time.Duration
	}{b, statusCode, t, d})
	fake.recordInvocation("CaptureRoutingResponseLatency", []interface{}{b, statusCode, t, d})
	fake.captureRoutingResponseLatencyMutex.Unlock()
	if fake.CaptureRoutingResponseLatencyStub != nil {
		fake.CaptureRoutingResponseLatencyStub(b, statusCode, t, d)
	}
}

func (fake *FakeVarzReporter) CaptureRoutingResponseLatencyCallCount() int {
	fake.captureRoutingResponseLatencyMutex.RLock()
	defer fake.captureRoutingResponseLatencyMutex.RUnlock()
	return len(fake.captureRoutingResponseLatencyArgsForCall)
}

func (fake *FakeVarzReporter) CaptureRoutingResponseLatencyArgsForCall(i int) (*route.Endpoint, int, time.Time, time.Duration) {
	fake.captureRoutingResponseLatencyMutex.RLock()
	defer fake.captureRoutingResponseLatencyMutex.RUnlock()
	return fake.captureRoutingResponseLatencyArgsForCall[i].b, fake.captureRoutingResponseLatencyArgsForCall[i].statusCode, fake.captureRoutingResponseLatencyArgsForCall[i].t, fake.captureRoutingResponseLatencyArgsForCall[i].d
}

func (fake *FakeVarzReporter) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.captureBadRequestMutex.RLock()
	defer fake.captureBadRequestMutex.RUnlock()
	fake.captureBadGatewayMutex.RLock()
	defer fake.captureBadGatewayMutex.RUnlock()
	fake.captureRoutingRequestMutex.RLock()
	defer fake.captureRoutingRequestMutex.RUnlock()
	fake.captureRoutingResponseLatencyMutex.RLock()
	defer fake.captureRoutingResponseLatencyMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeVarzReporter) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ metrics.VarzReporter = new(FakeVarzReporter)
