package helper

import "fmt"

func getErrorString(errorName, where, message string) string {
	return fmt.Sprintf("Error in %s %s: %s\n", where, errorName, message)
}

func HandleCrashingErr(err error){
	panic(err)
}

//AddressResolutionError is an error that should be thrown when network address cannot be resolved.
type AddressResolutionError struct {
	errorMessage string
	where        string
}

func (e *AddressResolutionError) Error() string {
	return getErrorString("AddressResolutionError", e.where, e.errorMessage)
}

//-------------------------------------------------------------------------------------
