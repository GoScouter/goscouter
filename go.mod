module goscouter

go 1.26.4

require (
	github.com/GoScouter/sdk v0.0.0-20260712154204-fa4dc6e57c6f
	golang.org/x/term v0.45.0
)

// Local co-development of the SDK. Remove once a version carrying the
// args-aware Scout contract (protocol v2) is published and required above.
replace github.com/GoScouter/sdk => ../sdk

require golang.org/x/sys v0.47.0 // indirect
