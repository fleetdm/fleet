package client

import "fmt"

const sameProfileNameErrorMsg = "Couldn't edit custom_settings. More than one configuration profile have the same name '%s' (PayloadDisplayName for .mobileconfig and file name for .json and .xml)."

func fmtDuplicateNameErrMsg(name string) string {
	return fmt.Sprintf(sameProfileNameErrorMsg, name)
}
