package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"musicclubbot/backend/internal/config"
	"musicclubbot/backend/internal/helpers"
	"musicclubbot/backend/proto"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TelegramUser represents user data from Telegram WebApp
type TelegramUser struct {
	ID           int64  `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
	PhotoURL     string `json:"photo_url,omitempty"`
}

// ChatMemberResponse represents Telegram API getChatMember response
type ChatMemberResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		Status string `json:"status"`
	} `json:"result"`
}

func (s *AuthService) TelegramWebAppAuth(ctx context.Context, req *proto.TelegramWebAppAuthRequest) (*proto.AuthSession, error) {
	cfg := ctx.Value("cfg").(config.Config)

	// 1. Verify Telegram WebApp initData
	log.Printf("[DEBUG] TelegramWebAppAuth called with initData: %s", req.InitData)
	user, err := verifyTelegramWebAppData(req.InitData, cfg.BotToken)
	if err != nil {
		log.Printf("[ERROR] Failed to verify Telegram WebApp data: %v, initData: %s", err, req.InitData)
		return nil, status.Error(codes.Unauthenticated, "invalid Telegram data")
	}

	// 2. Check chat membership
	isMember, err := checkChatMembership(user.ID, cfg.BotToken, cfg.ChatID)
	if err != nil {
		log.Printf("[ERROR] Failed to check chat membership for user %d: %v", user.ID, err)
		return nil, status.Error(codes.Internal, "failed to check chat membership")
	}

	log.Printf("[DEBUG] Chat membership check for user %d (@%s): isMember=%v, chatID=%s",
		user.ID, user.Username, isMember, cfg.ChatID)

	if !isMember {
		log.Printf("[WARN] User %d (@%s) attempted to access but is not a member of chat %s",
			user.ID, user.Username, cfg.ChatID)
		return nil, status.Error(codes.PermissionDenied, "you must be a member of the Music Club chat to use this app")
	}

	// 3. Get or create user in database
	db, err := helpers.DbFromCtx(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var userID uuid.UUID
	var displayName string
	var username string

	// Try to find existing user by Telegram ID
	err = db.QueryRowContext(ctx, `
		SELECT id, username, display_name FROM app_user WHERE tg_user_id = $1`,
		user.ID,
	).Scan(&userID, &username, &displayName)

	if err == sql.ErrNoRows {
		// Create new user
		displayName = user.FirstName
		if user.LastName != "" {
			displayName += " " + user.LastName
		}

		username = user.Username
		if username == "" {
			username = fmt.Sprintf("tg_%d", user.ID)
		}

		err = db.QueryRowContext(ctx, `
			INSERT INTO app_user (username, display_name, avatar_url, tg_user_id)
			VALUES ($1, $2, $3, $4)
			RETURNING id`,
			username,
			displayName,
			user.PhotoURL,
			user.ID,
		).Scan(&userID)

		if err != nil {
			log.Printf("[ERROR] Failed to create user: %v, username: %s, tg_user_id: %d", err, username, user.ID)
			return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
		}

		// Create default permissions
		_, err = db.ExecContext(ctx, `
			INSERT INTO user_permissions (user_id, edit_own_participation, edit_own_songs)
			VALUES ($1, TRUE, TRUE)`,
			userID,
		)

		if err != nil {
			return nil, status.Error(codes.Internal, "failed to create user permissions")
		}
	} else if err != nil {
		return nil, status.Error(codes.Internal, "database error")
	} else {
		// Update existing user info
		_, err = db.ExecContext(ctx, `
			UPDATE app_user
			SET display_name = $1, avatar_url = $2
			WHERE id = $3`,
			func() string {
				name := user.FirstName
				if user.LastName != "" {
					name += " " + user.LastName
				}
				return name
			}(),
			user.PhotoURL,
			userID,
		)

		if err != nil {
			// Ignore update errors
		}
	}

	// 4. Generate JWT tokens
	accessToken, err := GenerateAccessToken(ctx, userID, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate access token: %v", err)
	}

	refreshToken, err := GenerateRefreshToken()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate refresh token: %v", err)
	}

	// Store refresh token
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "begin tx: %v", err)
	}
	defer tx.Rollback()

	// Invalidate old refresh tokens
	_, err = tx.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invalidate old tokens: %v", err)
	}

	// Store new refresh token
	refreshExpiresAt := time.Now().Add(RefreshTokenExp)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at)
		VALUES (gen_random_uuid(), $1, $2, $3)`,
		userID, refreshToken, refreshExpiresAt)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "store refresh token: %v", err)
	}

	// 5. Get user permissions
	permissions, err := helpers.GetUserPermissions(ctx, tx, userID)
	if err != nil {
		permissions = &proto.PermissionSet{}
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "commit: %v", err)
	}

	// 6. Build profile
	profile := &proto.User{
		Id:          userID.String(),
		Username:    username,
		DisplayName: displayName,
		AvatarUrl:   user.PhotoURL,
		TelegramId:  uint64(user.ID),
	}

	return &proto.AuthSession{
		Tokens: &proto.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		},
		Iat:          uint64(time.Now().Unix()),
		Exp:          uint64(time.Now().Add(AccessTokenExp).Unix()),
		IsChatMember: isMember,
		Profile:      profile,
		Permissions:  permissions,
	}, nil
}

// verifyTelegramWebAppData validates the initData from Telegram WebApp
func verifyTelegramWebAppData(initData, botToken string) (*TelegramUser, error) {
	// Parse initData
	values, err := url.ParseQuery(initData)
	if err != nil {
		return nil, fmt.Errorf("invalid initData format: %w", err)
	}

	// Extract hash
	hash := values.Get("hash")
	if hash == "" {
		return nil, fmt.Errorf("missing hash")
	}

	// Remove hash from values
	values.Del("hash")

	// Build data-check-string
	var pairs []string
	for key, vals := range values {
		for _, val := range vals {
			pairs = append(pairs, fmt.Sprintf("%s=%s", key, val))
		}
	}
	sort.Strings(pairs)
	dataCheckString := strings.Join(pairs, "\n")

	// Compute secret_key = HMAC_SHA256("WebAppData", bot_token)
	secretKeyMac := hmac.New(sha256.New, []byte("WebAppData"))
	secretKeyMac.Write([]byte(botToken))
	secretKey := secretKeyMac.Sum(nil)

	// Compute hash = HMAC_SHA256(data-check-string, secret_key)
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(dataCheckString))
	computedHash := hex.EncodeToString(h.Sum(nil))

	log.Printf("[DEBUG] Computed hash: %s, Received hash: %s", computedHash, hash)

	// Verify hash
	if computedHash != hash {
		return nil, fmt.Errorf("hash verification failed")
	}

	// Parse user data
	userJSON := values.Get("user")
	if userJSON == "" {
		return nil, fmt.Errorf("missing user data")
	}

	var user TelegramUser
	if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
		return nil, fmt.Errorf("failed to parse user data: %w", err)
	}

	return &user, nil
}

// checkChatMembership checks if user is a member of the specified chat
func checkChatMembership(userID int64, botToken, chatID string) (bool, error) {
	url := fmt.Sprintf(
		"https://api.telegram.org/bot%s/getChatMember?chat_id=%s&user_id=%d",
		botToken,
		chatID,
		userID,
	)

	log.Printf("[DEBUG] Checking chat membership: userID=%d, chatID=%s", userID, chatID)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[ERROR] Telegram API request failed: %v", err)
		return false, fmt.Errorf("failed to call Telegram API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read Telegram API response: %v", err)
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("[DEBUG] Telegram API response: %s", string(body))

	var result ChatMemberResponse
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("[ERROR] Failed to parse Telegram API response: %v", err)
		return false, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Ok {
		log.Printf("[WARN] Telegram API returned ok=false for user %d in chat %s", userID, chatID)
		return false, nil
	}

	// Check if user is a member (not left, kicked, or restricted)
	status := result.Result.Status
	isMember := status == "creator" || status == "administrator" || status == "member"
	log.Printf("[DEBUG] User %d status in chat %s: %s (isMember=%v)", userID, chatID, status, isMember)

	return isMember, nil
}
