package triage

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/stretchr/testify/assert"
	"vmc-dns-sync/pkg/dns_api"
	"vmc-dns-sync/pkg/model"
	"os"
	"strings"
	"testing"
)

var updateDNSRecordsMock func(triageInput map[string]model.IPTriageSummary,
		awsHI dns_api.AwsHelperInterface) error
var updateRoute53RecordSetsMock func(r53SyncSet route53.ChangeResourceRecordSetsInput) error

type awsInterfaceTest struct {
}

func(a awsInterfaceTest) UpdateDNSRecords(triageInput map[string]model.IPTriageSummary, awsHI dns_api.AwsHelperInterface) error {
	return updateDNSRecordsMock(triageInput, awsHI)
}

func (a awsInterfaceTest) UpdateRoute53RecordSets(r53SyncSet route53.ChangeResourceRecordSetsInput) error {
	return updateRoute53RecordSetsMock(r53SyncSet)
}

func TestDefaultIsNotDryRun(t *testing.T) {
	os.Setenv("R53_UPDATE_DRY_RUN", "")
	assert.True(t, isDryRun())
	os.Setenv("R53_UPDATE_DRY_RUN", "some-invalid-value")
	assert.True(t, isDryRun())
	os.Setenv("R53_UPDATE_DRY_RUN", "true")
	assert.True(t, isDryRun())
	os.Setenv("R53_UPDATE_DRY_RUN", "FALSE")
	assert.False(t, isDryRun())
}

func TestIPv6(t *testing.T) {
	assert.False(t, isVMCv6("10.4.190.32"))
	assert.False(t, isVMCv6("anything"))
	assert.True(t, isVMCv6("fe80::3c00:2b77:5344:bb55"))
}

func Test_ThreePartMapping(t *testing.T) {
	var vmcMap, k8sMap, awsMap map[string]string

	vmcMap = make(map[string]string)
	k8sMap = make(map[string]string)
	awsMap = make(map[string]string)

	awsMap["aws-only-1"] = "1.2.3.40"
	awsMap["aws-only-2"] = "1.2.3.41"
	awsMap["aws-only-3"] = "1.2.3.42"
	awsMap["aws-vmc-same-1"] = "2.2.3.40"
	awsMap["aws-vmc-same-2"] = "2.2.3.41"
	awsMap["aws-vmc-same-3"] = "2.2.3.42"
	awsMap["aws-vmc-diff-1"] = "3.2.3.40"
	awsMap["aws-vmc-diff-2"] = "3.2.3.41"
	awsMap["aws-vmc-diff-3"] = "3.2.3.42"
	awsMap["vmw-v6-ip"] = "5.2.3.40"
	vmcMap["builder-same-1"] = "2.2.3.40"
	vmcMap["builder-same-2"] = "2.2.3.41"
	vmcMap["builder-same-3"] = "2.2.3.42"
	vmcMap["builder-diff-1"] = "3.2.3.43"
	vmcMap["builder-diff-2"] = "3.2.3.44"
	vmcMap["builder-diff-3"] = "3.2.3.45"
	vmcMap["vmc-only-1"] = "4.2.3.40"
	vmcMap["vmc-only-2"] = "4.2.3.41"
	vmcMap["vmc-only-3"] = "4.2.3.42"
	vmcMap["builder-v6"] = "fe80::3c00:2b77:5344:bb55"
	vmcMap["builder-v6-2"] = "fe80::3d00:2b77:5344:bb55"
	k8sMap["aws-vmc-same-1"] = "builder-same-1"
	k8sMap["aws-vmc-same-2"] = "builder-same-2"
	k8sMap["aws-vmc-same-3"] = "builder-same-3"
	k8sMap["aws-vmc-diff-1"] = "builder-diff-1"
	k8sMap["aws-vmc-diff-2"] = "builder-diff-2"
	k8sMap["aws-vmc-diff-3"] = "builder-diff-3"
	k8sMap["missing-host-1"] = "vmc-only-1"
	k8sMap["missing-host-2"] = "vmc-only-2"
	k8sMap["missing-host-3"] = "vmc-only-3"
	k8sMap["vmw-v6-ip"] = "builder-v6"
	k8sMap["vmw-v6-ip-2"] = "builder-v6-2"

	result := IPTriage(
			vmcMap, awsMap, k8sMap,
		)

	assert.Equal(t, result["missing-host-1"].R53IP, "")
	assert.Equal(t, result["missing-host-2"].R53IP, "")
	assert.Equal(t, result["missing-host-3"].R53IP, "")

	assert.Equal(t, result["missing-host-1"].VmwIP, "4.2.3.40")
	assert.Equal(t, result["missing-host-2"].VmwIP, "4.2.3.41")
	assert.Equal(t, result["missing-host-3"].VmwIP, "4.2.3.42")

	assert.Equal(t, result["missing-host-1"].Source, model.IPTriageSourceVMW)
	assert.Equal(t, result["missing-host-2"].Source, model.IPTriageSourceVMW)
	assert.Equal(t, result["missing-host-3"].Source, model.IPTriageSourceVMW)

	assert.Equal(t, result["missing-host-1"].Result, model.IPTriageAddR53)
	assert.Equal(t, result["missing-host-2"].Result, model.IPTriageAddR53)
	assert.Equal(t, result["missing-host-3"].Result, model.IPTriageAddR53)

	assert.Equal(t, result["aws-vmc-same-1"].R53IP, result["aws-vmc-same-1"].VmwIP)
	assert.Equal(t, result["aws-vmc-same-2"].R53IP, result["aws-vmc-same-2"].VmwIP)
	assert.Equal(t, result["aws-vmc-same-3"].R53IP, result["aws-vmc-same-3"].VmwIP)

	assert.Equal(t, result["aws-vmc-same-1"].R53IP, "2.2.3.40")
	assert.Equal(t, result["aws-vmc-same-2"].R53IP, "2.2.3.41")
	assert.Equal(t, result["aws-vmc-same-3"].R53IP, "2.2.3.42")

	assert.Equal(t, result["aws-vmc-same-1"].Source, model.IPTriageSourceBoth)
	assert.Equal(t, result["aws-vmc-same-2"].Source, model.IPTriageSourceBoth)
	assert.Equal(t, result["aws-vmc-same-3"].Source, model.IPTriageSourceBoth)

	assert.Equal(t, result["aws-vmc-same-1"].Result, model.IPTriageNoChange)
	assert.Equal(t, result["aws-vmc-same-2"].Result, model.IPTriageNoChange)
	assert.Equal(t, result["aws-vmc-same-3"].Result, model.IPTriageNoChange)

	assert.NotEqual(t, result["aws-vmc-diff-1"].R53IP, result["aws-vmc-diff-1"].VmwIP)
	assert.NotEqual(t, result["aws-vmc-diff-2"].R53IP, result["aws-vmc-diff-2"].VmwIP)
	assert.NotEqual(t, result["aws-vmc-diff-3"].R53IP, result["aws-vmc-diff-3"].VmwIP)

	assert.Equal(t, result["aws-vmc-diff-1"].R53IP, "3.2.3.40")
	assert.Equal(t, result["aws-vmc-diff-2"].R53IP, "3.2.3.41")
	assert.Equal(t, result["aws-vmc-diff-3"].R53IP, "3.2.3.42")

	assert.Equal(t, result["aws-vmc-diff-1"].VmwIP, "3.2.3.43")
	assert.Equal(t, result["aws-vmc-diff-2"].VmwIP, "3.2.3.44")
	assert.Equal(t, result["aws-vmc-diff-3"].VmwIP, "3.2.3.45")

	assert.Equal(t, result["aws-vmc-diff-1"].Source, model.IPTriageSourceBoth)
	assert.Equal(t, result["aws-vmc-diff-2"].Source, model.IPTriageSourceBoth)
	assert.Equal(t, result["aws-vmc-diff-3"].Source, model.IPTriageSourceBoth)

	assert.Equal(t, result["aws-vmc-diff-1"].Result, model.IPTriageUpdateR53)
	assert.Equal(t, result["aws-vmc-diff-2"].Result, model.IPTriageUpdateR53)
	assert.Equal(t, result["aws-vmc-diff-3"].Result, model.IPTriageUpdateR53)

	assert.Equal(t, result["aws-only-1"].VmwIP, "")
	assert.Equal(t, result["aws-only-2"].VmwIP, "")
	assert.Equal(t, result["aws-only-3"].VmwIP, "")

	assert.Equal(t, result["aws-only-1"].R53IP, "1.2.3.40")
	assert.Equal(t, result["aws-only-2"].R53IP, "1.2.3.41")
	assert.Equal(t, result["aws-only-3"].R53IP, "1.2.3.42")

	assert.Equal(t, result["aws-only-1"].Source, model.IPTriageSourceR53)
	assert.Equal(t, result["aws-only-2"].Source, model.IPTriageSourceR53)
	assert.Equal(t, result["aws-only-3"].Source, model.IPTriageSourceR53)

	assert.Equal(t, result["aws-only-1"].Result, model.IPTriageDeleteR53)
	assert.Equal(t, result["aws-only-2"].Result, model.IPTriageDeleteR53)
	assert.Equal(t, result["aws-only-3"].Result, model.IPTriageDeleteR53)

	assert.Equal(t, result["vmw-v6-ip"].R53IP, "5.2.3.40")
	assert.Equal(t, result["vmw-v6-ip"].VmwIP, "fe80::3c00:2b77:5344:bb55")
	assert.Equal(t, result["vmw-v6-ip"].Result, model.IPTriageDeleteR53)

	assert.Equal(t, result["vmw-v6-ip-2"].R53IP, "")
	assert.Equal(t, result["vmw-v6-ip-2"].VmwIP, "fe80::3d00:2b77:5344:bb55")
	assert.Equal(t, result["vmw-v6-ip-2"].Result, model.IPTriageNoChange)
}

func TestDetailedAWSFlow_DryRun(t *testing.T) {
	var a awsInterfaceTest
	triageResult := make(map[string]model.IPTriageSummary)

	os.Setenv("R53_UPDATE_DRY_RUN", "TRUE")
	result := SyncRoute53(triageResult, a)
	assert.True(t, strings.HasPrefix(result.Error(), "DNS002"))
}

func TestDetailedAWSFlow_NonDryRun(t *testing.T) {
	var a awsInterfaceTest
	var ao dns_api.AWSDNSAPI
	triageResult := make(map[string]model.IPTriageSummary)

	os.Setenv("R53_UPDATE_DRY_RUN", "FALSE")
	updateDNSRecordsMock = ao.UpdateDNSRecords
	updateRoute53RecordSetsMock = ao.UpdateRoute53RecordSets
	result := SyncRoute53(triageResult, a)
	assert.True(t, strings.HasPrefix(result.Error(), "DNS000"))
}

func TestDetailedAWSFlow_NoChange(t *testing.T) {
	var a awsInterfaceTest
	var ao dns_api.AWSDNSAPI
	triageResult := make(map[string]model.IPTriageSummary)

	os.Setenv("R53_UPDATE_DRY_RUN", "FALSE")
	updateDNSRecordsMock = ao.UpdateDNSRecords
	updateRoute53RecordSetsMock = func(r53SyncSet route53.ChangeResourceRecordSetsInput) error {
		return nil
	}
	triageResult["sample-domain-1"] = model.IPTriageSummary{
		Result: model.IPTriageNoChange,
	}
	triageResult["sample-domain-2"] = model.IPTriageSummary{
		Result: model.IPTriageNoChange,
	}
	result := SyncRoute53(triageResult, a)
	assert.True(t, strings.HasPrefix(result.Error(), "DNS000"))
}

func TestDetailedAWSFlow_NoError(t *testing.T) {
	var a awsInterfaceTest
	var ao dns_api.AWSDNSAPI
	triageResult := make(map[string]model.IPTriageSummary)

	os.Setenv("R53_UPDATE_DRY_RUN", "FALSE")
	updateDNSRecordsMock = ao.UpdateDNSRecords
	updateRoute53RecordSetsMock = func(r53SyncSet route53.ChangeResourceRecordSetsInput) error {
		return nil
	}
	triageResult["sample-domain-1"] = model.IPTriageSummary{
		Result: model.IPTriageUpdateR53,
	}
	triageResult["sample-domain-2"] = model.IPTriageSummary{
		Result: model.IPTriageDeleteR53,
	}

	result := SyncRoute53(triageResult, a)
	assert.True(t, result == nil)
}

func TestDetailedAWSFlow_R53Error(t *testing.T) {
	var a awsInterfaceTest
	var ao dns_api.AWSDNSAPI
	triageResult := make(map[string]model.IPTriageSummary)

	os.Setenv("R53_UPDATE_DRY_RUN", "FALSE")
	updateDNSRecordsMock = ao.UpdateDNSRecords
	updateRoute53RecordSetsMock = func(r53SyncSet route53.ChangeResourceRecordSetsInput) error {
		return fmt.Errorf("ok, I raised an error")
	}
	triageResult["sample-domain-1"] = model.IPTriageSummary{
		Result: model.IPTriageUpdateR53,
	}
	triageResult["sample-domain-2"] = model.IPTriageSummary{
		Result: model.IPTriageDeleteR53,
	}

	result := SyncRoute53(triageResult, a)
	assert.True(t, strings.HasPrefix(result.Error(), "DNS001"))
}