package gadb

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
)

func TestDevice_State(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range devices {
		s, err := d.State()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(d.Serial(), s)
	}
}

func TestDevice_DevicePath(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range devices {
		p, err := d.DevicePath()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(d.Serial(), p)
	}
}

func TestDevice_Product(t *testing.T) {
	adbClient, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := adbClient.List()
	if err != nil {
		t.Fatal(err)
	}

	for i := range devices {
		dev := devices[i]
		product, err := dev.Product()
		t.Log(dev.Serial(), product, err)
	}
}

func TestDevice_Model(t *testing.T) {
	adbClient, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := adbClient.List()
	if err != nil {
		t.Fatal(err)
	}

	for i := range devices {
		dev := devices[i]
		model, err := dev.Model()
		t.Log(dev.Serial(), model, err)
	}
}

func TestDevice_Usb(t *testing.T) {
	adbClient, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := adbClient.List()
	if err != nil {
		t.Fatal(err)
	}

	for i := range devices {
		dev := devices[i]
		usb, err := dev.Usb()
		t.Log(dev.Serial(), usb, err)
	}
}

func TestDevice_DeviceInfo(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range devices {
		t.Log(d.Serial(), d.DeviceInfo())
	}
}

func TestDevice_Forward(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) == 0 {
		t.SkipNow()
	}

	localPort := 61000
	err = devices[0].Forward(localPort, 6790)
	if err != nil {
		t.Fatal(err)
	}

	err = devices[0].ForwardKill(localPort)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDevice_ForwardList(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	for _, d := range devices {
		forwardList, err := d.ForwardList()
		if err != nil {
			t.Fatal(err)
		}
		t.Log(d.serial, "->", forwardList)
	}
}

func TestDevice_ForwardKill(t *testing.T) {
	t.Skip("Requires manual setup of a forwarding port")

	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) == 0 {
		t.SkipNow()
	}

	err = devices[0].ForwardKill(6790)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDevice_RunShellCommand(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) == 0 {
		t.SkipNow()
	}

	// for _, d := range devices {
	// 	dev := devices[i]
	// 	// cmdOutput, err := dev.RunShellCommand(`pm list packages  | grep  "bili"`)
	// 	// cmdOutput, err := dev.RunShellCommand(`pm list packages`, `| grep "bili"`)
	// 	// cmdOutput, err := dev.RunShellCommand("dumpsys activity | grep mFocusedActivity")
	// 	cmdOutput, err := dev.RunShellCommand("monkey", "-p", "tv.danmaku.bili", "-c", "android.intent.category.LAUNCHER", "1")
	// 	if err != nil {
	// 		t.Fatal(dev.serial, err)
	// 	}
	// 	t.Log("\n"+dev.serial, cmdOutput)
	// }

	//

	// cmdOutput, err := dev.RunShellCommand("monkey", "-p", "tv.danmaku.bili", "-c", "android.intent.category.LAUNCHER", "1")
	cmdOutput, err := devices[0].RunShellCommand("ls /sdcard")
	// cmdOutput, err := dev.RunShellCommandWithBytes("screencap -p")
	if err != nil {
		t.Fatal(devices[0].serial, err)
	}
	t.Log("\n⬇️"+devices[0].serial+"⬇️\n", cmdOutput)

}

func TestDevice_EnableAdbOverTCP(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) == 0 {
		t.SkipNow()
	}

	err = devices[0].EnableAdbOverTCP()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDevice_List(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) == 0 {
		t.SkipNow()
	}

	fileEntries, err := devices[0].List("/sdcard/Download")
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range fileEntries {
		t.Log(f.Name(), "\t", f.IsDir())
	}
}

func TestDevice_Push(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) == 0 {
		t.SkipNow()
	}

	mm := afero.NewMemMapFs()
	f, err := mm.Create("test.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	_, err = f.WriteString("Hello World")
	if err != nil {
		t.Fatal(err)
	}
	f.Seek(0, 0)

	err = devices[0].PushFile(f, "/sdcard/Download/push.txt", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	err = devices[0].Push(strings.NewReader("world"), "/sdcard/Download/hello.txt", time.Now())
	if err != nil {
		t.Fatal(err)
	}
}

func TestDevice_Pull(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatal(err)
	}

	devices, err := c.List()
	if err != nil {
		t.Fatal(err)
	}

	if len(devices) == 0 {
		t.SkipNow()
	}

	buffer := bytes.NewBufferString("")
	err = devices[0].Pull("/sdcard/Download/hello.txt", buffer)
	if err != nil {
		t.Fatal(err)
	}

	userHomeDir, _ := os.UserHomeDir()
	err = ioutil.WriteFile(userHomeDir+"/Desktop/hello.txt", buffer.Bytes(), defaultFileMode)
	if err != nil {
		t.Fatal(err)
	}
}
