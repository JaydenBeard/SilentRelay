package registry

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
)

// ConsulRegistry handles service registration with Consul
type ConsulRegistry struct {
	client     *api.Client
	serviceID  string
	serverID   string
	serverPort int
}

// NewConsulRegistry creates a new Consul registry
func NewConsulRegistry(addr, serverID, serverPort string) (*ConsulRegistry, error) {
	config := api.DefaultConfig()
	config.Address = addr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(serverPort)
	if err != nil {
		log.Printf("Warning: Failed to parse server port, using default 8080: %v", err)
		port = 8080
	}

	return &ConsulRegistry{
		client:     client,
		serviceID:  serverID,
		serverID:   serverID,
		serverPort: port,
	}, nil
}

// Register registers this server with Consul
func (c *ConsulRegistry) Register() error {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Warning: Failed to get hostname, using localhost: %v", err)
		hostname = "localhost"
	}

	registration := &api.AgentServiceRegistration{
		ID:      c.serviceID,
		Name:    "chat-server",
		Port:    c.serverPort,
		Address: hostname,
		Tags:    []string{"chat", "websocket"},
		Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", hostname, c.serverPort),
			Interval:                       "10s",
			Timeout:                        "3s",
			DeregisterCriticalServiceAfter: "30s",
		},
		Meta: map[string]string{
			"server_id": c.serverID,
		},
	}

	if err := c.client.Agent().ServiceRegister(registration); err != nil {
		return err
	}

	log.Printf("✅ Registered with Consul: %s", c.serviceID)
	return nil
}

// Deregister removes this server from Consul
func (c *ConsulRegistry) Deregister() error {
	if err := c.client.Agent().ServiceDeregister(c.serviceID); err != nil {
		return err
	}

	log.Printf("❌ Deregistered from Consul: %s", c.serviceID)
	return nil
}

// GetHealthyServers returns all healthy chat servers
func (c *ConsulRegistry) GetHealthyServers() ([]string, error) {
	services, _, err := c.client.Health().Service("chat-server", "", true, nil)
	if err != nil {
		return nil, err
	}

	servers := make([]string, 0, len(services))
	for _, service := range services {
		servers = append(servers, service.Service.ID)
	}
	return servers, nil
}

// WatchServices watches for changes in available servers
func (c *ConsulRegistry) WatchServices(callback func([]string)) {
	var lastIndex uint64

	for {
		services, meta, err := c.client.Health().Service("chat-server", "", true, &api.QueryOptions{
			WaitIndex: lastIndex,
			WaitTime:  5 * time.Minute,
		})
		if err != nil {
			log.Printf("Error watching Consul services: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if meta.LastIndex != lastIndex {
			lastIndex = meta.LastIndex

			servers := make([]string, 0, len(services))
			for _, service := range services {
				servers = append(servers, service.Service.ID)
			}
			callback(servers)
		}
	}
}
