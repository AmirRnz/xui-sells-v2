package i18n

import (
	"fmt"
	"strings"

	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

// MessageKey represents a localized string key.
type MessageKey string

const (
	MsgWelcomeCustomer           MessageKey = "welcome_customer"
	MsgWelcomeReseller           MessageKey = "welcome_reseller"
	MsgChooseLanguage            MessageKey = "choose_language"
	MsgLanguageUpdated           MessageKey = "language_updated"
	MsgMainMenu                  MessageKey = "main_menu"
	MsgBtnBuyService             MessageKey = "btn_buy_service"
	MsgBtnFreeTrial              MessageKey = "btn_free_trial"
	MsgBtnMyServices             MessageKey = "btn_my_services"
	MsgBtnWallet                 MessageKey = "btn_wallet"
	MsgBtnSupport                MessageKey = "btn_support"
	MsgBtnInviteFriends          MessageKey = "btn_invite_friends"
	MsgBtnWebPanel               MessageKey = "btn_web_panel"
	MsgBtnManageChildBot         MessageKey = "btn_manage_child_bot"
	MsgBtnMembership             MessageKey = "btn_membership"
	MsgBtnAdminMenu              MessageKey = "btn_admin_menu"
	MsgSelectPlan                MessageKey = "select_plan"
	MsgSelectDuration            MessageKey = "select_duration"
	MsgEnterUserCount            MessageKey = "enter_user_count"
	MsgEnterServiceName          MessageKey = "enter_service_name"
	MsgRandomNameOption          MessageKey = "random_name_option"
	MsgOrderSummary              MessageKey = "order_summary"
	MsgChoosePayment             MessageKey = "choose_payment"
	MsgPayWallet                 MessageKey = "pay_wallet"
	MsgPayDirect                 MessageKey = "pay_direct"
	MsgInsufficientWallet        MessageKey = "insufficient_wallet"
	MsgInsufficientResellerCredit MessageKey = "insufficient_reseller_credit"
	MsgSendReceiptPrompt         MessageKey = "send_receipt_prompt"
	MsgReceiptReceived           MessageKey = "receipt_received"
	MsgOrderApproved             MessageKey = "order_approved"
	MsgOrderRejected             MessageKey = "order_rejected"
	MsgSubDelivery               MessageKey = "sub_delivery"
	MsgTrialIssued               MessageKey = "trial_issued"
	MsgTrialCooldownError        MessageKey = "trial_cooldown_error"
	MsgTrialLimitError           MessageKey = "trial_limit_error"
	MsgUserCountDecreasedWarning MessageKey = "user_count_decreased_warning"
	MsgResellerOnboardingPrompt   MessageKey = "reseller_onboarding_prompt"
	MsgResellerOnboardingDone     MessageKey = "reseller_onboarding_done"
	MsgSetWebPasswordPrompt       MessageKey = "set_web_password_prompt"
	MsgWebPasswordSetSuccess      MessageKey = "web_password_set_success"
	MsgTicketPrompt               MessageKey = "ticket_prompt"
	MsgTicketSubmitted            MessageKey = "ticket_submitted"
	MsgSupportInfo                MessageKey = "support_info"
	MsgReferralInfo               MessageKey = "referral_info"
	MsgManualCreditNotice         MessageKey = "manual_credit_notice"
)

// BankCardMessage formats the payment bank card instruction according to P09.
func BankCardMessage(lang domain.Language, cardNumber, cardholderName string, amount money.Money) string {
	formattedAmount := money.FormatCurrency(amount, string(lang))
	if lang == domain.LangFA {
		return fmt.Sprintf("مبلغ قابل پرداخت: %s\n\nشماره کارت جهت واریز:\n%s\nنام صاحب کارت: %s\n\nلطفاً پس از واریز، تصویر رسید یا متن رسید (کد پیگیری) را ارسال کنید.",
			formattedAmount, cardNumber, cardholderName)
	}
	return fmt.Sprintf("Payable Amount: %s\n\nCard number for transfer:\n%s\nCardholder name: %s\n\nPlease send the receipt photo or transaction text/reference after payment.",
		formattedAmount, cardNumber, cardholderName)
}

// UserCreditNotification formats the manual credit message according to P10:
// "The notification must first state the amount credited, with the administrator's message on the next line."
func UserCreditNotification(lang domain.Language, amount money.Money, adminMsg string) string {
	formattedAmount := money.FormatCurrency(amount, string(lang))
	var firstLine string
	if lang == domain.LangFA {
		firstLine = fmt.Sprintf("حساب شما به مبلغ %s شارژ شد.", formattedAmount)
	} else {
		firstLine = fmt.Sprintf("Your account has been credited with %s.", formattedAmount)
	}
	adminMsg = strings.TrimSpace(adminMsg)
	if adminMsg != "" {
		return firstLine + "\n" + adminMsg
	}
	return firstLine
}

// Get returns the localized text for a MessageKey.
func Get(lang domain.Language, key MessageKey) string {
	if lang == domain.LangFA {
		if msg, ok := faMessages[key]; ok {
			return msg
		}
	} else {
		if msg, ok := enMessages[key]; ok {
			return msg
		}
	}
	// Fallback
	if msg, ok := enMessages[key]; ok {
		return msg
	}
	return string(key)
}

var faMessages = map[MessageKey]string{
	MsgWelcomeCustomer:     "سلام! به ربات خرید فیلترشکن خوش آمدید. لطفاً یکی از گزینه‌های زیر را انتخاب کنید:",
	MsgWelcomeReseller:     "سلام همکار گرامی! به پنل نمایندگی خوش آمدید.",
	MsgChooseLanguage:      "لطفاً زبان مورد نظر خود را انتخاب کنید:\nPlease select your language:",
	MsgLanguageUpdated:     "زبان ربات با موفقیت تغییر کرد.",
	MsgMainMenu:            "منوی اصلی:",
	MsgBtnBuyService:       "🛒 خرید سرویس",
	MsgBtnFreeTrial:        "🎁 تست رایگان",
	MsgBtnMyServices:       "📋 سرویس‌های من",
	MsgBtnWallet:           "💳 کیف پول",
	MsgBtnSupport:          "💬 پشتیبانی و تیکت",
	MsgBtnInviteFriends:    "👥 دعوت دوستان",
	MsgBtnWebPanel:         "🌐 پنل تحت وب",
	MsgBtnManageChildBot:   "🤖 مدیریت ربات مشتری",
	MsgBtnMembership:       "⭐ ارتقای سطح نمایندگی",
	MsgBtnAdminMenu:        "⚙️ منوی مدیریت",
	MsgSelectPlan:          "لطفاً پلن مورد نظر خود را انتخاب کنید:",
	MsgSelectDuration:      "مدت زمان اشتراک را انتخاب کنید:",
	MsgEnterUserCount:      "تعداد کاربر همزمان (محدودیت IP) را وارد کنید:",
	MsgEnterServiceName:    "لطفاً یک نام دلخواه برای اشتراک وارد کنید یا دکمه نام تصادفی را بزنید:",
	MsgRandomNameOption:    "🎲 نام تصادفی",
	MsgOrderSummary:        "مشخصات سفارش شما:\nپلن: %s\nمدت: %d ماه\nتعداد کاربر: %d\nمبلغ کل: %s",
	MsgChoosePayment:       "نحوه پرداخت را انتخاب کنید:",
	MsgPayWallet:           "پرداخت از کیف پول",
	MsgPayDirect:           "پرداخت مستقیم (کارت به کارت)",
	MsgInsufficientWallet:  "موجودی کیف پول شما کافی نیست. لطفاً ابتدا کیف پول خود را شارژ کنید یا پرداخت مستقیم را انتخاب نمایید.",
	MsgInsufficientResellerCredit: "خطا در پردازش سفارش: موجودی ارائه‌دهنده سرویس کافی نیست. لطفاً با پشتیبانی تماس بگیرید.",
	MsgSendReceiptPrompt:   "لطفاً عکس فیش واریزی یا شماره پیگیری را ارسال نمایید:",
	MsgReceiptReceived:     "رسید شما دریافت شد و برای تأیید به مدیریت ارسال گردید. پس از تأیید، سرویس به صورت خودکار تحویل داده خواهد شد.",
	MsgOrderApproved:       "سفارش شما با موفقیت تأیید شد!",
	MsgOrderRejected:       "متأسفانه سفارش شما رد شد.\nعلت: %s",
	MsgSubDelivery:         "🎉 سرویس شما با موفقیت ایجاد شد!\n\n🔗 لینک اشتراک:\n%s\n\nنکات مصرف:\n%s",
	MsgTrialIssued:         "🎁 تست رایگان شما با موفقیت فعال شد!\n\n🔗 لینک اشتراک:\n%s",
	MsgTrialCooldownError:  "شما به تازگی از این پلن تست دریافت کرده‌اید. لطفاً تا پایان دوره انتظار صبر کنید.",
	MsgTrialLimitError:     "شما به سقف مجاز دریافت تست در این بازه زمانی رسیده‌اید.",
	MsgUserCountDecreasedWarning: "⚠️ توجه: با کاهش تعداد کاربران، شناسه اشتراک تغییر کرده و لینک قبلی شما غیرفعال شده است.\nلطفاً لینک جدید زیر را در نرم‌افزار خود وارد نمایید:",
	MsgResellerOnboardingPrompt: "همکار گرامی، لطفاً ابتدا نام برند/سرویس خود را وارد کنید. این نام به عنوان گروه اختصاصی سرویس‌های شما در پنل ثبت می‌شود:",
	MsgResellerOnboardingDone:   "نام برند شما با موفقیت ثبت شد: %s",
	MsgSetWebPasswordPrompt:     "جهت ورود به پنل تحت وب، لطفاً یک رمز عبور امن وارد کنید:\n(نام کاربری شما شناسه تلگرام عددی شما خواهد بود)",
	MsgWebPasswordSetSuccess:    "رمز عبور پنل وب با موفقیت ذخیره شد. شما می‌توانید با شناسه عددی تلگرام خود وارد شوید.",
	MsgTicketPrompt:             "لطفاً پیام یا مشکل خود را مطرح کنید تا کارشناسان پشتیبانی پاسخ دهند:",
	MsgTicketSubmitted:          "تیکت شما با موفقیت ثبت شد. پاسخ از طریق همین ربات برای شما ارسال خواهد شد.",
	MsgSupportInfo:              "جهت ارتباط مستقیم با پشتیبانی می‌توانید به آیدی زیر پیام دهید:\n%s",
	MsgReferralInfo:             "با دعوت از دوستانتان، %d%% از هر خرید آن‌ها به عنوان پاداش به کیف پول شما واریز خواهد شد!\n\n🔗 لینک اختصاصی شما:\n%s",
}

var enMessages = map[MessageKey]string{
	MsgWelcomeCustomer:     "Hello! Welcome to the VPN Subscription Bot. Please choose an option below:",
	MsgWelcomeReseller:     "Welcome partner! This is your Reseller Bot Management Panel.",
	MsgChooseLanguage:      "Please select your language:\nلطفاً زبان خود را انتخاب کنید:",
	MsgLanguageUpdated:     "Bot language has been successfully updated.",
	MsgMainMenu:            "Main Menu:",
	MsgBtnBuyService:       "🛒 Buy Service",
	MsgBtnFreeTrial:        "🎁 Free Trial",
	MsgBtnMyServices:       "📋 My Services",
	MsgBtnWallet:           "💳 Wallet",
	MsgBtnSupport:          "💬 Support & Tickets",
	MsgBtnInviteFriends:    "👥 Invite Friends",
	MsgBtnWebPanel:         "🌐 Web Panel",
	MsgBtnManageChildBot:   "🤖 Manage Customer Bot",
	MsgBtnMembership:       "⭐ Reseller Membership",
	MsgBtnAdminMenu:        "⚙️ Admin Menu",
	MsgSelectPlan:          "Please select a VPN plan:",
	MsgSelectDuration:      "Select subscription duration:",
	MsgEnterUserCount:      "Enter number of concurrent users (IP limit):",
	MsgEnterServiceName:    "Enter a custom subscription name or choose random:",
	MsgRandomNameOption:    "🎲 Random Name",
	MsgOrderSummary:        "Order Summary:\nPlan: %s\nDuration: %d month(s)\nUsers: %d\nTotal Amount: %s",
	MsgChoosePayment:       "Select payment method:",
	MsgPayWallet:           "Pay from Wallet",
	MsgPayDirect:           "Direct Bank Transfer",
	MsgInsufficientWallet:  "Insufficient wallet balance. Please top up your wallet or choose direct payment.",
	MsgInsufficientResellerCredit: "Order processing error: Insufficient provider credit. Please contact support.",
	MsgSendReceiptPrompt:   "Please upload the payment receipt photo or send the reference text:",
	MsgReceiptReceived:     "Your receipt has been submitted for administrator review. Your service will be provisioned automatically once approved.",
	MsgOrderApproved:       "Your order has been approved!",
	MsgOrderRejected:       "Your order was rejected.\nReason: %s",
	MsgSubDelivery:         "🎉 Your service is ready!\n\n🔗 Subscription Link:\n%s\n\nUsage Notes:\n%s",
	MsgTrialIssued:         "🎁 Your free trial is active!\n\n🔗 Subscription Link:\n%s",
	MsgTrialCooldownError:  "You recently obtained a trial on this plan. Please wait for the cooldown period to end.",
	MsgTrialLimitError:     "You have reached the maximum trial quota for this time window.",
	MsgUserCountDecreasedWarning: "⚠️ Note: Decreasing the user count has changed your subscription ID. Your previous subscription link is now revoked.\nPlease update your VPN client with this new link:",
	MsgResellerOnboardingPrompt: "Welcome! Please enter your service/brand name. This will be used as the 3x-ui group name for all your clients:",
	MsgResellerOnboardingDone:   "Your brand name has been registered: %s",
	MsgSetWebPasswordPrompt:     "To log in to the Reseller Web Panel, please enter a password:\n(Your username will be your numeric Telegram ID)",
	MsgWebPasswordSetSuccess:    "Web panel password set successfully. You can now log in using your numeric Telegram ID.",
	MsgTicketPrompt:             "Please type your message or question for support:",
	MsgTicketSubmitted:          "Your support ticket has been submitted. Our team will reply through this bot.",
	MsgSupportInfo:              "For direct contact, please reach out to our Telegram support:\n%s",
	MsgReferralInfo:             "Invite your friends and earn %d%% of their purchases credited to your wallet!\n\n🔗 Your Referral Link:\n%s",
}
