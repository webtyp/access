//go:build wasm

package panel

import (
	"webtyp.com/dom"
	"webtyp.com/fetch"
	"webtyp.com/html"
	"webtyp.com/json"
	"webtyp.com/time"
	"github.com/veltylabs/iam/config"
)

func wireAudit() {
	loadAuditList()
}

func loadAuditList() {
	Fetch(fetch.Get(config.PathAdminAudit), func(resp *fetch.Response, err error) {
		if err != nil || resp.Status != 200 {
			setContainerText(IDAuditList, "Error al cargar la auditoría.")
			return
		}
		var auditResp config.AdminAuditResponse
		if err := json.Decode(resp.Body(), &auditResp); err != nil {
			setContainerText(IDAuditList, "Error decodificando la auditoría.")
			return
		}

		table := html.Table().Class("iam-table").Child(
			html.Thead().Child(
				html.Tr().Child(
					html.Th().Text("Fecha"),
					html.Th().Text("Actor"),
					html.Th().Text("Acción"),
					html.Th().Text("Objetivo"),
					html.Th().Text("Detalle"),
				),
			),
		)
		tbody := html.Tbody()
		for _, e := range auditResp.Entries {
			row := html.Tr().Child(
				html.Td().Text(formatEpoch(time.Unix(e.CreatedAt))),
				html.Td().Text(e.ActorEmail),
				html.Td().Text(e.Action),
				html.Td().Text(e.Target),
				html.Td().Text(e.Detail),
			)
			tbody.Child(row)
		}
		table.Child(tbody)
		_ = dom.Render(IDAuditList, table)
	})
}
