package app

import "mojian/internal/app/explorer"

// The Explorer's draft type and constants keep the app's vocabulary.
type treeDraft = explorer.Draft

const (
	draftFile   = explorer.DraftFile
	draftFolder = explorer.DraftFolder
	draftRename = explorer.DraftRename
)
