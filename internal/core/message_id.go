package core

import (
	crand "crypto/rand"
	"fmt"
	"time"
)

// GenerateMessageID 生成唯一消息 ID
// 格式: <unix_nano>_<random_8chars>
// 示例: 1709234567890123456_a1b2c3d4
func GenerateMessageID() string {
	ts := time.Now().UnixNano()
	randBytes := make([]byte, 4)
	crand.Read(randBytes)
	return fmt.Sprintf("%d_%x", ts, randBytes)
}
