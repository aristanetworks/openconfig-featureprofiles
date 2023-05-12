// Copyright 2023 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package qos_basic_test

import (
	"testing"
	"time"

	"github.com/openconfig/featureprofiles/internal/attrs"
	"github.com/openconfig/featureprofiles/internal/deviations"
	"github.com/openconfig/featureprofiles/internal/fptest"
	"github.com/openconfig/ondatra"
	"github.com/openconfig/ondatra/gnmi"
	"github.com/openconfig/ondatra/gnmi/oc"
	"github.com/openconfig/ygot/ygot"
)

var (
	intf1 = attrs.Attributes{
		Name:    "ate1",
		MAC:     "02:00:01:01:01:01",
		IPv4:    "198.51.100.1",
		IPv4Len: 31,
	}

	intf2 = attrs.Attributes{
		Name:    "ate2",
		MAC:     "02:00:01:02:01:01",
		IPv4:    "198.51.100.3",
		IPv4Len: 31,
	}

	intf3 = attrs.Attributes{
		Name:    "ate3",
		MAC:     "02:00:01:03:01:01",
		IPv4:    "198.51.100.5",
		IPv4Len: 31,
	}

	dutPort1 = attrs.Attributes{
		IPv4: "198.51.100.0",
	}
	dutPort2 = attrs.Attributes{
		IPv4: "198.51.100.2",
	}
	dutPort3 = attrs.Attributes{
		IPv4: "198.51.100.4",
	}
)

type trafficData struct {
	trafficRate           float64
	expectedThroughputPct float32
	frameSize             uint32
	dscp                  uint8
	queue                 string
	inputIntf             attrs.Attributes
}

func TestMain(m *testing.M) {
	fptest.RunTests(m)
}

// Test cases:
//  - Verify that there is no traffic loss:
//    1) Non-oversubscription traffic with 80% of linerate.
//    2) Non-oversubscription traffic with 98% of linerate.
//
// Topology:
//       ATE port 1
//        |
//       DUT--------ATE port 3
//        |
//       ATE port 2
//
//  Sample CLI command to get telemetry using gmic:
//   - gnmic -a ipaddr:10162 -u username -p password --skip-verify get \
//      --path /components/component --format flat
//   - gnmic tool info:
//     - https://github.com/karimra/gnmic/blob/main/README.md
//

func TestBasicConfigWithTraffic(t *testing.T) {
	dut := ondatra.DUT(t, "dut")
	dp1 := dut.Port(t, "port1")
	dp2 := dut.Port(t, "port2")
	dp3 := dut.Port(t, "port3")

	// Configure DUT interfaces and QoS.
	ConfigureDUTIntf(t, dut)
	switch dut.Vendor() {
	case ondatra.CISCO:
		ConfigureCiscoQos(t, dut)
	case ondatra.JUNIPER:
		ConfigureJuniperQos(t, dut)
	default:
		ConfigureQoS(t, dut)
	}

	// Configure ATE interfaces.
	ate := ondatra.ATE(t, "ate")
	ap1 := ate.Port(t, "port1")
	ap2 := ate.Port(t, "port2")
	ap3 := ate.Port(t, "port3")
	top := ate.OTG().NewConfig(t)

	intf1.AddToOTG(top, ap1, &dutPort1)
	intf2.AddToOTG(top, ap2, &dutPort2)
	intf3.AddToOTG(top, ap3, &dutPort3)
	ate.OTG().PushConfig(t, top)

	queueMap := map[ondatra.Vendor]map[string]string{
		ondatra.JUNIPER: {
			"NC1": "3",
			"AF4": "2",
			"AF3": "5",
			"AF2": "1",
			"AF1": "4",
			"BE1": "0",
			"BE0": "6",
		},
		ondatra.ARISTA: {
			"NC1": "NC1",
			"AF4": "AF4",
			"AF3": "AF3",
			"AF2": "AF2",
			"AF1": "AF1",
			"BE1": "BE1",
			"BE0": "BE0",
		},
		ondatra.CISCO: {
			"NC1": "a_NC1",
			"AF4": "b_AF4",
			"AF3": "c_AF3",
			"AF2": "d_AF2",
			"AF1": "e_AF1",
			"BE0": "f_BE0",
			"BE1": "g_BE1",
		},
		ondatra.NOKIA: {
			"NC1": "7",
			"AF4": "4",
			"AF3": "3",
			"AF2": "2",
			"AF1": "0",
			"BE1": "1",
			"BE0": "1",
		},
	}

	// Test case 1: Non-oversubscription traffic with 80% of linerate.
	//   - There should be no packet drop for all traffic classes.
	NonoversubscribedTrafficFlows1 := map[string]*trafficData{
		"intf1-nc1": {
			frameSize:             1000,
			trafficRate:           3,
			expectedThroughputPct: 100.0,
			dscp:                  56,
			queue:                 queueMap[dut.Vendor()]["NC1"],
			inputIntf:             intf1,
		},
		"intf1-af4": {
			frameSize:             1000,
			trafficRate:           24,
			expectedThroughputPct: 100.0,
			dscp:                  32,
			queue:                 queueMap[dut.Vendor()]["AF4"],
			inputIntf:             intf1,
		},
		"intf1-af3": {
			frameSize:             1000,
			trafficRate:           6,
			expectedThroughputPct: 100.0,
			dscp:                  24,
			queue:                 queueMap[dut.Vendor()]["AF3"],
			inputIntf:             intf1,
		},
		"intf1-af2": {
			frameSize:             1000,
			trafficRate:           4,
			expectedThroughputPct: 100.0,
			dscp:                  16,
			queue:                 queueMap[dut.Vendor()]["AF2"],
			inputIntf:             intf1,
		},
		"intf1-af1": {
			frameSize:             1000,
			trafficRate:           2,
			expectedThroughputPct: 100.0,
			dscp:                  8,
			queue:                 queueMap[dut.Vendor()]["AF1"],
			inputIntf:             intf1,
		},
		"intf1-be0": {
			frameSize:             1000,
			trafficRate:           0.5,
			expectedThroughputPct: 100.0,
			dscp:                  4,
			queue:                 queueMap[dut.Vendor()]["BE0"],
			inputIntf:             intf1,
		},
		"intf1-be1": {
			frameSize:             1000,
			trafficRate:           0.5,
			dscp:                  0,
			expectedThroughputPct: 100.0,
			queue:                 queueMap[dut.Vendor()]["BE1"],
			inputIntf:             intf1,
		},
		"intf2-nc1": {
			frameSize:             1000,
			trafficRate:           3,
			dscp:                  56,
			expectedThroughputPct: 100.0,
			queue:                 queueMap[dut.Vendor()]["NC1"],
			inputIntf:             intf2,
		},
		"intf2-af4": {
			frameSize:             1000,
			trafficRate:           24,
			dscp:                  32,
			expectedThroughputPct: 100.0,
			queue:                 queueMap[dut.Vendor()]["AF4"],
			inputIntf:             intf2,
		},
		"intf2-af3": {
			frameSize:             1000,
			trafficRate:           6,
			expectedThroughputPct: 100.0,
			dscp:                  24,
			queue:                 queueMap[dut.Vendor()]["AF3"],
			inputIntf:             intf2,
		},
		"intf2-af2": {
			frameSize:             1000,
			trafficRate:           4,
			expectedThroughputPct: 100.0,
			dscp:                  16,
			queue:                 queueMap[dut.Vendor()]["AF2"],
			inputIntf:             intf2,
		},
		"intf2-af1": {
			frameSize:             1000,
			trafficRate:           2,
			expectedThroughputPct: 100.0,
			dscp:                  8,
			queue:                 queueMap[dut.Vendor()]["AF1"],
			inputIntf:             intf2,
		},
		"intf2-be0": {
			frameSize:             1000,
			trafficRate:           0.5,
			dscp:                  4,
			expectedThroughputPct: 100.0,
			queue:                 queueMap[dut.Vendor()]["BE0"],
			inputIntf:             intf2,
		},
		"intf2-be1": {
			frameSize:             1000,
			trafficRate:           0.5,
			expectedThroughputPct: 100.0,
			dscp:                  0,
			queue:                 queueMap[dut.Vendor()]["BE1"],
			inputIntf:             intf2,
		},
	}

	// Test case 2: Non-oversubscription traffic with 98% of linerate.
	//   - There should be no packet drop for all traffic classes.
	NonoversubscribedTrafficFlows2 := map[string]*trafficData{
		"intf1-nc1": {
			frameSize:             1000,
			trafficRate:           5,
			expectedThroughputPct: 100.0,
			dscp:                  56,
			queue:                 queueMap[dut.Vendor()]["NC1"],
			inputIntf:             intf1,
		},
		"intf1-af4": {
			frameSize:             1000,
			trafficRate:           30,
			expectedThroughputPct: 100.0,
			dscp:                  32,
			queue:                 queueMap[dut.Vendor()]["AF4"],
			inputIntf:             intf1,
		},
		"intf1-af3": {
			frameSize:             1000,
			trafficRate:           7.5,
			expectedThroughputPct: 100.0,
			dscp:                  24,
			queue:                 queueMap[dut.Vendor()]["AF3"],
			inputIntf:             intf1,
		},
		"intf1-af2": {
			frameSize:             1000,
			trafficRate:           4,
			expectedThroughputPct: 100.0,
			dscp:                  16,
			queue:                 queueMap[dut.Vendor()]["AF2"],
			inputIntf:             intf1,
		},
		"intf1-af1": {
			frameSize:             1000,
			trafficRate:           2,
			expectedThroughputPct: 100.0,
			dscp:                  8,
			queue:                 queueMap[dut.Vendor()]["AF1"],
			inputIntf:             intf1,
		},
		"intf1-be0": {
			frameSize:             1000,
			trafficRate:           0.5,
			expectedThroughputPct: 100.0,
			dscp:                  4,
			queue:                 queueMap[dut.Vendor()]["BE0"],
			inputIntf:             intf1,
		},
		"intf1-be1": {
			frameSize:             1000,
			trafficRate:           0.5,
			dscp:                  0,
			expectedThroughputPct: 100.0,
			queue:                 queueMap[dut.Vendor()]["BE1"],
			inputIntf:             intf1,
		},
		"intf2-nc1": {
			frameSize:             1000,
			trafficRate:           4,
			dscp:                  56,
			expectedThroughputPct: 100.0,
			queue:                 queueMap[dut.Vendor()]["NC1"],
			inputIntf:             intf2,
		},
		"intf2-af4": {
			frameSize:             1000,
			trafficRate:           30,
			dscp:                  32,
			expectedThroughputPct: 100.0,
			queue:                 queueMap[dut.Vendor()]["AF4"],
			inputIntf:             intf2,
		},
		"intf2-af3": {
			frameSize:             1000,
			trafficRate:           7.5,
			expectedThroughputPct: 100.0,
			dscp:                  24,
			queue:                 queueMap[dut.Vendor()]["AF3"],
			inputIntf:             intf2,
		},
		"intf2-af2": {
			frameSize:             1000,
			trafficRate:           4,
			expectedThroughputPct: 100.0,
			dscp:                  16,
			queue:                 queueMap[dut.Vendor()]["AF2"],
			inputIntf:             intf2,
		},
		"intf2-af1": {
			frameSize:             1000,
			trafficRate:           2,
			expectedThroughputPct: 100.0,
			dscp:                  8,
			queue:                 queueMap[dut.Vendor()]["AF1"],
			inputIntf:             intf2,
		},
		"intf2-be0": {
			frameSize:             1000,
			trafficRate:           0.5,
			dscp:                  4,
			expectedThroughputPct: 100.0,
			queue:                 queueMap[dut.Vendor()]["BE0"],
			inputIntf:             intf2,
		},
		"intf2-be1": {
			frameSize:             1000,
			trafficRate:           0.5,
			expectedThroughputPct: 100.0,
			dscp:                  0,
			queue:                 queueMap[dut.Vendor()]["BE1"],
			inputIntf:             intf2,
		},
	}

	cases := []struct {
		desc         string
		trafficFlows map[string]*trafficData
	}{{
		desc:         "Non-oversubscription 80 percent of linerate traffic",
		trafficFlows: NonoversubscribedTrafficFlows1,
	}, {
		desc:         "Non-oversubscription 98 percent of linerate traffic",
		trafficFlows: NonoversubscribedTrafficFlows2,
	}}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			trafficFlows := tc.trafficFlows
			top.Flows().Clear()

			for trafficID, data := range trafficFlows {
				t.Logf("Configuring flow %s", trafficID)
				flow := top.Flows().Add().SetName(trafficID)
				flow.Metrics().SetEnable(true)
				flow.TxRx().Device().SetTxNames([]string{data.inputIntf.Name + ".IPv4"}).SetRxNames([]string{intf3.Name + ".IPv4"})
				ethHeader := flow.Packet().Add().Ethernet()
				ethHeader.Src().SetValue(data.inputIntf.MAC)

				ipHeader := flow.Packet().Add().Ipv4()
				ipHeader.Src().SetValue(data.inputIntf.IPv4)
				ipHeader.Dst().SetValue(intf3.IPv4)
				ipHeader.Priority().Dscp().Phb().SetValue(int32(data.dscp))

				flow.Size().SetFixed(int32(data.frameSize))
				flow.Rate().SetPercentage(float32(data.trafficRate))

			}
			ate.OTG().PushConfig(t, top)
			ate.OTG().StartProtocols(t)

			counters := make(map[string]map[string]uint64)
			var counterNames []string
			if !deviations.QOSDroppedOctets(dut) {
				counterNames = []string{

					"ateOutPkts", "ateInPkts", "dutQosPktsBeforeTraffic", "dutQosOctetsBeforeTraffic",
					"dutQosPktsAfterTraffic", "dutQosOctetsAfterTraffic", "dutQosDroppedPktsBeforeTraffic",
					"dutQosDroppedOctetsBeforeTraffic", "dutQosDroppedPktsAfterTraffic",
					"dutQosDroppedOctetsAfterTraffic",
				}
			} else {
				counterNames = []string{

					"ateOutPkts", "ateInPkts", "dutQosPktsBeforeTraffic", "dutQosOctetsBeforeTraffic",
					"dutQosPktsAfterTraffic", "dutQosOctetsAfterTraffic", "dutQosDroppedPktsBeforeTraffic",
					"dutQosDroppedPktsAfterTraffic",
				}

			}
			for _, name := range counterNames {
				counters[name] = make(map[string]uint64)

				// Set the initial counters to 0.
				for _, data := range trafficFlows {
					counters[name][data.queue] = 0
				}
			}

			// Get QoS egress packet counters before the traffic.
			for _, data := range trafficFlows {
				counters["dutQosPktsBeforeTraffic"][data.queue] = gnmi.Get(t, dut, gnmi.OC().Qos().Interface(dp3.Name()).Output().Queue(data.queue).TransmitPkts().State())
				counters["dutQosOctetsBeforeTraffic"][data.queue] = gnmi.Get(t, dut, gnmi.OC().Qos().Interface(dp3.Name()).Output().Queue(data.queue).TransmitOctets().State())
				counters["dutQosDroppedPktsBeforeTraffic"][data.queue] = gnmi.Get(t, dut, gnmi.OC().Qos().Interface(dp3.Name()).Output().Queue(data.queue).DroppedPkts().State())
				if !deviations.QOSDroppedOctets(dut) {
					counters["dutQosDroppedOctetsBeforeTraffic"][data.queue] = gnmi.Get(t, dut, gnmi.OC().Qos().Interface(dp3.Name()).Output().Queue(data.queue).DroppedOctets().State())
				}
			}

			t.Logf("Running traffic 1 on DUT interfaces: %s => %s ", dp1.Name(), dp3.Name())
			t.Logf("Running traffic 2 on DUT interfaces: %s => %s ", dp2.Name(), dp3.Name())
			t.Logf("Sending traffic flows: \n%v\n\n", trafficFlows)
			ate.OTG().StartTraffic(t)
			time.Sleep(30 * time.Second)
			ate.OTG().StopTraffic(t)
			time.Sleep(30 * time.Second)

			for trafficID, data := range trafficFlows {
				ateTxPkts := gnmi.Get(t, ate.OTG(), gnmi.OTG().Flow(trafficID).Counters().OutPkts().State())
				ateRxPkts := gnmi.Get(t, ate.OTG(), gnmi.OTG().Flow(trafficID).Counters().InPkts().State())
				counters["ateOutPkts"][data.queue] += ateTxPkts
				counters["ateInPkts"][data.queue] += ateRxPkts

				counters["dutQosPktsAfterTraffic"][data.queue] = gnmi.Get(t, dut, gnmi.OC().Qos().Interface(dp3.Name()).Output().Queue(data.queue).TransmitPkts().State())
				counters["dutQosOctetsAfterTraffic"][data.queue] = gnmi.Get(t, dut, gnmi.OC().Qos().Interface(dp3.Name()).Output().Queue(data.queue).TransmitOctets().State())
				counters["dutQosDroppedPktsAfterTraffic"][data.queue] = gnmi.Get(t, dut, gnmi.OC().Qos().Interface(dp3.Name()).Output().Queue(data.queue).DroppedPkts().State())
				if !deviations.QOSDroppedOctets(dut) {
					counters["dutQosDroppedOctetsAfterTraffic"][data.queue] = gnmi.Get(t, dut, gnmi.OC().Qos().Interface(dp3.Name()).Output().Queue(data.queue).DroppedOctets().State())
				}
				t.Logf("ateInPkts: %v, txPkts %v, Queue: %v", counters["ateInPkts"][data.queue], counters["dutQosPktsAfterTraffic"][data.queue], data.queue)

				if ateTxPkts == 0 {
					t.Fatalf("TxPkts == 0, want >0.")
				}
				lossPct := (float32)((float64(ateTxPkts-ateRxPkts) * 100.0) / float64(ateTxPkts))
				t.Logf("Get flow %q: lossPct: %.2f%% or rxPct: %.2f%%, want: %.2f%%\n\n", data.queue, lossPct, 100.0-lossPct, data.expectedThroughputPct)
				if got, want := 100.0-lossPct, data.expectedThroughputPct; got != want {
					t.Errorf("Get(throughput for queue %q): got %.2f%%, want %.2f%%", data.queue, got, want)
				}
			}

			// Check QoS egress packet counters are updated correctly.
			for _, name := range counterNames {
				t.Logf("QoS %s: %v", name, counters[name])
			}

			for _, data := range trafficFlows {
				dutPktCounterDiff := counters["dutQosPktsAfterTraffic"][data.queue] - counters["dutQosPktsBeforeTraffic"][data.queue]
				atePktCounterDiff := counters["ateInPkts"][data.queue]
				t.Logf("Queue %q: atePktCounterDiff: %v dutPktCounterDiff: %v", data.queue, atePktCounterDiff, dutPktCounterDiff)
				if dutPktCounterDiff < atePktCounterDiff {
					t.Errorf("Get dutPktCounterDiff for queue %q: got %v, want >= %v", data.queue, dutPktCounterDiff, atePktCounterDiff)
				}

				dutDropPktCounterDiff := counters["dutQosDroppedPktsAfterTraffic"][data.queue] - counters["dutQosDroppedPktsBeforeTraffic"][data.queue]
				t.Logf("Queue %q: dutDropPktCounterDiff: %v", data.queue, dutDropPktCounterDiff)
				if dutDropPktCounterDiff != 0 {
					t.Errorf("Get dutDropPktCounterDiff for queue %q: got %v, want 0", data.queue, dutDropPktCounterDiff)
				}

				dutOctetCounterDiff := counters["dutQosOctetsAfterTraffic"][data.queue] - counters["dutQosOctetsBeforeTraffic"][data.queue]
				ateOctetCounterDiff := counters["ateInPkts"][data.queue] * uint64(data.frameSize)
				t.Logf("Queue %q: ateOctetCounterDiff: %v dutOctetCounterDiff: %v", data.queue, ateOctetCounterDiff, dutOctetCounterDiff)
				if dutOctetCounterDiff < ateOctetCounterDiff {
					t.Errorf("Get dutOctetCounterDiff for queue %q: got %v, want >= %v", data.queue, dutOctetCounterDiff, ateOctetCounterDiff)
				}
				if !deviations.QOSDroppedOctets(dut) {
					dutDropOctetCounterDiff := counters["dutQosDroppedOctetsAfterTraffic"][data.queue] - counters["dutQosDroppedOctetsBeforeTraffic"][data.queue]
					t.Logf("Queue %q: dutDropOctetCounterDiff: %v", data.queue, dutDropOctetCounterDiff)
					if dutDropOctetCounterDiff != 0 {
						t.Errorf("Get dutDropOctetCounterDiff for queue %q: got %v, want 0", data.queue, dutDropOctetCounterDiff)
					}
				}
			}
		})
	}
}

func ConfigureDUTIntf(t *testing.T, dut *ondatra.DUTDevice) {
	t.Helper()
	dp1 := dut.Port(t, "port1")
	dp2 := dut.Port(t, "port2")
	dp3 := dut.Port(t, "port3")

	dutIntfs := []struct {
		desc      string
		intfName  string
		ipAddr    string
		prefixLen uint8
	}{{
		desc:      "Input interface port1",
		intfName:  dp1.Name(),
		ipAddr:    dutPort1.IPv4,
		prefixLen: 31,
	}, {
		desc:      "Input interface port2",
		intfName:  dp2.Name(),
		ipAddr:    dutPort2.IPv4,
		prefixLen: 31,
	}, {
		desc:      "Output interface port3",
		intfName:  dp3.Name(),
		ipAddr:    dutPort3.IPv4,
		prefixLen: 31,
	}}

	// Configure the interfaces.
	for _, intf := range dutIntfs {
		t.Logf("Configure DUT interface %s with attributes %v", intf.intfName, intf)
		i := &oc.Interface{
			Name:        ygot.String(intf.intfName),
			Description: ygot.String(intf.desc),
			Type:        oc.IETFInterfaces_InterfaceType_ethernetCsmacd,
			Enabled:     ygot.Bool(true),
		}
		i.GetOrCreateEthernet()
		s := i.GetOrCreateSubinterface(0).GetOrCreateIpv4()
		if *deviations.InterfaceEnabled && !*deviations.IPv4MissingEnabled {
			s.Enabled = ygot.Bool(true)
		}
		a := s.GetOrCreateAddress(intf.ipAddr)
		a.PrefixLength = ygot.Uint8(intf.prefixLen)
		gnmi.Replace(t, dut, gnmi.OC().Interface(intf.intfName).Config(), i)
	}
}

func ConfigureQoS(t *testing.T, dut *ondatra.DUTDevice) {
	t.Helper()
	dp1 := dut.Port(t, "port1")
	dp2 := dut.Port(t, "port2")
	dp3 := dut.Port(t, "port3")
	d := &oc.Root{}
	q := d.GetOrCreateQos()

	t.Logf("Create qos forwarding groups and queue name config")
	forwardingGroups := []struct {
		desc        string
		queueName   string
		targetGroup string
	}{{
		desc:        "forwarding-group-BE1",
		queueName:   "BE1",
		targetGroup: "target-group-BE1",
	}, {
		desc:        "forwarding-group-BE0",
		queueName:   "BE0",
		targetGroup: "target-group-BE0",
	}, {
		desc:        "forwarding-group-AF1",
		queueName:   "AF1",
		targetGroup: "target-group-AF1",
	}, {
		desc:        "forwarding-group-AF2",
		queueName:   "AF2",
		targetGroup: "target-group-AF2",
	}, {
		desc:        "forwarding-group-AF3",
		queueName:   "AF3",
		targetGroup: "target-group-AF3",
	}, {
		desc:        "forwarding-group-AF4",
		queueName:   "AF4",
		targetGroup: "target-group-AF4",
	}, {
		desc:        "forwarding-group-NC1",
		queueName:   "NC1",
		targetGroup: "target-group-NC1",
	}}

	t.Logf("qos forwarding groups config: %v", forwardingGroups)
	for _, tc := range forwardingGroups {
		fwdGroup := q.GetOrCreateForwardingGroup(tc.targetGroup)
		fwdGroup.SetName(tc.targetGroup)
		fwdGroup.SetOutputQueue(tc.queueName)
		queue := q.GetOrCreateQueue(tc.queueName)
		queue.SetName(tc.queueName)
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	t.Logf("Create qos queue management profile config")
	ecnConfig := struct {
		profileName  string
		ecnEnabled   bool
		minThreshold uint64
	}{
		profileName:  "ECNProfile",
		ecnEnabled:   true,
		minThreshold: uint64(80000),
	}
	t.Logf("qos queue management profile config: %v", ecnConfig)
	queueMgmtProfile := q.GetOrCreateQueueManagementProfile(ecnConfig.profileName)
	queueMgmtProfile.SetName("ECNProfile")
	wred := queueMgmtProfile.GetOrCreateWred()
	uniform := wred.GetOrCreateUniform()
	uniform.SetEnableEcn(ecnConfig.ecnEnabled)
	uniform.SetMinThreshold(ecnConfig.minThreshold)
	gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)

	t.Logf("Create qos Classifiers config")
	classifiers := []struct {
		desc        string
		name        string
		classType   oc.E_Qos_Classifier_Type
		termID      string
		targetGroup string
		dscpSet     []uint8
	}{{
		desc:        "classifier_ipv4_be1",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "0",
		targetGroup: "target-group-BE1",
		dscpSet:     []uint8{0, 1, 2, 3},
	}, {
		desc:        "classifier_ipv4_be0",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "1",
		targetGroup: "target-group-BE0",
		dscpSet:     []uint8{4, 5, 6, 7},
	}, {
		desc:        "classifier_ipv4_af1",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "2",
		targetGroup: "target-group-AF1",
		dscpSet:     []uint8{8, 9, 10, 11},
	}, {
		desc:        "classifier_ipv4_af2",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "3",
		targetGroup: "target-group-AF2",
		dscpSet:     []uint8{16, 17, 18, 19},
	}, {
		desc:        "classifier_ipv4_af3",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "4",
		targetGroup: "target-group-AF3",
		dscpSet:     []uint8{24, 25, 26, 27},
	}, {
		desc:        "classifier_ipv4_af4",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "5",
		targetGroup: "target-group-AF4",
		dscpSet:     []uint8{32, 33, 34, 35},
	}, {
		desc:        "classifier_ipv4_nc1",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "6",
		targetGroup: "target-group-NC1",
		dscpSet:     []uint8{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59},
	}, {
		desc:        "classifier_ipv6_be1",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "0",
		targetGroup: "target-group-BE1",
		dscpSet:     []uint8{0, 1, 2, 3},
	}, {
		desc:        "classifier_ipv6_be0",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "1",
		targetGroup: "target-group-BE0",
		dscpSet:     []uint8{4, 5, 6, 7},
	}, {
		desc:        "classifier_ipv6_af1",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "2",
		targetGroup: "target-group-AF1",
		dscpSet:     []uint8{8, 9, 10, 11},
	}, {
		desc:        "classifier_ipv6_af2",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "3",
		targetGroup: "target-group-AF2",
		dscpSet:     []uint8{16, 17, 18, 19},
	}, {
		desc:        "classifier_ipv6_af3",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "4",
		targetGroup: "target-group-AF3",
		dscpSet:     []uint8{24, 25, 26, 27},
	}, {
		desc:        "classifier_ipv6_af4",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "5",
		targetGroup: "target-group-AF4",
		dscpSet:     []uint8{32, 33, 34, 35},
	}, {
		desc:        "classifier_ipv6_nc1",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "6",
		targetGroup: "target-group-NC1",
		dscpSet:     []uint8{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59},
	}}

	t.Logf("qos Classifiers config: %v", classifiers)
	for _, tc := range classifiers {
		classifier := q.GetOrCreateClassifier(tc.name)
		classifier.SetName(tc.name)
		classifier.SetType(tc.classType)
		term, err := classifier.NewTerm(tc.termID)
		if err != nil {
			t.Fatalf("Failed to create classifier.NewTerm(): %v", err)
		}

		term.SetId(tc.termID)
		action := term.GetOrCreateActions()
		action.SetTargetGroup(tc.targetGroup)
		condition := term.GetOrCreateConditions()
		if tc.name == "dscp_based_classifier_ipv4" {
			condition.GetOrCreateIpv4().SetDscpSet(tc.dscpSet)
		} else if tc.name == "dscp_based_classifier_ipv6" {
			condition.GetOrCreateIpv6().SetDscpSet(tc.dscpSet)
		}
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	t.Logf("Create qos input classifier config")
	classifierIntfs := []struct {
		desc                string
		intf                string
		inputClassifierType oc.E_Input_Classifier_Type
		classifier          string
	}{{
		desc:                "Input Classifier Type IPV4",
		intf:                dp1.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV4,
		classifier:          "dscp_based_classifier_ipv4",
	}, {
		desc:                "Input Classifier Type IPV6",
		intf:                dp1.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV6,
		classifier:          "dscp_based_classifier_ipv6",
	}, {
		desc:                "Input Classifier Type IPV4",
		intf:                dp2.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV4,
		classifier:          "dscp_based_classifier_ipv4",
	}, {
		desc:                "Input Classifier Type IPV6",
		intf:                dp2.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV6,
		classifier:          "dscp_based_classifier_ipv6",
	}}

	t.Logf("qos input classifier config: %v", classifierIntfs)
	for _, tc := range classifierIntfs {
		i := q.GetOrCreateInterface(tc.intf)
		i.SetInterfaceId(tc.intf)
		c := i.GetOrCreateInput().GetOrCreateClassifier(tc.inputClassifierType)
		c.SetType(tc.inputClassifierType)
		c.SetName(tc.classifier)
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	t.Logf("Create qos scheduler policies config")
	schedulerPolicies := []struct {
		desc        string
		sequence    uint32
		setPriority bool
		priority    oc.E_Scheduler_Priority
		inputID     string
		inputType   oc.E_Input_InputType
		setWeight   bool
		weight      uint64
		queueName   string
		targetGroup string
	}{{
		desc:        "scheduler-policy-BE1",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "BE1",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(1),
		queueName:   "BE1",
		targetGroup: "target-group-BE1",
	}, {
		desc:        "scheduler-policy-BE0",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "BE0",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(1),
		queueName:   "BE0",
		targetGroup: "target-group-BE0",
	}, {
		desc:        "scheduler-policy-AF1",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "AF1",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(4),
		queueName:   "AF1",
		targetGroup: "target-group-AF1",
	}, {
		desc:        "scheduler-policy-AF2",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "AF2",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(8),
		queueName:   "AF2",
		targetGroup: "target-group-AF2",
	}, {
		desc:        "scheduler-policy-AF3",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "AF3",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(12),
		queueName:   "AF3",
		targetGroup: "target-group-AF3",
	}, {
		desc:        "scheduler-policy-AF4",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "AF4",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(48),
		queueName:   "AF4",
		targetGroup: "target-group-AF4",
	}, {
		desc:        "scheduler-policy-NC1",
		sequence:    uint32(0),
		setPriority: true,
		setWeight:   false,
		priority:    oc.Scheduler_Priority_STRICT,
		inputID:     "NC1",
		inputType:   oc.Input_InputType_QUEUE,
		queueName:   "NC1",
		targetGroup: "target-group-NC1",
	}}

	schedulerPolicy := q.GetOrCreateSchedulerPolicy("scheduler")
	schedulerPolicy.SetName("scheduler")
	t.Logf("qos scheduler policies config: %v", schedulerPolicies)
	for _, tc := range schedulerPolicies {
		s := schedulerPolicy.GetOrCreateScheduler(tc.sequence)
		s.SetSequence(tc.sequence)
		if tc.setPriority {
			s.SetPriority(tc.priority)
		}
		input := s.GetOrCreateInput(tc.inputID)
		input.SetId(tc.inputID)
		input.SetInputType(tc.inputType)
		input.SetQueue(tc.queueName)
		if tc.setWeight {
			input.SetWeight(tc.weight)
		}
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	t.Logf("Create qos output interface config")
	schedulerIntfs := []struct {
		desc       string
		queueName  string
		scheduler  string
		ecnProfile string
	}{{
		desc:       "output-interface-BE1",
		queueName:  "BE1",
		scheduler:  "scheduler",
		ecnProfile: "ECNProfile",
	}, {
		desc:       "output-interface-BE0",
		queueName:  "BE0",
		scheduler:  "scheduler",
		ecnProfile: "ECNProfile",
	}, {
		desc:       "output-interface-AF1",
		queueName:  "AF1",
		scheduler:  "scheduler",
		ecnProfile: "ECNProfile",
	}, {
		desc:       "output-interface-AF2",
		queueName:  "AF2",
		scheduler:  "scheduler",
		ecnProfile: "ECNProfile",
	}, {
		desc:       "output-interface-AF3",
		queueName:  "AF3",
		scheduler:  "scheduler",
		ecnProfile: "ECNProfile",
	}, {
		desc:       "output-interface-AF4",
		queueName:  "AF4",
		scheduler:  "scheduler",
		ecnProfile: "ECNProfile",
	}, {
		desc:       "output-interface-NC1",
		queueName:  "NC1",
		scheduler:  "scheduler",
		ecnProfile: "ECNProfile",
	}}

	t.Logf("qos output interface config: %v", schedulerIntfs)
	for _, tc := range schedulerIntfs {
		i := q.GetOrCreateInterface(dp3.Name())
		i.SetInterfaceId(dp3.Name())
		output := i.GetOrCreateOutput()
		schedulerPolicy := output.GetOrCreateSchedulerPolicy()
		schedulerPolicy.SetName(tc.scheduler)
		queue := output.GetOrCreateQueue(tc.queueName)
		queue.SetName(tc.queueName)
		queue.SetQueueManagementProfile(tc.ecnProfile)
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}
}
func ConfigureCiscoQos(t *testing.T, dut *ondatra.DUTDevice) {
	t.Helper()
	dp1 := dut.Port(t, "port1")
	dp2 := dut.Port(t, "port2")
	dp3 := dut.Port(t, "port3")
	d := &oc.Root{}
	q := d.GetOrCreateQos()

	t.Logf("Create qos Classifiers config")
	classifiers := []struct {
		desc        string
		name        string
		classType   oc.E_Qos_Classifier_Type
		termID      string
		targetGroup string
		dscpSet     []uint8
	}{{
		desc:        "classifier_ipv4_nc1",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "a_NC1",
		targetGroup: "a_NC1",
		dscpSet:     []uint8{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59},
	}, {
		desc:        "classifier_ipv4_af4",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "b_AF4",
		targetGroup: "b_AF4",
		dscpSet:     []uint8{32, 33, 34, 35},
	}, {
		desc:        "classifier_ipv4_af3",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "c_AF3",
		targetGroup: "c_AF3",
		dscpSet:     []uint8{24, 25, 26, 27},
	}, {
		desc:        "classifier_ipv4_af2",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "d_AF2",
		targetGroup: "d_AF2",
		dscpSet:     []uint8{16, 17, 18, 19},
	}, {
		desc:        "classifier_ipv4_af1",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "e_AF1",
		targetGroup: "e_AF1",
		dscpSet:     []uint8{8, 9, 10, 11},
	}, {
		desc:        "classifier_ipv4_be0",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "f_BE0",
		targetGroup: "f_BE0",
		dscpSet:     []uint8{4, 5, 6, 7},
	}, {
		desc:        "classifier_ipv4_be1",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "g_BE1",
		targetGroup: "g_BE1",
		dscpSet:     []uint8{0, 1, 2, 3},
	}, {
		desc:        "classifier_ipv6_nc1",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "a_NC1_ipv6",
		targetGroup: "a_NC1",
		dscpSet:     []uint8{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59},
	}, {
		desc:        "classifier_ipv6_af4",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "b_AF4_ipv6",
		targetGroup: "b_AF4",
		dscpSet:     []uint8{32, 33, 34, 35},
	}, {
		desc:        "classifier_ipv6_af3",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "c_AF3_ipv6",
		targetGroup: "c_AF3",
		dscpSet:     []uint8{24, 25, 26, 27},
	}, {
		desc:        "classifier_ipv6_af2",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "d_AF2_ipv6",
		targetGroup: "d_AF2",
		dscpSet:     []uint8{16, 17, 18, 19},
	}, {
		desc:        "classifier_ipv6_af1",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "e_AF1_ipv6",
		targetGroup: "e_AF1",
		dscpSet:     []uint8{8, 9, 10, 11},
	}, {
		desc:        "classifier_ipv6_be0",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "f_BE0_ipv6",
		targetGroup: "f_BE0",
		dscpSet:     []uint8{4, 5, 6, 7},
	}, {
		desc:        "classifier_ipv6_be1",
		name:        "dscp_based_classifier",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "g_BE1_ipv6",
		targetGroup: "g_BE1",
		dscpSet:     []uint8{0, 1, 2, 3},
	}}

	t.Logf("qos Classifiers config: %v", classifiers)
	queueName := []string{"a_NC1", "b_AF4", "c_AF3", "d_AF2", "e_AF1", "f_BE0", "g_BE1"}

	for _, queue := range queueName {
		q1 := q.GetOrCreateQueue(queue)
		q1.Name = ygot.String(queue)

	}
	gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	for _, tc := range classifiers {
		classifier := q.GetOrCreateClassifier(tc.name)
		classifier.SetName(tc.name)
		classifier.SetType(tc.classType)
		term, err := classifier.NewTerm(tc.termID)
		if err != nil {
			t.Fatalf("Failed to create classifier.NewTerm(): %v", err)
		}

		term.SetId(tc.termID)
		action := term.GetOrCreateActions()
		action.SetTargetGroup(tc.targetGroup)
		condition := term.GetOrCreateConditions()
		if tc.classType == oc.Qos_Classifier_Type_IPV4 {
			condition.GetOrCreateIpv4().SetDscpSet(tc.dscpSet)
		} else if tc.classType == oc.Qos_Classifier_Type_IPV6 {
			condition.GetOrCreateIpv6().SetDscpSet(tc.dscpSet)
		}
		fwdgroups := q.GetOrCreateForwardingGroup(tc.targetGroup)
		fwdgroups.Name = ygot.String(tc.targetGroup)
		fwdgroups.OutputQueue = ygot.String(tc.targetGroup)
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	t.Logf("Create qos input classifier config")
	classifierIntfs := []struct {
		desc                string
		intf                string
		inputClassifierType oc.E_Input_Classifier_Type
		classifier          string
	}{{
		desc:                "Input Classifier Type IPV4",
		intf:                dp1.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV4,
		classifier:          "dscp_based_classifier",
	}, {
		desc:                "Input Classifier Type IPV4",
		intf:                dp2.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV4,
		classifier:          "dscp_based_classifier",
	}}

	t.Logf("qos input classifier config: %v", classifierIntfs)
	for _, tc := range classifierIntfs {

		i := q.GetOrCreateInterface(tc.intf)
		i.InterfaceId = ygot.String(tc.intf)
		c := i.GetOrCreateInput()
		c.GetOrCreateClassifier(oc.Input_Classifier_Type_IPV4).Name = ygot.String(tc.classifier)
		c.GetOrCreateClassifier(oc.Input_Classifier_Type_IPV6).Name = ygot.String(tc.classifier)
		c.GetOrCreateClassifier(oc.Input_Classifier_Type_MPLS).Name = ygot.String(tc.classifier)

		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	t.Logf("Create qos scheduler policies config")
	schedulerPolicies := []struct {
		desc        string
		sequence    uint32
		priority    oc.E_Scheduler_Priority
		inputID     string
		inputType   oc.E_Input_InputType
		weight      uint64
		queueName   string
		targetGroup string
	}{{
		desc:        "scheduler-policy-BE1",
		sequence:    uint32(6),
		priority:    oc.Scheduler_Priority_UNSET,
		inputID:     "g_BE1",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(1),
		queueName:   "g_BE1",
		targetGroup: "target-group-BE1",
	}, {
		desc:        "scheduler-policy-BE0",
		sequence:    uint32(5),
		priority:    oc.Scheduler_Priority_UNSET,
		inputID:     "f_BE0",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(2),
		queueName:   "f_BE0",
		targetGroup: "target-group-BE0",
	}, {
		desc:        "scheduler-policy-AF1",
		sequence:    uint32(4),
		priority:    oc.Scheduler_Priority_UNSET,
		inputID:     "e_AF1",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(4),
		queueName:   "e_AF1",
		targetGroup: "target-group-AF1",
	}, {
		desc:        "scheduler-policy-AF2",
		sequence:    uint32(3),
		priority:    oc.Scheduler_Priority_UNSET,
		inputID:     "d_AF2",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(8),
		queueName:   "d_AF2",
		targetGroup: "target-group-AF2",
	}, {
		desc:        "scheduler-policy-AF3",
		sequence:    uint32(2),
		priority:    oc.Scheduler_Priority_UNSET,
		inputID:     "c_AF3",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(12),
		queueName:   "c_AF3",
		targetGroup: "target-group-AF3",
	}, {
		desc:        "scheduler-policy-AF4",
		sequence:    uint32(1),
		priority:    oc.Scheduler_Priority_UNSET,
		inputID:     "b_AF4",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(48),
		queueName:   "b_AF4",
		targetGroup: "target-group-AF4",
	}, {
		desc:        "scheduler-policy-NC1",
		sequence:    uint32(0),
		priority:    oc.Scheduler_Priority_STRICT,
		inputID:     "a_NC1",
		inputType:   oc.Input_InputType_QUEUE,
		weight:      uint64(7),
		queueName:   "a_NC1",
		targetGroup: "target-group-NC1",
	}}
	schedulerPolicy := q.GetOrCreateSchedulerPolicy("scheduler")
	schedulerPolicy.SetName("scheduler")
	t.Logf("qos scheduler policies config: %v", schedulerPolicies)
	for _, tc := range schedulerPolicies {
		s := schedulerPolicy.GetOrCreateScheduler(tc.sequence)
		s.SetSequence(tc.sequence)
		s.SetPriority(tc.priority)
		input := s.GetOrCreateInput(tc.inputID)
		input.SetId(tc.inputID)
		//input.SetInputType(tc.inputType)
		input.SetQueue(tc.queueName)
		input.SetWeight(tc.weight)
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)

	}

	t.Logf("Create qos output interface config")
	schedulerIntfs := []struct {
		desc      string
		queueName string
		scheduler string
	}{{
		desc:      "output-interface-BE1",
		queueName: "g_BE1",
		scheduler: "scheduler",
	}, {
		desc:      "output-interface-BE0",
		queueName: "f_BE0",
		scheduler: "scheduler",
	}, {
		desc:      "output-interface-AF1",
		queueName: "e_AF1",
		scheduler: "scheduler",
	}, {
		desc:      "output-interface-AF2",
		queueName: "d_AF2",
		scheduler: "scheduler",
	}, {
		desc:      "output-interface-AF3",
		queueName: "c_AF3",
		scheduler: "scheduler",
	}, {
		desc:      "output-interface-AF4",
		queueName: "b_AF4",
		scheduler: "scheduler",
	}, {
		desc:      "output-interface-NC1",
		queueName: "a_NC1",
		scheduler: "scheduler",
	}}

	t.Logf("qos output interface config: %v", schedulerIntfs)
	for _, tc := range schedulerIntfs {
		i := q.GetOrCreateInterface(dp3.Name())
		i.SetInterfaceId(dp3.Name())
		output := i.GetOrCreateOutput()
		schedulerPolicy := output.GetOrCreateSchedulerPolicy()
		schedulerPolicy.SetName(tc.scheduler)
		queue := output.GetOrCreateQueue(tc.queueName)
		queue.SetName(tc.queueName)
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}
}
func ConfigureJuniperQos(t *testing.T, dut *ondatra.DUTDevice) {
	t.Helper()
	dp1 := dut.Port(t, "port1")
	dp2 := dut.Port(t, "port2")
	dp3 := dut.Port(t, "port3")
	d := &oc.Root{}
	q := d.GetOrCreateQos()

	forwardingGroups := []struct {
		desc        string
		queueName   string
		targetGroup string
	}{{
		desc:        "forwarding-group-BE1",
		queueName:   "6",
		targetGroup: "target-group-BE1",
	}, {
		desc:        "forwarding-group-BE0",
		queueName:   "0",
		targetGroup: "target-group-BE0",
	}, {
		desc:        "forwarding-group-AF1",
		queueName:   "4",
		targetGroup: "target-group-AF1",
	}, {
		desc:        "forwarding-group-AF2",
		queueName:   "1",
		targetGroup: "target-group-AF2",
	}, {
		desc:        "forwarding-group-AF3",
		queueName:   "5",
		targetGroup: "target-group-AF3",
	}, {
		desc:        "forwarding-group-AF4",
		queueName:   "2",
		targetGroup: "target-group-AF4",
	}, {
		desc:        "forwarding-group-NC1",
		queueName:   "3",
		targetGroup: "target-group-NC1",
	}}

	t.Logf("qos forwarding groups config: %v", forwardingGroups)
	for _, tc := range forwardingGroups {
		fwdGroup := q.GetOrCreateForwardingGroup(tc.targetGroup)
		fwdGroup.SetName(tc.targetGroup)
		fwdGroup.SetOutputQueue(tc.queueName)
		queue := q.GetOrCreateQueue(tc.queueName)
		queue.SetName(tc.queueName)
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	if deviations.ECNProfileRequiredDefinition(dut) {
		t.Logf("Create qos queue management profile config")

		profileName := string("ECNProfile")
		ecnEnabled := bool(true)
		minThreshold := uint64(0)
		maxThreshold := uint64(55)
		maxDropProbabilityPercent := uint8(25)

		queueMgmtProfile := q.GetOrCreateQueueManagementProfile(profileName)
		queueMgmtProfile.SetName(profileName)
		wred := queueMgmtProfile.GetOrCreateWred()
		uniform := wred.GetOrCreateUniform()
		uniform.SetEnableEcn(ecnEnabled)
		uniform.SetMinThreshold(minThreshold)
		uniform.SetMaxThreshold(maxThreshold)
		uniform.SetMaxDropProbabilityPercent(maxDropProbabilityPercent)
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}
	t.Logf("Create qos Classifiers config")
	classifiers := []struct {
		desc        string
		name        string
		classType   oc.E_Qos_Classifier_Type
		termID      string
		targetGroup string
		dscpSet     []uint8
	}{{
		desc:        "classifier_ipv4_be1",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "0",
		targetGroup: "target-group-BE1",
		dscpSet:     []uint8{0, 1, 2, 3},
	}, {
		desc:        "classifier_ipv4_be0",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "1",
		targetGroup: "target-group-BE0",
		dscpSet:     []uint8{4, 5, 6, 7},
	}, {
		desc:        "classifier_ipv4_af1",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "2",
		targetGroup: "target-group-AF1",
		dscpSet:     []uint8{8, 9, 10, 11},
	}, {
		desc:        "classifier_ipv4_af2",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "3",
		targetGroup: "target-group-AF2",
		dscpSet:     []uint8{16, 17, 18, 19},
	}, {
		desc:        "classifier_ipv4_af3",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "4",
		targetGroup: "target-group-AF3",
		dscpSet:     []uint8{24, 25, 26, 27},
	}, {
		desc:        "classifier_ipv4_af4",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "5",
		targetGroup: "target-group-AF4",
		dscpSet:     []uint8{32, 33, 34, 35},
	}, {
		desc:        "classifier_ipv4_nc1",
		name:        "dscp_based_classifier_ipv4",
		classType:   oc.Qos_Classifier_Type_IPV4,
		termID:      "6",
		targetGroup: "target-group-NC1",
		dscpSet:     []uint8{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59},
	}, {
		desc:        "classifier_ipv6_be1",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "0",
		targetGroup: "target-group-BE1",
		dscpSet:     []uint8{0, 1, 2, 3},
	}, {
		desc:        "classifier_ipv6_be0",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "1",
		targetGroup: "target-group-BE0",
		dscpSet:     []uint8{4, 5, 6, 7},
	}, {
		desc:        "classifier_ipv6_af1",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "2",
		targetGroup: "target-group-AF1",
		dscpSet:     []uint8{8, 9, 10, 11},
	}, {
		desc:        "classifier_ipv6_af2",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "3",
		targetGroup: "target-group-AF2",
		dscpSet:     []uint8{16, 17, 18, 19},
	}, {
		desc:        "classifier_ipv6_af3",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "4",
		targetGroup: "target-group-AF3",
		dscpSet:     []uint8{24, 25, 26, 27},
	}, {
		desc:        "classifier_ipv6_af4",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "5",
		targetGroup: "target-group-AF4",
		dscpSet:     []uint8{32, 33, 34, 35},
	}, {
		desc:        "classifier_ipv6_nc1",
		name:        "dscp_based_classifier_ipv6",
		classType:   oc.Qos_Classifier_Type_IPV6,
		termID:      "6",
		targetGroup: "target-group-NC1",
		dscpSet:     []uint8{48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59},
	}}

	t.Logf("qos Classifiers config: %v", classifiers)
	for _, tc := range classifiers {
		classifier := q.GetOrCreateClassifier(tc.name)
		classifier.SetName(tc.name)
		classifier.SetType(tc.classType)
		term, err := classifier.NewTerm(tc.termID)
		if err != nil {
			t.Fatalf("Failed to create classifier.NewTerm(): %v", err)
		}

		term.SetId(tc.termID)
		action := term.GetOrCreateActions()
		action.SetTargetGroup(tc.targetGroup)
		condition := term.GetOrCreateConditions()
		if tc.name == "dscp_based_classifier_ipv4" {
			condition.GetOrCreateIpv4().SetDscpSet(tc.dscpSet)
		} else if tc.name == "dscp_based_classifier_ipv6" {
			condition.GetOrCreateIpv6().SetDscpSet(tc.dscpSet)
		}
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	t.Logf("Create qos input classifier config")
	classifierIntfs := []struct {
		desc                string
		intf                string
		inputClassifierType oc.E_Input_Classifier_Type
		classifier          string
	}{{
		desc:                "Input Classifier Type IPV4 for interface 1",
		intf:                dp1.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV4,
		classifier:          "dscp_based_classifier_ipv4",
	}, {
		desc:                "Input Classifier Type IPV6 for interface 1",
		intf:                dp1.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV6,
		classifier:          "dscp_based_classifier_ipv6",
	}, {
		desc:                "Input Classifier Type IPV4 for interface 2",
		intf:                dp2.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV4,
		classifier:          "dscp_based_classifier_ipv4",
	}, {
		desc:                "Input Classifier Type IPV6 for interface 2",
		intf:                dp2.Name(),
		inputClassifierType: oc.Input_Classifier_Type_IPV6,
		classifier:          "dscp_based_classifier_ipv6",
	}}

	t.Logf("qos input classifier config: %v", classifierIntfs)
	for _, tc := range classifierIntfs {
		i := q.GetOrCreateInterface(tc.intf)
		i.SetInterfaceId(tc.intf)
		if deviations.ExplicitInterfaceRefDefinition(dut) {
			i.GetOrCreateInterfaceRef().Interface = ygot.String(tc.intf)
			i.GetOrCreateInterfaceRef().Subinterface = ygot.Uint32(0)
		}
		c := i.GetOrCreateInput().GetOrCreateClassifier(tc.inputClassifierType)
		c.SetType(tc.inputClassifierType)
		c.SetName(tc.classifier)
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	t.Logf("Create qos scheduler policies config")
	schedulerPolicies := []struct {
		desc        string
		sequence    uint32
		setPriority bool
		priority    oc.E_Scheduler_Priority
		inputID     string
		setWeight   bool
		weight      uint64
		queueName   string
		targetGroup string
	}{{
		desc:        "scheduler-policy-BE1",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "BE1",
		weight:      uint64(1),
		queueName:   "6",
		targetGroup: "target-group-BE1",
	}, {
		desc:        "scheduler-policy-BE0",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "BE0",
		weight:      uint64(1),
		queueName:   "0",
		targetGroup: "target-group-BE0",
	}, {
		desc:        "scheduler-policy-AF1",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "AF1",
		weight:      uint64(4),
		queueName:   "4",
		targetGroup: "target-group-AF1",
	}, {
		desc:        "scheduler-policy-AF2",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "AF2",
		weight:      uint64(8),
		queueName:   "1",
		targetGroup: "target-group-AF2",
	}, {
		desc:        "scheduler-policy-AF3",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "AF3",
		weight:      uint64(12),
		queueName:   "5",
		targetGroup: "target-group-AF3",
	}, {
		desc:        "scheduler-policy-AF4",
		sequence:    uint32(1),
		setPriority: false,
		setWeight:   true,
		inputID:     "AF4",
		weight:      uint64(48),
		queueName:   "2",
		targetGroup: "target-group-AF4",
	}, {
		desc:        "scheduler-policy-NC1",
		sequence:    uint32(0),
		setPriority: true,
		setWeight:   false,
		priority:    oc.Scheduler_Priority_STRICT,
		inputID:     "NC1",
		queueName:   "3",
		targetGroup: "target-group-NC1",
	}}

	schedulerPolicy := q.GetOrCreateSchedulerPolicy("scheduler")
	schedulerPolicy.SetName("scheduler")
	t.Logf("qos scheduler policies config: %v", schedulerPolicies)
	for _, tc := range schedulerPolicies {
		s := schedulerPolicy.GetOrCreateScheduler(tc.sequence)
		s.SetSequence(tc.sequence)
		if tc.setPriority {
			s.SetPriority(tc.priority)
		}
		input := s.GetOrCreateInput(tc.inputID)
		input.SetId(tc.inputID)
		input.SetInputType(oc.Input_InputType_QUEUE)
		input.SetQueue(tc.queueName)
		if tc.setWeight {
			input.SetWeight(tc.weight)
		}
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}

	t.Logf("Create qos output interface config")
	schedulerIntfs := []struct {
		desc      string
		queueName string
	}{{
		desc:      "output-interface-BE1",
		queueName: "6",
	}, {
		desc:      "output-interface-BE0",
		queueName: "0",
	}, {
		desc:      "output-interface-AF1",
		queueName: "4",
	}, {
		desc:      "output-interface-AF2",
		queueName: "1",
	}, {
		desc:      "output-interface-AF3",
		queueName: "5",
	}, {
		desc:      "output-interface-AF4",
		queueName: "2",
	}, {
		desc:      "output-interface-NC1",
		queueName: "3",
	}}

	t.Logf("qos output interface config: %v", schedulerIntfs)
	for _, tc := range schedulerIntfs {
		i := q.GetOrCreateInterface(dp3.Name())
		i.SetInterfaceId(dp3.Name())
		if deviations.ExplicitInterfaceRefDefinition(dut) {
			i.GetOrCreateInterfaceRef().Interface = ygot.String(dp3.Name())
		}
		output := i.GetOrCreateOutput()
		schedulerPolicy := output.GetOrCreateSchedulerPolicy()
		schedulerPolicy.SetName("scheduler")
		queue := output.GetOrCreateQueue(tc.queueName)
		queue.SetName(tc.queueName)
		queue.SetQueueManagementProfile("ECNProfile")
		gnmi.Replace(t, dut, gnmi.OC().Qos().Config(), q)
	}
}