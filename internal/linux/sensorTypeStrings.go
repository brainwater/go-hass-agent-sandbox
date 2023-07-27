// Code generated by "stringer -type=sensorType -output sensorTypeStrings.go -linecomment"; DO NOT EDIT.

package linux

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[appActive-1]
	_ = x[appRunning-2]
	_ = x[battType-3]
	_ = x[battPercentage-4]
	_ = x[battTemp-5]
	_ = x[battVoltage-6]
	_ = x[battEnergy-7]
	_ = x[battEnergyRate-8]
	_ = x[battState-9]
	_ = x[battNativePath-10]
	_ = x[battLevel-11]
	_ = x[battModel-12]
	_ = x[memTotal-13]
	_ = x[memAvail-14]
	_ = x[memUsed-15]
	_ = x[swapTotal-16]
	_ = x[swapUsed-17]
	_ = x[swapFree-18]
	_ = x[connectionState-19]
	_ = x[connectionID-20]
	_ = x[connectionDevices-21]
	_ = x[connectionType-22]
	_ = x[connectionIPv4-23]
	_ = x[connectionIPv6-24]
	_ = x[addressIPv4-25]
	_ = x[addressIPv6-26]
	_ = x[wifiSSID-27]
	_ = x[wifiFrequency-28]
	_ = x[wifiSpeed-29]
	_ = x[wifiStrength-30]
	_ = x[wifiHWAddress-31]
	_ = x[bytesSent-32]
	_ = x[bytesRecv-33]
	_ = x[powerProfile-34]
	_ = x[boottime-35]
	_ = x[uptime-36]
	_ = x[load1-37]
	_ = x[load5-38]
	_ = x[load15-39]
	_ = x[screenLock-40]
	_ = x[problem-41]
}

const _sensorType_name = "Active AppRunning AppsBattery TypeBattery LevelBattery TemperatureBattery VoltageBattery EnergyBattery PowerBattery StateBattery PathBattery LevelBattery ModelMemory TotalMemory AvailableMemory UsedSwap Memory TotalSwap Memory UsedSwap Memory FreeConnection StateConnection IDConnection DeviceConnection TypeConnection IPv4Connection IPv6IPv4 AddressIPv6 AddressWi-Fi SSIDWi-Fi FrequencyWi-Fi Link SpeedWi-Fi Signal StrengthWi-Fi BSSIDBytes SentBytes RecievedPower ProfileLast RebootUptimeCPU load average (1 min)CPU load average (5 min)CPU load average (15 min)Screen LockProblems"

var _sensorType_index = [...]uint16{0, 10, 22, 34, 47, 66, 81, 95, 108, 121, 133, 146, 159, 171, 187, 198, 215, 231, 247, 263, 276, 293, 308, 323, 338, 350, 362, 372, 387, 403, 424, 435, 445, 459, 472, 483, 489, 513, 537, 562, 573, 581}

func (i sensorType) String() string {
	i -= 1
	if i < 0 || i >= sensorType(len(_sensorType_index)-1) {
		return "sensorType(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _sensorType_name[_sensorType_index[i]:_sensorType_index[i+1]]
}