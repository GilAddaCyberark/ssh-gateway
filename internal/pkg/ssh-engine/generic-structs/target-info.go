package generic_structs

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	guuid "github.com/google/uuid"
)

const AWS_PROVIDER_PREFIX = "aws"

type TargetInfo struct {
	TargetUser     string
	TargetPass     string
	TargetAddress  string
	TargetPort     int
	TargetProvider string
	TargetId       string
	AuthType       string
	SessionId      string
}

func GetTargetInfoByConnectionString(user string) (*TargetInfo, error) {

	var err error

	tarInfo := new(TargetInfo)
	tarInfo.SessionId = guuid.New().String()
	tarInfo.TargetPort = 22
	tarInfo.TargetId = ""
	tarInfo.TargetProvider = ""

	parts := strings.Split(user, "@")
	if len(parts) < 3 {
		return nil, errors.New("Unsupported user format: cannot be parsed into personal user, target, port")
	}

	tarInfo.TargetUser = parts[1]
	tarInfo.TargetAddress = parts[2]
	tarInfo.AuthType = "pass"

	// Handle Address
	if len(parts[2]) > 0 {
		// Split to port and address
		if strings.Contains(tarInfo.TargetAddress, ":") {
			portParts := strings.Split(parts[2], ":")
			tarInfo.TargetAddress = portParts[0]
			tarInfo.TargetPort, err = strconv.Atoi(portParts[1])
			if err != nil {
				return nil, err
			}
		}
		// Set Provider (Amazon, gcp, azure) , Remove  Prefix from target address
		instanceParts := strings.Split(tarInfo.TargetAddress, "#")
		if len(instanceParts) > 1 {
			if instanceParts[0] == AWS_PROVIDER_PREFIX {
				tarInfo.TargetAddress = instanceParts[1]
				tarInfo.TargetProvider = AWS_PROVIDER_PREFIX
			} else {
				return nil, fmt.Errorf("Unidentofied cloud provider %s", instanceParts[0])
			}
		}

		// Handle AWS Instance id or an IP Address
		if strings.HasPrefix(tarInfo.TargetAddress, "i-") {
			tarInfo.TargetId = tarInfo.TargetAddress
			tarInfo.TargetAddress = ""
			tarInfo.AuthType = "cert"
		}
	}

	if len(parts) > 3 {
		tarInfo.TargetPort, err = strconv.Atoi(parts[3])
		if err != nil {
			return nil, errors.New("Part of port exists, cannot be splited to port")
		}
	}
	return tarInfo, nil
}
