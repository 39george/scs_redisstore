package redisstore

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

const NoExpiration = 0

func getClient(t *testing.T) (context.Context, *redis.Client) {
	opts := &redis.Options{
		Addr:     os.Getenv("SCS_REDIS_TEST_DSN"),
		Password: os.Getenv("SCS_REDIS_TEST_PASS"),
		DB:       0,
	}
	conn := redis.NewClient(opts)
	ctx := context.Background()

	_, err := conn.FlushDB(ctx).Result()
	if err != nil {
		t.Fatal(err)
	}

	return ctx, conn
}

func TestFind(t *testing.T) {
	ctx, conn := getClient(t)

	r := New(conn)

	err := conn.Set(ctx, r.prefix+"session_token", "encoded_data", NoExpiration).Err()
	if err != nil {
		t.Fatal(err)
	}

	b, found, err := r.Find("session_token")
	if err != nil {
		t.Fatal(err)
	}
	if found != true {
		t.Fatalf("got %v: expected %v", found, true)
	}
	if bytes.Equal(b, []byte("encoded_data")) == false {
		t.Fatalf("got %v: expected %v", b, []byte("encoded_data"))
	}
}

func TestSaveNew(t *testing.T) {
	ctx, conn := getClient(t)

	r := New(conn)

	err := r.Commit("session_token", []byte("encoded_data"), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	data, err := conn.Get(ctx, r.prefix+"session_token").Bytes()
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(data, []byte("encoded_data")) == false {
		t.Fatalf("got %v: expected %v", data, []byte("encoded_data"))
	}
}

func TestFindMissing(t *testing.T) {
	_, conn := getClient(t)

	r := New(conn)

	_, found, err := r.Find("missing_session_token")
	if err != nil {
		t.Fatalf("got %v: expected %v", err, nil)
	}
	if found != false {
		t.Fatalf("got %v: expected %v", found, false)
	}
}

func TestSaveUpdated(t *testing.T) {
	ctx, conn := getClient(t)

	r := New(conn)

	err := conn.Set(ctx, r.prefix+"session_token", "encoded_data", NoExpiration).Err()
	if err != nil {
		t.Fatal(err)
	}

	err = r.Commit("session_token", []byte("new_encoded_data"), time.Now().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	data, err := conn.Get(ctx, r.prefix+"session_token").Bytes()
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(data, []byte("new_encoded_data")) == false {
		t.Fatalf("got %v: expected %v", data, []byte("new_encoded_data"))
	}
}

func TestExpiry(t *testing.T) {
	_, conn := getClient(t)

	r := New(conn)

	err := r.Commit("session_token", []byte("encoded_data"), time.Now().Add(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}

	_, found, _ := r.Find("session_token")
	if found != true {
		t.Fatalf("got %v: expected %v", found, true)
	}

	time.Sleep(200 * time.Millisecond)
	_, found, _ = r.Find("session_token")
	if found != false {
		t.Fatalf("got %v: expected %v", found, false)
	}
}

func TestDelete(t *testing.T) {
	ctx, conn := getClient(t)

	r := New(conn)

	err := conn.Set(ctx, r.prefix+"session_token", "encoded_data", NoExpiration).Err()
	if err != nil {
		t.Fatal(err)
	}

	err = r.Delete("session_token")
	if err != nil {
		t.Fatal(err)
	}

	data, err := conn.Get(ctx, r.prefix+"session_token").Bytes()
	if err != nil && !errors.Is(err, redis.Nil) {
		t.Fatal(err)
	}

	if data != nil {
		t.Fatalf("got %v: expected %v", data, nil)
	}
}
