package clusterinfo

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/binxio/gcloudconfig"
)

func TestListClusters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	creds, err := gcloudconfig.GetCredentials("")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := NewCache(ctx, creds.ProjectID, creds, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	clusters := cache.GetMap()
	if len(*clusters) == 0 {
		t.Fatalf("expected at least 1 cluster, found none")
	}

	for _, cluster := range *clusters {
		log.Printf("%v", cluster)
	}
}
