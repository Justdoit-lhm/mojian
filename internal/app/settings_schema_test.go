package app

import (
	"reflect"
	"testing"

	"mojian/internal/settings"
)

func TestSettingsTableMatchesPreferences(t *testing.T) {
	prefsType := reflect.TypeOf(prefs{})
	ids := map[string]bool{}
	for _, row := range settings.Rows {
		if row.ID == "" {
			t.Fatalf("%s has no id", row.Title)
		}
		if ids[row.ID] {
			t.Fatalf("duplicate settings row id %q", row.ID)
		}
		ids[row.ID] = true
		if row.Pref == "" {
			continue
		}
		field, ok := prefsType.FieldByName(row.Pref)
		if !ok {
			t.Fatalf("%s names unknown preference %q", row.ID, row.Pref)
		}
		switch row.Kind {
		case settings.Toggle:
			if field.Type.Kind() != reflect.Bool {
				t.Fatalf("%s: %s is %s, want bool", row.ID, row.Pref, field.Type)
			}
		case settings.Choice:
			if field.Type.Kind() != reflect.Int {
				t.Fatalf("%s: %s is %s, want int", row.ID, row.Pref, field.Type)
			}
		case settings.Slider:
			if k := field.Type.Kind(); k != reflect.Float64 && k != reflect.Int {
				t.Fatalf("%s: %s is %s, want number", row.ID, row.Pref, field.Type)
			}
		case settings.Text, settings.Font:
			if field.Type.Kind() != reflect.String {
				t.Fatalf("%s: %s is %s, want string", row.ID, row.Pref, field.Type)
			}
		}
	}
}
