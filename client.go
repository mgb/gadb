package gadb

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	// AdbServerPort is the default port for the adb server
	AdbServerPort = 5037

	// AdbDaemonPort is the default port for the adb daemon
	AdbDaemonPort = 5555
)

// Client contains the information needed to communicate with the adb server
type Client struct {
	host string
	port int
}

// ErrWarnings represents a list of warnings
type ErrWarnings []string

func (e ErrWarnings) Error() string {
	return "warnings: " + strings.Join(e, ", ")
}

// NewClient creates a new adb client
func NewClient() (Client, error) {
	return NewClientWithHost("localhost")
}

// NewClientWithHost creates a new adb client with the specified host
func NewClientWithHost(host string) (Client, error) {
	return NewClientWithHostAndPort(host, AdbServerPort)
}

// NewClientWithHostAndPort creates a new adb client with the specified host and port
func NewClientWithHostAndPort(host string, port int) (Client, error) {
	c := Client{
		host: host,
		port: port,
	}

	// Validate that we can communicate with the client
	tp, err := c.createTransport()
	if err != nil {
		return Client{}, err
	}
	tp.Close()

	return c, nil
}

// ServerVersion returns the version of the adb server
func (c Client) Version() (int, error) {
	resp, err := c.executeCommand("host:version")
	if err != nil {
		return 0, err
	}

	v, err := strconv.ParseInt(resp, 16, 64)
	if err != nil {
		return 0, err
	}

	return int(v), nil
}

// SerialList returns a list of serial numbers of all connected devices
func (c Client) SerialList() ([]string, error) {
	resp, err := c.executeCommand("host:devices")
	if err != nil {
		return nil, err
	}

	var serials []string
	for _, l := range strings.Split(resp, "\n") {
		f := strings.Fields(l)
		if len(f) < 2 {
			continue
		}
		serials = append(serials, f[0])
	}
	return serials, nil
}

// List returns a list of all connected devices
func (c Client) List() ([]Device, error) {
	resp, err := c.executeCommand("host:devices-l")
	if err != nil {
		return nil, err
	}

	var devices []Device
	var warnings []string
	for _, l := range strings.Split(resp, "\n") {
		line := strings.TrimSpace(l)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 || len(fields[0]) == 0 {
			warnings = append(warnings, fmt.Sprintf("invalid line: %q", line))
			continue
		}

		sliceAttrs := fields[2:]
		mapAttrs := map[string]string{}
		for _, field := range sliceAttrs {
			split := strings.Split(field, ":")
			key, val := split[0], split[1]
			mapAttrs[key] = val
		}
		devices = append(devices, Device{adbClient: c, serial: fields[0], attrs: mapAttrs})
	}

	if len(warnings) > 0 {
		return devices, ErrWarnings(warnings)
	}
	return devices, nil
}

// ForwardList returns a list of all forward connections
func (c Client) ForwardList() ([]DeviceForward, error) {
	resp, err := c.executeCommand("host:list-forward")
	if err != nil {
		return nil, err
	}

	var devices []DeviceForward
	for _, l := range strings.Split(resp, "\n") {
		line := strings.TrimSpace(l)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		devices = append(devices, DeviceForward{Serial: fields[0], Local: fields[1], Remote: fields[2]})
	}

	return devices, nil
}

// ForwadKillAll kills all forward connections
func (c Client) ForwardKillAll() error {
	return c.executeCommandWithoutResponse("host:killforward-all")
}

// ConnectHost connects to a device via TCP/IP
func (c Client) ConnectHost(ip string) error {
	return c.ConnectHostAndPort(ip, AdbDaemonPort)
}

// ConnectHostAndPort connects to a device via TCP/IP and port
func (c Client) ConnectHostAndPort(ip string, port int) error {
	resp, err := c.executeCommand("host:connect:" + net.JoinHostPort(ip, fmt.Sprint(port)))
	if err != nil {
		return err
	}

	if !strings.HasPrefix(resp, "connected to") && !strings.HasPrefix(resp, "already connected to") {
		return fmt.Errorf("adb connect: %s", resp)
	}
	return nil
}

// DisconnectHost disconnects from a device via TCP/IP
func (c Client) DisconnectHost(ip string) error {
	return c.disconnect(ip)
}

// DisconnectHostAndPort disconnects from a device via TCP/IP and port
func (c Client) DisconnectHostAndPort(ip string, port int) error {
	return c.disconnect(net.JoinHostPort(ip, fmt.Sprint(port)))
}

func (c Client) disconnect(hostAndPort string) error {
	resp, err := c.executeCommand("host:disconnect:" + hostAndPort)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(resp, "disconnected") {
		return fmt.Errorf("adb disconnect: %s", resp)
	}
	return nil
}

// DisconnectAll disconnects from all devices
func (c Client) DisconnectAll() error {
	resp, err := c.executeCommand("host:disconnect:")
	if err != nil {
		return err
	}

	if !strings.HasPrefix(resp, "disconnected everything") {
		return fmt.Errorf("adb disconnect all: %s", resp)
	}
	return nil
}

// KillServer kills the adb server
func (c Client) KillServer() error {
	tp, err := c.createTransport()
	if err != nil {
		return err
	}
	defer tp.Close()

	err = tp.Send("host:kill")
	if err != nil {
		return err
	}
	return nil
}

func (c Client) createTransport() (tp transport, err error) {
	return newTransport(net.JoinHostPort(c.host, fmt.Sprint(c.port)))
}

func (c Client) executeCommand(command string) (string, error) {
	tp, err := c.createTransport()
	if err != nil {
		return "", err
	}
	defer tp.Close()

	err = tp.Send(command)
	if err != nil {
		return "", err
	}

	err = tp.VerifyResponse()
	if err != nil {
		return "", err
	}

	resp, err := tp.UnpackString()
	if err != nil {
		return "", err
	}
	return resp, nil
}

func (c Client) executeCommandWithoutResponse(command string) error {
	tp, err := c.createTransport()
	if err != nil {
		return err
	}
	defer tp.Close()

	err = tp.Send(command)
	if err != nil {
		return err
	}

	err = tp.VerifyResponse()
	if err != nil {
		return err
	}

	return nil
}
