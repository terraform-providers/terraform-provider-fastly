package fastly

import (
	gofastly "github.com/fastly/go-fastly/fastly"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
	"strings"
)

var backendSchema = &schema.Schema{
	Type:     schema.TypeSet,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			// required fields
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "A name for this Backend",
			},
			"address": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "An IPv4, hostname, or IPv6 address for the Backend",
			},
			// Optional fields, defaults where they exist
			"auto_loadbalance": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Should this Backend be load balanced",
			},
			"between_bytes_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     10000,
				Description: "How long to wait between bytes in milliseconds",
			},
			"connect_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     1000,
				Description: "How long to wait for a timeout in milliseconds",
			},
			"error_threshold": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     0,
				Description: "Number of errors to allow before the Backend is marked as down",
			},
			"first_byte_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     15000,
				Description: "How long to wait for the first bytes in milliseconds",
			},
			"healthcheck": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The healthcheck name that should be used for this Backend",
			},
			"max_conn": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     200,
				Description: "Maximum number of connections for this Backend",
			},
			"port": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     80,
				Description: "The port number Backend responds on. Default 80",
			},
			"override_host": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The hostname to override the Host header",
			},
			"request_condition": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Name of a condition, which if met, will select this backend during a request.",
			},
			"shield": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "The POP of the shield designated to reduce inbound load.",
			},
			"use_ssl": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether or not to use SSL to reach the Backend",
			},
			"max_tls_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Maximum allowed TLS version on SSL connections to this backend.",
			},
			"min_tls_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Minimum allowed TLS version on SSL connections to this backend.",
			},
			"ssl_ciphers": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Comma sepparated list of ciphers",
			},
			"ssl_check_cert": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Be strict on checking SSL certs",
			},
			"ssl_hostname": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SSL certificate hostname",
				Deprecated:  "Use ssl_cert_hostname and ssl_sni_hostname instead.",
			},
			"ssl_ca_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "CA certificate attached to origin.",
			},
			"ssl_cert_hostname": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SSL certificate hostname for cert verification",
			},
			"ssl_sni_hostname": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SSL certificate hostname for SNI verification",
			},
			"ssl_client_cert": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SSL certificate file for client connections to the backend.",
				Sensitive:   true,
			},
			"ssl_client_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "SSL key file for client connections to backend.",
				Sensitive:   true,
			},

			"weight": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     100,
				Description: "The portion of traffic to send to a specific origins. Each origin receives weight/total of the traffic.",
			},
		},
	},
}

func processBackend(d *schema.ResourceData, conn *gofastly.Client, latestVersion int) error {
	ob, nb := d.GetChange("backend")
	if ob == nil {
		ob = new(schema.Set)
	}
	if nb == nil {
		nb = new(schema.Set)
	}

	obs := ob.(*schema.Set)
	nbs := nb.(*schema.Set)
	removeBackends := obs.Difference(nbs).List()
	addBackends := nbs.Difference(obs).List()

	// DELETE old Backends
	for _, bRaw := range removeBackends {
		bf := bRaw.(map[string]interface{})
		opts := gofastly.DeleteBackendInput{
			Service: d.Id(),
			Version: latestVersion,
			Name:    bf["name"].(string),
		}

		log.Printf("[DEBUG] Fastly Backend removal opts: %#v", opts)
		err := conn.DeleteBackend(&opts)
		if errRes, ok := err.(*gofastly.HTTPError); ok {
			if errRes.StatusCode != 404 {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	// Find and post new Backends
	for _, dRaw := range addBackends {
		df := dRaw.(map[string]interface{})
		opts := gofastly.CreateBackendInput{
			Service:             d.Id(),
			Version:             latestVersion,
			Name:                df["name"].(string),
			Address:             df["address"].(string),
			OverrideHost:        df["override_host"].(string),
			AutoLoadbalance:     gofastly.CBool(df["auto_loadbalance"].(bool)),
			SSLCheckCert:        gofastly.CBool(df["ssl_check_cert"].(bool)),
			SSLHostname:         df["ssl_hostname"].(string),
			SSLCACert:           df["ssl_ca_cert"].(string),
			SSLCertHostname:     df["ssl_cert_hostname"].(string),
			SSLSNIHostname:      df["ssl_sni_hostname"].(string),
			UseSSL:              gofastly.CBool(df["use_ssl"].(bool)),
			SSLClientKey:        df["ssl_client_key"].(string),
			SSLClientCert:       df["ssl_client_cert"].(string),
			MaxTLSVersion:       df["max_tls_version"].(string),
			MinTLSVersion:       df["min_tls_version"].(string),
			SSLCiphers:          strings.Split(df["ssl_ciphers"].(string), ","),
			Shield:              df["shield"].(string),
			Port:                uint(df["port"].(int)),
			BetweenBytesTimeout: uint(df["between_bytes_timeout"].(int)),
			ConnectTimeout:      uint(df["connect_timeout"].(int)),
			ErrorThreshold:      uint(df["error_threshold"].(int)),
			FirstByteTimeout:    uint(df["first_byte_timeout"].(int)),
			MaxConn:             uint(df["max_conn"].(int)),
			Weight:              uint(df["weight"].(int)),
			RequestCondition:    df["request_condition"].(string),
			HealthCheck:         df["healthcheck"].(string),
		}

		log.Printf("[DEBUG] Create Backend Opts: %#v", opts)
		_, err := conn.CreateBackend(&opts)
		if err != nil {
			return err
		}
	}
	return nil
}
