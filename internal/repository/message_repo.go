package repository

import (
	"errors"
	"sync"
	"tg-tsk-bot/internal/entity"
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
		return errors.New("erro, message id not be empty") //по хорошему бы генерацию айди добавить мне впадлу
	}
	// далее подобные темы дописать, которые могут вызвать ошибку при сохранениеи

	return nil
}
