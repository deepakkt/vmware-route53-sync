package main

import (
	"log"
	"time"
	"vmc-dns-sync/pkg/dns_api"
	"vmc-dns-sync/pkg/triage"
)

func main() {
	syncFrequency := dns_api.GetSyncFrequencySeconds()
	log.Printf("Starting DNS sync. We will sync at frequency of %d secs\n", syncFrequency)

	for {
		awsHelper := dns_api.AWSDNSAPI{}
		vmwNameToIPMap, err := dns_api.GetVMs()

		if err != nil {
			log.Println("Error fetching VMs")
			log.Println(err)
			log.Println("Let us retry next cycle")
			continue
		}
		awsDNSToR53IPMap := dns_api.GetR53DNStoIPMapping()
		k8sDNSToVMWNameMap := dns_api.GetDNStoVMMapping()

		result := triage.IPTriage(vmwNameToIPMap, awsDNSToR53IPMap, k8sDNSToVMWNameMap)

		err = triage.SyncRoute53(result, awsHelper)

		if err != nil {
			log.Println("Error doing final sync")
			log.Println(err)
			log.Println("Let us retry next cycle")
			time.Sleep(syncFrequency * time.Second)
			continue
		}

		log.Printf("Now sleeping for %d seconds\n", syncFrequency)
		time.Sleep(syncFrequency * time.Second)
	}
}
