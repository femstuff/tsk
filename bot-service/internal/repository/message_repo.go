package repository

import (
	"sync"
	"tg-tsk-bot/internal/entity"
	"time"
)

type MessageRepo interface {
	Save(message *entity.Message) error
	//на подумать че еще надо
}

type InMemoryMessage struct {
	//легкая заглушка впадлу бд подтягивать еще и поднимать да ну вообще нахуй его
	messages map[string]*entity.Message
	mu       sync.Mutex
}

func NewInMemoryMessage() *InMemoryMessage {
	return &InMemoryMessage{
		messages: make(map[string]*entity.Message),
	}
}

func (r *InMemoryMessage) Save(msg *entity.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if msg.ID == "" {
		msg.ID = generateID()
	}
	// далее подобные темы дописать, которые могут вызвать ошибку при сохранениеи

	return nil
}

func generateID() string {
	return time.Now().Format("20060102150405")
}
