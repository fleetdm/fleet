package fleet

// ScriptResult is a struct that represents the result of a script execution.
type ScriptResult struct {
	ScriptContents string `json:"script_contents"`
	ExitCode       int    `json:"exit_code"`
	Output         string `json:"output"`
	Message        string `json:"message"`
	Runtime        uint   `json:"runtime"`
}
