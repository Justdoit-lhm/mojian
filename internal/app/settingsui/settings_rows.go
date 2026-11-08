package settingsui

import (
	"reflect"

	"mojian/internal/settings"

	"github.com/egoist/mygo/ui"
)

// settingsGenericPage renders a page whose rows are declared in the
// settings table: sections and rows come from the same list the search
// index uses.
func (u *UI) settingsGenericPage(c *ui.Context, k *colors, cat int) {
	section := ""
	var rows []settings.Row
	flush := func() {
		if len(rows) == 0 {
			return
		}
		u.settingsSection(c, k, section, func() {
			for _, row := range rows {
				u.settingsRowSpec(c, k, row)
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
func (u *UI) settingsRowSpec(c *ui.Context, k *colors, row settings.Row) {
	switch row.Kind {
	case settings.Toggle:
		u.settingsSwitchRow(c, k, row.Title, row.Detail,
			func() bool {
				v := u.prefBool(row.Pref)
				if row.Invert {
					return !v
				}
				return v
			},
			func(v bool) {
				if row.Invert {
					v = !v
				}
				u.setPrefBool(row.Pref, v)
			})
	case settings.Choice:
		u.settingsChoiceRow(c, k, row.ID, row.Title,
			func() int { return u.prefChoice(row) },
			func(i int) { u.setPrefChoice(row, i) }, row.Options)
	case settings.Slider:
		u.settingsSliderRow(c, k, row.Title, row.Detail,
			func() float64 { return u.prefFloat(row.Pref) },
			func(v float64) { u.setPrefFloat(row.Pref, v) },
			row.Min, row.Max, row.Format)
	case settings.Text:
		ptr := u.prefStringPtr(row.Pref)
		if ptr == nil {
			return
		}
		u.settingRow(c, k, row.Title, row.Detail, func() {
			in := ui.TextInputBase(c, ptr).Width(230).FontSize(12).
				Placeholder(row.Placeholder)
			if in.Changed() {
				u.saveSettings()
				u.invalidate()
			}
		})
	case settings.Font:
		u.fontRow(c, k, row.Title, row.Detail, row.FontWhich, u.prefString(row.Pref))
	case settings.Info:
		u.settingsInfoRow(c, k, row.Title, u.settingsInfoValue(row.ID))
	}
}

// settingsInfoValue is the value a declared Info row shows.
func (u *UI) settingsInfoValue(id string) string {
	switch id {
	case "current-project":
		return u.projectPath()
	case "server":
		return u.remoteName()
	case "git-status":
		return "Working tree against HEAD"
	}
	return ""
}

// prefField reaches the prefs struct by its exported field name, which
// is what a settings.Row's Pref names.
func (u *UI) prefField(name string) reflect.Value {
	v := reflect.ValueOf(u.Prefs).Elem().FieldByName(name)
	if !v.IsValid() {
		panic("settings: unknown preference " + name)
	}
	return v
}

func (u *UI) prefBool(name string) bool { return u.prefField(name).Bool() }
func (u *UI) prefInt(name string) int {
	v := u.prefField(name)
	if v.Kind() == reflect.Int {
		return int(v.Int())
	}
	return 0
}
func (u *UI) prefFloat(name string) float64 {
	v := u.prefField(name)
	if v.Kind() == reflect.Float64 {
		return v.Float()
	}
	return float64(v.Int())
}
func (u *UI) prefString(name string) string { return u.prefField(name).String() }
func (u *UI) prefStringPtr(name string) *string {
	v := u.prefField(name)
	if v.Kind() != reflect.String {
		return nil
	}
	return v.Addr().Interface().(*string)
}
func (u *UI) setPrefBool(name string, value bool) { u.prefField(name).SetBool(value) }
func (u *UI) setPrefInt(name string, value int) {
	v := u.prefField(name)
	if v.Kind() == reflect.Int {
		v.SetInt(int64(value))
	}
}
func (u *UI) setPrefFloat(name string, value float64) {
	v := u.prefField(name)
	if v.Kind() == reflect.Float64 {
		v.SetFloat(value)
		return
	}
	v.SetInt(int64(value))
}

func (u *UI) prefChoice(row settings.Row) int {
	v := u.prefInt(row.Pref)
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

func (u *UI) setPrefChoice(row settings.Row, i int) {
	value := i
	if len(row.Values) > 0 && i >= 0 && i < len(row.Values) {
		value = row.Values[i]
	}
	if row.Pref == "Appearance" {
		u.Appearance = value
		u.syncAppearance()
		return
	}
	u.setPrefInt(row.Pref, value)
}
