package domain

import "context"

type BloomRepository interface {
	// Add 将 ID 加入过滤器
	Add(ctx context.Context, id int64) error

	// Exists 检查 ID 是否可能存在
	// 返回 true: 可能存在 (需要进一步查 Cache/DB)
	// 返回 false: 绝对不存在 (直接返回 404)
	Exists(ctx context.Context, id int64) (bool, error)

	// BulkAdd 用于大量添加 ID
	BulkAdd(ctx context.Context, ids []int64) error
}
