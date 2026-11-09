package settings

import (
	"reflect"
	"testing"

	"mojian/internal/prefs"
)

// TestSettingsTableNamesEachPreferenceOnce keeps every preference on one
// page: a second row for it is a second design for the same setting.
func TestSettingsTableNamesEachPreferenceOnce(t *testing.T) {
	seen := map[string]string{}
	for _, row := range Rows {
		if row.Pref == "" {
			continue
		}
		if other, ok := seen[row.Pref]; ok {
			t.Fatalf("%s and %s both edit %s", other, row.ID, row.Pref)
		}
		seen[row.Pref] = row.ID
	}
	prefsType := reflect.TypeOf(prefs.Prefs{})
	for i := range prefsType.NumField() {
		name := prefsType.Field(i).Name
		if _, ok := seen[name]; !ok {
			t.Fatalf("no row edits %s", name)
		}
	}
}

// TestSettingsCategoriesUseDistinctGlyphs keeps the rail readable: two
// pages with one picture look like one page.
func TestSettingsCategoriesUseDistinctGlyphs(t *testing.T) {
	seen := map[string]string{}
	for _, cat := range Categories {
		if other, ok := seen[cat.Glyph]; ok {
			t.Fatalf("%s and %s both use %s", other, cat.Title, cat.Glyph)
		}
		seen[cat.Glyph] = cat.Title
	}
}

func TestSettingsTableMatchesPreferences(t *testing.T) {
	prefsType := reflect.TypeOf(prefs.Prefs{})
	ids := map[string]bool{}
	for _, row := range Rows {
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
		case Toggle:
			if field.Type.Kind() != reflect.Bool {
				t.Fatalf("%s: %s is %s, want bool", row.ID, row.Pref, field.Type)
			}
		case Choice:
			if field.Type.Kind() != reflect.Int {
				t.Fatalf("%s: %s is %s, want int", row.ID, row.Pref, field.Type)
			}
		case Slider:
			if k := field.Type.Kind(); k != reflect.Float64 && k != reflect.Int {
				t.Fatalf("%s: %s is %s, want number", row.ID, row.Pref, field.Type)
			}
		case Text, Font:
			if field.Type.Kind() != reflect.String {
				t.Fatalf("%s: %s is %s, want string", row.ID, row.Pref, field.Type)
			}
		}
	}
}
