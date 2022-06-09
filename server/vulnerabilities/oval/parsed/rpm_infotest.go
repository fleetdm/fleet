package oval_parsed

type RpmInfoTest struct {
	Objects       []string
	States        []ObjectInfoState
	StateOperator OperatorType
	ObjectMatch   ObjectMatchType
	StateMatch    StateMatchType
}
