package clusterinfo

import (
	"context"
	"github.com/binxio/gcloudconfig"
	"log"
	"testing"
	"time"
)

func TestListClusters(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	creds, err := gcloudconfig.GetCredentials("")
	if err != nil {
		t.Fatal(err)
	}
	cache, err := NewClusterInfoCache(ctx, creds.ProjectID, creds, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	clusters := cache.GetClusterInfo()
	if len(*clusters) == 0 {
		t.Fatalf("expected at least 1 cluster, found none")
	}

	for _, cluster := range *clusters {
		log.Printf("%v", cluster)
	}
}
