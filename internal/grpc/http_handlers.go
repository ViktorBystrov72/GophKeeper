// Package grpc предоставляет HTTP обработчики для REST API.
package grpc

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/GophKeeper/internal/models"
	pb "github.com/GophKeeper/proto/gen/proto"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// HandleRegister обрабатывает HTTP запрос на регистрацию.
func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if err := s.validator.Struct(req); err != nil {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
	}

	resp, err := s.Register(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Registration failed", zap.Error(err))
		http.Error(w, "Registration failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleLogin обрабатывает HTTP запрос на вход.
func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if err := s.validator.Struct(req); err != nil {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	}

	resp, err := s.Login(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Login failed", zap.Error(err))
		http.Error(w, "Login failed", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleRefreshToken обрабатывает HTTP запрос на обновление токена.
func (s *Server) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.RefreshTokenRequest{
		Token: req.Token,
	}

	resp, err := s.RefreshToken(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Token refresh failed", zap.Error(err))
		http.Error(w, "Token refresh failed", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleCreateData обрабатывает HTTP запрос на создание данных.
func (s *Server) HandleCreateData(w http.ResponseWriter, r *http.Request) {
	var req models.CreateDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if err := s.validator.Struct(req); err != nil {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	// Получаем userID из контекста (проверяем авторизацию)
	_, ok := getUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Шифруем данные
	encryptedData, err := s.cryptoService.EncryptLargeData([]byte(req.Data.(string)))
	if err != nil {
		s.logger.Error("Failed to encrypt data", zap.Error(err))
		http.Error(w, "Failed to encrypt data", http.StatusInternalServerError)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.CreateDataRequest{
		Type:          convertToProtoDataType(req.Type),
		Name:          req.Name,
		Description:   req.Description,
		EncryptedData: encryptedData,
		Metadata:      req.Metadata,
	}

	resp, err := s.CreateData(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Failed to create data", zap.Error(err))
		http.Error(w, "Failed to create data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleGetData обрабатывает HTTP запрос на получение данных.
func (s *Server) HandleGetData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.GetDataRequest{
		Id: id,
	}

	resp, err := s.GetData(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Failed to get data", zap.Error(err))
		http.Error(w, "Failed to get data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleListData обрабатывает HTTP запрос на получение списка данных.
func (s *Server) HandleListData(w http.ResponseWriter, r *http.Request) {
	// Парсим параметры запроса
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	typeStr := r.URL.Query().Get("type")

	var limit, offset int32
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = int32(l)
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = int32(o)
		}
	}

	grpcReq := &pb.ListDataRequest{
		Limit:  limit,
		Offset: offset,
	}

	if typeStr != "" {
		// Конвертируем тип данных
		dataType := models.DataType(typeStr)
		protoType := convertToProtoDataType(dataType)
		grpcReq.Type = &protoType
	}

	resp, err := s.ListData(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Failed to list data", zap.Error(err))
		http.Error(w, "Failed to list data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleUpdateData обрабатывает HTTP запрос на обновление данных.
func (s *Server) HandleUpdateData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	var req models.UpdateDataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if err := s.validator.Struct(req); err != nil {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	// Шифруем данные
	encryptedData, err := s.cryptoService.EncryptLargeData([]byte(req.Data.(string)))
	if err != nil {
		s.logger.Error("Failed to encrypt data", zap.Error(err))
		http.Error(w, "Failed to encrypt data", http.StatusInternalServerError)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.UpdateDataRequest{
		Id:            id,
		Name:          req.Name,
		Description:   req.Description,
		EncryptedData: encryptedData,
		Metadata:      req.Metadata,
		Version:       req.Version,
	}

	resp, err := s.UpdateData(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Failed to update data", zap.Error(err))
		http.Error(w, "Failed to update data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleDeleteData обрабатывает HTTP запрос на удаление данных.
func (s *Server) HandleDeleteData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "ID is required", http.StatusBadRequest)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.DeleteDataRequest{
		Id: id,
	}

	resp, err := s.DeleteData(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Failed to delete data", zap.Error(err))
		http.Error(w, "Failed to delete data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleSyncData обрабатывает HTTP запрос на синхронизацию данных.
func (s *Server) HandleSyncData(w http.ResponseWriter, r *http.Request) {
	var req models.SyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.SyncDataRequest{
		LastSyncTime: timestamppb.New(req.LastSyncTime),
	}

	resp, err := s.SyncData(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Failed to sync data", zap.Error(err))
		http.Error(w, "Failed to sync data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleGenerateOTP обрабатывает HTTP запрос на генерацию OTP.
func (s *Server) HandleGenerateOTP(w http.ResponseWriter, r *http.Request) {
	var req models.OTPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if err := s.validator.Struct(req); err != nil {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.GenerateOTPRequest{
		Secret: req.Secret,
	}

	resp, err := s.GenerateOTP(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Failed to generate OTP", zap.Error(err))
		http.Error(w, "Failed to generate OTP", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleCreateOTPSecret обрабатывает HTTP запрос на создание OTP секрета.
func (s *Server) HandleCreateOTPSecret(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Issuer      string `json:"issuer"`
		AccountName string `json:"account_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем gRPC метод
	grpcReq := &pb.CreateOTPSecretRequest{
		Issuer:      req.Issuer,
		AccountName: req.AccountName,
	}

	resp, err := s.CreateOTPSecret(r.Context(), grpcReq)
	if err != nil {
		s.logger.Error("Failed to create OTP secret", zap.Error(err))
		http.Error(w, "Failed to create OTP secret", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
