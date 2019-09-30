package utils

import (
	"net"
	"strconv"
	"strings"
)

//ValidPort checks if the port given as argument is valid.
//A port is valid if it is between 0 and 65535
//It returns a boolean depending on the validity of the port.
func ValidPort(port string) bool {
	portInt, err := strconv.Atoi(port)
	return err == nil && 0 < portInt && portInt < 65535
}

//ValidIPv4 checks if the given string is a valid IPv4 address
func ValidIPv4(ip string) bool {
	valid := net.ParseIP(ip)
	return valid.To4() != nil
}

//ValidPortIP check if the given pair which as the form ip:port is valid
func ValidPortIP(pair string) bool {
	splitPair := strings.Split(pair, ":")

	if len(splitPair) != 2 {
		return false
	}

	return ValidIPv4(splitPair[0]) && ValidPort(splitPair[1])

}
