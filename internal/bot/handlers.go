package bot

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"text/template"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/alekslesik/telegram-bot-simple/internal/domain/learning"
	"github.com/alekslesik/telegram-bot-simple/internal/repository"
	contentservice "github.com/alekslesik/telegram-bot-simple/internal/service/content"
	learningflowservice "github.com/alekslesik/telegram-bot-simple/internal/service/learningflow"
)

// TelegramClient — минимум для Send и ответа на callback (answerCallbackQuery).
type TelegramClient interface {
	Send(tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

type Handlers struct {
	Bot      TelegramClient
	Logger   *slog.Logger
	Repo     repository.ContentRepository
	Content  flowContentBuilder
	Learning flowNavigator
	State    flowStateStore
}

type flowContentBuilder interface {
	BuildBlockPayload(ctx context.Context, blockID int64) (contentservice.BlockPayload, error)
}

type flowNavigator interface {
	NextStep(blockType learning.BlockType, current learning.FlowStep, action learningflowservice.Action) learning.FlowStep
}

type flowState struct {
	BlockID    int64
	ChapterID  int64
	Step       learning.FlowStep
	LastAnswer string
}

type flowStateStore interface {
	Get(userID int64) (flowState, bool)
	Save(userID int64, state flowState)
}

type inMemoryFlowStateStore struct {
	mu     sync.RWMutex
	byUser map[int64]flowState
}

func newInMemoryFlowStateStore() *inMemoryFlowStateStore {
	return &inMemoryFlowStateStore{
		byUser: make(map[int64]flowState),
	}
}

func (s *inMemoryFlowStateStore) Get(userID int64) (flowState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.byUser[userID]
	return state, ok
}

func (s *inMemoryFlowStateStore) Save(userID int64, state flowState) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.byUser[userID] = state
}

type Command struct {
	Name        string
	Description string
	ParseMode   string
	BuildText   func(msg *tgbotapi.Message) string
}

type UseCaseCategory struct {
	Title string
	Items []string
}

var commandButtons = map[string]string{
	"🚀 Старт":            "start",
	"📋 Демо-меню":        "menu",
	"🆘 Помощь":           "help",
	"ℹ️ О боте":          "about",
	"💼 Примеры задач":    "usecases",
	"🧩 Возможности":      "features",
	"✅ Проверка статуса": "ping",
	"🗣️ Повторить текст": "echo",
}

var learningMenuButtons = map[string]string{
	"📘 Введение":               "introduction",
	"📚 Алгоритмы (по порядку)": "algorithms",
	"🎲 Рандом задача":          "random",
	"📈 Мой прогресс":           "progress",
	"⚙️ Настройки":             "settings",
}

// demoInlineMenuKeyboard — те же пункты, что reply-клавиатура и меню у поля ввода.
func demoInlineMenuKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🚀 Старт", "cmd:start"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Демо-меню", "cmd:menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🆘 Помощь", "cmd:help"),
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ О боте", "cmd:about"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💼 Примеры задач", "cmd:usecases"),
			tgbotapi.NewInlineKeyboardButtonData("🧩 Возможности", "cmd:features"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Проверка статуса", "cmd:ping"),
			tgbotapi.NewInlineKeyboardButtonData("🗣️ Повторить текст", "cmd:echo"),
		),
	)
}

func commandKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("📘 Введение"),
			tgbotapi.NewKeyboardButton("📚 Алгоритмы (по порядку)"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("🎲 Рандом задача"),
			tgbotapi.NewKeyboardButton("📈 Мой прогресс"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("⚙️ Настройки"),
		),
	)
}

var useCases = []UseCaseCategory{
	{
		Title: "Салон / студия / услуги",
		Items: []string{
			"рассказать про услуги и цены",
			"принять заявку или запись",
			"отправить напоминание перед визитом",
		},
	},
	{
		Title: "Онлайн‑курсы / эксперты",
		Items: []string{
			"выдать материалы и инструкции",
			"собрать вопросы от учеников",
			"аккуратно предлагать доп. продукты",
		},
	},
	{
		Title: "Малый бизнес",
		Items: []string{
			"ответы на частые вопросы",
			"получение контакта для звонка",
			"быстрые опросы клиентов",
		},
	},
}

var usecasesTmpl = template.Must(template.New("usecases").Funcs(template.FuncMap{
	"add1": func(i int) int { return i + 1 },
}).Parse(
	`*Примеры задач, для которых подходит такой бот:*

{{- range $i, $c := . }}
{{ add1 $i }}. {{ $c.Title }}:
{{- range $c.Items }}
   — {{ . }}
{{- end }}

{{- end }}
Идея простая: всё, что менеджер делает руками в переписке, можно постепенно перенести в бота.`,
))

func renderUseCases() string {
	var buf bytes.Buffer
	_ = usecasesTmpl.Execute(&buf, useCases)
	return buf.String()
}

func (h *Handlers) commandRegistry() map[string]Command {
	commands := map[string]Command{
		"start": {
			Name:        "start",
			Description: "приветствие и краткое объяснение",
			BuildText: func(_ *tgbotapi.Message) string {
				return "Привет! Я демонстрационный Telegram‑бот для бизнеса.\n\n" +
					"Я показываю, как может выглядеть живой продукт для заказчика:\n" +
					"- приветствие новых клиентов\n" +
					"- ответы на типовые вопросы\n" +
					"- сбор заявок прямо в чат\n" +
					"- простая обратная связь.\n\n" +
					"Напиши /help, чтобы увидеть, что я уже умею."
			},
		},
		"menu": {
			Name:        "menu",
			Description: "показать демо-меню кнопками",
			BuildText: func(_ *tgbotapi.Message) string {
				return "Пожалуйста, выберите пункт меню."
			},
		},
		"about": {
			Name:        "about",
			Description: "чем полезен такой бот для бизнеса",
			ParseMode:   tgbotapi.ModeMarkdown,
			BuildText: func(_ *tgbotapi.Message) string {
				return "Этот бот — пример того, что вы можете получить как продукт.\n\n" +
					"Он подходит, если вам нужно:\n" +
					"- быстро отвечать клиентам 24/7\n" +
					"- разгрузить менеджеров от типовых вопросов\n" +
					"- собирать заявки и контакты прямо в Telegram\n" +
					"- аккуратно подводить людей к покупке или записи.\n\n" +
					"На основе этого бота можно добавить меню, оплату, интеграцию с CRM, базу знаний и любые сценарии под ваш бизнес."
			},
		},
		"features": {
			Name:        "features",
			Description: "какие функции можно добавить (заявки, меню, запись, опросы...)",
			ParseMode:   tgbotapi.ModeMarkdown,
			BuildText: func(_ *tgbotapi.Message) string {
				return "*Какие возможности можно добавить в такого бота:*\n\n" +
					"- Меню с разделами (услуги, цены, контакты)\n" +
					"- Приём заявок: имя, телефон, комментарий → вам в чат или CRM\n" +
					"- Запись на услуги по времени (простое расписание)\n" +
					"- Опросы и быстрый сбор обратной связи\n" +
					"- Отправка файлов, инструкций, прайсов\n" +
					"- Ограниченный доступ по списку клиентов или ролям.\n\n" +
					"Текущая версия — минимальный живой пример. Все перечисленное можно добавить в этот же бот под ваши задачи."
			},
		},
		"usecases": {
			Name:        "usecases",
			Description: "примеры задач, которые можно решить ботом",
			ParseMode:   tgbotapi.ModeMarkdown,
			BuildText: func(_ *tgbotapi.Message) string {
				return renderUseCases()
			},
		},
		"ping": {
			Name:        "ping",
			Description: "проверка, что бот онлайн",
			BuildText: func(_ *tgbotapi.Message) string {
				return "pong ✅ Бот запущен и готов работать с клиентами."
			},
		},
		"echo": {
			Name:        "echo",
			Description: "повторить ваш текст (пример простой команды)",
			BuildText: func(msg *tgbotapi.Message) string {
				args := strings.TrimSpace(msg.CommandArguments())
				if args == "" {
					return "Использование: /echo <текст, который нужно повторить>"
				}
				return args
			},
		},
	}

	commands["help"] = Command{
		Name:        "help",
		Description: "это сообщение с возможностями",
		ParseMode:   tgbotapi.ModeMarkdown,
		BuildText: func(_ *tgbotapi.Message) string {
			lines := []string{
				"Я бот, который помогает автоматизировать общение с клиентами.\n",
				"*Что я умею прямо сейчас:*",
			}

			order := []string{"start", "menu", "help", "about", "usecases", "features", "ping", "echo"}
			for _, name := range order {
				c := commands[name]
				label := "/" + c.Name
				if c.Name == "echo" {
					label = "/echo <текст>"
				}
				lines = append(lines, label+" — "+c.Description)
			}

			lines = append(lines, "", "Если просто написать сообщение — я отвечу тем же текстом. Это демонстрирует, как бот может принимать и обрабатывать любые обращения клиентов.")
			return strings.Join(lines, "\n")
		},
	}

	return commands
}

func (h *Handlers) HandleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID

	if msg.IsCommand() {
		h.HandleCommand(msg)
		return
	}

	if sectionCode, ok := learningMenuButtons[strings.TrimSpace(msg.Text)]; ok {
		h.handleLearningMenuSelection(msg, sectionCode)
		return
	}

	if cmdName, ok := commandButtons[strings.TrimSpace(msg.Text)]; ok {
		h.sendCommandReply(chatID, cmdName, msg)
		return
	}

	if h.captureAnswerText(msg) {
		return
	}

	reply := tgbotapi.NewMessage(chatID, "Ты написал: "+msg.Text)
	reply.ReplyMarkup = commandKeyboard()
	if _, err := h.Bot.Send(reply); err != nil {
		h.Logger.Error("failed to send message", "err", err)
	}
}

func (h *Handlers) HandleCommand(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	h.sendCommandReply(chatID, msg.Command(), msg)
}

func (h *Handlers) sendCommandReply(chatID int64, cmdName string, msg *tgbotapi.Message) {
	cmd, ok := h.commandRegistry()[cmdName]
	if !ok {
		reply := tgbotapi.NewMessage(chatID, "Неизвестная команда. Напиши /help, чтобы узнать, что я умею.")
		reply.ReplyMarkup = commandKeyboard()
		if _, err := h.Bot.Send(reply); err != nil {
			h.Logger.Error("failed to send unknown command reply", "err", err)
		}
		return
	}

	reply := tgbotapi.NewMessage(chatID, cmd.BuildText(msg))
	if cmd.ParseMode != "" {
		reply.ParseMode = cmd.ParseMode
	}
	if cmdName == "menu" {
		inline := demoInlineMenuKeyboard()
		reply.ReplyMarkup = &inline
	} else {
		reply.ReplyMarkup = commandKeyboard()
	}
	if _, err := h.Bot.Send(reply); err != nil {
		h.Logger.Error("failed to send command reply", "cmd", cmdName, "err", err)
	}
}

// HandleCallback — нажатия на inline-кнопки (те же команды, что в основном меню).
func (h *Handlers) HandleCallback(q *tgbotapi.CallbackQuery) {
	if q == nil || q.Message == nil {
		return
	}
	data := strings.TrimSpace(q.Data)
	if strings.HasPrefix(data, "flow:") {
		h.handleFlowCallback(q, data)
		return
	}
	if !strings.HasPrefix(data, "cmd:") {
		if _, err := h.Bot.Request(tgbotapi.NewCallback(q.ID, "")); err != nil {
			h.Logger.Error("failed to answer unknown callback", "err", err)
		}
		return
	}
	cmdName := strings.TrimPrefix(data, "cmd:")
	if _, err := h.Bot.Request(tgbotapi.NewCallback(q.ID, "")); err != nil {
		h.Logger.Error("failed to answer callback", "err", err)
	}
	fake := &tgbotapi.Message{
		Chat: q.Message.Chat,
		From: q.From,
	}
	h.sendCommandReply(q.Message.Chat.ID, cmdName, fake)
}

func (h *Handlers) ensureFlowStateStore() flowStateStore {
	if h.State == nil {
		h.State = newInMemoryFlowStateStore()
	}
	return h.State
}

func (h *Handlers) captureAnswerText(msg *tgbotapi.Message) bool {
	if msg == nil {
		return false
	}

	userID := messageUserID(msg)
	if userID == 0 {
		return false
	}

	state, ok := h.ensureFlowStateStore().Get(userID)
	if !ok || state.Step != learning.StepTask || strings.TrimSpace(msg.Text) == "" {
		return false
	}

	state.LastAnswer = strings.TrimSpace(msg.Text)
	h.ensureFlowStateStore().Save(userID, state)

	reply := tgbotapi.NewMessage(msg.Chat.ID, "Ответ сохранен. Нажмите «Проверить ответ», когда будете готовы, или пропустите проверку.")
	inline := taskInlineKeyboard(state.BlockID)
	reply.ReplyMarkup = inline
	if _, err := h.Bot.Send(reply); err != nil {
		h.Logger.Error("failed to send answer saved hint", "err", err)
	}

	return true
}

func (h *Handlers) handleLearningMenuSelection(msg *tgbotapi.Message, sectionCode string) {
	switch sectionCode {
	case "introduction", "algorithms":
		if err := h.startSection(context.Background(), msg.Chat.ID, messageUserID(msg), sectionCode); err != nil {
			h.sendInstructionalMessage(msg.Chat.ID, "Не удалось открыть раздел. Попробуйте еще раз чуть позже.")
			h.Logger.Error("failed to start learning section", "section", sectionCode, "err", err)
		}
	case "random":
		h.sendInstructionalMessage(msg.Chat.ID, "Рандом задач добавим следующим шагом. Пока выберите «Алгоритмы (по порядку)».")
	case "progress":
		h.sendInstructionalMessage(msg.Chat.ID, "Прогресс появится после сохранения решений. Пока можно продолжить обучение по главам.")
	case "settings":
		h.sendInstructionalMessage(msg.Chat.ID, "Настройки пока минимальные. Скоро здесь будут язык, темп и формат объяснений.")
	default:
		h.sendInstructionalMessage(msg.Chat.ID, "Выберите раздел из меню ниже.")
	}
}

func (h *Handlers) startSection(ctx context.Context, chatID, userID int64, sectionCode string) error {
	if h.Repo == nil || h.Content == nil || h.Learning == nil {
		return fmt.Errorf("learning flow dependencies are not configured")
	}

	section, err := h.Repo.GetSectionByCode(ctx, sectionCode)
	if err != nil {
		return fmt.Errorf("get section %q: %w", sectionCode, err)
	}

	chapters, err := h.Repo.ListChaptersBySection(ctx, section.ID)
	if err != nil {
		return fmt.Errorf("list chapters for section %q: %w", sectionCode, err)
	}
	if len(chapters) == 0 {
		return fmt.Errorf("section %q has no chapters", sectionCode)
	}

	blocks, err := h.Repo.ListChapterBlocks(ctx, chapters[0].ID)
	if err != nil {
		return fmt.Errorf("list chapter blocks %d: %w", chapters[0].ID, err)
	}
	if len(blocks) == 0 {
		return fmt.Errorf("chapter %d has no blocks", chapters[0].ID)
	}

	return h.renderBlockStep(ctx, chatID, userID, blocks[0], defaultStepForBlock(blocks[0].BlockType))
}

func (h *Handlers) handleFlowCallback(q *tgbotapi.CallbackQuery, data string) {
	if _, err := h.Bot.Request(tgbotapi.NewCallback(q.ID, "")); err != nil {
		h.Logger.Error("failed to answer flow callback", "err", err)
	}

	action, blockID, ok := parseFlowCallbackData(data)
	if !ok {
		h.sendInstructionalMessage(q.Message.Chat.ID, "Не понял действие. Используйте кнопки под текущим сообщением.")
		return
	}

	if h.Repo == nil || h.Content == nil || h.Learning == nil {
		h.sendInstructionalMessage(q.Message.Chat.ID, "Учебный сценарий пока не настроен. Попробуйте позже.")
		return
	}

	userID := callbackUserID(q)
	stateStore := h.ensureFlowStateStore()
	currentBlock, err := h.Repo.GetBlock(context.Background(), blockID)
	if err != nil {
		h.Logger.Error("failed to load current block", "block_id", blockID, "err", err)
		h.sendInstructionalMessage(q.Message.Chat.ID, "Не удалось загрузить следующий шаг. Попробуйте снова.")
		return
	}

	currentStep := defaultStepForBlock(currentBlock.BlockType)
	if saved, ok := stateStore.Get(userID); ok && saved.BlockID == currentBlock.ID {
		currentStep = saved.Step
	}
	if currentBlock.BlockType == learning.BlockTask && action == learningflowservice.ActionSkipReview && currentStep == learning.StepTask {
		currentStep = learning.StepReview
	}

	nextStep := h.Learning.NextStep(currentBlock.BlockType, currentStep, action)
	if nextStep == currentStep {
		h.sendInstructionalMessage(q.Message.Chat.ID, "Это действие сейчас недоступно. Используйте кнопки под сообщением.")
		return
	}

	switch nextStep {
	case learning.StepReview:
		h.renderReviewStep(q.Message.Chat.ID, userID, currentBlock)
	case learning.StepSolution:
		nextBlock, findErr := h.findNextBlockInChapter(context.Background(), currentBlock.ChapterID, currentBlock.ID, learning.BlockSolution)
		if findErr != nil {
			h.Logger.Error("failed to find solution block", "block_id", currentBlock.ID, "err", findErr)
			h.sendInstructionalMessage(q.Message.Chat.ID, "Не нашел решение для этого шага. Попробуйте другой блок.")
			return
		}
		if err := h.renderBlockStep(context.Background(), q.Message.Chat.ID, userID, nextBlock, learning.StepSolution); err != nil {
			h.Logger.Error("failed to render solution block", "block_id", nextBlock.ID, "err", err)
			h.sendInstructionalMessage(q.Message.Chat.ID, "Не удалось показать решение. Попробуйте снова.")
		}
	case learning.StepTask:
		nextBlock, findErr := h.findNextBlockInChapter(context.Background(), currentBlock.ChapterID, currentBlock.ID, learning.BlockTheory, learning.BlockTask)
		if findErr != nil {
			h.Logger.Error("failed to find next learning block", "block_id", currentBlock.ID, "err", findErr)
			h.sendInstructionalMessage(q.Message.Chat.ID, "Следующий шаг не найден. Вернитесь в меню и откройте главу заново.")
			return
		}
		if err := h.renderBlockStep(context.Background(), q.Message.Chat.ID, userID, nextBlock, defaultStepForBlock(nextBlock.BlockType)); err != nil {
			h.Logger.Error("failed to render next learning block", "block_id", nextBlock.ID, "err", err)
			h.sendInstructionalMessage(q.Message.Chat.ID, "Не удалось открыть следующий шаг. Попробуйте снова.")
		}
	default:
		h.sendInstructionalMessage(q.Message.Chat.ID, "Используйте кнопки под сообщением, чтобы продолжить обучение.")
	}
}

func parseFlowCallbackData(data string) (learningflowservice.Action, int64, bool) {
	parts := strings.Split(strings.TrimSpace(data), ":")
	if len(parts) != 3 || parts[0] != "flow" {
		return "", 0, false
	}

	action := learningflowservice.Action(parts[1])
	switch action {
	case learningflowservice.ActionNext,
		learningflowservice.ActionSubmitAnswer,
		learningflowservice.ActionSkipAnswer,
		learningflowservice.ActionSkipReview:
	default:
		return "", 0, false
	}

	blockID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || blockID <= 0 {
		return "", 0, false
	}

	return action, blockID, true
}

func defaultStepForBlock(blockType learning.BlockType) learning.FlowStep {
	switch blockType {
	case learning.BlockTheory:
		return learning.StepTheory
	case learning.BlockSolution:
		return learning.StepSolution
	default:
		return learning.StepTask
	}
}

func messageUserID(msg *tgbotapi.Message) int64 {
	if msg != nil && msg.From != nil {
		return msg.From.ID
	}
	if msg != nil && msg.Chat != nil {
		return msg.Chat.ID
	}
	return 0
}

func callbackUserID(q *tgbotapi.CallbackQuery) int64 {
	if q != nil && q.From != nil {
		return q.From.ID
	}
	if q != nil && q.Message != nil && q.Message.Chat != nil {
		return q.Message.Chat.ID
	}
	return 0
}

func (h *Handlers) renderReviewStep(chatID, userID int64, currentBlock learning.Block) {
	state, _ := h.ensureFlowStateStore().Get(userID)
	answer := strings.TrimSpace(state.LastAnswer)
	if answer == "" {
		answer = "Ответ пока не получен. Отправьте его сообщением или переходите к разбору."
	}

	reply := tgbotapi.NewMessage(chatID, fmt.Sprintf("Промежуточная проверка ответа:\n\n%s", answer))
	reply.ParseMode = tgbotapi.ModeMarkdown
	inline := reviewInlineKeyboard(currentBlock.ID)
	reply.ReplyMarkup = inline
	if _, err := h.Bot.Send(reply); err != nil {
		h.Logger.Error("failed to send review step", "block_id", currentBlock.ID, "err", err)
		return
	}

	h.ensureFlowStateStore().Save(userID, flowState{
		BlockID:    currentBlock.ID,
		ChapterID:  currentBlock.ChapterID,
		Step:       learning.StepReview,
		LastAnswer: state.LastAnswer,
	})
}

func (h *Handlers) renderBlockStep(ctx context.Context, chatID, userID int64, block learning.Block, step learning.FlowStep) error {
	payload, err := h.Content.BuildBlockPayload(ctx, block.ID)
	if err != nil {
		return fmt.Errorf("build block payload %d: %w", block.ID, err)
	}

	text, inline := renderStepMessage(block.ID, payload, step)
	reply := tgbotapi.NewMessage(chatID, text)
	reply.ParseMode = tgbotapi.ModeMarkdown
	reply.ReplyMarkup = inline
	if _, err := h.Bot.Send(reply); err != nil {
		return fmt.Errorf("send block message %d: %w", block.ID, err)
	}

	lastAnswer := ""
	if existing, ok := h.ensureFlowStateStore().Get(userID); ok && existing.BlockID == block.ID {
		lastAnswer = existing.LastAnswer
	}

	h.ensureFlowStateStore().Save(userID, flowState{
		BlockID:    block.ID,
		ChapterID:  block.ChapterID,
		Step:       step,
		LastAnswer: lastAnswer,
	})

	return nil
}

func renderStepMessage(blockID int64, payload contentservice.BlockPayload, step learning.FlowStep) (string, tgbotapi.InlineKeyboardMarkup) {
	title := strings.TrimSpace(payload.Title)
	if title == "" {
		title = fmt.Sprintf("Блок %d", blockID)
	}

	switch step {
	case learning.StepTheory:
		text := fmt.Sprintf("*%s*\n\n%s", title, strings.TrimSpace(payload.TheoryMD))
		return strings.TrimSpace(text), theoryInlineKeyboard(blockID)
	case learning.StepSolution:
		text := fmt.Sprintf("*%s*\n\n%s", title, strings.TrimSpace(payload.SolutionMD))
		return strings.TrimSpace(text), solutionInlineKeyboard(blockID)
	default:
		text := fmt.Sprintf("*%s*\n\n%s", title, strings.TrimSpace(payload.TaskMD))
		return strings.TrimSpace(text), taskInlineKeyboard(blockID)
	}
}

func theoryInlineKeyboard(blockID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Дальше", flowCallbackData(learningflowservice.ActionNext, blockID)),
		),
	)
}

func taskInlineKeyboard(blockID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Проверить ответ", flowCallbackData(learningflowservice.ActionSubmitAnswer, blockID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Пропустить ответ", flowCallbackData(learningflowservice.ActionSkipAnswer, blockID)),
			tgbotapi.NewInlineKeyboardButtonData("Показать решение", flowCallbackData(learningflowservice.ActionSkipReview, blockID)),
		),
	)
}

func reviewInlineKeyboard(blockID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Показать решение", flowCallbackData(learningflowservice.ActionSkipReview, blockID)),
		),
	)
}

func solutionInlineKeyboard(blockID int64) tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Следующий блок", flowCallbackData(learningflowservice.ActionNext, blockID)),
		),
	)
}

func flowCallbackData(action learningflowservice.Action, blockID int64) string {
	return fmt.Sprintf("flow:%s:%d", action, blockID)
}

func (h *Handlers) findNextBlockInChapter(ctx context.Context, chapterID, currentBlockID int64, allowedTypes ...learning.BlockType) (learning.Block, error) {
	blocks, err := h.Repo.ListChapterBlocks(ctx, chapterID)
	if err != nil {
		return learning.Block{}, fmt.Errorf("list chapter blocks %d: %w", chapterID, err)
	}

	seenCurrent := false
	for _, block := range blocks {
		if block.ID == currentBlockID {
			seenCurrent = true
			continue
		}
		if !seenCurrent {
			continue
		}
		for _, allowed := range allowedTypes {
			if block.BlockType == allowed {
				return block, nil
			}
		}
	}

	return learning.Block{}, fmt.Errorf("next block after %d not found", currentBlockID)
}

func (h *Handlers) sendInstructionalMessage(chatID int64, text string) {
	reply := tgbotapi.NewMessage(chatID, text)
	reply.ReplyMarkup = commandKeyboard()
	if _, err := h.Bot.Send(reply); err != nil {
		h.Logger.Error("failed to send instructional message", "err", err)
	}
}
