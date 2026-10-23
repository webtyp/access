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
	"github.com/veltylabs/iam/config"
)

var currentSelectedProjectID string

func wireRoles() {
	loadProjectsSelect()

	ref, ok := dom.Get(IDRolesProject)
	if ok {
		ref.OnChange(func(event dom.Event) {
			currentSelectedProjectID = ref.Value()
			loadRolesViewForProject(currentSelectedProjectID)
		})
	}
}

func loadProjectsSelect() {
	Fetch(fetch.Get(config.PathAdminProjects), func(resp *fetch.Response, err error) {
		if err != nil || resp.Status != 200 {
			return
		}
		var projResp config.AdminProjectsResponse
		if err := json.Decode(resp.Body(), &projResp); err != nil {
			return
		}

		if len(projResp.Projects) == 0 {
			_ = dom.Render(IDRolesProject, html.Option("", "-- Sin proyectos --"))
			currentSelectedProjectID = ""
			loadRolesViewForProject("")
			return
		}

		var opts []dom.Component
		for i, p := range projResp.Projects {
			opts = append(opts, html.Option(p.Id, p.Name+" ("+p.Id+")"))
			if i == 0 {
				currentSelectedProjectID = p.Id
			}
		}
		_ = dom.Render(IDRolesProject, html.Div().Child(opts...))
		ref, ok := dom.Get(IDRolesProject)
		if ok {
			ref.SetValue(currentSelectedProjectID)
		}
		loadRolesViewForProject(currentSelectedProjectID)
	})
}

func loadRolesViewForProject(projectID string) {
	if projectID == "" {
		setContainerText(IDRolesNew, "")
		setContainerText(IDRolesList, "Seleccioná un proyecto para administrar sus roles.")
		return
	}

	idGen, _ := config.NewIDs()
	req := &config.AdminCreateRoleRequest{ProjectId: projectID}
	f, err := form.New(IDRolesNew, req, idGen)
	if err == nil && f != nil {
		f.SubmitLabel("Crear rol")
		f.OnSubmit(func(data model.Fielder, done func(error)) {
			rReq, ok := data.(*config.AdminCreateRoleRequest)
			if !ok || fmt.TrimSpace(rReq.Code) == "" {
				done(fmt.Err("Role code required"))
				return
			}
			rReq.ProjectId = projectID
			var buf []byte
			if err := json.Encode(rReq, &buf); err != nil {
				done(err)
				return
			}
			Fetch(fetch.Post(config.PathAdminRoles).ContentTypeJSON().Body(buf), func(resp *fetch.Response, err error) {
				if err != nil || resp.Status != 201 {
					done(fmt.Err("Error creating role"))
					return
				}
				done(nil)
				loadRolesList(projectID)
			})
		})
		_ = dom.Render(IDRolesNew, f)
	}

	loadRolesList(projectID)
}

func loadRolesList(projectID string) {
	url := config.PathAdminRoles + "?project_id=" + projectID
	Fetch(fetch.Get(url), func(resp *fetch.Response, err error) {
		if err != nil || resp.Status != 200 {
			setContainerText(IDRolesList, "Error al cargar la lista de roles.")
			return
		}
		var rolesResp config.AdminRolesResponse
		if err := json.Decode(resp.Body(), &rolesResp); err != nil {
			setContainerText(IDRolesList, "Error decodificando los roles.")
			return
		}

		table := html.Table().Class("iam-table").Child(
			html.Thead().Child(
				html.Tr().Child(
					html.Th().Text("Código"),
					html.Th().Text("Nombre"),
					html.Th().Text("Descripción"),
					html.Th().Text("TTL de Sesión (seg)"),
					html.Th().Text("Usuarios"),
					html.Th().Text("Acciones"),
				),
			),
		)
		tbody := html.Tbody()
		for _, r := range rolesResp.Roles {
			rCode := r.Code
			rTTL := r.SessionTtl

			ttlInputID := "ttl-" + rCode
			ttlInput := html.Input("number").ID(ttlInputID).Attr("value", fmt.Sprintf("%d", rTTL)).Class("iam-input-sm")
			btnTTL := html.Button().Text("Guardar TTL")
			btnTTL.OnClick(func(event dom.Event) {
				ref, ok := dom.Get(ttlInputID)
				val := ""
				if ok {
					val = ref.Value()
				}
				var newTTL int64
				fmt.Sscanf(val, "%d", &newTTL)
				ttlReq := &config.AdminRoleTTLRequest{ProjectId: projectID, Code: rCode, SessionTtl: newTTL}
				var buf []byte
				_ = json.Encode(ttlReq, &buf)
				Fetch(fetch.Post(config.PathAdminRoleTTL).ContentTypeJSON().Body(buf), func(res *fetch.Response, err error) {
					if err == nil && res.Status == 200 {
						loadRolesList(projectID)
					}
				})
			})

			btnDelete := html.Button().Text("Borrar").Class("iam-btn-danger")
			btnDelete.OnClick(func(event dom.Event) {
				refReq := &config.AdminRoleRefRequest{ProjectId: projectID, Code: rCode}
				var buf []byte
				_ = json.Encode(refReq, &buf)
				Fetch(fetch.Post(config.PathAdminRoleDelete).ContentTypeJSON().Body(buf), func(res *fetch.Response, err error) {
					if err == nil && res.Status == 200 {
						loadRolesList(projectID)
					}
				})
			})

			row := html.Tr().Child(
				html.Td().Text(r.Code),
				html.Td().Text(r.Name),
				html.Td().Text(r.Description),
				html.Td().Child(ttlInput, btnTTL),
				html.Td().Text(fmt.Sprintf("%d", r.UserCount)),
				html.Td().Child(btnDelete),
			)
			tbody.Child(row)
		}
		table.Child(tbody)
		_ = dom.Render(IDRolesList, table)
	})
}
