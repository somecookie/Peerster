package helper

import "fmt"

//--------------------------------------------------------------------------------------
type NetworkError struct {
	errorMessage string
	location string
}

func (e *NetworkError) Error() string{
	return fmt.Sprintf("Error in %s NetworkError: %s\n", e.location, e.location)
}
//-------------------------------------------------------------------------------------
