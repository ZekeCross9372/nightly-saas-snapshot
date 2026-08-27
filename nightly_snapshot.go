package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func main() {
	bucket := flag.String("bucket", "saas-nightly-snapshots", "storage bucket")
	source := flag.String("source", "-", "JSON file to snapshot, or - for stdin")
	day := flag.String("date", time.Now().UTC().Format("2006-01-02"), "UTC snapshot date")
	flag.Parse()

	client, err := newInfrai()
	if err != nil {
		exit(err)
	}
	configureInfrai(client)

	ctx := context.Background()
	if err := infrai.bucket.create(ctx, *bucket); err != nil {
		exit(err)
	}

	data, err := readSnapshot(*source)
	if err != nil {
		exit(err)
	}
	key := "snapshots/" + *day + "/saas.json"

	head, err := infrai.object.head(ctx, *bucket, key)
	if err != nil {
		exit(err)
	}
	if head.Found {
		fmt.Printf("snapshot already present: %s\n", key)
		return
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	if err := infrai.object.put(ctx, *bucket, key, encoded); err != nil {
		exit(err)
	}

	url, err := infrai.object.presign(ctx, *bucket, key, "get", 900)
	if err != nil {
		exit(err)
	}
	fmt.Printf("stored %s\nreview URL: %s\n", key, url)
}

func readSnapshot(source string) ([]byte, error) {
	if source == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(source)
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
