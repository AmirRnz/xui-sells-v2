package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"xui-sells-v2/internal/domain"
)

// Update represents an incoming Telegram update.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from,omitempty"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text,omitempty"`
	Date      int64  `json:"date"`
}

// CallbackQuery represents an incoming callback query from an inline keyboard.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

// User represents a Telegram user.
type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type updatesResponse struct {
	OK     bool     `json:"ok"`
	Result []Update `json:"result"`
}

// HTTPSender implements Sender via direct Telegram Bot API HTTP calls.
type HTTPSender struct {
	token      string
	httpClient *http.Client
	apiBase    string
}

// NewHTTPSender constructs an HTTP sender for a given Telegram bot token.
func NewHTTPSender(token string) *HTTPSender {
	return &HTTPSender{
		token: token,
		httpClient: &http.Client{
			Timeout: 35 * time.Second,
		},
		apiBase: "https://api.telegram.org/bot" + token,
	}
}

// SendMessage sends a text message with optional reply markup.
func (s *HTTPSender) SendMessage(ctx context.Context, chatID int64, text string, replyMarkup any) error {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	if replyMarkup != nil {
		payload["reply_markup"] = replyMarkup
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal sendMessage payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiBase+"/sendMessage", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send message to Telegram: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// SendPhoto sends a photo with caption and optional reply markup.
func (s *HTTPSender) SendPhoto(ctx context.Context, chatID int64, photo []byte, caption string, replyMarkup any) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	_ = writer.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	if caption != "" {
		_ = writer.WriteField("caption", caption)
	}
	if replyMarkup != nil {
		if rmData, err := json.Marshal(replyMarkup); err == nil {
			_ = writer.WriteField("reply_markup", string(rmData))
		}
	}

	part, err := writer.CreateFormFile("photo", "qr.png")
	if err == nil {
		_, _ = part.Write(photo)
	}
	_ = writer.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiBase+"/sendPhoto", &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send photo to Telegram: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// AnswerCallbackQuery answers an inline keyboard callback query.
func (s *HTTPSender) AnswerCallbackQuery(ctx context.Context, callbackQueryID string, text string) error {
	payload := map[string]any{
		"callback_query_id": callbackQueryID,
	}
	if text != "" {
		payload["text"] = text
	}

	data, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiBase+"/answerCallbackQuery", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// GetUpdates polls Telegram for updates with long-polling timeout.
func (s *HTTPSender) GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=%d", s.apiBase, offset, timeoutSeconds)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ur updatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&ur); err != nil {
		return nil, err
	}
	return ur.Result, nil
}

// DispatchUpdate routes an incoming Telegram update to the appropriate handler.
func (b *BotInstance) DispatchUpdate(ctx context.Context, u Update) error {
	if u.Message != nil && u.Message.From != nil {
		tgID := u.Message.From.ID
		text := strings.TrimSpace(u.Message.Text)
		username := u.Message.From.Username
		firstName := u.Message.From.FirstName
		lastName := u.Message.From.LastName

		// 1. Core navigation commands
		if text == "/start" || strings.HasPrefix(text, "/start ") {
			return b.HandleStart(ctx, tgID, username, firstName, lastName, text)
		}
		if text == "/language" {
			return b.HandleLanguage(ctx, tgID)
		}
		if text == "/admin" || strings.HasPrefix(text, "/admin") {
			return b.HandleAdmin(ctx, tgID)
		}

		// 2. Active Wizard State Handling
		state := b.GetState(tgID)
		if state != nil && state.Step != StepNone {
			if b.Instance.IsReseller() {
				return b.ProcessResellerStep(ctx, tgID, text)
			}
			return b.ProcessPurchaseStep(ctx, tgID, text)
		}

		// 3. Reseller Onboarding / Menu
		if b.Instance.IsReseller() {
			if b.Instance.GroupName == "" {
				return b.HandleStart(ctx, tgID, username, firstName, lastName, text)
			}
			menuMsg := fmt.Sprintf("🌟 %s Reseller Panel\nService Group: %s\n\nCommands:\n/admin - Administrator Panel\n/language - Change Language", b.Instance.Name, b.Instance.GroupName)
			if b.Instance.DefaultLang == domain.LangFA {
				menuMsg = fmt.Sprintf("🌟 پنل نمایندگی %s\nنام گروه: %s\n\nدستورات:\n/admin - پنل مدیریت\n/language - تغییر زبان", b.Instance.Name, b.Instance.GroupName)
			}
			return b.Deps.Sender.SendMessage(ctx, tgID, menuMsg, nil)
		}

		// 4. Customer Default
		return b.HandleStart(ctx, tgID, username, firstName, lastName, text)
	}

	if u.CallbackQuery != nil {
		tgID := u.CallbackQuery.From.ID
		data := u.CallbackQuery.Data

		if b.Deps.Sender != nil {
			_ = b.Deps.Sender.AnswerCallbackQuery(ctx, u.CallbackQuery.ID, "")
		}

		switch {
		case data == "set_lang_fa":
			return b.SetUserLanguage(ctx, tgID, domain.LangFA)
		case data == "set_lang_en":
			return b.SetUserLanguage(ctx, tgID, domain.LangEN)
		case data == "admin_menu":
			return b.HandleAdmin(ctx, tgID)
		case strings.HasPrefix(data, "buy_plan_"):
			planIDStr := strings.TrimPrefix(data, "buy_plan_")
			if id, err := strconv.ParseInt(planIDStr, 10, 64); err == nil {
				return b.StartPurchaseWizard(ctx, tgID, id)
			}
		}
	}

	return nil
}

// StartPolling runs a persistent long-polling loop for the bot.
func (b *BotInstance) StartPolling(ctx context.Context) {
	sender, ok := b.Deps.Sender.(*HTTPSender)
	if !ok || sender == nil {
		return
	}

	var offset int64 = 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := sender.GetUpdates(ctx, offset, 20)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				continue
			}
		}

		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			_ = b.DispatchUpdate(ctx, u)
		}
	}
}
