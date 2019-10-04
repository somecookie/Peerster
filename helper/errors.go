package helper

import (
	"fmt"
	"log"
)

func getErrorString(errorName, where, message string) string {
	return fmt.Sprintf("Error in %s %s: %s\n", where, errorName, message)
}

func HandleCrashingErr(err error){
	if err != nil{
		panic(err)
	}
}

func LogError(err error){
	if err != nil{
		log.Println(err)
	}
}

//IllegalArgumentError is an error that should be thrown when illegal arguments are passed to a function/program.
type IllegalArgumentError struct {
	ErrorMessage string
	Where        string
}

func (e *IllegalArgumentError) Error() string {
	return getErrorString("AddressResolutionError", e.Where, e.ErrorMessage)
}

//-------------------------------------------------------------------------------------
