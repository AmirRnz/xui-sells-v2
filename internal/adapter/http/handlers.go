package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type LoginRequest struct {
	TelegramID int64  `json:"tg_id"`
	Password   string `json:"password"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type RejectOrderRequest struct {
	Reason string `json:"reason"`
}

type ReplyTicketRequest struct {
	Message string `json:"message"`
}

type ResellerUserView struct {
	TelegramID    int64               `json:"tg_id"`
	Username      string              `json:"username"`
	FirstName     string              `json:"first_name"`
	WalletBalance int64               `json:"wallet_balance"`
	ServiceName   string              `json:"service_name"`
	Tier          string              `json:"tier"`
	TierExpiresAt *string             `json:"tier_expires_at"`
}

type LoginResponse struct {
	Token string           `json:"token"`
	User  ResellerUserView `json:"user"`
}

// handleLogin handles reseller credentials authentication.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_request","message":"Invalid json"}`, http.StatusBadRequest)
		return
	}

	acc, err := s.resellerStore.GetAccountByTgID(r.Context(), req.TelegramID)
	if err != nil || acc == nil {
		http.Error(w, `{"error":"invalid_credentials","message":"Invalid Telegram ID or password"}`, http.StatusUnauthorized)
		return
	}

	valid, err := VerifyPassword(req.Password, acc.PasswordHash)
	if err != nil || !valid {
		http.Error(w, `{"error":"invalid_credentials","message":"Invalid Telegram ID or password"}`, http.StatusUnauthorized)
		return
	}

	sess, err := s.sessions.CreateSession(acc.TelegramID, acc.InstanceID, acc.UserID, acc.Tier, 7*24*time.Hour)
	if err != nil {
		http.Error(w, `{"error":"server_error","message":"Failed to create session"}`, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "reseller_token",
		Value:    sess.Token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  sess.ExpiresAt,
	})

	var expiryStr *string
	if acc.TierExpiresAt != nil {
		tStr := acc.TierExpiresAt.Format(time.RFC3339)
		expiryStr = &tStr
	}

	bal := int64(0)
	if s.walletRepo != nil {
		if b, err := s.walletRepo.GetBalance(r.Context(), acc.InstanceID, acc.UserID); err == nil {
			bal = b.Amount
		}
	}

	resp := LoginResponse{
		Token: sess.Token,
		User: ResellerUserView{
			TelegramID:    acc.TelegramID,
			Username:      acc.Username,
			FirstName:     acc.FirstName,
			WalletBalance: bal,
			ServiceName:   acc.ServiceName,
			Tier:          acc.Tier.String(),
			TierExpiresAt: expiryStr,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleGetMe returns current reseller profile information.
func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	sess := GetSessionFromContext(r.Context())
	if sess == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	acc, err := s.resellerStore.GetAccountByTgID(r.Context(), sess.TelegramID)
	if err != nil || acc == nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	var expiryStr *string
	if acc.TierExpiresAt != nil {
		tStr := acc.TierExpiresAt.Format(time.RFC3339)
		expiryStr = &tStr
	}

	bal := int64(0)
	if s.walletRepo != nil {
		if b, err := s.walletRepo.GetBalance(r.Context(), acc.InstanceID, acc.UserID); err == nil {
			bal = b.Amount
		}
	}

	resp := ResellerUserView{
		TelegramID:    acc.TelegramID,
		Username:      acc.Username,
		FirstName:     acc.FirstName,
		WalletBalance: bal,
		ServiceName:   acc.ServiceName,
		Tier:          acc.Tier.String(),
		TierExpiresAt: expiryStr,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleLogout ends the active session.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	sess := GetSessionFromContext(r.Context())
	if sess != nil {
		s.sessions.DeleteSession(sess.Token)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "reseller_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success":true}`))
}

// handleChangePassword allows resellers to update their panel password.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	sess := GetSessionFromContext(r.Context())
	if sess == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}

	acc, err := s.resellerStore.GetAccountByTgID(r.Context(), sess.TelegramID)
	if err != nil || acc == nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}

	valid, err := VerifyPassword(req.OldPassword, acc.PasswordHash)
	if err != nil || !valid {
		http.Error(w, `{"error":"invalid_password","message":"Current password incorrect"}`, http.StatusBadRequest)
		return
	}

	newHash, err := HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
		return
	}

	if err := s.resellerStore.UpdatePasswordHash(r.Context(), sess.TelegramID, newHash); err != nil {
		http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success":true,"message":"Password updated successfully"}`))
}

// handleGetStats serves operational metrics and charts for the web panel.
func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	sess := GetSessionFromContext(r.Context())
	if sess == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	stats, err := s.statsSvc.GetStats(r.Context(), sess.InstanceID)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"server_error","message":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(stats)
}

// handleGetServices returns active client subscriptions.
func (s *Server) handleGetServices(w http.ResponseWriter, r *http.Request) {
	sess := GetSessionFromContext(r.Context())
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	services, err := s.serviceStore.ListResellerServices(r.Context(), sess.InstanceID, search, status)
	if err != nil {
		http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(services)
}

// handleResetTraffic zeroes out client usage counters.
func (s *Server) handleResetTraffic(w http.ResponseWriter, r *http.Request) {
	param := chi.URLParam(r, "id")
	if param == "" {
		param = chi.URLParam(r, "email")
	}

	if err := s.serviceStore.ResetTraffic(r.Context(), param); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"api_error","message":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"success":true,"message":"Traffic reset for service %s"}`, param)))
}

// handleRotateSubID regenerates subId and generates a fresh subscription link.
func (s *Server) handleRotateSubID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, _ := strconv.ParseInt(idStr, 10, 64)

	svc, newURL, err := s.serviceStore.RotateSubID(r.Context(), id)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"api_error","message":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	resp := map[string]any{
		"success":          true,
		"sub_id":           svc.SubID,
		"subscription_url": newURL,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleGetOrders returns pending or historical purchase orders.
func (s *Server) handleGetOrders(w http.ResponseWriter, r *http.Request) {
	sess := GetSessionFromContext(r.Context())
	status := r.URL.Query().Get("status")

	orders, err := s.orderStore.ListOrders(r.Context(), sess.InstanceID, status)
	if err != nil {
		http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(orders)
}

// handleApproveOrder approves an order via the order service.
func (s *Server) handleApproveOrder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	orderID, _ := strconv.ParseInt(idStr, 10, 64)

	if err := s.orderStore.ApproveOrder(r.Context(), orderID); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"approval_error","message":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"success":true,"message":"Order %d approved successfully"}`, orderID)))
}

// handleRejectOrder rejects an order with recorded reasoning.
func (s *Server) handleRejectOrder(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	orderID, _ := strconv.ParseInt(idStr, 10, 64)

	var req RejectOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Reason == "" {
		req.Reason = "Rejected by administrator"
	}

	if err := s.orderStore.RejectOrder(r.Context(), orderID, req.Reason); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"rejection_error","message":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(fmt.Sprintf(`{"success":true,"message":"Order %d rejected: %s"}`, orderID, req.Reason)))
}

// handleGetChildBots returns child bots managed by this reseller.
func (s *Server) handleGetChildBots(w http.ResponseWriter, r *http.Request) {
	sess := GetSessionFromContext(r.Context())
	bots, err := s.childBotStore.ListChildBots(r.Context(), sess.InstanceID)
	if err != nil {
		http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(bots)
}

// handleGetTickets lists tickets for this reseller bot.
func (s *Server) handleGetTickets(w http.ResponseWriter, r *http.Request) {
	sess := GetSessionFromContext(r.Context())
	tickets, err := s.ticketSvc.ListTickets(r.Context(), sess.InstanceID, nil, nil)
	if err != nil {
		http.Error(w, `{"error":"server_error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tickets)
}

// handleReplyTicket posts a response to a support ticket.
func (s *Server) handleReplyTicket(w http.ResponseWriter, r *http.Request) {
	sess := GetSessionFromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	ticketID, _ := strconv.ParseInt(idStr, 10, 64)

	var req ReplyTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		http.Error(w, `{"error":"invalid_request","message":"Message is required"}`, http.StatusBadRequest)
		return
	}

	msg, err := s.ticketSvc.ReplyTicket(r.Context(), ticketID, sess.TelegramID, req.Message)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"ticket_error","message":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(msg)
}
