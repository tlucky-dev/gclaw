package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SecretType 敏感信息类型
type SecretType string

const (
	SecretTypeAPIKey     SecretType = "api_key"
	SecretTypePassword   SecretType = "password"
	SecretTypeToken      SecretType = "token"
	SecretTypePrivateKey SecretType = "private_key"
	SecretTypeCredential SecretType = "credential"
)

// SecretEntry 敏感信息条目
type SecretEntry struct {
	Key       string     `json:"key"`
	Value     string     `json:"value"` // 加密后的值
	Type      SecretType `json:"type"`
	CreatedAt int64      `json:"created_at"`
}

// SecretManager 敏感信息管理器
type SecretManager struct {
	mu          sync.RWMutex
	secrets     map[string]*SecretEntry
	encryptKey  []byte
	vaultPath   string
	autoSave    bool
}

// InputValidator 输入验证器
type InputValidator struct {
	maxLength         int
	allowedPatterns   []*regexp.Regexp
	blockedPatterns   []*regexp.Regexp
	injectionPatterns []*regexp.Regexp
}

// NewSecretManager 创建敏感信息管理器
func NewSecretManager(encryptKey string, vaultPath string) (*SecretManager, error) {
	if encryptKey == "" {
		// 生成随机密钥
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("failed to generate encryption key: %w", err)
		}
		encryptKey = hex.EncodeToString(key)
		fmt.Printf("WARNING: Generated new encryption key: %s\n", encryptKey)
		fmt.Println("Store this key securely - you will need it to decrypt secrets!")
	}

	keyBytes, err := hex.DecodeString(encryptKey)
	if err != nil {
		// 尝试使用原始字符串作为密钥
		hash := sha256.Sum256([]byte(encryptKey))
		keyBytes = hash[:]
	}

	sm := &SecretManager{
		secrets:    make(map[string]*SecretEntry),
		encryptKey: keyBytes,
		vaultPath:  vaultPath,
		autoSave:   true,
	}

	// 加载已有的密钥库
	if vaultPath != "" {
		if err := sm.LoadVault(); err != nil {
			fmt.Printf("Warning: failed to load vault: %v\n", err)
		}
	}

	return sm, nil
}

// Store 存储敏感信息（自动加密）
func (sm *SecretManager) Store(key string, value string, secretType SecretType) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	encrypted, err := sm.encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	sm.secrets[key] = &SecretEntry{
		Key:       key,
		Value:     encrypted,
		Type:      secretType,
		CreatedAt: time.Now().Unix(),
	}

	if sm.autoSave && sm.vaultPath != "" {
		return sm.SaveVault()
	}

	return nil
}

// Get 获取敏感信息（自动解密）
func (sm *SecretManager) Get(key string) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	entry, exists := sm.secrets[key]
	if !exists {
		return "", fmt.Errorf("secret '%s' not found", key)
	}

	decrypted, err := sm.decrypt(entry.Value)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return decrypted, nil
}

// Delete 删除敏感信息
func (sm *SecretManager) Delete(key string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.secrets, key)

	if sm.autoSave && sm.vaultPath != "" {
		return sm.SaveVault()
	}

	return nil
}

// List 列出所有敏感信息键（不暴露值）
func (sm *SecretManager) List() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	keys := make([]string, 0, len(sm.secrets))
	for k := range sm.secrets {
		keys = append(keys, k)
	}
	return keys
}

// SaveVault 保存密钥库到磁盘
func (sm *SecretManager) SaveVault() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.vaultPath == "" {
		return fmt.Errorf("vault path not configured")
	}

	// 确保目录存在
	dir := filepath.Dir(sm.vaultPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.Marshal(sm.secrets)
	if err != nil {
		return err
	}

	// 加密整个 vault
	encrypted, err := sm.encrypt(string(data))
	if err != nil {
		return err
	}

	return os.WriteFile(sm.vaultPath, []byte(encrypted), 0600)
}

// LoadVault 从磁盘加载密钥库
func (sm *SecretManager) LoadVault() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.vaultPath == "" {
		return fmt.Errorf("vault path not configured")
	}

	data, err := os.ReadFile(sm.vaultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在是正常的
		}
		return err
	}

	// 解密 vault
	decrypted, err := sm.decrypt(string(data))
	if err != nil {
		return err
	}

	var secrets map[string]*SecretEntry
	if err := json.Unmarshal([]byte(decrypted), &secrets); err != nil {
		return err
	}

	sm.secrets = secrets
	return nil
}

// encrypt 加密数据
func (sm *SecretManager) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(sm.encryptKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 解密数据
func (sm *SecretManager) decrypt(ciphertextStr string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertextStr)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(sm.encryptKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := data[:nonceSize]
	ct := data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// NewInputValidator 创建输入验证器
func NewInputValidator() *InputValidator {
	return &InputValidator{
		maxLength: 10000,
		allowedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^[\s\S]*$`), // 默认允许所有
		},
		blockedPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)<script[^>]*>`),
			regexp.MustCompile(`(?i)javascript:`),
			regexp.MustCompile(`(?i)data:text/html`),
		},
		injectionPatterns: []*regexp.Regexp{
			// 提示词注入模式
			regexp.MustCompile(`(?i)ignore\s+previous\s+instructions`),
			regexp.MustCompile(`(?i)forget\s+(all|everything)`),
			regexp.MustCompile(`(?i)bypass\s+(security|restrictions)`),
			regexp.MustCompile(`(?i)you\s+are\s+now\s+(in\s+)?(developer|debug)\s+mode`),
			regexp.MustCompile(`(?i)(system|developer)\s+message:`),
			regexp.MustCompile(`(?i)output\s+only\s+the`),
			regexp.MustCompile(`(?i)do\s+not\s+(follow|obey)\s+(your|the)\s+(rules|instructions)`),
			// SQL 注入
			regexp.MustCompile(`(?i)'\s*(OR|AND)\s+'?\d*'?\s*=\s*'?\d*`),
			regexp.MustCompile(`(?i);\s*(DROP|DELETE|UPDATE|INSERT)`),
			// 命令注入
			regexp.MustCompile(`[;&|` + "`" + `$()]`),
		},
	}
}

// Validate 验证用户输入
func (iv *InputValidator) Validate(input string) error {
	// 检查长度
	if len(input) > iv.maxLength {
		return fmt.Errorf("input exceeds maximum length of %d characters", iv.maxLength)
	}

	// 检查空输入
	if strings.TrimSpace(input) == "" {
		return fmt.Errorf("empty input not allowed")
	}

	// 检查阻止的模式
	for _, pattern := range iv.blockedPatterns {
		if pattern.MatchString(input) {
			return fmt.Errorf("input contains blocked pattern: %s", pattern.String())
		}
	}

	// 检查注入攻击模式
	for _, pattern := range iv.injectionPatterns {
		if pattern.MatchString(input) {
			return fmt.Errorf("potential injection attack detected: %s", pattern.String())
		}
	}

	return nil
}

// Sanitize 清理输入
func (iv *InputValidator) Sanitize(input string) string {
	// 移除潜在的恶意标签
	sanitized := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(input, "")
	
	// 移除 javascript: 协议
	sanitized = regexp.MustCompile(`(?i)javascript:`).ReplaceAllString(sanitized, "")
	
	// 移除 data: URL
	sanitized = regexp.MustCompile(`(?i)data:[^,]*,`).ReplaceAllString(sanitized, "")
	
	// 转义特殊字符
	sanitized = strings.ReplaceAll(sanitized, "&", "&amp;")
	sanitized = strings.ReplaceAll(sanitized, "<", "&lt;")
	sanitized = strings.ReplaceAll(sanitized, ">", "&gt;")
	
	return sanitized
}

// DetectInjection 检测是否为注入攻击
func (iv *InputValidator) DetectInjection(input string) (bool, string) {
	for _, pattern := range iv.injectionPatterns {
		if pattern.MatchString(input) {
			return true, pattern.String()
		}
	}
	return false, ""
}

// SetMaxLength 设置最大长度
func (iv *InputValidator) SetMaxLength(max int) {
	iv.maxLength = max
}

// AddBlockedPattern 添加阻止模式
func (iv *InputValidator) AddBlockedPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	iv.blockedPatterns = append(iv.blockedPatterns, re)
	return nil
}

// AddInjectionPattern 添加注入检测模式
func (iv *InputValidator) AddInjectionPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	iv.injectionPatterns = append(iv.injectionPatterns, re)
	return nil
}

// RedactSecrets 从文本中脱敏敏感信息
func RedactSecrets(text string, secrets map[string]string) string {
	result := text
	for key, value := range secrets {
		if value != "" {
			// 替换为占位符
			placeholder := fmt.Sprintf("[%s_REDACTED]", strings.ToUpper(key))
			result = strings.ReplaceAll(result, value, placeholder)
		}
	}
	return result
}

// IsSecurePath 检查路径是否安全（防止目录遍历）
func IsSecurePath(path string, baseDir string) bool {
	// 清理路径
	cleanPath := filepath.Clean(path)
	
	// 检查是否包含父目录引用
	if strings.Contains(cleanPath, "..") {
		return false
	}
	
	// 检查是否是绝对路径
	if filepath.IsAbs(cleanPath) && baseDir != "" {
		absBase, _ := filepath.Abs(baseDir)
		absPath, _ := filepath.Abs(cleanPath)
		return strings.HasPrefix(absPath, absBase)
	}
	
	return true
}

// GenerateSecureToken 生成安全令牌
func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
