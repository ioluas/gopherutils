package config

// BackupMethod controls how backups are made.
type BackupMethod int

const (
	BackupNone     BackupMethod = iota // never make backups
	BackupNumbered                     // make numbered backups
	BackupExisting                     // numbered if numbered backups exist, simple otherwise
	BackupSimple                       // always make simple backups
)

// UpdateMode controls which existing files are updated.
type UpdateMode int

const (
	UpdateReplaceAll   UpdateMode = iota // default: replace all existing files
	UpdateNone                           // none: do not replace existing files
	UpdateNoneFail                       // none-fail: fail if file exists
	UpdateReplaceOlder                   // older: replace if source is newer
)

// Config holds the configuration for the cp utility.
type Config struct {
	Sources        []string
	Dest           string
	Verbose        bool         // -v, --verbose: explain what is being done
	Backup         bool         // -b, --backup: make a backup of each existing destination file
	BackupMethod   BackupMethod // Control method for backups
	Suffix         string       // --suffix: override the usual backup suffix
	UpdateMode     UpdateMode   // --update: control which existing files are updated
	AttributesOnly bool         // --attributes-only: don't copy the file data, just the attributes
	Preserve       PreserveOptions
}

// PreserveOptions holds which attributes should be preserved.
type PreserveOptions struct {
	Mode       bool
	Ownership  bool
	Timestamps bool
	Links      bool
	Context    bool
	Xattr      bool
}
