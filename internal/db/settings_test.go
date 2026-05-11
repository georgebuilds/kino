package db

import "testing"

func TestSetSetting_Then_GetSetting(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetSetting("theme", "dark"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}

	got, err := d.GetSetting("theme")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if got != "dark" {
		t.Fatalf("GetSetting = %q, want %q", got, "dark")
	}
}

func TestGetSetting_MissingKey_ReturnsEmpty(t *testing.T) {
	d := newTestDB(t)

	got, err := d.GetSetting("nonexistent_key")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if got != "" {
		t.Fatalf("GetSetting(missing) = %q, want empty string", got)
	}
}

func TestSetSetting_Upsert(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetSetting("lang", "en"); err != nil {
		t.Fatalf("first SetSetting: %v", err)
	}
	if err := d.SetSetting("lang", "fr"); err != nil {
		t.Fatalf("second SetSetting: %v", err)
	}

	got, err := d.GetSetting("lang")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if got != "fr" {
		t.Fatalf("GetSetting after upsert = %q, want %q", got, "fr")
	}
}
