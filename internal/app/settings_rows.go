package app

import (
	"reflect"

	"mojian/internal/settings"

	"github.com/egoist/mygo/ui"
)

// settingsGenericPage renders a page whose rows are declared in the
// settings table: sections and rows come from the same list the search
// index uses.
func (a *app) settingsGenericPage(c *ui.Context, k *colors, cat int) {
	section := ""
	var rows []settings.Row
	flush := func() {
		if len(rows) == 0 {
			return
		}
		a.settingsSection(c, k, section, func() {
			for _, row := range rows {
				a.settingsRowSpec(c, k, row)
			}
		})
	}
	for _, row := range settings.Rows {
		if row.Cat != cat {
			continue
		}
		if row.Section != section {
			flush()
			section, rows = row.Section, nil
		}
		rows = append(rows, row)
	}
	flush()
}

// settingsRowSpec draws one declared row with the control its Kind names.
func (a *app) settingsRowSpec(c *ui.Context, k *colors, row settings.Row) {
	switch row.Kind {
	case settings.Toggle:
		a.settingsSwitchRow(c, k, row.Title, row.Detail,
			func() bool {
				v := a.prefBool(row.Pref)
				if row.Invert {
					return !v
				}
				return v
			},
			func(v bool) {
				if row.Invert {
					v = !v
				}
				a.setPrefBool(row.Pref, v)
			})
	case settings.Choice:
		a.settingsChoiceRow(c, k, row.ID, row.Title,
			func() int { return a.prefChoice(row) },
			func(i int) { a.setPrefChoice(row, i) }, row.Options)
	case settings.Slider:
		a.settingsSliderRow(c, k, row.Title, row.Detail,
			func() float64 { return a.prefFloat(row.Pref) },
			func(v float64) { a.setPrefFloat(row.Pref, v) },
			row.Min, row.Max, row.Format)
	case settings.Text:
		ptr := a.prefStringPtr(row.Pref)
		if ptr == nil {
			return
		}
		a.settingRow(c, k, row.Title, row.Detail, func() {
			in := ui.TextInputBase(c, ptr).Width(230).FontSize(12).
				Placeholder(row.Placeholder)
			if in.Changed() {
				a.saveSettings()
				a.invalidate()
			}
		})
	case settings.Font:
		a.fontRow(c, k, row.Title, row.Detail, row.FontWhich, a.prefString(row.Pref))
	case settings.Info:
		a.settingsInfoRow(c, k, row.Title, a.settingsInfoValue(row.ID))
	}
}

// settingsInfoValue is the value a declared Info row shows.
func (a *app) settingsInfoValue(id string) string {
	switch id {
	case "current-project":
		return a.projectPath()
	case "server":
		return a.remoteName()
	case "git-status":
		return "Working tree against HEAD"
	}
	return ""
}

// prefField reaches the prefs struct by its exported field name, which
// is what a settings.Row's Pref names.
func (a *app) prefField(name string) reflect.Value {
	v := reflect.ValueOf(&a.prefs).Elem().FieldByName(name)
	if !v.IsValid() {
		panic("settings: unknown preference " + name)
	}
	return v
}

func (a *app) prefBool(name string) bool { return a.prefField(name).Bool() }
func (a *app) prefInt(name string) int {
	v := a.prefField(name)
	if v.Kind() == reflect.Int {
		return int(v.Int())
	}
	return 0
}
func (a *app) prefFloat(name string) float64 {
	v := a.prefField(name)
	if v.Kind() == reflect.Float64 {
		return v.Float()
	}
	return float64(v.Int())
}
func (a *app) prefString(name string) string { return a.prefField(name).String() }
func (a *app) prefStringPtr(name string) *string {
	v := a.prefField(name)
	if v.Kind() != reflect.String {
		return nil
	}
	return v.Addr().Interface().(*string)
}
func (a *app) setPrefBool(name string, value bool) { a.prefField(name).SetBool(value) }
func (a *app) setPrefInt(name string, value int) {
	v := a.prefField(name)
	if v.Kind() == reflect.Int {
		v.SetInt(int64(value))
	}
}
func (a *app) setPrefFloat(name string, value float64) {
	v := a.prefField(name)
	if v.Kind() == reflect.Float64 {
		v.SetFloat(value)
		return
	}
	v.SetInt(int64(value))
}

func (a *app) prefChoice(row settings.Row) int {
	v := a.prefInt(row.Pref)
	if len(row.Values) > 0 {
		for i, want := range row.Values {
			if v == want {
				return i
			}
		}
		return 0
	}
	return v
}

func (a *app) setPrefChoice(row settings.Row, i int) {
	value := i
	if len(row.Values) > 0 && i >= 0 && i < len(row.Values) {
		value = row.Values[i]
	}
	if row.Pref == "Appearance" {
		a.Appearance = value
		a.syncAppearance()
		return
	}
	a.setPrefInt(row.Pref, value)
}
