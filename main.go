package main

import (
	"flag"
	"fmt"
	"github.com/wmbest2/android/adb"
	"github.com/wmbest2/android/pidcat"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

func runOnDevice(wg *sync.WaitGroup, d *adb.Device, params []string) {
	defer wg.Done()
	adb.ShellSync(d, params...)
}

func runCmdOnDevice(wg *sync.WaitGroup, d *adb.Device, params ...[]string) {
	defer wg.Done()
	for _, cmd := range params {
		trace("Started with params %s", cmd)
		adb.ShellSync(d, cmd...)
	}
}

func runCommands(devices []*adb.Device, params ...[]string) []byte {
	var wg sync.WaitGroup

	if len(devices) == 0 {
		return []byte("No devices found\n")
	}

	for _, d := range devices {
		wg.Add(1)
		go runCmdOnDevice(&wg, d, params...)
	}
	wg.Wait()
	return []byte("")
}

func runOnAll(devices []*adb.Device, params ...string) []byte {
	defer un(trace("Started with params %s", params))
	var wg sync.WaitGroup

	if len(devices) == 0 {
		return []byte("No devices found\n")
	}

	for _, d := range devices {
		wg.Add(1)
		go runOnDevice(&wg, d, params)
	}
	wg.Wait()
	return []byte("")
}

func flagFromBool(f bool, s string) *string {
	result := fmt.Sprintf("-%s", s)
	if !f {
		result = ""
	}
	return &result
}

func runAndPrint(t adb.Transporter, args ...string) {
	output := adb.Shell(t, args...)
	out_ok := true
	for {
		var v interface{}
		if !out_ok {
			break
		}
		switch v, out_ok = <-output; v.(type) {
		case []byte:
			fmt.Printf("%s\n", v.([]byte))
		}
	}
}

func trace(s string, args ...interface{}) (string, time.Time) {
	log.Println("START:", fmt.Sprintf(s, args...))
	return fmt.Sprintf("args[%s]", args...), time.Now()
}

func un(s string, startTime time.Time) {
	endTime := time.Now()
	log.Println("  END:", s, "ElapsedTime in seconds:", endTime.Sub(startTime))
}

func install(file string) {
	f, _ := os.Open(file)
	devices := adb.ListDevices(nil)
	fmt.Printf("%s:\n", time.Now())
	stat, _ := f.Stat()
	loc := fmt.Sprintf("/sdcard/tmp/%s", stat.Name())
	err := adb.PushDevices(devices, f, loc)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%s: %s\n", time.Now(), loc)

	runCommands(devices, []string{"pm", "install", "-r", loc}, []string{"rm", loc})
}

func uninstall(args ...string) {
	devices := adb.ListDevices(nil)
	runCommands(devices, append([]string{"pm"}, args...))
}

func screenshot(t adb.Transporter, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		return
	}
	f.Write(adb.ShellSync(t, "screencap", "-p"))
	f.Close()
}

func logcat(t adb.Transporter) {
	lc.Parse(flag.Args()[1:])

	if *lc_clear {
		pidcat.Clear(t)
	}

	p := pidcat.NewPidCat(*lc_pidcat, 22)
	p.SetAppFilters(strings.Split(lc.Arg(0), ":")...)
	p.UpdateAppFilters(t)

	cmd := fmt.Sprintf("export ANDROID_LOG_TAGS=\"%s\" ; logcat", lc_tags)
	out := adb.Shell(t, cmd)
	for line := range out {
		fmt.Print(p.Sprint(string(line)))
	}
}

var (
	s = flag.String("s", "", "directs command to the device or emulator with the given serial number or qualifier. Overrides ANDROID_SERIAL  environment variable.")
	p = flag.String("p", "", "directs command to the device or emulator with the given serial number or qualifier. Overrides ANDROID_SERIAL  environment variable.")
	a = flag.Bool("a", false, "directs adb to listen on all interfaces for a connection")
	d = flag.Bool("d", false, "directs command to the only connected USB device returns an error if more than one USB device is present.")
	e = flag.Bool("e", false, "directs command to the device or emulator with the given serial number or qualifier. Overrides ANDROID_SERIAL  environment variable.")
	H = flag.String("H", "", "directs command to the device or emulator with the given serial number or qualifier. Overrides ANDROID_SERIAL  environment variable.")
	P = flag.String("P", "", "directs command to the device or emulator with the given serial number or qualifier. Overrides ANDROID_SERIAL  environment variable.")

	lc        = flag.NewFlagSet("logcat [options] PACKAGES(colon-separated)", flag.ExitOnError)
	lc_clear  = lc.Bool("clear", false, "clear (flush) the entire log and exit")
	lc_tags   = lc.String("tags", "", "a space separated list of tag filters i.e. \"*:S MyTag:V\"")
	lc_pidcat = lc.Bool("pretty-print", true, "pretty print logcat output")
)

func init() {
	flag.Parse()
}

func main() {
	aFlag := flagFromBool(*a, "a")
	dFlag := flagFromBool(*d, "d")
	eFlag := flagFromBool(*e, "e")

	allParams := []*string{aFlag, dFlag, eFlag, p, H, P}
	params := make([]string, 0, 7)
	for _, param := range allParams {
		if *param != "" {
			params = append(params, []string{*param}...)
		}
	}

	l := len(params) + len(flag.Args())
	args := make([]string, 0, l)
	args = append(args, params...)
	args = append(args, flag.Args()...)

	var out []byte
	t := adb.Transporter(adb.Default)
	if *s != "" {
		device := adb.Default.FindDevice(*s)
		t = adb.Transporter(&device)
	}

	switch flag.Arg(0) {
	case "push":
		f, _ := os.Open(flag.Arg(1))
		adb.Push(t, f, flag.Arg(2))
	case "pull":
		f, _ := os.Create(flag.Arg(2))
		err := adb.Pull(t, f, flag.Arg(1))
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
	case "logcat":
		logcat(t)
	case "install":
		install(flag.Arg(1))
	case "uninstall":
		uninstall(args...)
	case "screencap":
		screenshot(t, flag.Arg(1))
	case "devices":
		devices := adb.ListDevices(nil)
		fmt.Println("List of devices attached")

		if len(devices) == 0 {
			out = []byte("No devices found\n")
		} else {
			for _, d := range devices {
				out = append(out, []byte(fmt.Sprintln(d.String()))...)
			}
			out = append(out, []byte(fmt.Sprintln("\n"))...)
		}
	case "ls":
		adb.Ls(t, flag.Arg(1))
	default:
		runAndPrint(t, flag.Args()...)
	}
	fmt.Print(string(out))
}
