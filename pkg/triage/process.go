package triage

import (
	"fmt"
	"log"
	"strings"
	"vmc-dns-sync/pkg/dns_api"
	"vmc-dns-sync/pkg/model"
)

func isDryRun() bool {
	dryRun := dns_api.GetEnv("R53_UPDATE_DRY_RUN")

	return dryRun != "FALSE"
}

func mapVMCNameToIP(k8sDNSToBuilderMap, vmcBuilderToIPMap map[string]string) map[string]string {
	result := make(map[string]string)

	for key := range k8sDNSToBuilderMap {
		builder := k8sDNSToBuilderMap[key]

		if vmcIP, ok := vmcBuilderToIPMap[builder]; ok {
			result[key] = vmcIP
		}
	}

	return result
}

func isVMCv6(ip string) bool {
	// checks if vmware has reported ipv6 instead of ipv4
	return strings.Contains(ip, "::")
}

func IPTriage(vmcBuilderToIPMap, awsDNSToIPMap, k8sDNSToBuilderMap map[string]string) map[string]model.IPTriageSummary {
	result := make(map[string]model.IPTriageSummary)

	vmcIPMap := mapVMCNameToIP(k8sDNSToBuilderMap, vmcBuilderToIPMap)

	for key := range awsDNSToIPMap {
		var currentTriage model.IPTriageSummary

		currentTriage.HttpEntry = key
		currentTriage.R53IP = awsDNSToIPMap[key]
		currentTriage.Source = model.IPTriageSourceR53
		currentTriage.Result = model.IPTriageDeleteR53

		result[key] = currentTriage
	}

	for key := range vmcIPMap {
		if currentTriage, ok := result[key]; ok {
			currentTriage.Source = model.IPTriageSourceBoth
			currentTriage.VmwIP = vmcIPMap[key]

			if currentTriage.VmwIP == currentTriage.R53IP {
				currentTriage.Result = model.IPTriageNoChange
			} else if isVMCv6(currentTriage.VmwIP) {
				log.Printf("VMW %s has reported v6 IP %s. We will ignore/delete R53 instead of updating", currentTriage.HttpEntry, currentTriage.VmwIP)
				currentTriage.Result = model.IPTriageDeleteR53
			} else {
				currentTriage.Result = model.IPTriageUpdateR53
			}
			result[key] = currentTriage
		} else {
			var currentTriage model.IPTriageSummary

			currentTriage.HttpEntry = key
			currentTriage.VmwIP = vmcIPMap[key]
			if !isVMCv6(currentTriage.VmwIP) {
				currentTriage.Result = model.IPTriageAddR53
			} else {
				log.Printf("VMW %s has reported v6 IP %s. We will ignore/delete R53 instead of updating", currentTriage.HttpEntry, currentTriage.VmwIP)
				currentTriage.Result = model.IPTriageNoChange
			}
			currentTriage.Source = model.IPTriageSourceVMW

			result[key] = currentTriage
		}
	}

	return result
}

func SyncRoute53(triageResult map[string]model.IPTriageSummary,
				awsHelper dns_api.AwsHelperInterface) error {
	log.Println("Starting final sync")

	for key := range triageResult {
		log.Printf("Host - %s, R53 - %s, VMW - %s\n", key, triageResult[key].R53IP, triageResult[key].VmwIP)

		switch triageResult[key].Result {
		case model.IPTriageNoChange:
			log.Println("No change in IP - no action needed")
		case model.IPTriageUpdateR53:
			log.Println("R53 IP is different. Updating R53")
		case model.IPTriageDeleteR53:
			log.Println("R53 DNS not found on VMW or VMW invalid IP. Delete R53")
		case model.IPTriageAddR53:
			log.Println("VMC IP not found on R53. Add R53")
		}
	}

	if isDryRun() {
		log.Println("Configuration mentions dry run. No updates made to route 53")
		return fmt.Errorf("DNS002: Dry run - no action taken")
	}

	log.Println("Not a dry run. Commencing final sync to R53")
	err := awsHelper.UpdateDNSRecords(triageResult, awsHelper)

	if err != nil {
		log.Printf("We encountered an update error %v\n", err)
		log.Println("We will not panic. The next cycle may be successful")
	}

	return err
}


