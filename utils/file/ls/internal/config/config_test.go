package config

import "testing"

func TestQuotingStyleString(t *testing.T) {
	tests := []struct {
		style    QuotingStyle
		expected string
	}{
		{QuotingStyleLiteral, "literal"},
		{QuotingStyleLocale, "locale"},
		{QuotingStyleShell, "shell"},
		{QuotingStyleShellAlways, "shell-always"},
		{QuotingStyleShellEscape, "shell-escape"},
		{QuotingStyleShellEscapeAlways, "shell-escape-always"},
		{QuotingStyleC, "c"},
		{QuotingStyleEscape, "escape"},
		{QuotingStyle(99), "literal"},
	}

	for _, tt := range tests {
		if got := tt.style.String(); got != tt.expected {
			t.Fatalf("QuotingStyle(%d).String() = %q, want %q", tt.style, got, tt.expected)
		}
	}
}

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

func TestFormatModeDefaults(t *testing.T) {
	if FormatDefault != 0 {
		t.Fatalf("expected FormatDefault to be zero, got %d", FormatDefault)
	}
	if FormatDefault == FormatColumnate || FormatDefault == FormatOnePerLine || FormatColumnate == FormatOnePerLine {
		t.Fatal("expected format modes to be distinct")
	}
}
