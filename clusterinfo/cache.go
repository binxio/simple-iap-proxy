package clusterinfo

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
	"log"
	"strings"
	"sync"
	"time"
)

type ClusterInfo struct {
	Name                 string
	Endpoint             string
	ClusterCaCertificate string
	RootCAs              *x509.CertPool
}

// map from endpoint to name and certificate
type ClusterInfoMap map[string]*ClusterInfo

type ClusterInfoCache struct {
	ctx         context.Context
	projectId   string
	credentials *google.Credentials
	refresh     time.Duration
	clusterInfo *ClusterInfoMap
	mutex       sync.Mutex
}

func NewClusterInfoCache(ctx context.Context, projectId string, credentials *google.Credentials, refresh time.Duration) (*ClusterInfoCache, error) {
	cache := &ClusterInfoCache{
		ctx:         ctx,
		credentials: credentials,
		projectId:   projectId,
		refresh:     refresh,
	}
	clusterInfo, err := cache.retrieveClusters()
	if err != nil {
		return nil, err
	}
	cache.clusterInfo = clusterInfo
	go cache.run()
	return cache, nil
}

func (c *ClusterInfoCache) GetClusterInfoForEndpoint(endpoint string) *ClusterInfo {
	host := strings.Split(endpoint, ":")
	if r, ok := (*c.clusterInfo)[host[0]]; ok {
		return r
	} else {
		return nil
	}
}

// thread safe get cluster info
func (c *ClusterInfoCache) getClusterInfo() *ClusterInfoMap {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.clusterInfo
}

// thread safe set cluster info
func (c *ClusterInfoCache) setClusterInfo(m *ClusterInfoMap) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.clusterInfo = m
}

// returns a copy of the cluster info map
func (c *ClusterInfoCache) GetClusterInfo() *ClusterInfoMap {
	result := make(ClusterInfoMap)
	for k, v := range *c.getClusterInfo() {
		result[k] = &ClusterInfo{
			Endpoint:             v.Endpoint,
			Name:                 v.Name,
			ClusterCaCertificate: v.ClusterCaCertificate,
			RootCAs:              v.RootCAs,
		}
	}
	return &result
}

func (c *ClusterInfoCache) run() {
	for {
		select {
		case <-c.ctx.Done():
			log.Printf("INFO: cluster info cache shutting down")
			return
		case <-time.After(c.refresh):
			break
		}
		if clusterInfo, err := c.retrieveClusters(); err == nil {
			c.setClusterInfo(clusterInfo)
		} else {
			log.Printf("ERROR: failed to refresh cluster information, %s", err)
		}
	}
}

// creates a ca cert pool from the clusterCaCertificate
func createCertPool(name string, clusterCaCertificate string) *x509.CertPool {
	result := x509.NewCertPool()
	cert, err := base64.StdEncoding.DecodeString(clusterCaCertificate)
	if err == nil {
		if ok := result.AppendCertsFromPEM(cert); !ok {
			log.Printf("ERROR: failed to add CA certificates of cluster %s to pool", name)
		}
	} else {
		log.Printf("ERROR: failed to decode CA certificate of cluster %s, %s", name, err)
	}
	return result
}

func (c *ClusterInfoCache) retrieveClusters() (*ClusterInfoMap, error) {
	result := make(ClusterInfoMap)

	service, err := container.NewService(c.ctx,
		option.WithTokenSource(c.credentials.TokenSource))
	if err != nil {
		return nil, err
	}
	parent := fmt.Sprintf("projects/%s/locations/-", c.projectId)
	response, err := service.Projects.Locations.Clusters.List(parent).Do()
	if err != nil {
		return nil, err
	}
	for _, cluster := range response.Clusters {
		if cluster.Status != "RUNNING" {
			log.Printf("INFO: skipping cluster %s in status %s", cluster.Name, cluster.Status)
			continue
		}
		result[cluster.Endpoint] = &ClusterInfo{
			Name:                 cluster.Name,
			Endpoint:             cluster.Endpoint,
			ClusterCaCertificate: cluster.MasterAuth.ClusterCaCertificate,
			RootCAs:              createCertPool(cluster.Name, cluster.MasterAuth.ClusterCaCertificate),
		}
	}
	log.Printf("INFO: refreshed cluster information. Found %d running clusters", len(result))
	return &result, nil
}
