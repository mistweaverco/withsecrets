package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/mistweaverco/withsecrets/internal/config"
	"github.com/mistweaverco/withsecrets/internal/lib/secrets"
)

type Model struct {
	ctx        context.Context
	configPath string

	cfg       *config.SecretsConfig
	globalCfg *config.GlobalConfig

	screen Screen
	errMsg string

	winW int
	winH int

	envList list.Model

	selectedEnvName string
	selectedEnv     *config.Environment

	secretTable table.Model
	allRows     []secretRow
	maskValues  bool

	filter textinput.Model

	viewValue string

	editTarget *secretRow
	editForm   *huh.Form
	editValue  string
	editSave   bool

	createForm        *huh.Form
	createEnvVar      string
	createSecretKey   string
	createValue       string
	createDesc        string
	createReplication string // "global" | "user-managed"
	createLocations   []string
	createAction      string // "create" | "cancel"
	createSummaryTick int
	createSummaryKey  string

	// For GCP Secret Manager, "regions" are called "locations".
	// gcpLocationsAll is the full supported set (loaded from API).
	// gcpLocations is what we currently show in the create form (may be filtered by defaults).
	gcpLocationsAll []string
	gcpLocations    []string

	confirmText string
	deleteForm  *huh.Form
	deleteYes   bool

	errorForm   *huh.Form
	errorReturn Screen
	errorTitle  string
	errorText   string

	busyText string
	spinner  spinner.Model
}

func (m *Model) sizeFormToModalBody(f *huh.Form) *huh.Form {
	if f == nil {
		return nil
	}
	if m.winW <= 0 || m.winH <= 0 {
		return f
	}
	innerW, innerH := panelInnerSize(m.winW, m.winH, panelStyle())
	// viewModal renders: title + "\n\n" + body
	bodyH := clampMin(innerH-2, 1)
	return f.WithWidth(innerW).WithHeight(bodyH)
}

func New(ctx context.Context, configPath string) (*Model, error) {
	cfg, err := config.LoadSecretsConfig(configPath)
	if err != nil {
		return nil, err
	}

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return nil, err
	}

	envNames := make([]string, 0, len(cfg.Environments))
	for name := range cfg.Environments {
		envNames = append(envNames, name)
	}
	sort.Strings(envNames)

	items := make([]list.Item, 0, len(envNames))
	for _, n := range envNames {
		items = append(items, envItem{name: n})
	}

	delegate := vhsEnvDelegate()
	l := list.New(items, delegate, 0, 0)
	l.Title = "Environments"
	l.SetShowHelp(true)
	l.Styles = vhsListStyles()

	filter := textinput.New()
	filter.Placeholder = "Filter secrets…"
	filter.CharLimit = 256
	filter.Prompt = "/ "
	filter.SetStyles(vhsSecretsFilterStyles())

	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Env Var", Width: 28},
			{Title: "Value", Width: 32},
			{Title: "Provider", Width: 10},
			{Title: "Ref", Width: 28},
		}),
		table.WithRows(nil),
		table.WithFocused(true),
		table.WithStyles(secretsTableStyles()),
	)

	return &Model{
		ctx:               ctx,
		configPath:        configPath,
		cfg:               cfg,
		globalCfg:         globalCfg,
		screen:            screenEnvs,
		envList:           l,
		secretTable:       t,
		maskValues:        true,
		filter:            filter,
		createReplication: "global",
		createAction:      "create",
		spinner:           spinner.New(spinner.WithSpinner(spinner.Line)),
	}, nil
}

func (m *Model) Init() tea.Cmd {
	// Ensure we get an initial WindowSizeMsg even on terminals/platforms where
	// Bubble Tea may only deliver size updates after the first resize.
	return func() tea.Msg { return tea.RequestWindowSize() }
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Capture window size globally so we can size models
	// even when switching screens (Bubble Tea only sends this on resize/start).
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.winW = ws.Width
		m.winH = ws.Height
		innerW, innerH := panelInnerSize(ws.Width, ws.Height, panelStyle())
		// Keep the various lists/tables sized to panel inner area.
		m.envList.SetSize(innerW, innerH)
	}

	switch m.screen {
	case screenEnvs:
		return m.updateEnvs(msg)
	case screenSecrets:
		return m.updateSecrets(msg)
	case screenView:
		return m.updateView(msg)
	case screenEdit:
		return m.updateEdit(msg)
	case screenCreate:
		return m.updateCreate(msg)
	case screenConfirmDelete:
		return m.updateConfirmDelete(msg)
	case screenError:
		return m.updateError(msg)
	case screenBusy:
		return m.updateBusy(msg)
	default:
		return m, nil
	}
}

func (m *Model) View() tea.View {
	// Render "best effort" at any size. We avoid forcing minimum dimensions and
	// instead shrink content to whatever the terminal can actually show.
	switch m.screen {
	case screenEnvs:
		box := fitPanelToWindow(panelStyle(), m.winW, m.winH)
		v := tea.NewView(box.Render(m.envList.View()))
		v.AltScreen = true
		return v
	case screenSecrets:
		v := tea.NewView(m.viewSecrets())
		v.AltScreen = true
		return v
	case screenView:
		v := tea.NewView(m.viewModal("View secret", m.viewValue+"\n\n(esc to go back)"))
		v.AltScreen = true
		return v
	case screenEdit:
		if m.editForm == nil {
			v := tea.NewView(m.viewModal("Edit secret", "Loading…"))
			v.AltScreen = true
			return v
		}
		v := tea.NewView(m.viewModal("Edit secret", m.editForm.View()))
		v.AltScreen = true
		return v
	case screenCreate:
		if m.createForm == nil {
			v := tea.NewView(m.viewModal("Create secret & mapping", "Loading…"))
			v.AltScreen = true
			return v
		}
		v := tea.NewView(m.viewModal("Create secret & mapping", m.createForm.View()))
		v.AltScreen = true
		return v
	case screenConfirmDelete:
		if m.deleteForm == nil {
			v := tea.NewView(m.viewModal("Confirm delete", "Loading…"))
			v.AltScreen = true
			return v
		}
		v := tea.NewView(m.viewModal("Confirm delete", m.deleteForm.View()))
		v.AltScreen = true
		return v
	case screenError:
		if m.errorForm == nil {
			v := tea.NewView(m.viewModal("Error", "Loading…"))
			v.AltScreen = true
			return v
		}
		title := m.errorTitle
		if strings.TrimSpace(title) == "" {
			title = "Error"
		}
		v := tea.NewView(m.viewModal(title, m.errorForm.View()))
		v.AltScreen = true
		return v
	case screenBusy:
		s := m.spinner.View()
		body := strings.TrimSpace(s + " " + m.busyText)
		v := tea.NewView(m.viewModal("Working…", body))
		v.AltScreen = true
		return v
	default:
		v := tea.NewView("")
		v.AltScreen = true
		return v
	}
}

func (m *Model) reloadSecrets() error {
	if m.selectedEnv == nil {
		return fmt.Errorf("no environment selected")
	}

	factory := secrets.NewSecretManagerFactory()
	values, err := factory.GetSecretsForEnvironmentWithCache(m.ctx, m.selectedEnv, m.configPath, m.selectedEnvName)
	if err != nil {
		return err
	}

	items := m.selectedEnv.GetEnvItems()
	rows := make([]secretRow, 0, len(items))
	for _, it := range items {
		provider := it.Provider
		if provider == "" {
			provider = m.selectedEnv.Provider
		}
		project := it.Project
		if project == "" {
			project = m.selectedEnv.Project
		}

		refKind := "value"
		ref := ""
		if it.SecretKey != "" {
			refKind = "secret-key"
			ref = it.SecretKey
		} else if it.SecretPath != "" {
			refKind = "secret-path"
			ref = it.SecretPath
		}

		val := values[it.EnvironmentVariable]
		rows = append(rows, secretRow{
			envVar:   it.EnvironmentVariable,
			value:    val,
			item:     it,
			provider: provider,
			project:  project,
			refKind:  refKind,
			ref:      ref,
		})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].envVar < rows[j].envVar })
	m.allRows = rows
	m.applyFilterToTable()
	return nil
}

func (m *Model) applyFilterToTable() {
	q := strings.TrimSpace(strings.ToLower(m.filter.Value()))

	filtered := make([]secretRow, 0, len(m.allRows))
	for _, r := range m.allRows {
		if q == "" || strings.Contains(strings.ToLower(r.envVar), q) || strings.Contains(strings.ToLower(r.ref), q) {
			filtered = append(filtered, r)
		}
	}

	trows := make([]table.Row, 0, len(filtered))
	for _, r := range filtered {
		val := r.value
		if m.maskValues {
			val = mask(val)
		}
		ref := r.ref
		if ref == "" {
			ref = r.refKind
		} else {
			ref = r.refKind + ":" + ref
		}
		trows = append(trows, table.Row{r.envVar, val, r.provider, ref})
	}
	m.secretTable.SetRows(trows)
}

func mask(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return strings.Repeat("•", len(v))
	}
	return strings.Repeat("•", 8)
}

func (m *Model) viewSecrets() string {
	header := sectionHeaderStyle().Render(fmt.Sprintf("Environment: %s", m.selectedEnvName))
	help := helpStyle().Render("enter:view  e:edit  n:new  d:delete  /:filter  m:mask  esc:back  q:quit")
	if m.errMsg != "" {
		help = errorStyle().Render("Error: "+m.errMsg) + "\n" + help
	}

	filterLine := ""
	if m.filter.Focused() {
		filterLine = m.filter.View()
	} else {
		if v := m.filter.Value(); strings.TrimSpace(v) != "" {
			filterLine = "/ " + v
		} else {
			filterLine = ""
		}
	}

	parts := []string{header}
	if filterLine != "" {
		parts = append(parts, filterLine)
	}
	parts = append(parts, m.secretTable.View(), help)
	content := strings.Join(parts, "\n\n")

	box := fitPanelToWindow(panelStyle(), m.winW, m.winH)
	return box.Render(content)
}

func (m *Model) updateSecrets(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		innerW, innerH := panelInnerSize(msg.Width, msg.Height, panelStyle())

		// Size the table to exactly fill available inner height.
		// viewSecrets uses blocks separated by "\n\n", which contributes one empty
		// line between blocks. So total height is:
		// sum(blockHeights) + (numBlocks-1).
		headerH := lipgloss.Height(sectionHeaderStyle().Render("Environment: X"))
		helpH := lipgloss.Height(helpStyle().Render("enter:view  e:edit  n:new  d:delete  /:filter  m:mask  esc:back  q:quit"))
		filterVisible := m.filter.Focused() || strings.TrimSpace(m.filter.Value()) != ""
		filterH := 0
		if filterVisible {
			filterH = 1 // single-line text input / display
		}
		numBlocks := 3
		if filterVisible {
			numBlocks = 4
		}
		nonTableH := headerH + filterH + helpH + (numBlocks - 1)
		m.secretTable.SetWidth(innerW)
		m.setSecretTableColumns(innerW)
		m.secretTable.SetHeight(clampMin(innerH-nonTableH, 1))
		m.filter.SetWidth(clamp(clampMin(innerW, 1), 1, 60))
		return m, nil
	case tea.KeyMsg:
		// Clear any previous error once the user interacts.
		if m.errMsg != "" {
			m.errMsg = ""
		}
		if m.filter.Focused() {
			var cmd tea.Cmd
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "esc", "enter":
				m.filter.Blur()
				return m, nil
			}
			m.filter, cmd = m.filter.Update(msg)
			m.applyFilterToTable()
			return m, cmd
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.filter.Focused() {
				m.filter.Blur()
				return m, nil
			}
			m.screen = screenEnvs
			return m, nil
		case "/":
			m.filter.Focus()
			return m, nil
		case "m":
			m.maskValues = !m.maskValues
			m.applyFilterToTable()
			return m, nil
		case "enter":
			r, ok := m.selectedRow()
			if !ok {
				return m, nil
			}
			m.viewValue = fmt.Sprintf("%s\n\n%s", r.envVar, r.value)
			m.screen = screenView
			return m, nil
		case "e":
			r, ok := m.selectedRow()
			if !ok {
				return m, nil
			}
			if r.refKind != "secret-key" {
				m.errMsg = "edit is only supported for secret-key mappings"
				return m, nil
			}
			m.editTarget = &r
			m.editValue = r.value
			m.editSave = false
			m.editForm = m.newEditForm()
			m.editForm = m.sizeFormToModalBody(m.editForm)
			m.screen = screenEdit
			return m, m.editForm.Init()
		case "d":
			r, ok := m.selectedRow()
			if !ok {
				return m, nil
			}
			if r.refKind != "secret-key" {
				m.errMsg = "delete is only supported for secret-key mappings"
				return m, nil
			}
			m.editTarget = &r
			m.confirmText = fmt.Sprintf("Delete provider secret '%s'?\n\nEnv var: %s\nProvider: %s", r.ref, r.envVar, r.provider)
			m.deleteYes = false
			m.deleteForm = m.newDeleteForm()
			m.deleteForm = m.sizeFormToModalBody(m.deleteForm)
			m.screen = screenConfirmDelete
			return m, m.deleteForm.Init()
		case "n":
			m.createEnvVar = ""
			m.createSecretKey = ""
			m.createValue = ""
			m.createDesc = ""
			m.createReplication = "global"
			m.createLocations = nil
			m.createAction = "create"
			m.createSummaryTick = 0
			m.createSummaryKey = ""
			// Lazy-load GCP locations for region multiselect.
			if err := m.ensureGCPLocationsLoaded(); err != nil {
				m.errMsg = err.Error()
				return m, nil
			}
			m.applyCreateDefaults()
			m.createForm = m.newCreateForm()
			m.createForm = m.sizeFormToModalBody(m.createForm)
			m.screen = screenCreate
			return m, m.createForm.Init()
		}
	}

	var cmd tea.Cmd
	m.secretTable, cmd = m.secretTable.Update(msg)
	return m, cmd
}

func (m *Model) selectedRow() (secretRow, bool) {
	i := m.secretTable.Cursor()
	if i < 0 || i >= len(m.secretTable.Rows()) {
		return secretRow{}, false
	}
	envVar := m.secretTable.Rows()[i][0]
	for _, r := range m.allRows {
		if r.envVar == envVar {
			return r, true
		}
	}
	return secretRow{}, false
}

func (m *Model) updateView(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "enter":
			m.screen = screenSecrets
			return m, nil
		}
	}
	return m, nil
}

func (m *Model) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		km := msg.(tea.KeyMsg).String()
		if km == "esc" {
			m.screen = screenSecrets
			m.editTarget = nil
			m.editForm = nil
			return m, nil
		}
		if (km == "alt+up" || km == "alt+down") && m.editForm != nil {
			if km == "alt+up" {
				return m, m.editForm.PrevField()
			}
			return m, m.editForm.NextField()
		}
		if (km == "up" || km == "down") && m.editForm != nil {
			if cmd, ok := formArrowNavCmd(m.editForm, km); ok {
				return m, cmd
			}
		}
	}

	if m.editForm == nil {
		m.editForm = m.newEditForm()
		m.editForm = m.sizeFormToModalBody(m.editForm)
		return m, m.editForm.Init()
	}

	var cmd tea.Cmd
	var mdl huh.Model
	mdl, cmd = m.editForm.Update(msg)
	if f, ok := mdl.(*huh.Form); ok {
		m.editForm = f
	}

	if m.editForm.State == huh.StateCompleted {
		if m.editTarget != nil && m.editSave {
			row := *m.editTarget
			val := m.editValue
			m.busyText = "Saving secret…"
			m.screen = screenBusy
			m.editForm = nil
			m.editTarget = nil
			return m, tea.Batch(
				m.spinner.Tick,
				func() tea.Msg {
					return editDoneMsg{err: m.saveEdit(row, val)}
				},
			)
		}
		m.screen = screenSecrets
		m.editForm = nil
		m.editTarget = nil
	}

	return m, cmd
}

func (m *Model) saveEdit(row secretRow, newValue string) error {
	factory := secrets.NewSecretManagerFactory()
	sm, err := factory.CreateSecretManager(m.ctx, row.provider, row.project)
	if err != nil {
		return err
	}
	defer sm.Close()

	mut, err := secrets.AsMutator(sm)
	if err != nil {
		return err
	}

	return mut.UpdateSecret(row.ref, newValue)
}

func (m *Model) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		// Constrain the form to the modal body area so it can scroll internally
		// and the action field remains reachable even on small terminals.
		if m.createForm != nil {
			innerW, innerH := panelInnerSize(ws.Width, ws.Height, panelStyle())
			// viewModal renders: title + "\n\n" + body
			bodyH := clampMin(innerH-2, 1)
			m.createForm = m.createForm.WithWidth(innerW).WithHeight(bodyH)
		}
		return m, nil
	}

	if _, ok := msg.(tea.KeyMsg); ok {
		km := msg.(tea.KeyMsg).String()
		if km == "esc" {
			m.screen = screenSecrets
			m.createForm = nil
			return m, nil
		}
		if (km == "alt+up" || km == "alt+down") && m.createForm != nil {
			if km == "alt+up" {
				return m, m.createForm.PrevField()
			}
			return m, m.createForm.NextField()
		}
		if (km == "up" || km == "down") && m.createForm != nil {
			if cmd, ok := formArrowNavCmd(m.createForm, km); ok {
				return m, cmd
			}
		}
	}

	// Ensure locations exist before rendering the form (so the multiselect has options).
	if err := m.ensureGCPLocationsLoaded(); err != nil {
		return m.openError(screenCreate, "Create failed", err.Error())
	}

	if m.createForm == nil {
		m.createForm = m.newCreateForm()
		m.createForm = m.sizeFormToModalBody(m.createForm)
		return m, m.createForm.Init()
	}

	var cmd tea.Cmd
	var mdl huh.Model
	mdl, cmd = m.createForm.Update(msg)
	if f, ok := mdl.(*huh.Form); ok {
		m.createForm = f
	}
	// Recompute the Summary note only when replication/locations change.
	{
		locs := append([]string(nil), m.createLocations...)
		sort.Strings(locs)
		key := m.createReplication + "|" + strings.Join(locs, ",")
		if key != m.createSummaryKey {
			m.createSummaryKey = key
			m.createSummaryTick++
		}
	}

	if m.createForm.State == huh.StateCompleted {
		switch m.createAction {
		case "create":
			in := m.snapshotCreateInput()
			m.busyText = "Creating secret…"
			m.screen = screenBusy
			m.createForm = nil
			return m, tea.Batch(
				m.spinner.Tick,
				func() tea.Msg {
					return createDoneMsg{err: m.doCreateFromForm(in)}
				},
			)
		default:
			// Explicit cancel: return to overview.
			m.screen = screenSecrets
			m.createForm = nil
			return m, nil
		}
	}

	return m, cmd
}

func (m *Model) updateConfirmDelete(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		km := msg.(tea.KeyMsg).String()
		if km == "esc" {
			m.screen = screenSecrets
			m.editTarget = nil
			m.deleteForm = nil
			return m, nil
		}
		if (km == "alt+up" || km == "alt+down") && m.deleteForm != nil {
			if km == "alt+up" {
				return m, m.deleteForm.PrevField()
			}
			return m, m.deleteForm.NextField()
		}
		if (km == "up" || km == "down") && m.deleteForm != nil {
			if cmd, ok := formArrowNavCmd(m.deleteForm, km); ok {
				return m, cmd
			}
		}
	}

	if m.deleteForm == nil {
		m.deleteForm = m.newDeleteForm()
		m.deleteForm = m.sizeFormToModalBody(m.deleteForm)
		return m, m.deleteForm.Init()
	}

	var cmd tea.Cmd
	var mdl huh.Model
	mdl, cmd = m.deleteForm.Update(msg)
	if f, ok := mdl.(*huh.Form); ok {
		m.deleteForm = f
	}

	if m.deleteForm.State == huh.StateCompleted {
		if m.editTarget != nil && m.deleteYes {
			row := *m.editTarget
			m.busyText = "Deleting secret…"
			m.screen = screenBusy
			m.deleteForm = nil
			m.editTarget = nil
			return m, tea.Batch(
				m.spinner.Tick,
				func() tea.Msg {
					return deleteDoneMsg{err: m.doDelete(row)}
				},
			)
		}
		m.screen = screenSecrets
		m.deleteForm = nil
		m.editTarget = nil
	}

	return m, cmd
}

func (m *Model) updateError(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		if km := msg.(tea.KeyMsg).String(); km == "esc" {
			m.screen = m.errorReturn
			m.errorForm = nil
			m.errorTitle = ""
			m.errorText = ""
			switch m.screen {
			case screenCreate:
				if m.createForm != nil {
					return m, m.createForm.Init()
				}
			case screenEdit:
				if m.editForm != nil {
					return m, m.editForm.Init()
				}
			case screenConfirmDelete:
				if m.deleteForm != nil {
					return m, m.deleteForm.Init()
				}
			}
			return m, nil
		}
	}

	if m.errorForm == nil {
		m.errorForm = m.newErrorForm(m.errorTitle, m.errorText)
		return m, m.errorForm.Init()
	}

	var cmd tea.Cmd
	var mdl huh.Model
	mdl, cmd = m.errorForm.Update(msg)
	if f, ok := mdl.(*huh.Form); ok {
		m.errorForm = f
	}

	if m.errorForm.State == huh.StateCompleted {
		m.screen = m.errorReturn
		m.errorForm = nil
		m.errorTitle = ""
		m.errorText = ""
		switch m.screen {
		case screenCreate:
			if m.createForm != nil {
				return m, m.createForm.Init()
			}
		case screenEdit:
			if m.editForm != nil {
				return m, m.editForm.Init()
			}
		case screenConfirmDelete:
			if m.deleteForm != nil {
				return m, m.deleteForm.Init()
			}
		}
	}

	return m, cmd
}

func (m *Model) doDelete(row secretRow) error {
	factory := secrets.NewSecretManagerFactory()
	sm, err := factory.CreateSecretManager(m.ctx, row.provider, row.project)
	if err != nil {
		return err
	}
	defer sm.Close()

	mut, err := secrets.AsMutator(sm)
	if err != nil {
		return err
	}

	if err := mut.DeleteSecret(row.ref, true); err != nil {
		return err
	}

	// Also remove the mapping from ws.yaml so it doesn't reappear on refresh.
	if err := config.RemoveEnvMapping(m.configPath, m.selectedEnvName, row.envVar); err != nil {
		return err
	}

	// Reload config/env so subsequent actions use updated inheritance/mappings.
	if cfg, err := config.LoadSecretsConfig(m.configPath); err == nil {
		m.cfg = cfg
		if env, err := m.cfg.GetEnvironment(m.selectedEnvName); err == nil {
			m.selectedEnv = env
		}
	}

	return nil
}

func (m *Model) openError(returnTo Screen, title, text string) (tea.Model, tea.Cmd) {
	m.errorReturn = returnTo
	m.errorTitle = title
	m.errorText = text
	m.errorForm = m.newErrorForm(title, text)
	m.screen = screenError
	return m, m.errorForm.Init()
}

func (m *Model) setSecretTableColumns(innerW int) {
	// bubbles/table does not automatically reflow column widths when SetWidth is
	// called, so we recompute widths to avoid horizontal overflow on small terminals.
	//
	// Note: bubbles/table renders each cell with padding and inserts spacing between
	// columns. Column.Width applies to the cell content area, but the final rendered
	// width includes extra "chrome". We therefore reserve a conservative overhead
	// budget so the table never exceeds innerW.
	const cols = 4
	// Default table styles include left+right cell padding (commonly 1 each),
	// plus at least 1 char gap between columns.
	cellPaddingLR := 2 // 1 left + 1 right (conservative)
	colGaps := cols - 1
	overhead := cols*cellPaddingLR + colGaps
	avail := innerW - overhead
	if avail < cols { // at least 1 char per column
		avail = cols
	}

	// Minimums for usability.
	minEnv, minVal, minProv, minRef := 8, 10, 8, 10
	minTotal := minEnv + minVal + minProv + minRef
	if avail < minTotal {
		// If extremely narrow, shrink everything but keep provider readable-ish.
		minEnv, minVal, minRef = 6, 6, 6
		minTotal = minEnv + minVal + minProv + minRef
	}

	envW, valW, provW, refW := minEnv, minVal, minProv, minRef
	remaining := avail - (envW + valW + provW + refW)
	if remaining < 0 {
		remaining = 0
	}

	// Distribute extra width with a bias towards Value and Ref.
	// Order: Value, Ref, Env Var, Provider.
	add := func(w *int, n int) {
		if n <= 0 {
			return
		}
		*w += n
		remaining -= n
	}
	for remaining > 0 {
		add(&valW, min(remaining, 2))
		if remaining == 0 {
			break
		}
		add(&refW, min(remaining, 2))
		if remaining == 0 {
			break
		}
		add(&envW, 1)
		if remaining == 0 {
			break
		}
		add(&provW, 1)
	}

	m.secretTable.SetColumns([]table.Column{
		{Title: "Env Var", Width: envW},
		{Title: "Value", Width: valW},
		{Title: "Provider", Width: provW},
		{Title: "Ref", Width: refW},
	})
}
