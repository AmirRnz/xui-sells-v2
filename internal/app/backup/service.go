package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/crypto/pbkdf2"

	"xui-sells-v2/internal/domain"
)

var (
	MagicHeader = []byte("XUIS")
	Version     = byte(1)

	ErrInvalidBackupFormat = errors.New("invalid backup file format")
	ErrDecryptionFailed    = errors.New("backup decryption failed: incorrect password or corrupted file")
	ErrChecksumMismatch    = errors.New("backup integrity verification failed: checksum mismatch")
	ErrScopeMismatch       = errors.New("backup scope does not match target")
	ErrPasswordRequired    = errors.New("password cannot be empty")
)

const (
	pbkdf2Iterations = 64_000
	keyLength        = 32 // AES-256
	saltLength       = 16
	nonceLength      = 12 // GCM standard
)

type BackupScope string

const (
	ScopeGlobal   BackupScope = "global"
	ScopeInstance BackupScope = "instance"
)

// Manifest contains metadata and cryptographic integrity proof of a backup archive.
type Manifest struct {
	Version     string      `json:"version"`
	Scope       BackupScope `json:"scope"`
	InstanceID  *int64      `json:"instance_id,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	ChecksumSHA string      `json:"checksum_sha"`
}

// BackupData bundles domain entities for backup and restore.
type BackupData struct {
	Instances     []domain.Instance                   `json:"instances,omitempty"`
	Plans         []domain.Plan                       `json:"plans,omitempty"`
	Services      []domain.Service                    `json:"services,omitempty"`
	Orders        []domain.Order                      `json:"orders,omitempty"`
	Transactions  []domain.WalletTransaction          `json:"transactions,omitempty"`
	Reservations  []domain.ResellerFundingReservation `json:"reservations,omitempty"`
	Tickets       []domain.Ticket                     `json:"tickets,omitempty"`
	MediaFiles    map[string][]byte                   `json:"media_files,omitempty"`
}

// DataProvider abstracts persistence operations for backup dumping and restoration.
type DataProvider interface {
	GetGlobalData(ctx context.Context) (*BackupData, error)
	GetInstanceData(ctx context.Context, instanceID int64) (*BackupData, error)
	RestoreGlobalData(ctx context.Context, data *BackupData) error
	RestoreInstanceData(ctx context.Context, instanceID int64, data *BackupData) error
}

// Service coordinates encrypted backup generation, checksum verification, and roundtrip restoration.
type Service struct {
	provider DataProvider
}

// NewService creates a new backup service.
func NewService(provider DataProvider) *Service {
	return &Service{provider: provider}
}

// CreateBackup generates an encrypted, checksummed tar.gz archive.
// Adheres strictly to P23: For instance backups, preserves child metadata and relations
// without duplicating parent/shared wallet transactions or reservation balances.
func (s *Service) CreateBackup(ctx context.Context, scope BackupScope, instanceID *int64, password string) ([]byte, error) {
	if password == "" {
		return nil, ErrPasswordRequired
	}

	var data *BackupData
	var err error

	if scope == ScopeGlobal {
		data, err = s.provider.GetGlobalData(ctx)
	} else {
		if instanceID == nil {
			return nil, errors.New("instanceID must be provided for instance backup")
		}
		data, err = s.provider.GetInstanceData(ctx, *instanceID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve backup data: %w", err)
	}

	// P23 Enforcement: Instance backup must not duplicate shared financial ledger state
	if scope == ScopeInstance {
		data.Transactions = nil
		data.Reservations = nil
	}

	// 1. Serialize data
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize backup data: %w", err)
	}

	// 2. Calculate data payload SHA-256
	h := sha256.New()
	h.Write(dataBytes)
	checksumHex := hex.EncodeToString(h.Sum(nil))

	// 3. Create manifest
	manifest := Manifest{
		Version:     "2.0",
		Scope:       scope,
		InstanceID:  instanceID,
		CreatedAt:   time.Now().UTC(),
		ChecksumSHA: checksumHex,
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize manifest: %w", err)
	}

	// 4. Archive into tar.gz
	tarGzBuf := new(bytes.Buffer)
	gzWriter := gzip.NewWriter(tarGzBuf)
	tarWriter := tar.NewWriter(gzWriter)

	// Write manifest.json
	if err := writeTarEntry(tarWriter, "manifest.json", manifestBytes); err != nil {
		return nil, err
	}
	// Write data.json
	if err := writeTarEntry(tarWriter, "data.json", dataBytes); err != nil {
		return nil, err
	}

	// Write media files
	for filename, content := range data.MediaFiles {
		entryName := fmt.Sprintf("media/%s", filename)
		if err := writeTarEntry(tarWriter, entryName, content); err != nil {
			return nil, err
		}
	}

	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := gzWriter.Close(); err != nil {
		return nil, err
	}

	// 5. Encrypt with AES-256-GCM using PBKDF2 derived key
	plainArchive := tarGzBuf.Bytes()
	salt := make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, keyLength, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GCM: %w", err)
	}

	nonce := make([]byte, nonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, plainArchive, nil)

	// 6. Assemble envelope: Magic (4) + Version (1) + Salt (16) + Nonce (12) + Ciphertext
	out := new(bytes.Buffer)
	out.Write(MagicHeader)
	out.WriteByte(Version)
	out.Write(salt)
	out.Write(nonce)
	out.Write(ciphertext)

	return out.Bytes(), nil
}

// RestoreBackup decrypts, verifies integrity, and restores backup data.
func (s *Service) RestoreBackup(ctx context.Context, encryptedData []byte, password string, targetInstanceID *int64) (*Manifest, error) {
	manifest, data, err := s.DecryptAndUnpack(encryptedData, password)
	if err != nil {
		return nil, err
	}

	if manifest.Scope == ScopeGlobal {
		if err := s.provider.RestoreGlobalData(ctx, data); err != nil {
			return nil, fmt.Errorf("failed to restore global data: %w", err)
		}
	} else if manifest.Scope == ScopeInstance {
		id := manifest.InstanceID
		if targetInstanceID != nil {
			id = targetInstanceID
		}
		if id == nil {
			return nil, errors.New("no instance ID available for instance restore")
		}
		if err := s.provider.RestoreInstanceData(ctx, *id, data); err != nil {
			return nil, fmt.Errorf("failed to restore instance data: %w", err)
		}
	}

	return manifest, nil
}

// DecryptAndUnpack performs envelope decryption, SHA-256 verification, and archive extraction.
func (s *Service) DecryptAndUnpack(encryptedData []byte, password string) (*Manifest, *BackupData, error) {
	if password == "" {
		return nil, nil, ErrPasswordRequired
	}

	minLen := len(MagicHeader) + 1 + saltLength + nonceLength
	if len(encryptedData) < minLen {
		return nil, nil, ErrInvalidBackupFormat
	}

	// 1. Verify Magic
	if !bytes.Equal(encryptedData[:len(MagicHeader)], MagicHeader) {
		return nil, nil, ErrInvalidBackupFormat
	}
	offset := len(MagicHeader)

	// Version
	ver := encryptedData[offset]
	if ver != Version {
		return nil, nil, fmt.Errorf("unsupported backup version: %d", ver)
	}
	offset++

	salt := encryptedData[offset : offset+saltLength]
	offset += saltLength

	nonce := encryptedData[offset : offset+nonceLength]
	offset += nonceLength

	ciphertext := encryptedData[offset:]

	// 2. Derive key and decrypt
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, keyLength, sha256.New)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, ErrDecryptionFailed
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, ErrDecryptionFailed
	}

	plainArchive, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, nil, ErrDecryptionFailed
	}

	// 3. Decompress tar.gz
	gzReader, err := gzip.NewReader(bytes.NewReader(plainArchive))
	if err != nil {
		return nil, nil, fmt.Errorf("corrupted gzip archive: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var manifestBytes []byte
	var dataBytes []byte
	mediaFiles := make(map[string][]byte)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("corrupted tar archive: %w", err)
		}

		content, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read tar entry %s: %w", header.Name, err)
		}

		switch header.Name {
		case "manifest.json":
			manifestBytes = content
		case "data.json":
			dataBytes = content
		default:
			if len(header.Name) > 6 && header.Name[:6] == "media/" {
				mediaFiles[header.Name[6:]] = content
			}
		}
	}

	if len(manifestBytes) == 0 || len(dataBytes) == 0 {
		return nil, nil, errors.New("archive missing required manifest or data files")
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// 4. Verify SHA-256 Checksum
	h := sha256.New()
	h.Write(dataBytes)
	actualChecksum := hex.EncodeToString(h.Sum(nil))

	if actualChecksum != manifest.ChecksumSHA {
		return nil, nil, ErrChecksumMismatch
	}

	var data BackupData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return nil, nil, fmt.Errorf("failed to parse backup data: %w", err)
	}
	data.MediaFiles = mediaFiles

	return &manifest, &data, nil
}

// CreateBackupToFile helper to write backup directly to a file on disk.
func (s *Service) CreateBackupToFile(ctx context.Context, scope BackupScope, instanceID *int64, password, filePath string) error {
	data, err := s.CreateBackup(ctx, scope, instanceID, password)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

// RestoreBackupFromFile helper to restore backup directly from a file on disk.
func (s *Service) RestoreBackupFromFile(ctx context.Context, filePath, password string, targetInstanceID *int64) (*Manifest, error) {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup file: %w", err)
	}
	return s.RestoreBackup(ctx, bytes, password, targetInstanceID)
}

func writeTarEntry(tw *tar.Writer, name string, content []byte) error {
	hdr := &tar.Header{
		Name:     name,
		Mode:     0600,
		Size:     int64(len(content)),
		ModTime:  time.Now().UTC(),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(content)
	return err
}
