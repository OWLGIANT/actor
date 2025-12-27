package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/example/microshop/pkg/config"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type ServiceDiscovery struct {
	client *clientv3.Client
	config *config.EtcdConfig
}

type ServiceInstance struct {
	Name string
	Host string
	Port int
}

func NewServiceDiscovery(cfg *config.EtcdConfig) (*ServiceDiscovery, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   cfg.Endpoints,
		DialTimeout: cfg.DialTimeout * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	return &ServiceDiscovery{
		client: cli,
		config: cfg,
	}, nil
}

func (sd *ServiceDiscovery) Register(ctx context.Context, instance *ServiceInstance) error {
	key := fmt.Sprintf("%s%s/%s:%d", sd.config.Prefix, instance.Name, instance.Host, instance.Port)
	value := fmt.Sprintf("%s:%d", instance.Host, instance.Port)

	// Create lease with 30 second TTL
	lease, err := sd.client.Grant(ctx, 30)
	if err != nil {
		return fmt.Errorf("failed to create lease: %w", err)
	}

	// Register service with keep-alive
	_, err = sd.client.Put(ctx, key, value, clientv3.WithLease(lease.ID))
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// Keep alive
	ch, kaerr := sd.client.KeepAlive(ctx, lease.ID)
	if kaerr != nil {
		return fmt.Errorf("failed to keep alive: %w", kaerr)
	}

	go func() {
		for ka := range ch {
			_ = ka
		}
	}()

	return nil
}

func (sd *ServiceDiscovery) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	key := fmt.Sprintf("%s%s/", sd.config.Prefix, serviceName)

	resp, err := sd.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, fmt.Errorf("failed to discover service: %w", err)
	}

	var instances []*ServiceInstance
	for _, kv := range resp.Kvs {
		addr := string(kv.Value)
		// Parse addr (simplified)
		instances = append(instances, &ServiceInstance{
			Name: serviceName,
			Host: addr,
		})
	}

	return instances, nil
}

func (sd *ServiceDiscovery) Deregister(ctx context.Context, instance *ServiceInstance) error {
	key := fmt.Sprintf("%s%s/%s:%d", sd.config.Prefix, instance.Name, instance.Host, instance.Port)
	_, err := sd.client.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}
	return nil
}

func (sd *ServiceDiscovery) Close() error {
	return sd.client.Close()
}
