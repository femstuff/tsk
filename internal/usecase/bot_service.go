package usecase

import (
	"tg-tsk-bot/internal/entity"
)

type MessageRepo interface {
	Save(message *entity.Message) error
	//на подумать че еще надо
}
