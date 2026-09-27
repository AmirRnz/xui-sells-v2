package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"xui-sells-v2/internal/app/i18n"
	"xui-sells-v2/internal/app/pricing"
	"xui-sells-v2/internal/app/provisioning"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

// HandleStart handles the /start command.
// Strictly adheres to P25: Never shows a language selector unless the user explicitly executes /language.
// Handles referral link parameter /start ref_<tg_id> (P15).
func (b *BotInstance) HandleStart(ctx context.Context, tgID int64, username, firstName, lastName, text string) error {
	user := b.GetUser(tgID)
	if user == nil {
		lang := b.Instance.DefaultLang
		if lang == "" {
			lang = domain.LangFA
		}

		var referrerID *int64
		parts := strings.Fields(text)
		if len(parts) > 1 && strings.HasPrefix(parts[1], "ref_") {
			refStr := strings.TrimPrefix(parts[1], "ref_")
			if id, err := strconv.ParseInt(refStr, 10, 64); err == nil && id != tgID {
				referrerID = &id
			}
		}

		newUser := domain.User{
			InstanceID:         b.Instance.ID,
			TelegramID:         tgID,
			Username:           username,
			FirstName:          firstName,
			LastName:           lastName,
			Language:           lang,
			ReferrerTelegramID: referrerID,
		}
		b.SaveUser(newUser)
		user = &newUser
	}

	// Reseller bot mandatory first step (P16): choosing service name / group name!
	if b.Instance.IsReseller() && b.Instance.GroupName == "" {
		b.SetState(tgID, &WizardState{Step: StepResellerSetGroupName})
		msg := "Welcome! As a reseller, please enter your Service Name. This will be used as your group name in 3x-ui:"
		if user.Language == domain.LangFA {
			msg = "خوش آمدید! لطفاً نام سرویس خود را وارد کنید (این نام به عنوان نام گروه در پنل ثبت می‌شود):"
		}
		return b.Deps.Sender.SendMessage(ctx, tgID, msg, nil)
	}

	welcome := fmt.Sprintf("Welcome to %s!", b.Instance.Name)
	if user.Language == domain.LangFA {
		welcome = fmt.Sprintf("به ربات %s خوش آمدید!", b.Instance.Name)
	}

	return b.Deps.Sender.SendMessage(ctx, tgID, welcome, nil)
}

// HandleLanguage handles the explicit /language command.
// P25: This is the ONLY place where the Persian / English language selector keyboard is presented.
func (b *BotInstance) HandleLanguage(ctx context.Context, tgID int64) error {
	user := b.GetUser(tgID)
	lang := domain.LangFA
	if user != nil {
		lang = user.Language
	}

	msg := i18n.Get(lang, i18n.MsgChooseLanguage)
	keyboard := map[string]any{
		"inline_keyboard": [][]map[string]string{
			{
				{"text": "فارسی 🇮🇷", "callback_data": "set_lang_fa"},
				{"text": "English 🇬🇧", "callback_data": "set_lang_en"},
			},
		},
	}

	return b.Deps.Sender.SendMessage(ctx, tgID, msg, keyboard)
}

// SetUserLanguage applies an explicit language choice.
func (b *BotInstance) SetUserLanguage(ctx context.Context, tgID int64, lang domain.Language) error {
	user := b.GetUser(tgID)
	if user == nil {
		user = &domain.User{
			InstanceID: b.Instance.ID,
			TelegramID: tgID,
			Language:   lang,
		}
	} else {
		user.Language = lang
	}
	b.SaveUser(*user)

	ack := "Language changed to English."
	if lang == domain.LangFA {
		ack = "زبان به فارسی تغییر یافت."
	}
	return b.Deps.Sender.SendMessage(ctx, tgID, ack, nil)
}

// StartPurchaseWizard initiates the multi-step buying flow (P03).
func (b *BotInstance) StartPurchaseWizard(ctx context.Context, tgID int64, planID int64) error {
	plan, ok := b.GetPlan(planID)
	if !ok {
		return b.Deps.Sender.SendMessage(ctx, tgID, "Selected plan not found.", nil)
	}

	b.SetState(tgID, &WizardState{
		Step:           StepBuySelectDuration,
		SelectedPlanID: plan.ID,
	})

	user := b.GetUser(tgID)
	msg := fmt.Sprintf("Selected: %s\nPlease select duration in months:", plan.Name)
	if user != nil && user.Language == domain.LangFA {
		msg = fmt.Sprintf("پلن انتخابی: %s\nلطفاً مدت زمان (ماه) را انتخاب کنید:", plan.Name)
	}

	return b.Deps.Sender.SendMessage(ctx, tgID, msg, nil)
}

// ProcessPurchaseStep handles sequential steps in the purchase wizard (P03).
func (b *BotInstance) ProcessPurchaseStep(ctx context.Context, tgID int64, input string) error {
	state := b.GetState(tgID)
	plan, ok := b.GetPlan(state.SelectedPlanID)
	if !ok {
		b.ClearState(tgID)
		return b.Deps.Sender.SendMessage(ctx, tgID, "Session expired, please select plan again.", nil)
	}

	user := b.GetUser(tgID)
	lang := domain.LangFA
	if user != nil {
		lang = user.Language
	}

	switch state.Step {
	case StepBuySelectDuration:
		dur, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || dur <= 0 {
			return b.Deps.Sender.SendMessage(ctx, tgID, "Please enter a valid duration in months (e.g. 1, 2, 3):", nil)
		}
		state.DurationMonths = dur
		state.Step = StepBuyEnterUsers
		b.SetState(tgID, state)

		prompt := fmt.Sprintf("Enter number of users/devices (1 to %d):", plan.MaxUsers)
		if lang == domain.LangFA {
			prompt = fmt.Sprintf("تعداد کاربر / دستگاه را وارد کنید (۱ تا %d):", plan.MaxUsers)
		}
		return b.Deps.Sender.SendMessage(ctx, tgID, prompt, nil)

	case StepBuyEnterUsers:
		users, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || users < 1 || (plan.MaxUsers > 0 && users > plan.MaxUsers) {
			return b.Deps.Sender.SendMessage(ctx, tgID, fmt.Sprintf("Invalid user count. Must be 1 to %d:", plan.MaxUsers), nil)
		}
		state.UserCount = users
		state.Step = StepBuyEnterName
		b.SetState(tgID, state)

		prompt := "Enter custom name for this subscription (or type 'random'):"
		if lang == domain.LangFA {
			prompt = "یک نام برای این اشتراک وارد کنید (یا بنویسید 'تصادفی'):"
		}
		return b.Deps.Sender.SendMessage(ctx, tgID, prompt, nil)

	case StepBuyEnterName:
		name := strings.TrimSpace(input)
		if strings.ToLower(name) == "random" || name == "تصادفی" || name == "" {
			name = provisioning.GenerateSubID()[:8]
		}
		state.SubName = name

		// Calculate exact quote using pricing engine (Single source of truth!)
		quote, err := pricing.CalculateQuote(plan, state.UserCount, state.DurationMonths)
		if err != nil {
			return b.Deps.Sender.SendMessage(ctx, tgID, fmt.Sprintf("Pricing error: %s", err.Error()), nil)
		}

		state.Step = StepBuySelectPayment
		b.SetState(tgID, state)

		totalFormatted := money.FormatCurrency(quote.Total, string(lang))
		summary := fmt.Sprintf("Order Summary:\nPlan: %s\nDuration: %d months\nUsers: %d\nTotal Payable: %s\n\nChoose payment method: 'wallet' or 'card'",
			plan.Name, state.DurationMonths, state.UserCount, totalFormatted)

		if lang == domain.LangFA {
			summary = fmt.Sprintf("خلاصه سفارش:\nپلن: %s\nمدت زمان: %d ماه\nتعداد کاربر: %d\nمبلغ قابل پرداخت: %s\n\nروش پرداخت را انتخاب کنید: 'کیف پول' یا 'کارت'",
				plan.Name, state.DurationMonths, state.UserCount, totalFormatted)
		}
		return b.Deps.Sender.SendMessage(ctx, tgID, summary, nil)

	case StepBuySelectPayment:
		method := strings.ToLower(strings.TrimSpace(input))

		quote, _ := pricing.CalculateQuote(plan, state.UserCount, state.DurationMonths)

		if method == "card" || method == "کارت" || method == "direct" {
			// Direct Card Payment with structured Persian bank card layout (P09)
			prompt := i18n.BankCardMessage(lang, b.Instance.CardNumber, b.Instance.CardHolder, quote.Total)

			state.Step = StepBuySubmitReceipt
			b.SetState(tgID, state)
			return b.Deps.Sender.SendMessage(ctx, tgID, prompt, nil)
		}

		if method == "wallet" || method == "کیف پول" {
			return b.ExecuteWalletCheckout(ctx, tgID, plan, state, quote.Total)
		}

		return b.Deps.Sender.SendMessage(ctx, tgID, "Please type 'wallet' or 'card':", nil)

	case StepBuySubmitReceipt:
		// Save order pending administrator approval (P04)
		quote, _ := pricing.CalculateQuote(plan, state.UserCount, state.DurationMonths)
		orderRecord := domain.Order{
			InstanceID:     b.Instance.ID,
			UserID:         user.ID,
			TelegramID:     tgID,
			PlanID:         &plan.ID,
			Type:           domain.OrderTypePurchase,
			DurationMonths: state.DurationMonths,
			UserCount:      state.UserCount,
			Amount:         quote.Total,
			PaymentMethod:  domain.PaymentMethodDirect,
			ReceiptNotes:   input,
			Status:         domain.OrderStatusPendingApproval,
		}

		if b.Deps.OrderSvc != nil {
			_, _ = b.Deps.OrderSvc.CreateOrder(ctx, orderRecord)
		}

		b.ClearState(tgID)
		ack := "Receipt submitted! Your order is pending administrator approval."
		if lang == domain.LangFA {
			ack = "رسید شما ثبت شد! سفارش در انتظار بررسی و تایید مدیر می‌باشد."
		}
		return b.Deps.Sender.SendMessage(ctx, tgID, ack, nil)
	}

	return nil
}

// ExecuteWalletCheckout processes wallet payment and handles shared reseller credit on child bots (P20).
func (b *BotInstance) ExecuteWalletCheckout(ctx context.Context, tgID int64, plan domain.Plan, state *WizardState, total money.Money) error {
	user := b.GetUser(tgID)
	lang := b.Instance.DefaultLang
	if user != nil {
		lang = user.Language
	}

	// Shared Reseller Credit Check (P20 & D06):
	// If this is a child customer bot, reseller wallet in parent instance must fund wholesale cost!
	if b.Instance.IsChild() && b.Instance.ParentInstanceID != nil {
		parentID := *b.Instance.ParentInstanceID
		wholesaleAmount := plan.BasePrice // Or wholesale rate

		if b.Deps.WalletSvc != nil {
			_, err := b.Deps.WalletSvc.ReserveResellerCredit(ctx, parentID, b.Instance.ID, b.Instance.AdminTelegramID, b.Instance.AdminTelegramID, 0, wholesaleAmount)
			if err != nil {
				// P20: Customer receives error, Reseller is notified
				customerErr := "Insufficient provider credit. Please contact support."
				if lang == domain.LangFA {
					customerErr = "اعتبار ارائه‌دهنده ناکافی است. لطفاً با پشتیبانی تماس بگیرید."
				}
				_ = b.Deps.Sender.SendMessage(ctx, tgID, customerErr, nil)

				// Notify reseller
				resellerAlert := fmt.Sprintf("⚠️ Alert: Insufficient wallet balance to fund child bot customer order (%s). Please top up your reseller wallet.", b.Instance.Name)
				_ = b.Deps.Sender.SendMessage(ctx, b.Instance.AdminTelegramID, resellerAlert, nil)

				b.ClearState(tgID)
				return err
			}
		}
	}

	// Debit customer wallet
	if b.Deps.WalletSvc != nil {
		_, err := b.Deps.WalletSvc.Debit(ctx, b.Instance.ID, user.ID, tgID, total, domain.TxTypePurchase, "checkout", "VPN subscription purchase")
		if err != nil {
			errText := "Insufficient wallet balance. Please top up your wallet or pay via bank card."
			if lang == domain.LangFA {
				errText = "موجودی کیف پول شما کافی نیست. لطفاً کیف پول را شارژ کنید یا از پرداخت کارتی استفاده نمایید."
			}
			return b.Deps.Sender.SendMessage(ctx, tgID, errText, nil)
		}
	}

	// Provision immediately upon wallet payment approval
	b.ClearState(tgID)
	return b.ProvisionAndDeliver(ctx, tgID, plan, state.DurationMonths, state.UserCount, state.SubName)
}

// RequestFreeTrial processes ordinary customer free trial requests (P02, P18).
func (b *BotInstance) RequestFreeTrial(ctx context.Context, tgID int64, plan domain.Plan, tier domain.TierVariant) error {
	user := b.GetUser(tgID)
	lang := b.Instance.DefaultLang
	if user != nil {
		lang = user.Language
	}

	instIDStr := strconv.FormatInt(b.Instance.ID, 10)

	// Check eligibility
	if b.Deps.TrialSvc != nil {
		if err := b.Deps.TrialSvc.ConsumeTrial(ctx, instIDStr, tgID, plan, tier); err != nil {
			errMsg := fmt.Sprintf("Trial not available: %s", err.Error())
			if lang == domain.LangFA {
				errMsg = "شما در حال حاضر مجاز به دریافت تست رایگان نیستید (محدودیت تعداد یا زمان انتظار فعال است)."
			}
			return b.Deps.Sender.SendMessage(ctx, tgID, errMsg, nil)
		}
	}

	return b.ProvisionAndDeliver(ctx, tgID, plan, 0, 1, "Trial")
}

// ProvisionAndDeliver creates 3x-ui client, generates QR code, and sends credentials to user.
func (b *BotInstance) ProvisionAndDeliver(ctx context.Context, tgID int64, plan domain.Plan, durationMonths, userCount int, subName string) error {
	user := b.GetUser(tgID)
	lang := b.Instance.DefaultLang
	if user != nil {
		lang = user.Language
	}

	if b.Deps.ProvSvc != nil {
		req := provisioning.ProvisionRequest{
			InstanceID:     b.Instance.ID,
			TelegramID:     tgID,
			PlanID:         plan.ID,
			GroupName:      b.Instance.GroupName,
			InboundIDs:     plan.InboundIDs,
			TotalBytes:     plan.TrafficBytes,
			LimitIP:        userCount,
			DurationMonths: durationMonths,
			PanelURL:       b.Instance.PanelURL,
		}

		svc, qrBytes, err := b.Deps.ProvSvc.ProvisionNewSubscription(ctx, req)
		if err != nil {
			return b.Deps.Sender.SendMessage(ctx, tgID, fmt.Sprintf("Provisioning error: %s", err.Error()), nil)
		}

		subURL := svc.SubID
		caption := fmt.Sprintf("✅ Subscription Active!\nPlan: %s\nUsers: %d\nSub ID: %s", plan.Name, userCount, subURL)
		if lang == domain.LangFA {
			caption = fmt.Sprintf("✅ اشتراک شما فعال شد!\nپلن: %s\nتعداد کاربر: %d\nشناسه: %s", plan.Name, userCount, subURL)
		}

		if qrBytes != nil {
			return b.Deps.Sender.SendPhoto(ctx, tgID, qrBytes, caption, nil)
		}
		return b.Deps.Sender.SendMessage(ctx, tgID, caption, nil)
	}

	return nil
}

// HandleDecreaseUserCount enforces P05 & Contract Section 5:
// Lowering user count automatically rotates subId, invalidates previous link upstream, and delivers new link + QR.
func (b *BotInstance) HandleDecreaseUserCount(ctx context.Context, tgID int64, service domain.Service, newCount int) error {
	user := b.GetUser(tgID)
	lang := b.Instance.DefaultLang
	if user != nil {
		lang = user.Language
	}

	if b.Deps.ProvSvc != nil {
		updatedSvc, rotated, qrBytes, err := b.Deps.ProvSvc.ChangeUserCount(ctx, service, newCount, b.Instance.PanelURL)
		if err != nil {
			return b.Deps.Sender.SendMessage(ctx, tgID, fmt.Sprintf("Error updating user count: %s", err.Error()), nil)
		}

		if rotated {
			warning := fmt.Sprintf("⚠️ Notice: User count reduced to %d. Your previous subscription link is now INVALID.\nHere is your new subscription link:\n%s", newCount, updatedSvc.SubID)
			if lang == domain.LangFA {
				warning = fmt.Sprintf("⚠️ توجه: تعداد کاربر به %d کاهش یافت. لینک اشتراک قبلی شما باطل شد.\nلینک اشتراک و بارکد جدید شما:", newCount)
			}

			if qrBytes != nil {
				return b.Deps.Sender.SendPhoto(ctx, tgID, qrBytes, warning, nil)
			}
			return b.Deps.Sender.SendMessage(ctx, tgID, warning, nil)
		}
	}
	return nil
}
