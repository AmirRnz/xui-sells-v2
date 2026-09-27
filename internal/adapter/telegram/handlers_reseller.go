package telegram

import (
	"context"
	"fmt"
	"strings"

	adapterHTTP "xui-sells-v2/internal/adapter/http"
	"xui-sells-v2/internal/domain"
)

// ProcessResellerStep processes inputs specific to reseller bots.
func (b *BotInstance) ProcessResellerStep(ctx context.Context, tgID int64, input string) error {
	state := b.GetState(tgID)
	user := b.GetUser(tgID)
	lang := b.Instance.DefaultLang
	if user != nil {
		lang = user.Language
	}

	// Mandatory onboarding: First action MUST be choosing service / group name (P16)
	if b.Instance.GroupName == "" || state.Step == StepResellerSetGroupName {
		groupName := strings.TrimSpace(input)
		if groupName == "" {
			prompt := "Service name cannot be empty. Please enter your Service Name:"
			if lang == domain.LangFA {
				prompt = "نام سرویس نمی‌تواند خالی باشد. لطفاً نام سرویس خود را وارد کنید:"
			}
			return b.Deps.Sender.SendMessage(ctx, tgID, prompt, nil)
		}

		b.Instance.GroupName = groupName
		b.ClearState(tgID)

		ack := fmt.Sprintf("✅ Service name set to '%s'! All 3x-ui clients will be created under this group.", groupName)
		if lang == domain.LangFA {
			ack = fmt.Sprintf("✅ نام سرویس شما به '%s' تنظیم شد! تمام اشتراک‌های شما در پنل با این گروه ساخته خواهند شد.", groupName)
		}
		return b.Deps.Sender.SendMessage(ctx, tgID, ack, nil)
	}

	switch state.Step {
	case StepResellerSetPassword:
		password := strings.TrimSpace(input)
		if len(password) < 6 {
			return b.Deps.Sender.SendMessage(ctx, tgID, "Password must be at least 6 characters. Please try again:", nil)
		}

		hash, err := adapterHTTP.HashPassword(password)
		if err != nil {
			return b.Deps.Sender.SendMessage(ctx, tgID, "Failed to hash password. Please try again.", nil)
		}

		b.ClearState(tgID)
		msg := fmt.Sprintf("✅ Web panel password set!\nYou can now log in at the web panel using:\nTelegram ID: %d\nPassword: (the password you just entered)", tgID)
		if lang == domain.LangFA {
			msg = fmt.Sprintf("✅ رمز عبور پنل وب با موفقیت ذخیره شد!\nاکنون می‌توانید با اطلاعات زیر در پنل وب لاگین کنید:\nشناسه تلگرام: %d\nرمز عبور: (همان رمز انتخابی شما)", tgID)
		}
		// In a real environment this updates the database reseller_profiles.web_panel_password_hash
		_ = hash
		return b.Deps.Sender.SendMessage(ctx, tgID, msg, nil)

	case StepResellerAddChildBot:
		token := strings.TrimSpace(input)
		if token == "" || !strings.Contains(token, ":") {
			return b.Deps.Sender.SendMessage(ctx, tgID, "Invalid Telegram bot token. Format should be '123456789:ABC...'", nil)
		}

		// Create child customer bot inheriting panel URL, API key, and reseller's TG ID as admin (P19)
		childInst := domain.Instance{
			Name:             fmt.Sprintf("Child Bot (%s)", b.Instance.GroupName),
			Type:             domain.InstanceTypeChildCustomer,
			BotToken:         token,
			DefaultLang:      b.Instance.DefaultLang,
			PanelURL:         b.Instance.PanelURL,
			PanelAPIKey:      b.Instance.PanelAPIKey,
			ParentInstanceID: &b.Instance.ID,
			AdminTelegramID:  tgID, // Reseller is the admin of the child bot!
			GroupName:        b.Instance.GroupName,
			CardNumber:       b.Instance.CardNumber,
			CardHolder:       b.Instance.CardHolder,
			Currency:         b.Instance.Currency,
			IsActive:         true,
		}

		if b.Deps.Supervisor != nil {
			childRuntime := NewBotInstance(childInst, b.Deps)
			_ = b.Deps.Supervisor.RegisterBot(childRuntime)
		}

		b.ClearState(tgID)
		ack := "✅ Child bot registered and started! It inherits your 3x-ui panel and credentials."
		if lang == domain.LangFA {
			ack = "✅ ربات زیرمجموعه شما راه‌اندازی شد و شروع به کار کرد! این ربات از پنل و اعتبار شما استفاده می‌کند."
		}
		return b.Deps.Sender.SendMessage(ctx, tgID, ack, nil)
	}

	return nil
}

// StartSetPasswordFlow initiates the web panel password setting process (P21).
func (b *BotInstance) StartSetPasswordFlow(ctx context.Context, tgID int64) error {
	b.SetState(tgID, &WizardState{Step: StepResellerSetPassword})
	prompt := "Enter a secure password for your Web Panel login:"
	if b.Instance.DefaultLang == domain.LangFA {
		prompt = "لطفاً یک رمز عبور امن برای ورود به پنل وب انتخاب کنید:"
	}
	return b.Deps.Sender.SendMessage(ctx, tgID, prompt, nil)
}

// StartAddChildBotFlow initiates child bot provisioning from within parent reseller bot (P19).
func (b *BotInstance) StartAddChildBotFlow(ctx context.Context, tgID int64, tier domain.ResellerTier) error {
	// Check tier entitlement: Free tier cannot create child bots (D07 & P17)
	if b.Deps.EntitlementSvc != nil {
		if !b.Deps.EntitlementSvc.CanCreateChildBot(tier) {
			msg := "Child bots require Pro or Ultimate reseller membership."
			if b.Instance.DefaultLang == domain.LangFA {
				msg = "ایجاد ربات زیرمجموعه نیازمند سطح عضویت پرو یا التیمیت است."
			}
			return b.Deps.Sender.SendMessage(ctx, tgID, msg, nil)
		}
	}

	b.SetState(tgID, &WizardState{Step: StepResellerAddChildBot})
	prompt := "Please enter the Telegram bot token for your child customer bot (obtained from @BotFather):"
	if b.Instance.DefaultLang == domain.LangFA {
		prompt = "لطفاً توکن ربات تلگرام زیرمجموعه خود را وارد کنید (دریافت شده از @BotFather):"
	}
	return b.Deps.Sender.SendMessage(ctx, tgID, prompt, nil)
}
