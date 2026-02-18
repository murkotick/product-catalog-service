package domain

// ChangeTracker tracks which fields have been modified in an aggregate.
// This enables repositories to generate optimized UPDATE statements
// that only modify changed fields rather than updating entire rows.
type ChangeTracker struct {
	dirtyFields map[string]bool
}

// NewChangeTracker creates a new ChangeTracker instance.
func NewChangeTracker() *ChangeTracker {
	return &ChangeTracker{
		dirtyFields: make(map[string]bool),
	}
}

// MarkDirty marks a field as dirty (modified).
func (ct *ChangeTracker) MarkDirty(field string) {
	ct.dirtyFields[field] = true
}

// Dirty checks if a specific field has been marked dirty.
func (ct *ChangeTracker) Dirty(field string) bool {
	return ct.dirtyFields[field]
}

// HasChanges returns true if any fields have been marked dirty.
func (ct *ChangeTracker) HasChanges() bool {
	return len(ct.dirtyFields) > 0
}

// DirtyFields returns a slice of all field names that have been marked dirty.
func (ct *ChangeTracker) DirtyFields() []string {
	fields := make([]string, 0, len(ct.dirtyFields))
	for field := range ct.dirtyFields {
		fields = append(fields, field)
	}
	return fields
}

// Clear removes all dirty field markers.
func (ct *ChangeTracker) Clear() {
	ct.dirtyFields = make(map[string]bool)
}

// Count returns the number of dirty fields.
func (ct *ChangeTracker) Count() int {
	return len(ct.dirtyFields)
}
