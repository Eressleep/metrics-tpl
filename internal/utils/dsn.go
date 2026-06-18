package utils

import (
	"fmt"
	"strings"
)

func MaskDSN(dsn string) string {
	if dsn == "" {
		return ""
	}

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		if idx := strings.Index(dsn, "@"); idx > 0 {
			if protoIdx := strings.Index(dsn, "://"); protoIdx > 0 {
				protocol := dsn[:protoIdx+3]
				rest := dsn[protoIdx+3:]

				if atIdx := strings.Index(rest, "@"); atIdx > 0 {
					credPart := rest[:atIdx]
					hostPart := rest[atIdx+1:]

					if colonIdx := strings.Index(credPart, ":"); colonIdx > 0 {
						user := credPart[:colonIdx]
						return fmt.Sprintf("%s%s:***@%s", protocol, user, hostPart)
					}

					return fmt.Sprintf("%s***@%s", protocol, hostPart)
				}
			}
		}

		return "postgres://***:***@***/***"
	}

	return "***:***@***"
}
