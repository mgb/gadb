package gadb

import (
	"testing"
)

func TestClient_Version(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	v, err := c.Version()
	if err != nil {
		t.Fatal(err)
	}

	t.Log(v)
}

func TestClient_SerialList(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	serials, err := c.SerialList()
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range serials {
		t.Log(s)
	}
}

func TestClient_List(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range devices {
		t.Log(d.serial, d.DeviceInfo())
	}
}

func TestClient_ForwardList(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	deviceForwardList, err := c.ForwardList()
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range deviceForwardList {
		t.Log(d)
	}
}

func TestClient_ForwardKillAll(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	err = c.ForwardKillAll()
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_ConnectHost(t *testing.T) {
	t.Skip("Requires manual setup of a host to connect to")

	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	err = c.ConnectHost("192.168.1.28")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_DisconnectHost(t *testing.T) {
	t.Skip("Requires manual setup of a host to connect to")

	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	err = c.DisconnectHost("192.168.1.28")
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_DisconnectAll(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	err = c.DisconnectAll()
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_KillServer(t *testing.T) {
	t.Skip("Killing server makes all other unit test fail, so skip it")

	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	err = c.KillServer()
	if err != nil {
		t.Fatal(err)
	}
}
