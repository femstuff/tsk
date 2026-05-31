package bitrixclient

import (
	"fmt"
	"strings"
)

const bitrixTaskScopeHint = "В Bitrix24: Разработчикам → Локальное приложение → ваше приложение → Права доступа → включите «Задачи (task)» и «Пользователи (user)» → сохраните. Затем в мобильном приложении выйдите из Bitrix24 и войдите снова."

const bitrixCRMScopeHint = "В Bitrix24: Разработчикам → Локальное приложение → ваше приложение → Права доступа → включите «CRM (crm)» → сохраните. Затем в мобильном приложении выйдите из Bitrix24 и войдите снова."

// TasksListUserError переводит ответ Bitrix REST в понятное сообщение для пользователя.
func TasksListUserError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "insufficient_scope"):
		return fmt.Errorf("нет прав «Задачи» у OAuth-приложения Bitrix24. %s", bitrixTaskScopeHint)
	case strings.Contains(msg, "invalid_scope"):
		return fmt.Errorf("запрошенные права превышают настройки локального приложения Bitrix24. %s", bitrixTaskScopeHint)
	case strings.Contains(msg, "user_access_error"):
		return fmt.Errorf("ваш пользователь не имеет доступа к приложению в Bitrix24 — попросите администратора портала открыть доступ")
	default:
		return err
	}
}

// DealsListUserError переводит ответ Bitrix REST по сделкам в понятное сообщение.
func DealsListUserError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "insufficient_scope"):
		return fmt.Errorf("нет прав «CRM» у OAuth-приложения Bitrix24. %s", bitrixCRMScopeHint)
	case strings.Contains(msg, "invalid_scope"):
		return fmt.Errorf("запрошенные права CRM превышают настройки локального приложения Bitrix24. %s", bitrixCRMScopeHint)
	case strings.Contains(msg, "user_access_error"):
		return fmt.Errorf("ваш пользователь не имеет доступа к CRM в Bitrix24 — попросите администратора портала открыть доступ")
	default:
		return err
	}
}

// HasTaskScope проверяет, что в scope OAuth-токена есть доступ к задачам.
// Локальные приложения Bitrix24 часто возвращают scope=app — в этом случае считаем права выданными.
func HasTaskScope(scope string) bool {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" || scope == "app" {
		return true
	}
	for _, part := range strings.FieldsFunc(scope, func(r rune) bool {
		return r == ',' || r == ' ' || r == ';'
	}) {
		part = strings.TrimSpace(part)
		switch part {
		case "task", "tasks", "tasks_extended", "tasksmobile":
			return true
		default:
			if strings.HasPrefix(part, "task.") || strings.HasPrefix(part, "tasks.") {
				return true
			}
		}
	}
	return false
}

// NotifyListUserError переводит ответ Bitrix REST по уведомлениям.
func NotifyListUserError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "insufficient_scope"):
		return fmt.Errorf("нет прав «Чат и уведомления (im)» у OAuth-приложения Bitrix24. Включите scope im и notifications, затем перелогиньтесь")
	case strings.Contains(msg, "invalid_scope"):
		return fmt.Errorf("запрошенные права уведомлений превышают настройки локального приложения Bitrix24")
	default:
		return err
	}
}
