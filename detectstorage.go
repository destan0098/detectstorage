package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

//const ipAdd = "http://10.30.20.1/serial.txt"

const ipAdd = "http://10.10.20.1/serial.txt"

type DeviceInfo struct {
	Device    string `json:"device"`
	Type      string `json:"type"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	BusDevice string `json:"bus_device"`
	IP        string `json:"ip"`
	Allow     bool   `json:"allow"`
}

func main() {
	switch runtime.GOOS {
	case "windows":
		listMassStorageDevicesWindows()
	case "linux":
		listMassStorageDevicesLinux()
	default:
		log.Println("Unsupported OS.")
	}
}

func listMassStorageDevicesWindows() {
	devices := listUSBMassStorageWindows()
	allowList := fetchAllowList(ipAdd)
	if len(devices) == 0 {
		fmt.Println("No USB devices found.")
		os.Exit(0)
	}

	for device, name := range devices {

		info := detectDeviceInfo(device, "N/A", name, allowList)
		fmt.Println(formatDeviceInfo(info))
	}
}

func listUSBMassStorageWindows() map[string]string {
	devices := make(map[string]string)

	cmd := exec.Command("wmic", "diskdrive", "where", "InterfaceType='USB'", "get", "Model,SerialNumber,InterfaceType,Size,MediaType")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error listing USB devices on Windows: %v", err)
		return devices
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		log.Println("No USB devices found in WMIC output.")
		return devices
	}

	// **حذف خط اول (هدر)**
	lines = lines[1:]

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)

		// **اطمینان از اینکه تعداد ستون‌های کافی وجود دارد**
		if len(parts) < 2 {
			continue
		}

		// **استخراج Model و SerialNumber**
		serial := parts[len(parts)-2]                    // ستون قبل از آخرین مقدار (Size یا MediaType)
		model := strings.Join(parts[:len(parts)-2], " ") // بقیه‌ی مقدار به‌عنوان Model

		devices[serial] = model
	}

	return devices

}

func listMassStorageDevicesLinux() {
	devices := listUSBMassStorageLinux()
	allowList := fetchAllowList(ipAdd)
	if len(devices) == 0 {
		fmt.Println("No USB devices found.")
		os.Exit(0)
	}

	for device, info := range devices {
		detectedInfo := detectDeviceInfo(device, info.BusDevice, info.Name, allowList)
		fmt.Println(formatDeviceInfo(detectedInfo))
	}

}

func extractWindowsLikeSerial(serial string) string {
	if len(serial) > 20 {
		return serial[:20]
	}
	return serial
}
func listUSBMassStorageLinux() map[string]struct {
	BusDevice string
	Name      string
} {
	devices := make(map[string]struct {
		BusDevice string
		Name      string
	})

	cmd := exec.Command("lsusb")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error listing USB devices on Linux: %v", err)
		return devices
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}

		busID := parts[1]
		deviceID := parts[3][:len(parts[3])-1]
		devicePath := fmt.Sprintf("/dev/bus/usb/%s/%s", busID, deviceID)

		udevCmd := exec.Command("udevadm", "info", "--query=all", "--name="+devicePath)
		udevOutput, err := udevCmd.Output()
		if err != nil {
			log.Printf("Error getting details for device %s: %v", devicePath, err)
			continue
		}

		var serial, interfaces, model string
		udevLines := strings.Split(string(udevOutput), "\n")
		for _, udevLine := range udevLines {
			if strings.HasPrefix(udevLine, "E: ID_SERIAL_SHORT=") {
				serial = strings.TrimSpace(strings.TrimPrefix(udevLine, "E: ID_SERIAL_SHORT="))
				serial = extractWindowsLikeSerial(serial)
			}
			if strings.HasPrefix(udevLine, "E: ID_USB_INTERFACES=") {
				interfaces = strings.TrimSpace(strings.TrimPrefix(udevLine, "E: ID_USB_INTERFACES="))
			}
			if strings.HasPrefix(udevLine, "E: ID_MODEL=") {
				model = strings.TrimSpace(strings.TrimPrefix(udevLine, "E: ID_MODEL="))
			}
		}

		if strings.Contains(interfaces, ":08") && serial != "" {
			devices[serial] = struct {
				BusDevice string
				Name      string
			}{
				BusDevice: devicePath,
				Name:      model,
			}
		}
	}

	return devices
}

func fetchAllowList(url string) map[string]bool {
	allowList := make(map[string]bool)
	//
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching allow list: %v", err)
		return allowList
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			allowList[line] = true
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading allow list response: %v", err)
	}

	return allowList
}

func detectDeviceInfo(deviceKey, busDevice, name string, allowList map[string]bool) DeviceInfo {
	return DeviceInfo{
		Device:    deviceKey,
		Type:      "Mass Storage Device",
		Name:      name,
		Status:    "connected",
		BusDevice: busDevice,
		IP:        getLocalIP(),
		Allow:     allowList[deviceKey],
	}
}

func formatDeviceInfo(info DeviceInfo) string {
	return fmt.Sprintf(
		"Device: %s\nType: %s\nName: %s\nStatus: %s\nBus Device: %s\nIP: %s\nAllow: %t",
		info.Device, info.Type, info.Name, info.Status, info.BusDevice, info.IP, info.Allow,
	)
}

func getLocalIP() string {
	var ips []string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("Error getting local IP: %v", err)
		return "unknown"
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ips = append(ips, ipNet.IP.String())
			}
		}
	}

	if len(ips) == 0 {
		return "unknown"
	}

	return strings.Join(ips, ", ")
}
