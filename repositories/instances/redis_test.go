package instances

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/verbeux-ai/whatsmiau/models"
)

func newTestRepository(t *testing.T) (*RedisInstance, *redis.Client, *miniredis.Miniredis) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRedis(client), client, server
}

func TestCreateIsAtomic(t *testing.T) {
	repo, _, _ := newTestRepository(t)
	ctx := context.Background()

	if err := repo.Create(ctx, &models.Instance{ID: "same"}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := repo.Create(ctx, &models.Instance{ID: "same"}); !errors.Is(err, ErrorAlreadyExists) {
		t.Fatalf("second create error = %v, want ErrorAlreadyExists", err)
	}
}

type deleteBeforeConditionalSetHook struct {
	server *miniredis.Miniredis
	key    string
}

func (h deleteBeforeConditionalSetHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	if cmd.Name() == "set" && strings.EqualFold(cmd.Args()[len(cmd.Args())-1].(string), "XX") {
		h.server.Del(h.key)
	}
	return ctx, nil
}

func (deleteBeforeConditionalSetHook) AfterProcess(context.Context, redis.Cmder) error {
	return nil
}

func (deleteBeforeConditionalSetHook) BeforeProcessPipeline(ctx context.Context, _ []redis.Cmder) (context.Context, error) {
	return ctx, nil
}

func (deleteBeforeConditionalSetHook) AfterProcessPipeline(context.Context, []redis.Cmder) error {
	return nil
}

func TestUpdateDoesNotResurrectDeletedKey(t *testing.T) {
	repo, client, server := newTestRepository(t)
	ctx := context.Background()
	instance := &models.Instance{ID: "race"}

	if err := repo.Create(ctx, instance); err != nil {
		t.Fatalf("create: %v", err)
	}
	client.AddHook(deleteBeforeConditionalSetHook{server: server, key: repo.key(instance.ID)})

	_, err := repo.Update(ctx, instance.ID, &models.Instance{RemoteJID: "123@s.whatsapp.net"})
	if !errors.Is(err, ErrorNotFound) {
		t.Fatalf("update error = %v, want ErrorNotFound", err)
	}
	if server.Exists(repo.key(instance.ID)) {
		t.Fatal("conditional update resurrected the deleted Redis key")
	}
}

func TestDeleteReportsMissingInstance(t *testing.T) {
	repo, _, _ := newTestRepository(t)
	if err := repo.Delete(context.Background(), "missing"); !errors.Is(err, ErrorNotFound) {
		t.Fatalf("delete error = %v, want ErrorNotFound", err)
	}
}
