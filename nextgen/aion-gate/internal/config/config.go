// Package config — загрузка config.yaml (зеркало etc/config.txt AuthGateD, ver 41007, §4 дока).
package config

import (
	"os"

	"aion-gate/internal/ship"

	"gopkg.in/yaml.v3"
)

// Gate — зеркало ключей оригинального config.txt (+ welcome-вары и ship-заготовка).
type Gate struct {
	ServerPort int    `yaml:"serverPort"`
	AuthAddr   string `yaml:"authAddr"`
	AuthPort   int    `yaml:"authPort"`

	NumThread     int `yaml:"numThread"`
	NumIOThread   int `yaml:"numIOThread"`
	AcceptCallNum int `yaml:"acceptCallNum"`
	SocketLimit   int `yaml:"socketLimit"`

	SessionTimeoutMin  int    `yaml:"sessionTimeout"` // минуты (5)
	UseForbiddenIPList bool   `yaml:"useForbiddenIPList"`
	BlockIPsFile       string `yaml:"blockIPsFile"`
	LogDirectory       string `yaml:"logDirectory"`
	CheckGameGuard     bool   `yaml:"checkGameGuard"`
	UseGameGuard       bool   `yaml:"useGameGuard"`
	UseNotifyCSResult  bool   `yaml:"useNotifyCSResult"`
	LoginType          int    `yaml:"loginType"`
	CompanyCode        string `yaml:"companyCode"`
	LogdPort           int    `yaml:"logdport"`
	UseLogd            bool   `yaml:"useLogd"`

	TryIntervalSec       int  `yaml:"tryInterval"` // 60
	TryCount             int  `yaml:"tryCount"`    // 20
	TryBlockIntervalSec  int  `yaml:"tryBlockInterval"`
	DumpPacket           bool `yaml:"dumpPacket"`
	GgNumActive          int  `yaml:"ggNumActive"`
	UseGCSideExtendAccount bool `yaml:"useGCSideExtendAccount"`
	AppLaunchBanDelay    int  `yaml:"appLaunchBanDelay"`
	UseReportMail        bool `yaml:"useReportMail"`
	AuthReconnectInterval int `yaml:"authReconnectInterval"` // 0 = не реконнектит (как оригинал)

	// Welcome-вары (§5.1: происхождение plaintext[0]=0x23 и b0/b1/b2 — capture-зона
	// @0x43c1b0 содержала UTF-16 «er»; дефолты ниже = гипотеза, верифицировать capture'ом).
	WelcomePlainByte int    `yaml:"welcomePlainByte"`
	WelcomeB0        int    `yaml:"welcomeB0"`
	WelcomeB1        int    `yaml:"welcomeB1"`
	WelcomeB2        int    `yaml:"welcomeB2"`
	// WelcomeFixture — hex ПОЛНОГО фрейма (len+c200...) из дампа ОРИГИНАЛА:
	// если задан — шлём его байт-в-байт вместо сборки (тупой реплей, A/B-тест).
	// Приоритет выше welcomeProbe.
	WelcomeFixture   string `yaml:"welcomeFixture"`
	// WelcomeProbe — round-robin пробных вариантов welcome (оракул: frame-32 = принят).
	WelcomeProbe     bool   `yaml:"welcomeProbe"`
	VariantHoldSec   int    `yaml:"variantHoldSec"` // удержание варианта (мин. 180 = пауза на логин)

	// Ship — телеметрия TELEMETRY-SPEC (syslog RFC5424/http/file; не критичный путь).
	Ship ship.Cfg `yaml:"ship"`
}

// FillDefaults — значения из оригинального config.txt (§4 дока).
func (g *Gate) FillDefaults() {
	set := func(dst *int, v int) { if *dst == 0 { *dst = v } }
	setb := func(dst *string, v string) { if *dst == "" { *dst = v } }
	set(&g.ServerPort, 2106)
	setb(&g.AuthAddr, "127.0.0.1")
	set(&g.AuthPort, 2110)
	set(&g.NumThread, 32)
	set(&g.NumIOThread, 96)
	set(&g.AcceptCallNum, 200)
	set(&g.SocketLimit, 2000)
	set(&g.SessionTimeoutMin, 5)
	setb(&g.BlockIPsFile, "etc/BlockIPs.txt")
	setb(&g.LogDirectory, "log")
	set(&g.LoginType, 2)
	setb(&g.CompanyCode, "2")
	set(&g.LogdPort, 3999)
	set(&g.TryIntervalSec, 60)
	set(&g.TryCount, 20)
	set(&g.TryBlockIntervalSec, 120)
	set(&g.GgNumActive, 50)
	set(&g.AppLaunchBanDelay, 5)
	set(&g.WelcomePlainByte, 0x23)
	set(&g.WelcomeB0, 'e')
	set(&g.WelcomeB1, 'r')
	set(&g.WelcomeB2, 0)
	set(&g.VariantHoldSec, 180) // минимум 3 минуты на вариант (пауза на логин)
}

// Config — верхний уровень config.yaml.
type Config struct {
	Gate Gate `yaml:",inline"`
}

// Load — читает yaml; отсутствие файла не фатально (работаем на дефолтах config.txt).
func Load(path string) (*Config, error) {
	c := &Config{}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			c.Gate.FillDefaults()
			return c, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(raw, c); err != nil {
		return nil, err
	}
	c.Gate.FillDefaults()
	return c, nil
}
