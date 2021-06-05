package model

const (
	IPTriageNoChange  = iota
	IPTriageDeleteR53 = iota
	IPTriageUpdateR53 = iota
	IPTriageAddR53    = iota
)

const (
	IPTriageSourceR53  = iota
	IPTriageSourceVMW  = iota
	IPTriageSourceBoth = iota
)

type IPTriageSummary struct {
	HttpEntry string
	R53IP     string
	VmwIP     string
	Source    int
	Result    int
}

