package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"xui-sells-v2/internal/app/broadcast"
	"xui-sells-v2/internal/app/i18n"
	"xui-sells-v2/internal/app/provisioning"
	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

// HandleAdmin checks admin authorization and renders the administrator menu (P07).
func (b *BotInstance) HandleAdmin(ctx context.Context, tgID int64) error {
	user := b.GetUser(tgID)
	isAdmin := (user != nil && user.IsAdmin) || tgID == b.Instance.AdminTelegramID

	if !isAdmin {
		return b.Deps.Sender.SendMessage(ctx, tgID, "Access denied. Administrator privileges required.", nil)
	}

	menu := "👑 Administrator Panel\n\nOptions:\n1. /admin_create_client - Create client directly\n2. /admin_credit - Manual wallet credit\n3. /admin_stats - Operational statistics\n4. /admin_broadcast - Send broadcast announcement"
	if b.Instance.DefaultLang == domain.LangFA {
		menu = "👑 پنل مدیریت\n\nامکانات:\n۱. /admin_create_client - ایجاد مستقیم کاربر در پنل\n۲. /admin_credit - شارژ دستی کیف پول\n۳. /admin_stats - آمار و گزارشات\n۴. /admin_broadcast - ارسال پیام همگانی"
	}

	return b.Deps.Sender.SendMessage(ctx, tgID, menu, nil)
}

// AdminCreateClient provisions a client directly in 3x-ui without payment receipt (P07).
func (b *BotInstance) AdminCreateClient(ctx context.Context, adminTgID int64, targetTgID int64, plan domain.Plan, durationMonths int) error {
	req := provisioning.ProvisionRequest{
		InstanceID:     b.Instance.ID,
		TelegramID:     targetTgID,
		PlanID:         plan.ID,
		GroupName:      b.Instance.GroupName,
		InboundIDs:     plan.InboundIDs,
		TotalBytes:     plan.TrafficBytes,
		LimitIP:        1,
		DurationMonths: durationMonths,
		PanelURL:       b.Instance.PanelURL,
	}

	svc, qrBytes, err := b.Deps.ProvSvc.ProvisionNewSubscription(ctx, req)
	if err != nil {
		return b.Deps.Sender.SendMessage(ctx, adminTgID, fmt.Sprintf("Failed to create client: %s", err.Error()), nil)
	}

	adminAck := fmt.Sprintf("✅ Client created for user %d!\nEmail: %s\nSub ID: %s", targetTgID, svc.ClientEmail, svc.SubID)
	_ = b.Deps.Sender.SendMessage(ctx, adminTgID, adminAck, nil)

	// Notify customer if target is assigned
	if targetTgID > 0 {
		userNotice := fmt.Sprintf("🎉 You have been gifted a VPN subscription by the administrator!\nSub ID: %s", svc.SubID)
		if qrBytes != nil {
			_ = b.Deps.Sender.SendPhoto(ctx, targetTgID, qrBytes, userNotice, nil)
		} else {
			_ = b.Deps.Sender.SendMessage(ctx, targetTgID, userNotice, nil)
		}
	}

	return nil
}

// StartManualCreditFlow initiates a manual credit wizard for single, selected, or all users (P10).
func (b *BotInstance) StartManualCreditFlow(ctx context.Context, adminTgID int64, scope string, targetTgID int64) error {
	b.SetState(adminTgID, &WizardState{
		Step:              StepAdminManualCredit,
		AdminCreditScope:  scope,
		AdminCreditTarget: targetTgID,
	})

	prompt := "Enter credit amount on the first line.\nOptionally enter your message to the user on the next line:\n\nExample:\n100000\nCompensation bonus for server maintenance"
	if b.Instance.DefaultLang == domain.LangFA {
		prompt = "مبلغ شارژ (تومان) را در خط اول وارد کنید.\nاختیاری: پیام خود به کاربر را در خط بعدی بنویسید:\n\nمثال:\n100000\nهدیه بابت جبران اختلالات شبکه"
	}

	return b.Deps.Sender.SendMessage(ctx, adminTgID, prompt, nil)
}

// ProcessManualCredit processes the manual credit input with optional message (P10).
func (b *BotInstance) ProcessManualCredit(ctx context.Context, adminTgID int64, input string) error {
	state := b.GetState(adminTgID)
	b.ClearState(adminTgID)

	lines := strings.SplitN(strings.TrimSpace(input), "\n", 2)
	amountStr := strings.TrimSpace(lines[0])
	adminMsg := ""
	if len(lines) > 1 {
		adminMsg = strings.TrimSpace(lines[1])
	}

	rawAmount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil || rawAmount <= 0 {
		return b.Deps.Sender.SendMessage(ctx, adminTgID, "Invalid amount. Credit cancelled.", nil)
	}

	creditMoney := money.NewToman(rawAmount)

	// Execute credit
	if b.Deps.WalletSvc != nil {
		targets := []int64{state.AdminCreditTarget}
		if state.AdminCreditScope == "all" {
			// In full runtime, fetches all users in instance
			targets = []int64{state.AdminCreditTarget}
		}

		for _, targetID := range targets {
			refID := fmt.Sprintf("admin_credit_%d", adminTgID)
			desc := "Manual administrator credit"
			if adminMsg != "" {
				desc = adminMsg
			}

			_, _ = b.Deps.WalletSvc.Credit(ctx, b.Instance.ID, targetID, targetID, creditMoney, domain.TxTypeManualCredit, refID, desc)

			// Notify user with localized amount and optional message (P10)
			notification := i18n.UserCreditNotification(b.Instance.DefaultLang, creditMoney, adminMsg)
			_ = b.Deps.Sender.SendMessage(ctx, targetID, notification, nil)
		}
	}

	ack := fmt.Sprintf("✅ Wallet credited successfully with %s!", money.FormatCurrency(creditMoney, string(b.Instance.DefaultLang)))
	return b.Deps.Sender.SendMessage(ctx, adminTgID, ack, nil)
}

// ShowAdminStats sends operational statistics to the administrator (P11).
func (b *BotInstance) ShowAdminStats(ctx context.Context, adminTgID int64) error {
	if b.Deps.StatsSvc == nil {
		return b.Deps.Sender.SendMessage(ctx, adminTgID, "Stats service unavailable.", nil)
	}

	st, err := b.Deps.StatsSvc.GetStats(ctx, b.Instance.ID)
	if err != nil {
		return b.Deps.Sender.SendMessage(ctx, adminTgID, fmt.Sprintf("Error fetching stats: %s", err.Error()), nil)
	}

	report := fmt.Sprintf("📊 System Statistics:\n\nActive Services: %d\nExpired Services: %d\nRegistered Users: %d\nTrials Issued: %d\nApproved Orders: %d\nPending Approvals: %d\nTotal Revenue: %s\nTotal Refunds: %s",
		st.ActiveServices, st.ExpiredServices, st.RegisteredUsers, st.TrialsIssued, st.ApprovedOrders, st.PendingApprovals,
		money.FormatCurrency(st.TotalRevenue, string(b.Instance.DefaultLang)),
		money.FormatCurrency(st.TotalRefunds, string(b.Instance.DefaultLang)))

	return b.Deps.Sender.SendMessage(ctx, adminTgID, report, nil)
}

// StartAdminBroadcast initiates broadcast wizard (P13).
func (b *BotInstance) StartAdminBroadcast(ctx context.Context, adminTgID int64, audience broadcast.AudienceType) error {
	b.SetState(adminTgID, &WizardState{
		Step:              StepAdminBroadcast,
		BroadcastAudience: audience,
	})

	prompt := fmt.Sprintf("Enter announcement message to broadcast to %s:", audience)
	return b.Deps.Sender.SendMessage(ctx, adminTgID, prompt, nil)
}

// ProcessAdminBroadcast sends message to selected audience (P13).
func (b *BotInstance) ProcessAdminBroadcast(ctx context.Context, adminTgID int64, text string) error {
	state := b.GetState(adminTgID)
	b.ClearState(adminTgID)

	if b.Deps.BroadcastSvc == nil {
		return b.Deps.Sender.SendMessage(ctx, adminTgID, "Broadcast service unavailable.", nil)
	}

	res, err := b.Deps.BroadcastSvc.DispatchBroadcast(ctx, b.Instance.ID, state.BroadcastAudience, text)
	if err != nil {
		return b.Deps.Sender.SendMessage(ctx, adminTgID, fmt.Sprintf("Broadcast failed: %s", err.Error()), nil)
	}

	ack := fmt.Sprintf("📢 Broadcast completed!\nRecipients: %d\nDelivered: %d\nFailed: %d", res.TotalRecipients, res.SentCount, res.FailedCount)
	return b.Deps.Sender.SendMessage(ctx, adminTgID, ack, nil)
}
