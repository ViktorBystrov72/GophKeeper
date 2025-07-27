// Package grpc содержит gRPC сервер и обработчики для GophKeeper.
package grpc

import (
	"context"
	"time"

	"github.com/GophKeeper/internal/auth"
	"github.com/GophKeeper/internal/crypto"
	"github.com/GophKeeper/internal/models"
	"github.com/GophKeeper/internal/otp"
	"github.com/GophKeeper/internal/storage"
	pb "github.com/GophKeeper/proto/gen/proto"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server реализует gRPC сервер GophKeeper.
type Server struct {
	pb.UnimplementedGophKeeperServer
	storage       storage.Storage
	authService   *auth.Service
	cryptoService *crypto.Service
	otpService    *otp.Service
	validator     *validator.Validate
	logger        *zap.Logger
}

// NewServer создает новый gRPC сервер.
func NewServer(
	storage storage.Storage,
	authService *auth.Service,
	cryptoService *crypto.Service,
	otpService *otp.Service,
	logger *zap.Logger,
) *Server {
	return &Server{
		storage:       storage,
		authService:   authService,
		cryptoService: cryptoService,
		otpService:    otpService,
		validator:     validator.New(),
		logger:        logger,
	}
}

// NewGRPCServer создает новый gRPC сервер.
func NewGRPCServer(server *Server) *grpc.Server {
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(AuthInterceptor(server.authService, server.logger)),
	)
	pb.RegisterGophKeeperServer(grpcServer, server)
	return grpcServer
}

// Register регистрирует нового пользователя.
func (s *Server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.AuthResponse, error) {
	// Валидируем запрос
	if req.Username == "" || len(req.Username) < 3 {
		return nil, status.Error(codes.InvalidArgument, "username must be at least 3 characters")
	}
	if req.Password == "" || len(req.Password) < 6 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 6 characters")
	}

	// Хешируем пароль
	hashedPassword, err := s.authService.HashPassword(req.Password)
	if err != nil {
		s.logger.Error("Failed to hash password", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to process password")
	}

	// Создаем пользователя
	user := &models.User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
	}

	if err := s.storage.CreateUser(ctx, user); err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		if err.Error() == "username already exists" {
			return nil, status.Error(codes.AlreadyExists, "username already exists")
		}
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	// Генерируем токен
	token, expiresAt, err := s.authService.GenerateToken(user.ID, user.Username)
	if err != nil {
		s.logger.Error("Failed to generate token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb.AuthResponse{
		Token:     token,
		ExpiresAt: timestamppb.New(expiresAt),
		User: &pb.User{
			Id:        user.ID.String(),
			Username:  user.Username,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		},
	}, nil
}

// Login аутентифицирует пользователя.
func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.AuthResponse, error) {
	// Валидируем запрос
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}

	// Получаем пользователя
	user, err := s.storage.GetUserByUsername(ctx, req.Username)
	if err != nil {
		s.logger.Warn("User not found", zap.String("username", req.Username))
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Проверяем пароль
	if !s.authService.CheckPassword(req.Password, user.PasswordHash) {
		s.logger.Warn("Invalid password", zap.String("username", req.Username))
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}

	// Генерируем токен
	token, expiresAt, err := s.authService.GenerateToken(user.ID, user.Username)
	if err != nil {
		s.logger.Error("Failed to generate token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate token")
	}

	return &pb.AuthResponse{
		Token:     token,
		ExpiresAt: timestamppb.New(expiresAt),
		User: &pb.User{
			Id:        user.ID.String(),
			Username:  user.Username,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		},
	}, nil
}

// RefreshToken обновляет JWT токен.
func (s *Server) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.AuthResponse, error) {
	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	token, expiresAt, err := s.authService.RefreshToken(req.Token)
	if err != nil {
		s.logger.Warn("Failed to refresh token", zap.Error(err))
		return nil, status.Error(codes.Unauthenticated, "invalid or expired token")
	}

	// Получаем claims для информации о пользователе
	claims, err := s.authService.ValidateToken(token)
	if err != nil {
		s.logger.Error("Failed to validate new token", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to validate token")
	}

	// Получаем пользователя
	user, err := s.storage.GetUserByID(ctx, claims.UserID)
	if err != nil {
		s.logger.Error("Failed to get user", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &pb.AuthResponse{
		Token:     token,
		ExpiresAt: timestamppb.New(expiresAt),
		User: &pb.User{
			Id:        user.ID.String(),
			Username:  user.Username,
			CreatedAt: timestamppb.New(user.CreatedAt),
			UpdatedAt: timestamppb.New(user.UpdatedAt),
		},
	}, nil
}

// CreateData создает новую запись данных.
func (s *Server) CreateData(ctx context.Context, req *pb.CreateDataRequest) (*pb.DataEntryResponse, error) {
	// Получаем пользователя из контекста (добавляется middleware)
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	// Валидируем запрос
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if len(req.EncryptedData) == 0 {
		return nil, status.Error(codes.InvalidArgument, "data is required")
	}

	// Преобразуем тип данных
	dataType := convertProtoDataType(req.Type)
	if dataType == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid data type")
	}

	// Создаем запись
	entry := &models.DataEntry{
		UserID:        userID,
		Type:          models.DataType(dataType),
		Name:          req.Name,
		Description:   req.Description,
		EncryptedData: req.EncryptedData,
		Metadata:      req.Metadata,
	}

	if err := s.storage.CreateDataEntry(ctx, entry); err != nil {
		s.logger.Error("Failed to create data entry", zap.Error(err))
		if err.Error() == "entry with this name already exists" {
			return nil, status.Error(codes.AlreadyExists, "entry with this name already exists")
		}
		return nil, status.Error(codes.Internal, "failed to create data entry")
	}

	return &pb.DataEntryResponse{
		DataEntry: convertToProtoDataEntry(entry),
	}, nil
}

// GetData получает запись данных по ID.
func (s *Server) GetData(ctx context.Context, req *pb.GetDataRequest) (*pb.DataEntryResponse, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	entryID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entry ID")
	}

	entry, err := s.storage.GetDataEntry(ctx, userID, entryID)
	if err != nil {
		s.logger.Error("Failed to get data entry", zap.Error(err))
		return nil, status.Error(codes.NotFound, "data entry not found")
	}

	return &pb.DataEntryResponse{
		DataEntry: convertToProtoDataEntry(entry),
	}, nil
}

// ListData получает список записей данных.
func (s *Server) ListData(ctx context.Context, req *pb.ListDataRequest) (*pb.ListDataResponse, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	var dataType *models.DataType
	if req.Type != nil && *req.Type != pb.DataType_DATA_TYPE_UNSPECIFIED {
		dt := convertProtoDataType(*req.Type)
		if dt != "" {
			modelType := models.DataType(dt)
			dataType = &modelType
		}
	}

	entries, err := s.storage.GetDataEntries(ctx, userID, dataType)
	if err != nil {
		s.logger.Error("Failed to get data entries", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get data entries")
	}

	protoEntries := make([]*pb.DataEntry, len(entries))
	for i, entry := range entries {
		protoEntries[i] = convertToProtoDataEntry(&entry)
	}

	return &pb.ListDataResponse{
		DataEntries: protoEntries,
		Total:       int32(len(entries)),
	}, nil
}

// UpdateData обновляет запись данных.
func (s *Server) UpdateData(ctx context.Context, req *pb.UpdateDataRequest) (*pb.DataEntryResponse, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	entryID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entry ID")
	}

	// Получаем существующую запись
	entry, err := s.storage.GetDataEntry(ctx, userID, entryID)
	if err != nil {
		return nil, status.Error(codes.NotFound, "data entry not found")
	}

	// Обновляем поля
	entry.Name = req.Name
	entry.Description = req.Description
	entry.EncryptedData = req.EncryptedData
	entry.Metadata = req.Metadata
	entry.Version = req.Version

	if err := s.storage.UpdateDataEntry(ctx, entry); err != nil {
		s.logger.Error("Failed to update data entry", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update data entry")
	}

	return &pb.DataEntryResponse{
		DataEntry: convertToProtoDataEntry(entry),
	}, nil
}

// DeleteData удаляет запись данных.
func (s *Server) DeleteData(ctx context.Context, req *pb.DeleteDataRequest) (*pb.DeleteDataResponse, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	entryID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entry ID")
	}

	if err := s.storage.DeleteDataEntry(ctx, userID, entryID); err != nil {
		s.logger.Error("Failed to delete data entry", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to delete data entry")
	}

	return &pb.DeleteDataResponse{
		Success: true,
	}, nil
}

// SyncData синхронизирует данные клиента с сервером.
func (s *Server) SyncData(ctx context.Context, req *pb.SyncDataRequest) (*pb.SyncDataResponse, error) {
	userID, ok := getUserIDFromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	lastSyncTime := req.LastSyncTime.AsTime()

	// Получаем измененные записи
	entries, err := s.storage.GetDataEntriesAfter(ctx, userID, lastSyncTime)
	if err != nil {
		s.logger.Error("Failed to get data entries after sync time", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to sync data")
	}

	// Получаем удаленные записи
	deletedIDs, err := s.storage.GetDeletedEntriesAfter(ctx, userID, lastSyncTime)
	if err != nil {
		s.logger.Error("Failed to get deleted entries", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to sync data")
	}

	protoEntries := make([]*pb.DataEntry, len(entries))
	for i, entry := range entries {
		protoEntries[i] = convertToProtoDataEntry(&entry)
	}

	deletedStringIDs := make([]string, len(deletedIDs))
	for i, id := range deletedIDs {
		deletedStringIDs[i] = id.String()
	}

	return &pb.SyncDataResponse{
		DataEntries:  protoEntries,
		DeletedIds:   deletedStringIDs,
		LastSyncTime: timestamppb.New(time.Now()),
	}, nil
}

// GenerateOTP генерирует OTP код.
func (s *Server) GenerateOTP(ctx context.Context, req *pb.GenerateOTPRequest) (*pb.GenerateOTPResponse, error) {
	if req.Secret == "" {
		return nil, status.Error(codes.InvalidArgument, "secret is required")
	}

	code, err := s.otpService.GenerateCode(req.Secret)
	if err != nil {
		s.logger.Error("Failed to generate OTP code", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate OTP code")
	}

	timeRemaining := s.otpService.GetTimeRemaining()
	expiresAt := time.Now().Add(time.Duration(timeRemaining) * time.Second)

	return &pb.GenerateOTPResponse{
		Code:          code,
		ExpiresAt:     timestamppb.New(expiresAt),
		TimeRemaining: int32(timeRemaining),
	}, nil
}

// CreateOTPSecret создает новый OTP секрет.
func (s *Server) CreateOTPSecret(ctx context.Context, req *pb.CreateOTPSecretRequest) (*pb.CreateOTPSecretResponse, error) {
	if req.Issuer == "" {
		return nil, status.Error(codes.InvalidArgument, "issuer is required")
	}
	if req.AccountName == "" {
		return nil, status.Error(codes.InvalidArgument, "account name is required")
	}

	secret, err := s.otpService.GenerateSecret()
	if err != nil {
		s.logger.Error("Failed to generate OTP secret", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate OTP secret")
	}

	qrCodeURL, err := s.otpService.GenerateQRCodeURL(secret, req.Issuer, req.AccountName)
	if err != nil {
		s.logger.Error("Failed to generate QR code URL", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate QR code URL")
	}

	backupCodes, err := s.otpService.GenerateBackupCodes(10)
	if err != nil {
		s.logger.Error("Failed to generate backup codes", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to generate backup codes")
	}

	return &pb.CreateOTPSecretResponse{
		Secret:      secret,
		QrCodeUrl:   qrCodeURL,
		BackupCodes: backupCodes,
	}, nil
}

// getUserIDFromContext извлекает ID пользователя из контекста.
func getUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userIDValue := ctx.Value(UserIDKey)
	if userIDValue == nil {
		return uuid.Nil, false
	}

	userID, ok := userIDValue.(uuid.UUID)
	return userID, ok
}

// convertProtoDataType преобразует proto тип данных в строку.
func convertProtoDataType(protoType pb.DataType) string {
	switch protoType {
	case pb.DataType_DATA_TYPE_CREDENTIALS:
		return string(models.DataTypeCredentials)
	case pb.DataType_DATA_TYPE_TEXT:
		return string(models.DataTypeText)
	case pb.DataType_DATA_TYPE_BINARY:
		return string(models.DataTypeBinary)
	case pb.DataType_DATA_TYPE_CARD:
		return string(models.DataTypeCard)
	default:
		return ""
	}
}

// convertToProtoDataType преобразует строку в proto тип данных.
func convertToProtoDataType(dataType models.DataType) pb.DataType {
	switch dataType {
	case models.DataTypeCredentials:
		return pb.DataType_DATA_TYPE_CREDENTIALS
	case models.DataTypeText:
		return pb.DataType_DATA_TYPE_TEXT
	case models.DataTypeBinary:
		return pb.DataType_DATA_TYPE_BINARY
	case models.DataTypeCard:
		return pb.DataType_DATA_TYPE_CARD
	default:
		return pb.DataType_DATA_TYPE_UNSPECIFIED
	}
}

// convertToProtoDataEntry преобразует модель DataEntry в proto DataEntry.
func convertToProtoDataEntry(entry *models.DataEntry) *pb.DataEntry {
	return &pb.DataEntry{
		Id:            entry.ID.String(),
		Type:          convertToProtoDataType(entry.Type),
		Name:          entry.Name,
		Description:   entry.Description,
		EncryptedData: entry.EncryptedData,
		Metadata:      entry.Metadata,
		CreatedAt:     timestamppb.New(entry.CreatedAt),
		UpdatedAt:     timestamppb.New(entry.UpdatedAt),
		Version:       entry.Version,
	}
}
