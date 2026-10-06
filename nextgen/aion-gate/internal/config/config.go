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

	// Welcome-статики сняты дизasm'ом: plaintext[0:4]=fc, [168:172]=image-константы 65650072 —
	// конфиг-полей для них больше нет (ночь-4 финал; welcomePlainByte/B0/B1/B2 удалены).
	// WelcomeFixture — hex ПОЛНОГО фрейма (len+c200...) из дампа ОРИГИНАЛА:
	// если задан — шлём его байт-в-байт вместо сборки (тупой реплей, A/B-тест).
	// Приоритет выше welcomeProbe.
	WelcomeFixture   string `yaml:"welcomeFixture"`
	// WelcomeProbe — round-robin пробных вариантов welcome (оракул: frame-32 = принят).
	WelcomeProbe     bool   `yaml:"welcomeProbe"`
	VariantHoldSec   int    `yaml:"variantHoldSec"` // удержание варианта (мин. 180 = пауза на логин)
	// WelcomeForceVariant — >=0: ВСЕГДА этот вариант (закрепление найденного: 4 = raw-модуль,
	// единственный, где живой клиент СРЕАГИРОВАЛ «ошибка авторизации» 07:59:50 06.10).
	WelcomeForceVariant int `yaml:"welcomeForceVariant"`
	// WelcomeTestCC — >0: в ответ на AUTH_GG слать клиенту cc-код (тест сообщений:
	// клиент показывает 22 как «аккаунт заблокирован» — живое наблюдение 06.10).
	WelcomeTestCC    int    `yaml:"welcomeTestCC"`

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
	set(&g.VariantHoldSec, 180) // минимум 3 минуты на вариант (пауза на логин)
	set(&g.WelcomeForceVariant, -1) // -1 = по кругу (probe); >=0 = всегда этот вариант
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
