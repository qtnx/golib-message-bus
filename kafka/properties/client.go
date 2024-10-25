package properties

import (
	"strings"

	"github.com/golibs-starter/golib/config"
)

func NewClient(loader config.Loader) (*Client, error) {
	props := Client{}
	err := loader.Bind(&props)
	return &props, err
}

type Client struct {
	Version          string   `default:"2.1.1"`
	BootstrapServers []string `default:"[\"localhost:9092\"]"`
	SecurityProtocol string   // TLS
	ClientId         string
	Debug            bool
	Tls              *Tls
	Admin            Admin
	Producer         Producer
	Consumer         Consumer
}

func (p Client) Prefix() string {
	return "app.kafka"
}

func (p *Client) PostBinding() error {
	// Overwrite admin configuration
	if len(p.Admin.ClientId) == 0 {
		p.Admin.ClientId = p.ClientId
	}

	// if commas in the bootstrap servers, then split them
	if len(p.BootstrapServers) > 0 && strings.Contains(p.BootstrapServers[0], ",") {
		p.BootstrapServers = strings.Split(p.BootstrapServers[0], ",")
	}

	if len(p.Admin.BootstrapServers) == 0 {
		p.Admin.BootstrapServers = p.BootstrapServers
	}
	if len(p.Admin.SecurityProtocol) == 0 {
		p.Admin.SecurityProtocol = p.SecurityProtocol
	}
	if p.Admin.Tls == nil {
		p.Admin.Tls = p.Tls
	}

	// Overwrite producer configuration
	if len(p.Producer.ClientId) == 0 {
		p.Producer.ClientId = p.ClientId
	}
	if len(p.Producer.BootstrapServers) == 0 {
		p.Producer.BootstrapServers = p.BootstrapServers
	}
	if len(p.Producer.SecurityProtocol) == 0 {
		p.Producer.SecurityProtocol = p.SecurityProtocol
	}
	if p.Producer.Tls == nil {
		p.Producer.Tls = p.Tls
	}

	// Overwrite consumer configuration
	if len(p.Consumer.ClientId) == 0 {
		p.Consumer.ClientId = p.ClientId
	}
	if len(p.Consumer.BootstrapServers) == 0 {
		p.Consumer.BootstrapServers = p.BootstrapServers
	}
	if len(p.Consumer.SecurityProtocol) == 0 {
		p.Consumer.SecurityProtocol = p.SecurityProtocol
	}
	if p.Consumer.Tls == nil {
		p.Consumer.Tls = p.Tls
	}
	return nil
}
