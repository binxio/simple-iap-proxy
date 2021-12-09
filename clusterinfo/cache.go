package clusterinfo

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
)

// ConnectInfo provides basie GKE cluster connect information
type ConnectInfo struct {
	Name                 string
	Endpoint             string
	ClusterCaCertificate string
	RootCAs              *x509.CertPool
}

// Map provides a lookup for cluster connection information base on the hostname
type Map map[string]*ConnectInfo

// Cache provides access to a cached cluster information map
type Cache struct {
	ctx         context.Context
	projectID   string
	credentials *google.Credentials
	refresh     time.Duration
	clusterInfo *Map
	mutex       sync.Mutex
}

// NewCache creates a cluster info cache which is refreshed every `refresh`
func NewCache(ctx context.Context, projectID string, credentials *google.Credentials, refresh time.Duration) (*Cache, error) {
	cache := &Cache{
		ctx:         ctx,
		credentials: credentials,
		projectID:   projectID,
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

// GetConnectInfoForEndpoint returns connect information for the host, or nil if not found
func (c *Cache) GetConnectInfoForEndpoint(endpoint string) *ConnectInfo {
	host := strings.Split(endpoint, ":")
	if r, ok := (*c.clusterInfo)[host[0]]; ok {
		return r
	}
	return nil
}

// thread safe get cluster info
func (c *Cache) getClusterInfo() *Map {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.clusterInfo
}

// thread safe set cluster info
func (c *Cache) setClusterInfo(m *Map) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.clusterInfo = m
}

// GetMap returns a copy of the cluster info map
func (c *Cache) GetMap() *Map {
	result := make(Map)
	for k, v := range *c.getClusterInfo() {
		result[k] = &ConnectInfo{
			Endpoint:             v.Endpoint,
			Name:                 v.Name,
			ClusterCaCertificate: v.ClusterCaCertificate,
			RootCAs:              v.RootCAs,
		}
	}
	return &result
}

func (c *Cache) run() {
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

func (c *Cache) retrieveClusters() (*Map, error) {
	result := make(Map)

	service, err := container.NewService(c.ctx,
		option.WithTokenSource(c.credentials.TokenSource))
	if err != nil {
		return nil, err
	}
	parent := fmt.Sprintf("projects/%s/locations/-", c.projectID)
	response, err := service.Projects.Locations.Clusters.List(parent).Do()
	if err != nil {
		return nil, err
	}
	for _, cluster := range response.Clusters {
		if cluster.Status != "RUNNING" {
			log.Printf("INFO: skipping cluster %s in status %s", cluster.Name, cluster.Status)
			continue
		}
		result[cluster.Endpoint] = &ConnectInfo{
			Name:                 cluster.Name,
			Endpoint:             cluster.Endpoint,
			ClusterCaCertificate: cluster.MasterAuth.ClusterCaCertificate,
			RootCAs:              createCertPool(cluster.Name, cluster.MasterAuth.ClusterCaCertificate),
		}
	}
	log.Printf("INFO: refreshed cluster information. Found %d running clusters", len(result))
	return &result, nil
}
