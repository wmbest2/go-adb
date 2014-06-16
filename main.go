package main

import (
	"flag"
	"fmt"
	"github.com/wmbest2/android/adb"
	"os"
	"sync"
)

func runOnDevice(wg *sync.WaitGroup, d *adb.Device, params *[]string) {
	defer wg.Done()
	v := adb.ShellSync(d, *params...)
	fmt.Printf("%s\n", string(v))
}

func runOnAll(params []string) []byte {
	var wg sync.WaitGroup
	devices := adb.ListDevices(nil)

	if len(devices) == 0 {
		return []byte("No devices found\n")
	}

	for _, d := range devices {
		wg.Add(1)
		fmt.Printf("%s\n", d)
		go runOnDevice(&wg, d, &params)
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

func runAndPrint(args ...string) {
	a := adb.Default
	output := adb.Shell(a, args...)
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

func main() {

	s := flag.String("s", "", "directs command to the device or emulator with the given\nserial number or qualifier. Overrides ANDROID_SERIAL\n environment variable.")
	p := flag.String("p", "", "directs command to the device or emulator with the given\nserial number or qualifier. Overrides ANDROID_SERIAL\n environment variable.")
	a := flag.Bool("a", false, "directs adb to listen on all interfaces for a connection")
	d := flag.Bool("d", false, "directs command to the only connected USB device\nreturns an error if more than one USB device is present.")
	e := flag.Bool("e", false, "directs command to the device or emulator with the given\nserial number or qualifier. Overrides ANDROID_SERIAL\n environment variable.")
	H := flag.String("H", "", "directs command to the device or emulator with the given\nserial number or qualifier. Overrides ANDROID_SERIAL\n environment variable.")
	P := flag.String("P", "", "directs command to the device or emulator with the given\nserial number or qualifier. Overrides ANDROID_SERIAL\n environment variable.")

	flag.Parse()

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
	if *s != "" {
		runAndPrint(os.Args[1:]...)
	} else {
		switch flag.Arg(0) {
		case "install":
			out = runOnAll(args)
		case "uninstall":
			out = runOnAll(args)
		case "devices":
			fmt.Println("List of devices attached")
			devices := adb.ListDevices(nil)

			if len(devices) == 0 {
				out = []byte("No devices found\n")
			} else {
				for _, d := range devices {
					out = append(out, []byte(fmt.Sprintln(d.String()))...)
				}
				out = append(out, []byte(fmt.Sprintln("\n"))...)
			}
		default:
			runAndPrint(flag.Args()...)
		}
	}
	fmt.Print(string(out))
}
