package i18n

import (
	"strings"
	"testing"

	"xui-sells-v2/internal/domain"
	"xui-sells-v2/internal/domain/money"
)

func TestBankCardMessageStructurePersian(t *testing.T) {
	amount := money.FromTomanInput(200) // 200,000 Toman -> 200 هزار تومان
	cardNumber := "6037997112345678"
	cardholderName := "علی محمدی"

	msg := BankCardMessage(domain.LangFA, cardNumber, cardholderName, amount)

	// Must preserve exact structure specified in P09:
	// شماره کارت جهت واریز:
	// X
	// نام صاحب کارت: X
	if !strings.Contains(msg, "شماره کارت جهت واریز:\n6037997112345678") {
		t.Errorf("Persian card message missing required card number structure: %s", msg)
	}
	if !strings.Contains(msg, "نام صاحب کارت: علی محمدی") {
		t.Errorf("Persian card message missing cardholder name structure: %s", msg)
	}
	if !strings.Contains(msg, "200 هزار تومان") {
		t.Errorf("Persian card message missing formatted toman amount: %s", msg)
	}
}

func TestBankCardMessageStructureEnglish(t *testing.T) {
	amount := money.FromTomanInput(200) // 200,000 Toman -> 200 thousand toman
	cardNumber := "6037997112345678"
	cardholderName := "John Doe"

	msg := BankCardMessage(domain.LangEN, cardNumber, cardholderName, amount)

	if !strings.Contains(msg, "Card number for transfer:\n6037997112345678") {
		t.Errorf("English card message missing required card number structure: %s", msg)
	}
	if !strings.Contains(msg, "Cardholder name: John Doe") {
		t.Errorf("English card message missing cardholder name structure: %s", msg)
	}
	if !strings.Contains(msg, "200 thousand toman") {
		t.Errorf("English card message missing formatted amount: %s", msg)
	}
}

func TestUserCreditNotification(t *testing.T) {
	amount := money.NewUSD(100) // $1.00
	adminMsg := "Bonus reward for contest"

	// English test: P10 example
	// "Your account has been credited with 1 dollar.\n<Administrator's message>"
	msgEN := UserCreditNotification(domain.LangEN, amount, adminMsg)
	expectedEN := "Your account has been credited with $1.00.\nBonus reward for contest"
	if msgEN != expectedEN {
		t.Errorf("Expected:\n%s\nGot:\n%s", expectedEN, msgEN)
	}

	// Persian test
	amountToman := money.FromTomanInput(1000) // 1,000,000 Toman -> 1 میلیون تومان
	adminMsgFA := "پاداش وفاداری"
	msgFA := UserCreditNotification(domain.LangFA, amountToman, adminMsgFA)
	expectedFA := "حساب شما به مبلغ 1 میلیون تومان شارژ شد.\nپاداش وفاداری"
	if msgFA != expectedFA {
		t.Errorf("Expected:\n%s\nGot:\n%s", expectedFA, msgFA)
	}
}

func TestMessageFallbacks(t *testing.T) {
	fa := Get(domain.LangFA, MsgWelcomeCustomer)
	if !strings.Contains(fa, "سلام") {
		t.Errorf("Expected Persian welcome, got: %s", fa)
	}

	en := Get(domain.LangEN, MsgWelcomeCustomer)
	if !strings.Contains(en, "Hello") {
		t.Errorf("Expected English welcome, got: %s", en)
	}
}
