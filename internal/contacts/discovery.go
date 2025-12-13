package contacts

import (
	"database/sql"
	"log"

	"github.com/google/uuid"
	"github.com/jaydenbeard/messaging-app/internal/security"
)

// ContactMatch represents a matched contact
type ContactMatch struct {
	ContactHash   string     `json:"contact_hash"`
	MatchedUserID *uuid.UUID `json:"matched_user_id,omitempty"`
	DisplayName   *string    `json:"display_name,omitempty"`
	Username      *string    `json:"username,omitempty"`
	AvatarURL     *string    `json:"avatar_url,omitempty"`
}

// DiscoveryService handles privacy-preserving contact discovery
type DiscoveryService struct {
	db *sql.DB
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(db *sql.DB) *DiscoveryService {
	return &DiscoveryService{db: db}
}

// UploadContactHashes uploads hashed phone numbers for contact discovery
// Client sends SHA-256 hashes of normalized phone numbers
// Server matches against registered users without seeing actual numbers
func (d *DiscoveryService) UploadContactHashes(userID uuid.UUID, hashes []string) ([]ContactMatch, error) {
	matches := make([]ContactMatch, 0, len(hashes))

	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			log.Printf("Warning: failed to rollback transaction: %v", err)
		}
	}()

	for _, hash := range hashes {
		// Check if hash matches a registered user
		var matchedUserID uuid.UUID
		var displayName, username, avatarURL sql.NullString

		err := tx.QueryRow(`
			SELECT user_id, display_name, username, avatar_url
			FROM users WHERE phone_hash = $1 AND is_active = true AND user_id != $2
		`, hash, userID).Scan(&matchedUserID, &displayName, &username, &avatarURL)

		match := ContactMatch{ContactHash: hash}

		if err == nil {
			// Found a match
			match.MatchedUserID = &matchedUserID
			if displayName.Valid {
				match.DisplayName = &displayName.String
			}
			if username.Valid {
				match.Username = &username.String
			}
			if avatarURL.Valid {
				match.AvatarURL = &avatarURL.String
			}

			// Store the contact relationship
			_, err = tx.Exec(`
				INSERT INTO user_contacts (user_id, contact_hash, matched_user_id)
				VALUES ($1, $2, $3)
				ON CONFLICT (user_id, contact_hash) DO UPDATE SET matched_user_id = $3
			`, userID, hash, matchedUserID)
			if err != nil {
				return nil, err
			}
		} else if err == sql.ErrNoRows {
			// No match - still store for future matching
			_, err = tx.Exec(`
				INSERT INTO user_contacts (user_id, contact_hash)
				VALUES ($1, $2)
				ON CONFLICT (user_id, contact_hash) DO NOTHING
			`, userID, hash)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}

		matches = append(matches, match)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return matches, nil
}

// CheckNewUserAgainstContacts checks if a new user matches any stored contact hashes
// Called when a new user registers to update their contacts' lists
func (d *DiscoveryService) CheckNewUserAgainstContacts(newUserID uuid.UUID, phoneNumber string) error {
	phoneHash := security.HashPhoneNumber(phoneNumber)

	// Find all users who have this phone hash in their contacts
	_, err := d.db.Exec(`
		UPDATE user_contacts SET matched_user_id = $1
		WHERE contact_hash = $2 AND matched_user_id IS NULL
	`, newUserID, phoneHash)

	return err
}

// GetMatchedContacts returns all matched contacts for a user
func (d *DiscoveryService) GetMatchedContacts(userID uuid.UUID) ([]ContactMatch, error) {
	rows, err := d.db.Query(`
		SELECT c.contact_hash, c.matched_user_id, u.display_name, u.username, u.avatar_url
		FROM user_contacts c
		LEFT JOIN users u ON c.matched_user_id = u.user_id
		WHERE c.user_id = $1 AND c.matched_user_id IS NOT NULL
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var contacts []ContactMatch
	for rows.Next() {
		var match ContactMatch
		var matchedUserID uuid.UUID
		var displayName, username, avatarURL sql.NullString

		if err := rows.Scan(&match.ContactHash, &matchedUserID, &displayName, &username, &avatarURL); err != nil {
			return nil, err
		}

		match.MatchedUserID = &matchedUserID
		if displayName.Valid {
			match.DisplayName = &displayName.String
		}
		if username.Valid {
			match.Username = &username.String
		}
		if avatarURL.Valid {
			match.AvatarURL = &avatarURL.String
		}

		contacts = append(contacts, match)
	}

	return contacts, nil
}

// SetContactNickname sets a custom nickname for a contact
func (d *DiscoveryService) SetContactNickname(userID uuid.UUID, contactHash, nickname string) error {
	_, err := d.db.Exec(`
		UPDATE user_contacts SET nickname = $1
		WHERE user_id = $2 AND contact_hash = $3
	`, nickname, userID, contactHash)
	return err
}

// RemoveContact removes a contact from the user's list
func (d *DiscoveryService) RemoveContact(userID uuid.UUID, contactHash string) error {
	_, err := d.db.Exec(`
		DELETE FROM user_contacts WHERE user_id = $1 AND contact_hash = $2
	`, userID, contactHash)
	return err
}

// HashPhoneNumbers is a helper that clients can use (though they should hash locally)
func HashPhoneNumbers(phoneNumbers []string) []string {
	hashes := make([]string, len(phoneNumbers))
	for i, phone := range phoneNumbers {
		hashes[i] = security.HashPhoneNumber(phone)
	}
	return hashes
}
