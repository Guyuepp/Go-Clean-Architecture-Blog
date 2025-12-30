package redis

import (
	"context"
	"fmt"
	"hash/crc32"
	"hash/fnv"

	"github.com/Guyuepp/Go-Clean-Architecture-Blog/domain"
	"github.com/redis/go-redis/v9"
)

const (
	KeyArticleBloom = "bloom:article:ids"
)

type redisBloomRepo struct {
	client       *redis.Client
	BloomBitSize uint64
}

var _ domain.BloomRepository = (*redisBloomRepo)(nil)

func NewRedisBloomRepo(client *redis.Client, bitSize uint64) *redisBloomRepo {
	return &redisBloomRepo{
		client:       client,
		BloomBitSize: bitSize,
	}
}

func (r *redisBloomRepo) Add(ctx context.Context, id int64) error {
	offsets := r.getOffset(id)
	pipe := r.client.Pipeline()
	for _, offset := range offsets {
		pipe.SetBit(ctx, KeyArticleBloom, int64(offset), 1)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r *redisBloomRepo) Exists(ctx context.Context, id int64) (bool, error) {
	offsets := r.getOffset(id)
	pipe := r.client.Pipeline()
	for _, offset := range offsets {
		pipe.GetBit(ctx, KeyArticleBloom, int64(offset))
	}
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	for _, cmd := range cmds {
		val, err := cmd.(*redis.IntCmd).Result()
		if err != nil {
			return false, err
		}
		if val == 0 {
			return false, nil
		}
	}

	return true, nil
}

func (r *redisBloomRepo) getOffset(id int64) []uint64 {
	data := fmt.Appendf(nil, "%d", id)
	offsets := make([]uint64, 3) // 假设 k=3

	// Hash 1: CRC32
	offsets[0] = uint64(crc32.ChecksumIEEE(data)) % r.BloomBitSize

	// Hash 2: FNV64
	h := fnv.New64()
	h.Write(data)
	offsets[1] = h.Sum64() % r.BloomBitSize

	// Hash 3: 线性混合
	offsets[2] = (offsets[0] + offsets[1] + 0xABC) % r.BloomBitSize

	return offsets
}

func (r *redisBloomRepo) BulkAdd(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	pipe := r.client.Pipeline()
	for _, id := range ids {
		offsets := r.getOffset(id)
		for _, offset := range offsets {
			pipe.SetBit(ctx, KeyArticleBloom, int64(offset), 1)
		}
	}

	_, err := pipe.Exec(ctx)
	return err
}
