package config

import "testing"

func TestBlockSizeModeDefaults(t *testing.T) {
	if BlockSizeModeBytes != 0 {
		t.Fatalf("expected BlockSizeModeBytes to be zero, got %d", BlockSizeModeBytes)
	}
	if BlockSizeModeBytes == BlockSizeModeHuman || BlockSizeModeBytes == BlockSizeModeSI || BlockSizeModeHuman == BlockSizeModeSI {
		t.Fatal("expected block size modes to be distinct")
	}
}

func TestTimeFieldDefaults(t *testing.T) {
	if TimeFieldMod != 0 {
		t.Fatalf("expected TimeFieldMod to be zero, got %d", TimeFieldMod)
	}
	if TimeFieldMod == TimeFieldAccess || TimeFieldMod == TimeFieldChange || TimeFieldMod == TimeFieldBirth {
		t.Fatal("expected time fields to be distinct from TimeFieldMod")
	}
	if TimeFieldAccess == TimeFieldChange || TimeFieldAccess == TimeFieldBirth || TimeFieldChange == TimeFieldBirth {
		t.Fatal("expected time fields to be distinct")
	}
}

func TestTimeStyleKindDefaults(t *testing.T) {
	if TimeStyleLocale != 0 {
		t.Fatalf("expected TimeStyleLocale to be zero, got %d", TimeStyleLocale)
	}
	if TimeStyleLocale == TimeStyleFullISO || TimeStyleLocale == TimeStyleLongISO || TimeStyleLocale == TimeStyleISO || TimeStyleLocale == TimeStyleCustom {
		t.Fatal("expected time style kinds to be distinct from TimeStyleLocale")
	}
	if TimeStyleFullISO == TimeStyleLongISO || TimeStyleFullISO == TimeStyleISO || TimeStyleFullISO == TimeStyleCustom {
		t.Fatal("expected time style kinds to be distinct")
	}
}
