package main

type RequestSeed struct {
	// Request Parameters
	Header   map[string]interface{}
	Param    map[string]interface{}
	Data     map[string]interface{}
	JsonData string
}

type MethodSelector struct {
	// for display html
	Method   string
	Selected bool
}

type TeamSelector struct {
	// for display html
	Team     string
	Selected bool
}
