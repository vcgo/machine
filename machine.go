package machine

import (
	"fmt"
	"net"
	"strings"
	"syscall"
	"unsafe"

	"github.com/StackExchange/wmi"
	"github.com/vcgo/van"
)

var (
	advapi = syscall.NewLazyDLL("Advapi32.dll")
	kernel = syscall.NewLazyDLL("Kernel32.dll")
)

func GetUniqueId() string {
	var str string
	str = GetBiosInfo()               // BIOS
	str += "|" + GetMotherboardInfo() // 主板
	str += "|" + GetCpuInfo()         // CPU，相同CPU此参数相同
	str += "|" + GetMemory()          // 内存数
	str += "|" + GetDiskInfo()        // 硬盘，按盘符、总量
	str += "|" + GetMac()             // Mac 地址
	return van.Md5(str)
}

func GetStr() string {
	var str string
	str = GetBiosInfo()               // BIOS
	str += "|" + GetMotherboardInfo() // 主板
	str += "|" + GetCpuInfo()         // CPU，相同CPU此参数相同
	str += "|" + GetMemory()          // 内存数
	str += "|" + GetDiskInfo()        // 硬盘，按盘符、总量
	str += "|" + GetMac()             // Mac 地址
	return str
}

// dist
func GetDiskInfo() (infos string) {
	// https://docs.microsoft.com/zh-cn/windows/win32/cimwin32prov/win32-diskdrive
	var storageinfo []struct {
		Caption      string
		SerialNumber string
	}
	err := wmi.Query("Select * from Win32_DiskDrive ", &storageinfo)
	if err != nil {
		return "nil disk"
	}
	res := ""
	for _, v := range storageinfo {
		res += "Caption:" + v.Caption + ";SerialNumber:" + v.SerialNumber + ";"
	}
	return res
}

//CPU信息
//简单的获取方法fmt.Sprintf("Num:%d Arch:%s\n", runtime.NumCPU(), runtime.GOARCH)
func GetCpuInfo() string {
	var size uint32 = 128
	var buffer = make([]uint16, size)
	var index = uint32(copy(buffer, syscall.StringToUTF16("Num:")) - 1)
	nums := syscall.StringToUTF16Ptr("NUMBER_OF_PROCESSORS")
	arch := syscall.StringToUTF16Ptr("PROCESSOR_ARCHITECTURE")
	r, err := syscall.GetEnvironmentVariable(nums, &buffer[index], size-index)
	if err != nil {
		return ""
	}
	index += r
	index += uint32(copy(buffer[index:], syscall.StringToUTF16(" Arch:")) - 1)
	r, err = syscall.GetEnvironmentVariable(arch, &buffer[index], size-index)
	if err != nil {
		return syscall.UTF16ToString(buffer[:index])
	}
	index += r
	return syscall.UTF16ToString(buffer[:index+r])
}

type memoryStatusEx struct {
	cbSize                  uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64 // in bytes
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

//内存信息
func GetMemory() string {
	GlobalMemoryStatusEx := kernel.NewProc("GlobalMemoryStatusEx")
	var memInfo memoryStatusEx
	memInfo.cbSize = uint32(unsafe.Sizeof(memInfo))
	mem, _, _ := GlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memInfo)))
	if mem == 0 {
		return ""
	}
	return fmt.Sprint(memInfo.ullTotalPhys)
}

type intfInfo struct {
	Name string
	Ipv4 []string
	Ipv6 []string
}

func GetMac() string {
	// 获取本机的MAC地址
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, inter := range interfaces {
			mac := inter.HardwareAddr //获取本机MAC地址
			return mac.String()
		}
	}
	return ""
}

//网卡信息
func GetIntfs() []intfInfo {
	intf, err := net.Interfaces()
	if err != nil {
		return []intfInfo{}
	}
	var is = make([]intfInfo, len(intf))
	for i, v := range intf {
		ips, err := v.Addrs()
		if err != nil {
			continue
		}
		is[i].Name = v.Name
		for _, ip := range ips {
			if strings.Contains(ip.String(), ":") {
				is[i].Ipv6 = append(is[i].Ipv6, ip.String())
			} else {
				is[i].Ipv4 = append(is[i].Ipv4, ip.String())
			}
		}
	}
	return is
}

//主板信息
func GetMotherboardInfo() string {
	var s = []struct {
		Product string
	}{}
	err := wmi.Query("SELECT  Product  FROM Win32_BaseBoard WHERE (Product IS NOT NULL)", &s)
	if err != nil {
		return ""
	}
	return s[0].Product
}

//BIOS信息
func GetBiosInfo() string {
	var s = []struct {
		Name string
	}{}
	err := wmi.Query("SELECT Name FROM Win32_BIOS WHERE (Name IS NOT NULL)", &s) // WHERE (BIOSVersion IS NOT NULL)
	if err != nil {
		return ""
	}
	return s[0].Name
}
