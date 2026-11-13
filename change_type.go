package gitutil

// StatusChangeType represents git change types from status codes.
// Values correspond to git status --porcelain format codes.
type StatusChangeType byte

const (
	UnknownChangeType   StatusChangeType = 0
	ModifiedChangeType  StatusChangeType = 'M' // Modified
	AddedChangeType     StatusChangeType = 'A' // Added (new file)
	DeletedChangeType   StatusChangeType = 'D' // Deleted
	RenamedChangeType   StatusChangeType = 'R' // Renamed
	CopiedChangeType    StatusChangeType = 'C' // Copied
	UntrackedChangeType StatusChangeType = '?' // Untracked (new, not in index)
)

// String returns the full name of the change type for general purpose use.
func (ct StatusChangeType) String() string {
	switch ct {
	case ModifiedChangeType:
		return "Modified"
	case AddedChangeType:
		return "Added"
	case DeletedChangeType:
		return "Deleted"
	case RenamedChangeType:
		return "Renamed"
	case CopiedChangeType:
		return "Copied"
	case UntrackedChangeType:
		return "Untracked"
	default:
		return "Unknown"
	}
}

// Label returns a short display label for the change type (suitable for table columns).
func (ct StatusChangeType) Label() string {
	switch ct {
	case ModifiedChangeType:
		return "Mod"
	case AddedChangeType:
		return "Add"
	case DeletedChangeType:
		return "Del"
	case RenamedChangeType:
		return "Ren"
	case CopiedChangeType:
		return "Cpy"
	case UntrackedChangeType:
		return "New"
	default:
		return "???"
	}
}

// ParseChangeType converts a git status code byte to StatusChangeType.
// Returns error if the code is not a recognized git status code.
func ParseChangeType(code byte) (ct StatusChangeType, err error) {
	switch code {
	case 'M':
		ct = ModifiedChangeType
	case 'A':
		ct = AddedChangeType
	case 'D':
		ct = DeletedChangeType
	case 'R':
		ct = RenamedChangeType
	case 'C':
		ct = CopiedChangeType
	case '?':
		ct = UntrackedChangeType
	case ' ', 0:
		ct = UnknownChangeType
	default:
		err = NewErr(
			ErrInvalidGitStatusCode,
			"code", code,
		)
		goto end
	}

end:
	return ct, err
}
