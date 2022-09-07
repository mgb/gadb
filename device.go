package gadb

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	dirBit          = 1 << 14
	defaultFileMode = os.FileMode(0o664)
)

// DeviceFileInfo is the information of a file on the device
type DeviceFileInfo struct {
	Name         string
	Mode         os.FileMode
	Size         uint32
	LastModified time.Time
}

// IsDir returns true if the file is a directory
func (info DeviceFileInfo) IsDir() bool {
	return info.Mode&dirBit != 0
}

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
func (d Device) Product() string {
	return d.attrs["product"]
}

// Model returns the model name of the device
func (d Device) Model() string {
	return d.attrs["model"]
}

// Usb returns the usb information of the device
func (d Device) Usb() string {
	return d.attrs["usb"]
}

func (d Device) transportId() string {
	return d.attrs["transport_id"]
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
func (d Device) IsUsb() bool {
	return d.Usb() != ""
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

	_, err := d.adbClient.executeCommand(command, true)
	return err
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
	local := fmt.Sprintf("tcp:%d", localPort)
	_, err := d.adbClient.executeCommand(fmt.Sprintf("host-serial:%s:killforward:%s", d.serial, local), true)
	return err
}

// RunShellCommand runs a shell command on the device
func (d Device) RunShellCommand(cmd string, args ...string) (string, error) {
	raw, err := d.RunShellCommandWithBytes(cmd, args...)
	if err != nil {
		return string(raw), err
	}
	return string(raw), nil
}

// RunShellCommandWithBytes runs a shell command on the device and returns the raw bytes
func (d Device) RunShellCommandWithBytes(cmd string, args ...string) ([]byte, error) {
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

	_, err := d.executeCommand(fmt.Sprintf("tcpip:%d", port[0]), true)
	return err
}

func (d Device) createDeviceTransport() (transport, error) {
	tp, err := newTransport(fmt.Sprintf("%s:%d", d.adbClient.host, d.adbClient.port))
	if err != nil {
		return transport{}, err
	}

	err = tp.Send(fmt.Sprintf("host:transport:%s", d.serial))
	if err != nil {
		return transport{}, err
	}

	err = tp.VerifyResponse()
	if err != nil {
		return transport{}, err
	}
	return tp, nil
}

func (d Device) executeCommand(command string, onlyVerifyResponse ...bool) ([]byte, error) {
	if len(onlyVerifyResponse) == 0 {
		onlyVerifyResponse = []bool{false}
	}

	tp, err := d.createDeviceTransport()
	if err != nil {
		return nil, err
	}
	defer tp.Close()

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

	raw, err := tp.ReadBytesAll()
	if err != nil {
		return raw, err
	}
	return raw, nil
}

// List returns the list of files in the directory
func (d Device) List(remotePath string) ([]DeviceFileInfo, error) {
	tp, err := d.createDeviceTransport()
	if err != nil {
		return nil, err
	}
	defer tp.Close()

	sync, err := tp.CreateSyncTransport()
	if err != nil {
		return nil, err
	}
	defer sync.Close()

	err = sync.Send("LIST", remotePath)
	if err != nil {
		return nil, err
	}

	var devFileInfos []DeviceFileInfo
	for {
		entry, err := sync.ReadDirectoryEntry()
		if err != nil {
			return nil, err
		}
		if entry == (DeviceFileInfo{}) {
			break
		}

		devFileInfos = append(devFileInfos, entry)
	}

	return devFileInfos, nil
}

// PushFile pushes a file to the device
func (d Device) PushFile(local *os.File, remotePath string, modification ...time.Time) error {
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
