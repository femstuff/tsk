package entity

import "time"

type Message struct {
	ID        string
	Text      string
	ChatID    int64
	Status    string
	CreatedAt time.Time
}
