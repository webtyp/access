//go:build wasm

package panel

import (
	"webtyp.com/dom"
	"webtyp.com/fetch"
	"webtyp.com/fmt"
	"webtyp.com/form"
	"webtyp.com/html"
	"webtyp.com/json"
	"webtyp.com/model"
	"webtyp.com/time"
	"github.com/veltylabs/iam/config"
)

func wireProjects() {
	loadProjectsList()

	idGen, _ := config.NewIDs()
	f, err := form.New(IDProjectsNew, &config.AdminCreateProjectRequest{}, idGen)
	if err == nil && f != nil {
		f.SubmitLabel("Crear proyecto")
		f.OnSubmit(func(data model.Fielder, done func(error)) {
			req, ok := data.(*config.AdminCreateProjectRequest)
			if !ok || fmt.TrimSpace(req.Name) == "" {
				done(fmt.Err("Invalid name"))
				return
			}
			var buf []byte
			if err := json.Encode(req, &buf); err != nil {
				done(err)
				return
			}
			Fetch(fetch.Post(config.PathAdminProjects).ContentTypeJSON().Body(buf), func(resp *fetch.Response, err error) {
				if err != nil || resp.Status != 201 {
					done(fmt.Err("Failed to create project"))
					return
				}
				done(nil)
				loadProjectsList()
			})
		})
		_ = dom.Render(IDProjectsNew, f)
	}
}

func setContainerText(id, text string) {
	_ = dom.Render(id, html.Div().Text(text))
}

func loadProjectsList() {
	Fetch(fetch.Get(config.PathAdminProjects), func(resp *fetch.Response, err error) {
		if err != nil || resp.Status != 200 {
			setContainerText(IDProjectsList, "Error al cargar la lista de proyectos.")
			return
		}
		var projResp config.AdminProjectsResponse
		if err := json.Decode(resp.Body(), &projResp); err != nil {
			setContainerText(IDProjectsList, "Error decodificando la lista.")
			return
		}

		table := html.Table().Class("iam-table").Child(
			html.Thead().Child(
				html.Tr().Child(
					html.Th().Text("ID"),
					html.Th().Text("Nombre"),
					html.Th().Text("Estado"),
					html.Th().Text("Creado"),
					html.Th().Text("Acciones"),
				),
			),
		)
		tbody := html.Tbody()
		for _, p := range projResp.Projects {
			pID := p.Id
			pActive := p.Active

			statusText := "Inactivo"
			if pActive {
				statusText = "Activo"
			}

			keyInputID := "key-" + pID
			inputKey := html.Input("text").ID(keyInputID).Attr("placeholder", "Nueva clave pública")
			btnKey := html.Button().Text("Cambiar clave pública")
			btnKey.OnClick(func(event dom.Event) {
				ref, ok := dom.Get(keyInputID)
				if !ok || ref.Value() == "" {
					return
				}
				req := &config.AdminProjectKeyRequest{ProjectId: pID, PublicKey: ref.Value()}
				var buf []byte
				_ = json.Encode(req, &buf)
				Fetch(fetch.Post(config.PathAdminProjectKey).ContentTypeJSON().Body(buf), func(r *fetch.Response, err error) {
					if err == nil && r.Status == 200 {
						loadProjectsList()
					}
				})
			})

			toggleText := "Activar"
			if pActive {
				toggleText = "Desactivar"
			}
			btnToggle := html.Button().Text(toggleText)
			btnToggle.OnClick(func(event dom.Event) {
				req := &config.AdminSetActiveRequest{ProjectId: pID, Active: !pActive}
				var buf []byte
				_ = json.Encode(req, &buf)
				Fetch(fetch.Post(config.PathAdminProjectActive).ContentTypeJSON().Body(buf), func(r *fetch.Response, err error) {
					if err == nil && r.Status == 200 {
						loadProjectsList()
					}
				})
			})

			row := html.Tr().Child(
				html.Td().Text(p.Id),
				html.Td().Text(p.Name),
				html.Td().Text(statusText),
				html.Td().Text(formatEpoch(time.Unix(p.CreatedAt))),
				html.Td().Child(inputKey, btnKey, btnToggle),
			)
			tbody.Child(row)
		}
		table.Child(tbody)
		_ = dom.Render(IDProjectsList, table)
	})
}

func formatEpoch(sec time.Unix) string {
	if sec == 0 {
		return "-"
	}
	return time.FormatDateTime(sec)
}
