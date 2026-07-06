package lldp

import (
	"reflect"
	"testing"
)

func TestBuildPortOverview(t *testing.T) {
	ports := []EthernetPort{
		{Name: "lan2", Speed: "1000F", Bridge: "br-lan"},
		{Name: "lan1", Speed: "1000F", Bridge: "br-lan"},
		{Name: "wan", Speed: "", Bridge: ""},
	}
	rows := []Row{
		{Port: "lan1", Name: "sw-core", Mac: "AA:BB:CC:DD:EE:01"}, // known via devices, upper-case
		{Port: "lan1", Name: "printer", Mac: "11:22:33:44:55:66"}, // unknown, same port
		{Port: "eth9", Name: "sw-edge", Mac: "aa:bb:cc:dd:ee:02"}, // port not in ethernet list
	}
	macOwners := map[string]string{
		"aa:bb:cc:dd:ee:01": "dev_core",
		"AA:BB:CC:DD:EE:02": "dev_edge",
	}

	got := BuildPortOverview(ports, rows, macOwners)
	want := []Port{
		{Port: "eth9", Speed: "", Bridge: "", Neighbors: []OverviewNeighbor{
			{Name: "sw-edge", Mac: "aa:bb:cc:dd:ee:02", KnownDeviceId: "dev_edge"},
		}},
		{Port: "lan1", Speed: "1000F", Bridge: "br-lan", Neighbors: []OverviewNeighbor{
			{Name: "printer", Mac: "11:22:33:44:55:66", KnownDeviceId: ""},
			{Name: "sw-core", Mac: "AA:BB:CC:DD:EE:01", KnownDeviceId: "dev_core"},
		}},
		{Port: "lan2", Speed: "1000F", Bridge: "br-lan", Neighbors: nil},
		{Port: "wan", Speed: "", Bridge: "", Neighbors: nil},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("BuildPortOverview mismatch:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestBuildPortOverviewEmpty(t *testing.T) {
	got := BuildPortOverview(nil, nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %+v", got)
	}
}

func TestBuildPortOverviewNaturalSort(t *testing.T) {
	ports := []EthernetPort{
		{Name: "lan10"},
		{Name: "lan2"},
		{Name: "lan1"},
		{Name: "eth0"},
		{Name: "wan"},
	}
	got := BuildPortOverview(ports, nil, nil)
	var order []string
	for _, p := range got {
		order = append(order, p.Port)
	}
	want := []string{"eth0", "lan1", "lan2", "lan10", "wan"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("port order mismatch:\n got: %v\nwant: %v", order, want)
	}
}
