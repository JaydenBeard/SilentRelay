package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"golang.org/x/crypto/hkdf"
	"io"
)

// BIP39 word list (subset for 24-word mnemonic)
// In production, use the full 2048 word list
var wordList = []string{
	"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
	"absurd", "abuse", "access", "accident", "account", "accuse", "achieve", "acid",
	"acoustic", "acquire", "across", "act", "action", "actor", "actress", "actual",
	"adapt", "add", "addict", "address", "adjust", "admit", "adult", "advance",
	"advice", "aerobic", "affair", "afford", "afraid", "again", "age", "agent",
	"agree", "ahead", "aim", "air", "airport", "aisle", "alarm", "album",
	"alcohol", "alert", "alien", "all", "alley", "allow", "almost", "alone",
	"alpha", "already", "also", "alter", "always", "amateur", "amazing", "among",
	"amount", "amused", "analyst", "anchor", "ancient", "anger", "angle", "angry",
	"animal", "ankle", "announce", "annual", "another", "answer", "antenna", "antique",
	"anxiety", "any", "apart", "apology", "appear", "apple", "approve", "april",
	"arch", "arctic", "area", "arena", "argue", "arm", "armed", "armor",
	"army", "around", "arrange", "arrest", "arrive", "arrow", "art", "artefact",
	"artist", "artwork", "ask", "aspect", "assault", "asset", "assist", "assume",
	"asthma", "athlete", "atom", "attack", "attend", "attitude", "attract", "auction",
	"audit", "august", "aunt", "author", "auto", "autumn", "average", "avocado",
	"avoid", "awake", "aware", "away", "awesome", "awful", "awkward", "axis",
	"baby", "bachelor", "bacon", "badge", "bag", "balance", "balcony", "ball",
	"bamboo", "banana", "banner", "bar", "barely", "bargain", "barrel", "base",
	"basic", "basket", "battle", "beach", "bean", "beauty", "because", "become",
	"beef", "before", "begin", "behave", "behind", "believe", "below", "belt",
	"bench", "benefit", "best", "betray", "better", "between", "beyond", "bicycle",
	"bid", "bike", "bind", "biology", "bird", "birth", "bitter", "black",
	"blade", "blame", "blanket", "blast", "bleak", "bless", "blind", "blood",
	"blossom", "blouse", "blue", "blur", "blush", "board", "boat", "body",
	"boil", "bomb", "bone", "bonus", "book", "boost", "border", "boring",
	"borrow", "boss", "bottom", "bounce", "box", "boy", "bracket", "brain",
	"brand", "brass", "brave", "bread", "breeze", "brick", "bridge", "brief",
	"bright", "bring", "brisk", "broccoli", "broken", "bronze", "broom", "brother",
	"brown", "brush", "bubble", "buddy", "budget", "buffalo", "build", "bulb",
	"bulk", "bullet", "bundle", "bunker", "burden", "burger", "burst", "bus",
	"business", "busy", "butter", "buyer", "buzz", "cabbage", "cabin", "cable",
}

var (
	ErrInvalidRecoveryKey = errors.New("invalid recovery key")
	ErrRecoveryKeyLength  = errors.New("recovery key must be 24 words")
)

// GenerateRecoveryKey creates a new 24-word recovery key (BIP39-style mnemonic)
func GenerateRecoveryKey() (string, []byte, error) {
	// Generate 256 bits of entropy (for 24 words)
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return "", nil, err
	}

	// Convert entropy to word indices
	words := make([]string, 24)
	for i := 0; i < 24; i++ {
		// Use 11 bits per word (2048 possible words, but we use first 256)
		// Simplified: just use byte values mod word list length
		idx := int(entropy[i]) % len(wordList)
		words[i] = wordList[idx]
	}

	mnemonic := strings.Join(words, " ")

	return mnemonic, entropy, nil
}

// ValidateRecoveryKey checks if a recovery key is valid and returns the entropy
func ValidateRecoveryKey(mnemonic string) ([]byte, error) {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(mnemonic)))

	if len(words) != 24 {
		return nil, ErrRecoveryKeyLength
	}

	// Build word index map
	wordIndex := make(map[string]int)
	for i, word := range wordList {
		wordIndex[word] = i
	}

	// Convert words back to entropy
	entropy := make([]byte, 32)
	for i, word := range words {
		idx, ok := wordIndex[word]
		if !ok {
			return nil, ErrInvalidRecoveryKey
		}
		entropy[i] = byte(idx)
	}

	return entropy, nil
}

// DeriveKeyFromRecoveryKey derives an encryption key from the recovery key
func DeriveKeyFromRecoveryKey(entropy []byte, salt []byte, keyLen int) ([]byte, error) {
	if len(entropy) != 32 {
		return nil, errors.New("invalid entropy length")
	}

	// Use HKDF to derive key
	reader := hkdf.New(sha256.New, entropy, salt, []byte("secure-messenger-backup-key"))
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}

	return key, nil
}

// HashRecoveryKey creates a hash of the recovery key for verification
// This is stored in the database - the actual key is never stored
func HashRecoveryKey(entropy []byte) string {
	hash := sha256.Sum256(entropy)
	return hex.EncodeToString(hash[:])
}

// VerifyRecoveryKeyHash checks if a recovery key matches its hash
func VerifyRecoveryKeyHash(entropy []byte, expectedHash string) bool {
	actualHash := HashRecoveryKey(entropy)
	return actualHash == expectedHash
}

// EncryptMasterKey encrypts the user's master key with recovery key
func EncryptMasterKey(masterKey []byte, recoveryEntropy []byte) ([]byte, error) {
	// Derive encryption key from recovery key
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	encKey, err := DeriveKeyFromRecoveryKey(recoveryEntropy, salt, 32)
	if err != nil {
		return nil, err
	}

	// Encrypt with AES-256-GCM
	encrypted, err := EncryptAESGCM(masterKey, encKey)
	if err != nil {
		return nil, err
	}

	// Prepend salt
	result := make([]byte, len(salt)+len(encrypted))
	copy(result[:len(salt)], salt)
	copy(result[len(salt):], encrypted)

	return result, nil
}

// DecryptMasterKey decrypts the master key using recovery key
func DecryptMasterKey(encryptedMasterKey []byte, recoveryEntropy []byte) ([]byte, error) {
	if len(encryptedMasterKey) < 16 {
		return nil, errors.New("invalid encrypted data")
	}

	// Extract salt
	salt := encryptedMasterKey[:16]
	ciphertext := encryptedMasterKey[16:]

	// Derive decryption key
	decKey, err := DeriveKeyFromRecoveryKey(recoveryEntropy, salt, 32)
	if err != nil {
		return nil, err
	}

	// Decrypt
	return DecryptAESGCM(ciphertext, decKey)
}
