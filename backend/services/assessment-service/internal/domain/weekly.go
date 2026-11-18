package domain

type WeeklyEntry struct {
	ID          string
	CompletedAt int64
	Result      Result
}
type WeeklyHistory struct {
	Entries    []WeeklyEntry
	NextDue    int64
	Due        bool
	NextCursor string
}
