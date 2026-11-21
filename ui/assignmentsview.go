package ui

import (
	"webtyp.com/access"
	"webtyp.com/crud"
	. "webtyp.com/dom"
	. "webtyp.com/html"
	"webtyp.com/lang"
	"webtyp.com/model"
	"webtyp.com/router"
	"webtyp.com/widget"
)

// NameAssignmentsView is the widget identity of the multi-role assignments screen.
const NameAssignmentsView = widget.Name("accessassignments")

const (
	PartAssignHeader = widget.Part("header")
	PartAssignList   = widget.Part("list")
	PartAssignCard   = widget.Part("card")
	PartAssignPerson = widget.Part("person")
	PartAssignEmail  = widget.Part("email")
	PartAssignChips  = widget.Part("chips")
	PartAssignChip   = widget.Part("chip")
	PartAssignToggle = widget.Part("toggle")
	PartAssignInvite = widget.Part("invite")
	PartAssignInput  = widget.Part("input")
	PartAssignBtn    = widget.Part("btn")
	PartAssignError  = widget.Part("error")
)

var (
	clsAssignRoot   = NameAssignmentsView.Root()
	clsAssignHeader = NameAssignmentsView.Class(PartAssignHeader)
	clsAssignList   = NameAssignmentsView.Class(PartAssignList)
	clsAssignCard   = NameAssignmentsView.Class(PartAssignCard)
	clsAssignPerson = NameAssignmentsView.Class(PartAssignPerson)
	clsAssignEmail  = NameAssignmentsView.Class(PartAssignEmail)
	clsAssignChips  = NameAssignmentsView.Class(PartAssignChips)
	clsAssignChip   = NameAssignmentsView.Class(PartAssignChip)
	clsAssignToggle = NameAssignmentsView.Class(PartAssignToggle)
	clsAssignInvite = NameAssignmentsView.Class(PartAssignInvite)
	clsAssignInput  = NameAssignmentsView.Class(PartAssignInput)
	clsAssignBtn    = NameAssignmentsView.Class(PartAssignBtn)
	clsAssignError  = NameAssignmentsView.Class(PartAssignError)
)

const (
	textAssignSearch  lang.Text = "Filter by person..."
	textInviteEmail   lang.Text = "Invite by e-mail"
	textInviteAction  lang.Text = "Assign"
	textNoRoles       lang.Text = "No roles assigned"
	textPickRole      lang.Text = "Select role"
)

// AssignmentsView lists one row per person with coloured chips for all their roles
// and lets an administrator toggle multiple roles per person directly.
type AssignmentsView struct {
	Element
	caller router.Caller

	people      []*access.Person
	roles       []*access.RoleRecord
	assignments []*access.Assignment

	search      *SignalString
	expandedID  *SignalString
	inviteEmail *SignalString
	inviteRole  *SignalString
	errMsg      *SignalString
	body        *SignalNodes
	roleSelect  *SignalNodes
}

func (v *AssignmentsView) WidgetName() widget.Name { return NameAssignmentsView }
func (v *AssignmentsView) WidgetKind() widget.Kind { return widget.Region }

// NewAssignmentsView creates the multi-role assignments screen component.
func NewAssignmentsView(caller router.Caller) Component {
	return &AssignmentsView{
		caller:      caller,
		search:      NewString(""),
		expandedID:  NewString(""),
		inviteEmail: NewString(""),
		inviteRole:  NewString(""),
		errMsg:      NewString(""),
		body:        NewNodes(),
		roleSelect:  NewNodes(),
	}
}

func (v *AssignmentsView) Init(_ Ctx) {
	v.reload()
}

func (v *AssignmentsView) reload() {
	v.errMsg.Set("")
	var people access.PersonList
	var roles access.RoleRecordList
	var assigns access.AssignmentList
	pending := 3

	done := func(err error) {
		if err != nil {
			v.errMsg.Set(err.Error())
		}
		pending--
		if pending == 0 {
			v.people = people
			v.roles = roles
			v.assignments = assigns
			v.renderRoleOptions()
			v.rebuildList()
		}
	}

	readAll := func(entity string, into model.Decodable) {
		op := router.Operation{Module: access.ModelName, Entity: entity, Action: model.Read}
		v.caller.Call(op, &crud.ReadArgs{Limit: 500}, into, done)
	}
	readAll((&access.Person{}).ModelName(), &people)
	readAll((&access.RoleRecord{}).ModelName(), &roles)
	readAll((&access.Assignment{}).ModelName(), &assigns)
}

func (v *AssignmentsView) renderRoleOptions() {
	opts := make([]*Element, 0, len(v.roles)+1)
	opts = append(opts, Option("", lang.Translate(textPickRole).String()))
	for _, r := range v.roles {
		opts = append(opts, Option(r.Code, r.Name))
	}
	v.roleSelect.Set(opts)
}

func (v *AssignmentsView) toneFor(roleCode string) string {
	for _, r := range v.roles {
		if r.Code == roleCode && r.Tone != "" {
			return r.Tone
		}
	}
	return ""
}

func toneCSSValue(tone string) string {
	if tone == "" {
		return ""
	}
	if tone[0] == '#' {
		return tone
	}
	return "var(--palette-" + tone + ")"
}

func (v *AssignmentsView) roleNameFor(roleCode string) string {
	for _, r := range v.roles {
		if r.Code == roleCode && r.Name != "" {
			return r.Name
		}
	}
	return roleCode
}

func (v *AssignmentsView) assignmentID(userID, roleCode string) string {
	for _, a := range v.assignments {
		if a.UserId == userID && a.RoleCode == roleCode {
			return a.Id
		}
	}
	return ""
}

func (v *AssignmentsView) userRoles(userID string) []string {
	var out []string
	for _, a := range v.assignments {
		if a.UserId == userID {
			out = append(out, a.RoleCode)
		}
	}
	return out
}

func (v *AssignmentsView) toggleRole(userID, roleCode string) {
	v.errMsg.Set("")
	if id := v.assignmentID(userID, roleCode); id != "" {
		op := router.Operation{Module: access.ModelName, Entity: (&access.Assignment{}).ModelName(), Action: model.Delete}
		v.caller.Call(op, &crud.DeleteArgs{IDs: []string{id}}, nil, func(err error) {
			if err != nil {
				v.errMsg.Set(err.Error())
				return
			}
			v.reload()
		})
		return
	}
	rec := &access.Assignment{UserId: userID, RoleCode: roleCode}
	op := router.Operation{Module: access.ModelName, Entity: rec.ModelName(), Action: model.Create}
	v.caller.Call(op, &crud.CreateArgs{Records: []model.Model{rec}}, nil, func(err error) {
		if err != nil {
			v.errMsg.Set(err.Error())
			return
		}
		v.reload()
	})
}

func (v *AssignmentsView) inviteByEmail() {
	email := v.inviteEmail.Get()
	role := v.inviteRole.Get()
	if email == "" || role == "" {
		return
	}
	v.errMsg.Set("")
	rec := &access.Assignment{Email: email, RoleCode: role}
	op := router.Operation{Module: access.ModelName, Entity: rec.ModelName(), Action: model.Create}
	v.caller.Call(op, &crud.CreateArgs{Records: []model.Model{rec}}, nil, func(err error) {
		if err != nil {
			v.errMsg.Set(err.Error())
			return
		}
		v.inviteEmail.Set("")
		v.reload()
	})
}

func containsFold(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	lh := toLowerASCII(haystack)
	ln := toLowerASCII(needle)
	if len(ln) > len(lh) {
		return false
	}
	for i := 0; i <= len(lh)-len(ln); i++ {
		if lh[i:i+len(ln)] == ln {
			return true
		}
	}
	return false
}

func toLowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}

func (v *AssignmentsView) rebuildList() {
	q := v.search.Get()
	exp := v.expandedID.Get()
	cards := make([]*Element, 0, len(v.people))

	for _, p := range v.people {
		if !containsFold(p.Name, q) && !containsFold(p.Email, q) {
			continue
		}
		uid := p.Id
		held := v.userRoles(uid)
		isExpanded := exp == uid

		chips := Div().Set(clsAssignChips.AsAttr())
		if len(held) == 0 {
			chips.Child(Span().Set(clsAssignEmail.AsAttr()).Text(lang.Translate(textNoRoles).String()))
		} else {
			for _, rc := range held {
				chip := Span().Set(clsAssignChip.AsAttr()).Text(v.roleNameFor(rc))
				if val := toneCSSValue(v.toneFor(rc)); val != "" {
					chip.Attr("data-toned", "1").BindStyleProp("--tone-color", NewString(val))
				}
				chips.Child(chip)
			}
		}

		header := Div().Set(clsAssignPerson.AsAttr()).
			Child(Strong().Text(p.Name)).
			Child(Span().Set(clsAssignEmail.AsAttr()).Text(p.Email)).
			OnClick(func(Event) {
				if v.expandedID.Get() == uid {
					v.expandedID.Set("")
				} else {
					v.expandedID.Set(uid)
				}
				v.rebuildList()
			})

		heldKey := ""
		for _, rc := range held {
			heldKey += "," + rc
		}
		card := Div().Set(clsAssignCard.AsAttr()).Key("person-" + uid + "-" + boolStr(isExpanded) + heldKey).
			Child(header, chips)

		if isExpanded {
			toggles := Div().Set(clsAssignChips.AsAttr())
			for _, r := range v.roles {
				code := r.Code
				active := v.assignmentID(uid, code) != ""
				btn := Button().
					Attr("type", "button").
					Set(clsAssignToggle.AsAttr()).
					Attr("aria-pressed", boolStr(active)).
					Text(r.Name)
				if active {
					if val := toneCSSValue(v.toneFor(code)); val != "" {
						btn.Attr("data-toned", "1").BindStyleProp("--tone-color", NewString(val))
					}
				}
				btn.OnClick(func(Event) { v.toggleRole(uid, code) })
				toggles.Child(btn)
			}
			card.Child(toggles)
		}

		cards = append(cards, card)
	}

	v.body.Set(cards)
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func (v *AssignmentsView) Render() *Element {
	searchInput := NewElement("input").
		Attr("type", "search").
		Attr("name", "access-assign-search").
		Attr("placeholder", lang.Translate(textAssignSearch).String()).
		Set(clsAssignInput.AsAttr()).
		Bind(v.search).
		OnInput(func(e Event) {
			v.search.Set(e.TargetValue())
			v.rebuildList()
		})

	emailInput := NewElement("input").
		Attr("type", "email").
		Attr("name", "access-invite-email").
		Attr("placeholder", lang.Translate(textInviteEmail).String()).
		Set(clsAssignInput.AsAttr()).
		Bind(v.inviteEmail).
		OnInput(func(e Event) { v.inviteEmail.Set(e.TargetValue()) })

	roleSel := NewElement("select").
		Attr("name", "access-invite-role").
		Set(clsAssignInput.AsAttr()).
		Bind(v.inviteRole).
		OnChange(func(e Event) { v.inviteRole.Set(e.TargetValue()) }).
		BindChildren(v.roleSelect)

	inviteBtn := Button().
		Attr("type", "button").
		Set(clsAssignBtn.AsAttr()).
		Text(lang.Translate(textInviteAction).String()).
		OnClick(func(Event) { v.inviteByEmail() })

	inviteBar := Div().Set(clsAssignInvite.AsAttr()).
		Child(emailInput, roleSel, inviteBtn)

	errLine := Show(DeriveBool(func() bool { return v.errMsg.Get() != "" }), func() *Element {
		return P().Set(clsAssignError.AsAttr()).BindText(v.errMsg)
	})

	header := Div().Set(clsAssignHeader.AsAttr()).Child(searchInput, inviteBar)
	list := Div().Set(clsAssignList.AsAttr()).BindChildren(v.body)

	return Div().Set(clsAssignRoot.AsAttr()).Child(header, errLine, list)
}
