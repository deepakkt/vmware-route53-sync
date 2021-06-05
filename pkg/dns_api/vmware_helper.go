package dns_api

import (
	"context"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"log"
	"net/url"
	"strings"
)

func processOverride(u *url.URL) {
	envUsername := GetEnv("VMWARE_USERNAME")
	envPassword := GetEnv("VMWARE_PASSWORD")

	// Override username if provided
	if envUsername != "" {
		var password string
		var ok bool

		if u.User != nil {
			password, ok = u.User.Password()
		}

		if ok {
			u.User = url.UserPassword(envUsername, password)
		} else {
			u.User = url.User(envUsername)
		}
	}

	// Override password if provided
	if envPassword != "" {
		var username string

		if u.User != nil {
			username = u.User.Username()
		}

		u.User = url.UserPassword(username, envPassword)
	}
}

// NewClient creates a govmomi.Client for use
func NewClient(ctx context.Context) (*govmomi.Client, error) {
	var urlFlag = GetEnv("VMWARE_SDDC_URL")

	if urlFlag == "" {
		log.Panic("Vmware SDDC URL expected via env var VMWARE_SDDC_URL is missing")
	}
	insecureConnect := strings.ToLower(GetEnv("VMWARE_VERIFY_SSL")) != "false"

	// Parse URL from string
	u, err := soap.ParseURL(urlFlag)
	if err != nil {
		return nil, err
	}

	// Override username and/or password as required
	processOverride(u)

	// Connect and log in to ESX or vCenter
	return govmomi.NewClient(ctx, u, insecureConnect)
}

func GetVMs() (map[string]string, error) {
	vmMap := make(map[string]string)

	log.Println("Initializing context")
	ctx := context.TODO()

	log.Println("Attempting to create vmware connection")
	c, err := NewClient(ctx)
	if err != nil {
		log.Printf("Sorry - vmware connection attempt failed: %v\n", err)
		return vmMap, err
	}
	log.Println("vmware connection succeeded. Now fetching VMs")

	m := view.NewManager(c.Client)
	v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder,
		[]string{"VirtualMachine"}, true)

	if err != nil {
		log.Printf("Sorry - retrieval of VMs failed: %v\n", err)
		return vmMap, err
	}
	defer v.Destroy(ctx)

	// Retrieve summary property for all machines
	// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.VirtualMachine.html
	var vms []mo.VirtualMachine
	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"summary"}, &vms)
	if err != nil {
		log.Printf("Sorry - retrieval of VMs failed: %v\n", err)
		return vmMap, err
	}

	for _, vm := range vms {
		vmIP := vm.Summary.Guest.IpAddress
		vmName := vm.Summary.Config.Name


		log.Printf("Fetched VM %s\n", vmName)

		if vmIP == "" {
			log.Printf("VM %s has no IP.It is either a frozen VM or a template. Let's skip\n", vmName)
			continue
		}

		log.Printf("Adding %s - %s\n", vmName, vmIP)
		vmMap[vmName] = vmIP
	}

	return vmMap, nil
}