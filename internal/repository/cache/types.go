package cache

import "time"

// DataWithLogicalExpire 支持逻辑过期的数据结构
type DataWithLogicalExpire struct {
	Data      any       `json:"data"`
	ExpireAt  time.Time `json:"expire_at"`  // 逻辑过期时间
	CreatedAt time.Time `json:"created_at"` // 创建时间，用于调试
}

// IsLogicalExpired 检查是否逻辑过期
func (d *DataWithLogicalExpire) IsLogicalExpired() bool {
	return time.Now().After(d.ExpireAt)
}

// NewDataWithLogicalExpire 创建带逻辑过期的数据
func NewDataWithLogicalExpire(data any, ttl time.Duration) *DataWithLogicalExpire {
	now := time.Now()
	return &DataWithLogicalExpire{
		Data:      data,
		ExpireAt:  now.Add(ttl),
		CreatedAt: now,
	}
}
