package requests

import (
	"encoding/json"

	"github.com/GoScouter/sdk"
)

const MethodReport = "report"

type ReportRequest struct {
	Namespace sdk.ModuleNamespace `json:"namespace"`
	Data      json.RawMessage     `json:"data"`
}
