package global

import (
	"log"
	"time"
)

// Config
type ServerConfig struct {

	Service struct {
		Name			string	`json:"name" binding:"required"`
		Mode			string	`json:"mode" binding:"required"`
		APIKey			string	`json:"api_key" binding:"required"`
		APISecret		string	`json:"api_secret" binding:"required"`
		Timezone		string	`json:"timezone" binding:"required"`
	} `json:"service" binding:"required"`

	WWW struct {
		HttpHost 		string	`json:"http_host" binding:"required"`
		HttpSSLChain 	string	`json:"http_ssl_chain" binding:"required"`
		HttpSSLPrivKey 	string	`json:"http_ssl_privkey" binding:"required"`
	} `json:"www" binding:"required"`

	Database struct {
		Driver       	string `json:"driver" binding:"required"`
		User         	string `json:"user" binding:"required"`
		Password        string `json:"password" binding:"required"`
		ConnectString   string `json:"connectString" binding:"required"`
		MaxOpenConns 	int    `json:"max_open_conns binding:"required"`
		MaxIdleConns 	int    `json:"max_idle_conns binding:"required"`
	} `json:"database" binding:"required"`

	Redis struct {
		Host			string	`json:"host" binding:"required"`
		Password		string	`json:"password" binding:"required"`
		MaxIdleConns 	int    `json:"max_idle_conns binding:"required"`
		MaxActiveConns 	int    `json:"max_active_conns binding:"required"`
	} `json:"redis" binding:"required"`
}

// Header Struct
type HeaderParameter struct {
	XNonce     			int64 	`header:"X-OBSWRT-NONCE" binding:"required"`
	XAccess    			string	`header:"X-OBSWRT-ACCESS" binding:"required"`
	XSignature 			string	`header:"X-OBSWRT-SIGNATURE" binding:"required"`
}

// Grid Struct
type GridInfo struct {
	X					int64
	Y					int64
	Lng					float64
	Lat					float64
	Land				string
	Temp				string
	Distance			float64
}

// Const Valiable
const ConfigFile string = "config/server.json"

const DBContextTimeout time.Duration = 5 * time.Second

// Global Variable
var Config ServerConfig
var FLog *log.Logger

