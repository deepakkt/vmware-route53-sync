
package dns_api

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"vmc-dns-sync/pkg/model"

	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

type AwsHelperInterface interface {
	UpdateDNSRecords(triageInput map[string]model.IPTriageSummary, awsHI AwsHelperInterface) error
	UpdateRoute53RecordSets(r53SyncSet route53.ChangeResourceRecordSetsInput) error
}

type AWSDNSAPI struct {
}

type batchPair struct {
	start int
	end int
}

type dnsStruct struct {
	dnsName string
	dnsAction string
	dnsIP string
	dnsOldIP string
}

func getUpdateBatchSize() int {
// use default 25 if nothing is present, else use env var
	batchSize := GetEnv("R53_UPDATE_BATCH_SIZE")

	if batchSize == "" {
		return 25
	}

	batchSizeInt, err := strconv.Atoi(batchSize)

	if err != nil {
		return 25
	}

	return batchSizeInt
}

func isDNSLengthOK(httpEntry string) bool {
	workName := getAWSAName(httpEntry)

	entries := strings.Split(workName, ".")

	return len(entries[0]) < 64
}

func getHostedZoneID() string {
	return GetEnv("R53_HOSTED_ZONE_ID")
}

func getAWSRegion() string {
	region := GetEnv("R53_SYNC_REGION")

	if region == "" {
		return "us-east-1"
	}

	return region
}

func getAWSAName(routeName string) string {
	// clean up prefix http and suffix "." if present
	return strings.TrimSuffix(
		strings.TrimPrefix(routeName,
			"http://"), ".",
		)
}

func getAWSAction(triageAction int) string {
// translate our triage action to AWS term
	switch triageAction {
	case model.IPTriageAddR53, model.IPTriageUpdateR53:
		return "UPSERT"
	case model.IPTriageDeleteR53:
		return "DELETE"
	}

	// we shouldn't come here
	return "UNKNOWN"
}

func getBatchPairs(sampleSize, batchSize int) []batchPair {
// If there are many R53 updates to be performed, it is not
// efficient or advisable to call the API for every update.
// It is recommended to batch them together. This function
// will give the batch endpoints based on sample size and batch size

	var pairList []batchPair

	start := 0
	for {
		end := start + batchSize

		if end >= sampleSize {
			pairList = append(pairList, batchPair{
				start: start,
				end:   sampleSize,
			})
			break
		}

		pairList = append(pairList, batchPair{
			start: start,
			end:   end,
		})
		start = end

	}

	return pairList
}

func createRoute53Session(awsRegion string) *route53.Route53 {
	awsSession := session.Must(session.NewSession())
	*awsSession.Config.Region = awsRegion
	return route53.New(awsSession)
}

func getRoute53Records() map[string]string {
// get route 53 records we are interested in and translate it
// into a simple route-ip dictionary
	manager := createRoute53Session(getAWSRegion())
	hostedZoneID := getHostedZoneID()

	if  hostedZoneID == "" {
		log.Panic("Hosted zone was expected and not provided via env var R53_HOSTED_ZONE_ID")
	}

	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
		StartRecordName: aws.String("A"),
	}

	hostedZone, _ := manager.ListResourceRecordSets(input)

	var record *route53.ResourceRecordSet
	var i int
	var dnsMap map[string]string
	dnsMap = make(map[string]string)

	for i, record = range hostedZone.ResourceRecordSets {
		log.Println(*record.Name, *record.ResourceRecords[0].Value)

		httpIP := *record.ResourceRecords[0].Value
		if strings.HasPrefix(httpIP, "ns") {
			log.Println("Ignoring NS record", httpIP)
			continue
		}

		httpRoute := "http://" + strings.TrimRight(*record.Name, ".")
		log.Printf("Adding route to map: %s - %s\n", httpRoute, httpIP)
		dnsMap[httpRoute] = httpIP
	}

	log.Printf("Processed %d records", i + 1)
	return dnsMap
}

// GetR53DNStoIPMapping - get dict map of http to ip
func GetR53DNStoIPMapping() map[string]string {

	log.Println("Syncing Route 53 entries")
	recordSets := getRoute53Records()

	return recordSets
}

func getR53UpdateSet(entries []dnsStruct) route53.ChangeResourceRecordSetsInput {
	var finalReturn route53.ChangeResourceRecordSetsInput
	var changeBatch route53.ChangeBatch
	var changeList []*route53.Change

	hostedZone := getHostedZoneID()
	finalReturn.HostedZoneId = &hostedZone


	for _, eachDNS := range entries{
		var currentChange route53.Change
		var recordSet route53.ResourceRecordSet
		var ipValue string

		if eachDNS.dnsAction == "DELETE" {
			ipValue = eachDNS.dnsOldIP
		} else {
			ipValue = eachDNS.dnsIP
		}

		log.Printf("Action: %s, DNS: %s, IP: %s\n", eachDNS.dnsAction, eachDNS.dnsName, ipValue)

		recordSet.Name = aws.String(eachDNS.dnsName)
		recordSet.ResourceRecords = []*route53.ResourceRecord{
			{
				Value: aws.String(ipValue),
			},
		}
		recordSet.TTL = aws.Int64(60)
		recordSet.Type = aws.String("A")

		currentChange.Action = aws.String(eachDNS.dnsAction)
		currentChange.ResourceRecordSet = &recordSet

		changeList = append(changeList, &currentChange)
	}

	changeBatch.Changes = changeList
	finalReturn.ChangeBatch = &changeBatch

	return finalReturn
}

func(a AWSDNSAPI) UpdateRoute53RecordSets(r53SyncSet route53.ChangeResourceRecordSetsInput) error {
	manager := createRoute53Session(getAWSRegion())

	_, err := manager.ChangeResourceRecordSets(&r53SyncSet)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case route53.ErrCodeNoSuchHostedZone:
				log.Println(route53.ErrCodeNoSuchHostedZone, aerr.Error())
			case route53.ErrCodeNoSuchHealthCheck:
				log.Println(route53.ErrCodeNoSuchHealthCheck, aerr.Error())
			case route53.ErrCodeInvalidChangeBatch:
				log.Println(route53.ErrCodeInvalidChangeBatch, aerr.Error())
			case route53.ErrCodeInvalidInput:
				log.Println(route53.ErrCodeInvalidInput, aerr.Error())
			case route53.ErrCodePriorRequestNotComplete:
				log.Println(route53.ErrCodePriorRequestNotComplete, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
	}

	return err
}

func(a AWSDNSAPI) UpdateDNSRecords(triageInput map[string]model.IPTriageSummary,
							awsHI AwsHelperInterface) error {
	var r53DNSList []dnsStruct

	for _, triage := range triageInput {
		if triage.Result != model.IPTriageNoChange {
			if isDNSLengthOK(triage.HttpEntry) {
				r53DNSList = append(
					r53DNSList,
					dnsStruct{
						dnsName:   getAWSAName(triage.HttpEntry),
						dnsAction: getAWSAction(triage.Result),
						dnsIP:     triage.VmwIP,
						dnsOldIP:  triage.R53IP,
					},
				)
			} else {
				log.Printf("%s is too long. AWS will reject this - so let us instead\n", triage.HttpEntry)
			}
		}
	}

	if len(r53DNSList) == 0 {
		log.Println("No action encountered after processing. All records seem to be in sync. Exiting without action")
		return fmt.Errorf("DNS000: No action to take")
	}

	errorPresent := false
	updatePairs := getBatchPairs(len(r53DNSList), getUpdateBatchSize())

	for i, eachPair := range updatePairs {
		log.Printf("Set %d, Start Range %d, End Range %d\n", i + 1, eachPair.start, eachPair.end - 1)
		updatableRecordSet := getR53UpdateSet(r53DNSList[eachPair.start:eachPair.end])
		err := awsHI.UpdateRoute53RecordSets(updatableRecordSet)
		if err != nil {
			errorPresent = true
			log.Println("Last set sync failed. Error was logged already. Not panicking...")
		} else {
			log.Println("Last set was successful.")
		}
	}

	if errorPresent {
		return fmt.Errorf("DNS001: At least one set of updates failed")
	}

	return nil
}


