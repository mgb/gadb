package gadb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	defaultFileMode = os.FileMode(0o664)
)

// DeviceState is the state of the device
type DeviceState string

// List of DeviceStates
const (
	StateUnknown      DeviceState = "UNKNOWN"
	StateOnline       DeviceState = "online"
	StateOffline      DeviceState = "offline"
	StateDisconnected DeviceState = "disconnected"
)

var deviceStateStrings = map[string]DeviceState{
	"":        StateDisconnected,
	"offline": StateOffline,
	"device":  StateOnline,
}

func deviceStateConv(k string) DeviceState {
	deviceState, ok := deviceStateStrings[k]
	if !ok {
		return StateUnknown
	}
	return deviceState
}

// DeviceForward is the forward information of a device
type DeviceForward struct {
	Serial string
	Local  string
	Remote string
	// LocalProtocol string
	// RemoteProtocol string
}

// Device is the representation of a device
type Device struct {
	adbClient Client
	serial    string
	attrs     map[string]string
}

// Product returns the product name of the device
func (d Device) Product() (string, error) {
	if d.HasAttribute("product") {
		return d.attrs["product"], nil
	}
	return "", errors.New("does not have attribute: product")
}

// Model returns the model name of the device
func (d Device) Model() (string, error) {
	if d.HasAttribute("model") {
		return d.attrs["model"], nil
	}
	return "", errors.New("does not have attribute: model")
}

// Usb returns the usb information of the device
func (d Device) Usb() (string, error) {
	if d.HasAttribute("usb") {
		return d.attrs["usb"], nil
	}
	return "", errors.New("does not have attribute: usb")
}

func (d Device) HasAttribute(key string) bool {
	_, ok := d.attrs[key]
	return ok
}

func (d Device) transportId() (string, error) {
	if d.HasAttribute("transport_id") {
		return d.attrs["transport_id"], nil
	}
	return "", errors.New("does not have attribute: transport_id")
}

// DeviceInfo returns the information of the device
func (d Device) DeviceInfo() map[string]string {
	return d.attrs
}

// Serial returns the serial number of the device
func (d Device) Serial() string {
	return d.serial
}

// IsUsb returns true if the device is connected via USB
func (d Device) IsUsb() (bool, error) {
	usb, err := d.Usb()
	if err != nil {
		return false, err
	}

	return usb != "", nil
}

// State returns the state of the device
func (d Device) State() (DeviceState, error) {
	resp, err := d.adbClient.executeCommand(fmt.Sprintf("host-serial:%s:get-state", d.serial))
	if err != nil {
		return StateUnknown, err
	}
	return deviceStateConv(resp), nil
}

// DevicePath returns the path of the device
func (d Device) DevicePath() (string, error) {
	resp, err := d.adbClient.executeCommand(fmt.Sprintf("host-serial:%s:get-devpath", d.serial))
	if err != nil {
		return "", err
	}
	return resp, nil
}

// Forward forwards a local port to a remote port on the device
func (d Device) Forward(localPort, remotePort int, noRebind ...bool) error {
	command := ""
	local := fmt.Sprintf("tcp:%d", localPort)
	remote := fmt.Sprintf("tcp:%d", remotePort)

	if len(noRebind) != 0 && noRebind[0] {
		command = fmt.Sprintf("host-serial:%s:forward:norebind:%s;%s", d.serial, local, remote)
	} else {
		command = fmt.Sprintf("host-serial:%s:forward:%s;%s", d.serial, local, remote)
	}

	return d.adbClient.executeCommandWithoutResponse(command)
}

// ForwardList returns the list of forwards on the device
func (d Device) ForwardList() ([]DeviceForward, error) {
	forwardList, err := d.adbClient.ForwardList()
	if err != nil {
		return nil, err
	}

	var deviceForwardList []DeviceForward
	for i := range forwardList {
		if forwardList[i].Serial == d.serial {
			deviceForwardList = append(deviceForwardList, forwardList[i])
		}
	}
	return deviceForwardList, nil
}

// ForwardKill kills a forward on the device
func (d Device) ForwardKill(localPort int) error {
	return d.adbClient.executeCommandWithoutResponse(
		fmt.Sprintf("host-serial:%s:killforward:%s:%d",
			d.serial,
			"tcp",
			localPort,
		),
	)
}

// RunShellCommand runs a shell command on the device
func (d Device) RunShellCommand(cmd string, args ...string) (string, error) {
	b, err := d.RunShellCommandStreaming(cmd, args...)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// RunShellCommandStreaming runs a shell command on the device and returns the output
func (d Device) RunShellCommandStreaming(cmd string, args ...string) ([]byte, error) {
	cmd = fmt.Sprintf("%s %s", cmd, strings.Join(args, " "))
	if strings.TrimSpace(cmd) == "" {
		return nil, errors.New("adb shell: command cannot be empty")
	}

	raw, err := d.executeCommand(fmt.Sprintf("shell:%s", cmd))
	if err != nil {
		return raw, err
	}
	return raw, nil
}

// EnableAdbOverTCP enables adb over tcp
func (d Device) EnableAdbOverTCP(port ...int) error {
	if len(port) == 0 {
		port = []int{AdbDaemonPort}
	}

	r, err := d.executeCommandStreaming(fmt.Sprintf("tcpip:%d", port[0]))
	if err != nil {
		return err
	}
	defer r.Close()

	return nil
}

func (d Device) createDeviceTransport() (transport, error) {
	tp, err := newTransport(fmt.Sprintf("%s:%d", d.adbClient.host, d.adbClient.port))
	if err != nil {
		return transport{}, fmt.Errorf("failed to create transport: %w", err)
	}

	err = tp.Send(fmt.Sprintf("host:transport:%s", d.serial))
	if err != nil {
		return transport{}, fmt.Errorf("failed to send transport command: %w", err)
	}

	err = tp.VerifyResponse()
	if err != nil {
		return transport{}, fmt.Errorf("failed to verify transport response: %w", err)
	}
	return tp, nil
}

func (d Device) executeCommand(command string) ([]byte, error) {
	r, err := d.executeCommandStreaming(command)
	if err != nil {
		return nil, fmt.Errorf("failed to exec command: %w", err)
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read cmd response: %w", err)
	}
	return b, nil
}

func (d Device) executeCommandStreaming(command string, onlyVerifyResponse ...bool) (resp io.ReadCloser, err error) {
	if len(onlyVerifyResponse) == 0 {
		onlyVerifyResponse = []bool{false}
	}

	tp, err := d.createDeviceTransport()
	if err != nil {
		return nil, err
	}
	// Only close connection if we don't respond with a socket
	defer func() {
		if resp == nil {
			tp.Close()
		}
	}()

	err = tp.Send(command)
	if err != nil {
		return nil, err
	}

	err = tp.VerifyResponse()
	if err != nil {
		return nil, err
	}

	if onlyVerifyResponse[0] {
		return nil, nil
	}

	resp = tp.sock
	return
}

// List returns the list of files in the directory
func (d Device) List(remotePath string) ([]os.FileInfo, error) {
	tp, err := d.createDeviceTransport()
	if err != nil {
		return nil, fmt.Errorf("failed to create device transport: %w", err)
	}
	defer tp.Close()

	sync, err := tp.CreateSyncTransport()
	if err != nil {
		return nil, fmt.Errorf("failed to create sync transport: %w", err)
	}
	defer sync.Close()

	err = sync.Send("LIST", remotePath)
	if err != nil {
		return nil, fmt.Errorf("failed to send list command: %w", err)
	}

	var devFileInfos []os.FileInfo
	for {
		entry, ok, err := sync.ReadDirectoryEntry()
		if err != nil {
			return nil, fmt.Errorf("failed to read directory entry: %w", err)
		}
		if !ok {
			break
		}

		devFileInfos = append(devFileInfos, entry)
	}

	return devFileInfos, nil
}

// FileWithStat represents a reader that also can call Stat() on
type FileWithStat interface {
	Stat() (os.FileInfo, error)
	io.Reader
}

// PushFile pushes a file to the device
func (d Device) PushFile(local FileWithStat, remotePath string, modification ...time.Time) error {
	if len(modification) == 0 {
		stat, err := local.Stat()
		if err != nil {
			return err
		}
		modification = []time.Time{stat.ModTime()}
	}

	return d.Push(local, remotePath, modification[0], defaultFileMode)
}

// Push pushes a file to the device
func (d Device) Push(source io.Reader, remotePath string, modification time.Time, mode ...os.FileMode) error {
	if len(mode) == 0 {
		mode = []os.FileMode{defaultFileMode}
	}

	tp, err := d.createDeviceTransport()
	if err != nil {
		return err
	}
	defer tp.Close()

	sync, err := tp.CreateSyncTransport()
	if err != nil {
		return err
	}
	defer sync.Close()

	data := fmt.Sprintf("%s,%d", remotePath, mode[0])
	err = sync.Send("SEND", data)
	if err != nil {
		return err
	}

	err = sync.SendStream(source)
	if err != nil {
		return err
	}

	err = sync.SendStatus("DONE", uint32(modification.Unix()))
	if err != nil {
		return err
	}

	err = sync.VerifyStatus()
	if err != nil {
		return err
	}
	return nil
}

// Pull pulls a file from the device
func (d Device) Pull(remotePath string, dest io.Writer) error {
	tp, err := d.createDeviceTransport()
	if err != nil {
		return err
	}
	defer tp.Close()

	sync, err := tp.CreateSyncTransport()
	if err != nil {
		return err
	}
	defer sync.Close()

	err = sync.Send("RECV", remotePath)
	if err != nil {
		return err
	}

	err = sync.WriteStream(dest)
	if err != nil {
		return err
	}
	return nil
}

func (d Device) Logcat(dst io.Writer, exitChan chan bool) error {
	var tp transport
	var err error
	if tp, err = d.createDeviceTransport(); err != nil {
		return err
	}
	defer func() { _ = tp.Close() }()

	if err = tp.Send("shell:logcat"); err != nil {
		return err
	}
	if err = tp.VerifyResponse(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		r := NewReader(ctx, tp.sock)
		io.Copy(dst, r)
	}()
	<-exitChan
	cancel()
	return err
}

func (d Device) Logcat2File(file string, exitChan chan bool) error {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_SYNC|os.O_APPEND, 0755)
	if err != nil {
		return err
	}
	defer f.Close()
	return d.Logcat(f, exitChan)
}

func (d Device) LogcatClear() error {
	_, err := d.executeCommand("shell:logcat -c")
	return err
}
