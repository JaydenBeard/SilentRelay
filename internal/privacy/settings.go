package privacy

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Settings represents a user's privacy settings
type Settings struct {
	UserID                      uuid.UUID `json:"user_id"`
	ShowReadReceipts            bool      `json:"show_read_receipts"`
	ShowLastSeen                bool      `json:"show_last_seen"`
	ShowTypingIndicator         bool      `json:"show_typing_indicator"`
	WhoCanSeeProfile            string    `json:"who_can_see_profile"`           // everyone, contacts, nobody
	WhoCanAddToGroups           string    `json:"who_can_add_to_groups"`         // everyone, contacts, nobody
	DisappearingMessagesDefault *int      `json:"disappearing_messages_default"` // seconds, null = disabled
	UpdatedAt                   time.Time `json:"updated_at"`
}

// SettingsService handles privacy settings
type SettingsService struct {
	db *sql.DB
}

// NewSettingsService creates a new settings service
func NewSettingsService(db *sql.DB) *SettingsService {
	return &SettingsService{db: db}
}

// GetSettings retrieves a user's privacy settings
func (s *SettingsService) GetSettings(userID uuid.UUID) (*Settings, error) {
	settings := &Settings{UserID: userID}

	err := s.db.QueryRow(`
		SELECT show_read_receipts, show_last_seen, show_typing_indicator,
		       who_can_see_profile, who_can_add_to_groups, disappearing_messages_default, updated_at
		FROM privacy_settings WHERE user_id = $1
	`, userID).Scan(
		&settings.ShowReadReceipts,
		&settings.ShowLastSeen,
		&settings.ShowTypingIndicator,
		&settings.WhoCanSeeProfile,
		&settings.WhoCanAddToGroups,
		&settings.DisappearingMessagesDefault,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return defaults
		return &Settings{
			UserID:              userID,
			ShowReadReceipts:    true,
			ShowLastSeen:        true,
			ShowTypingIndicator: true,
			WhoCanSeeProfile:    "everyone",
			WhoCanAddToGroups:   "everyone",
			UpdatedAt:           time.Now(),
		}, nil
	}

	return settings, err
}

// UpdateSettings updates a user's privacy settings
func (s *SettingsService) UpdateSettings(userID uuid.UUID, updates map[string]interface{}) error {
	// Upsert settings
	_, err := s.db.Exec(`
		INSERT INTO privacy_settings (user_id, show_read_receipts, show_last_seen, 
			show_typing_indicator, who_can_see_profile, who_can_add_to_groups, 
			disappearing_messages_default, updated_at)
		VALUES ($1, 
			COALESCE($2, true), COALESCE($3, true), COALESCE($4, true),
			COALESCE($5, 'everyone'), COALESCE($6, 'everyone'), $7, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			show_read_receipts = COALESCE($2, privacy_settings.show_read_receipts),
			show_last_seen = COALESCE($3, privacy_settings.show_last_seen),
			show_typing_indicator = COALESCE($4, privacy_settings.show_typing_indicator),
			who_can_see_profile = COALESCE($5, privacy_settings.who_can_see_profile),
			who_can_add_to_groups = COALESCE($6, privacy_settings.who_can_add_to_groups),
			disappearing_messages_default = COALESCE($7, privacy_settings.disappearing_messages_default),
			updated_at = NOW()
	`,
		userID,
		updates["show_read_receipts"],
		updates["show_last_seen"],
		updates["show_typing_indicator"],
		updates["who_can_see_profile"],
		updates["who_can_add_to_groups"],
		updates["disappearing_messages_default"],
	)

	return err
}

// CanSendReadReceipt checks if read receipts should be sent to a user
func (s *SettingsService) CanSendReadReceipt(readerID, senderID uuid.UUID) (bool, error) {
	var showReadReceipts bool
	err := s.db.QueryRow(`
		SELECT COALESCE(show_read_receipts, true) 
		FROM privacy_settings WHERE user_id = $1
	`, readerID).Scan(&showReadReceipts)

	if err == sql.ErrNoRows {
		return true, nil // Default is to show
	}
	return showReadReceipts, err
}

// CanSeeLastSeen checks if a user can see another user's last seen
func (s *SettingsService) CanSeeLastSeen(viewerID, targetID uuid.UUID) (bool, error) {
	var showLastSeen bool
	var whoCanSee string

	err := s.db.QueryRow(`
		SELECT COALESCE(show_last_seen, true), COALESCE(who_can_see_profile, 'everyone')
		FROM privacy_settings WHERE user_id = $1
	`, targetID).Scan(&showLastSeen, &whoCanSee)

	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	if !showLastSeen {
		return false, nil
	}

	switch whoCanSee {
	case "nobody":
		return false, nil
	case "contacts":
		return s.areContacts(viewerID, targetID)
	default:
		return true, nil
	}
}

// CanAddToGroup checks if a user can be added to a group by another user
func (s *SettingsService) CanAddToGroup(adderID, targetID uuid.UUID) (bool, error) {
	var whoCanAdd string

	err := s.db.QueryRow(`
		SELECT COALESCE(who_can_add_to_groups, 'everyone')
		FROM privacy_settings WHERE user_id = $1
	`, targetID).Scan(&whoCanAdd)

	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	switch whoCanAdd {
	case "nobody":
		return false, nil
	case "contacts":
		return s.areContacts(adderID, targetID)
	default:
		return true, nil
	}
}

// areContacts checks if two users are in each other's contacts
func (s *SettingsService) areContacts(user1, user2 uuid.UUID) (bool, error) {
	var count int
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM user_contacts 
		WHERE user_id = $1 AND matched_user_id = $2
	`, user1, user2).Scan(&count)

	return count > 0, err
}

// GetDisappearingMessageDefault gets the default disappearing message duration
func (s *SettingsService) GetDisappearingMessageDefault(userID uuid.UUID) (*int, error) {
	var duration *int
	err := s.db.QueryRow(`
		SELECT disappearing_messages_default FROM privacy_settings WHERE user_id = $1
	`, userID).Scan(&duration)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return duration, err
}
