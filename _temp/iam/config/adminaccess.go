package config

import (
	"webtyp.com/env"
	"webtyp.com/fmt"
)

const EnvAdminEmails = "IAM_ADMIN_EMAILS"

// PanelAdminList parsea EnvAdminEmails (por coma, sin vacíos ni espacios). nil si no está.
func PanelAdminList() []string {
	val := env.Get(EnvAdminEmails)
	if val == "" {
		return nil
	}
	raw := fmt.Split(val, ",")
	var list []string
	for _, item := range raw {
		trimmed := fmt.TrimSpace(item)
		if trimmed != "" {
			list = append(list, trimmed)
		}
	}
	if len(list) == 0 {
		return nil
	}
	return list
}
