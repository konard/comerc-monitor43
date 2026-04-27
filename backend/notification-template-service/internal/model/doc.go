// Package model содержит доменные модели notification-template-service.
//
// Модели представляют собой бизнес-сущности с валидацией и бизнес-методами.
//
// Основные модели:
//
//	Template - шаблон уведомления
//	TemplateVariable - переменная шаблона
//	RenderedTemplate - отрендеренный шаблон
//	ValidationResult - результат валидации шаблона
//
// Все модели имеют бизнес-методы для валидации и трансформации:
//
//	template.Validate() - проверяет корректность шаблона
//	template.Render(variables) - рендерит шаблон с переменными
//	template.IsDefaultFor(channel, type) - проверяет является ли шаблон дефолтным
//	template.CanDelete() - проверяет можно ли удалить шаблон
package model
