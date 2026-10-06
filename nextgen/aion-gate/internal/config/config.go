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
	// WelcomeWaitAuthdMs — >0: ждать [03] от authd до welcome (V≠0 в welcome; тест гипотезы
	// «клиент отвергает V=0 → ошибка авторизации»). 0 = сразу (текущее поведение).
	WelcomeWaitAuthdMs int `yaml:"welcomeWaitAuthdMs"`
	// Фиксированная RSA-пара (эксперимент Pub.key клиента): если заданы — пул использует её.
	RsaFixedN string `yaml:"rsaFixedN"`
	RsaFixedD string `yaml:"rsaFixedD"`
	// IP игрового мира для 42b server-info (ответ на 26b[05]) и classic serverlist.
	WorldIP   string `yaml:"worldIP"`
	WorldPort int    `yaml:"worldPort"` // default 7777 (0x1e61)
	// ServerID — id сервера в эмуляции play-ok (26b[02] → [07][pk1][pk2][serverID][6×0],
	// К-6/P2-6); default 1.
	ServerID int `yaml:"serverID"`
	// SmAuthGgWire — форма SM_AUTH_GG (П3): 42 = live-форма [0b][sid][27×0] (wire 42,
	// ДЕФОЛТ — live-принят, НЕ ТРОГАТЬ); 50 = эталонная форма 7.7 (SM_AUTH_GG.java:
	// D sid + B35 = pt 40 → wire 50) для A/B-сверки с оригом в fork-режиме.
	SmAuthGgWire int `yaml:"smAuthGgWire"`

	// РЕЖИМ (байон-48 06.10, §7 П4):
	//   authgate — живой 7.7 EU флоу (welcome 194/EncryptPrimary/key1, cc-plaintext,
	//              длина-диспетчер) — НЕ ТРОГАЕТСЯ (дефолт);
	//   classic  — beyond-aion-совместимый флоу 4.8: SM_INIT-192/encXORPass/static-ключ,
	//              диспетчер по (op,state), LOGIN_OK/SERVER_LIST/PLAY_OK по раскладке гита;
	//   fork     — форк-прокси: клиент ↔ наш гейт ↔ оригинальный AuthGateD (forkOrig*),
	//              всё релеится raw, логируется расшифровка ориг-ответов и shadow-ответов
	//              НАШЕГО движка — прямое сравнение «оригинал vs наш» (без authd-коннекта!).
	Mode string `yaml:"mode"`
	// rsaExponent — экспонента RSA-ключей гейта (П2): 17 (Pub.key[1]=0x11) или 65537
	// (F4 из гита beyond-aion 4.8). Калибруется живым логином (валидная раскладка = тот e).
	RsaExponent int `yaml:"rsaExponent"`
	// loginDecbufLen — длина decbuf в authd-blob "cbdb": 34 (asm оригинала arg3=0x22,
	// = user14+pwd16+otp4 из раскладки гита; дефолт) | 32 | 128 (полный m).
	LoginDecbufLen int `yaml:"loginDecbufLen"`
	// forkOrig* — куда форк-прокси форвардит клиента (оригинальный AuthGateD).
	ForkOrigAddr string `yaml:"forkOrigAddr"`
	ForkOrigPort int    `yaml:"forkOrigPort"`
	// ggXorTail — classic: SM_AUTH_GG в форме гита (wire 50) вместо живой 32Б (wire 42).
	GgXorTail bool `yaml:"ggXorTail"`



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
	set(&g.WelcomeWaitAuthdMs, 0)
	setb(&g.Mode, "authgate")
	set(&g.RsaExponent, 17)      // гипотеза Pub.key[1]; альтернатива 65537 (F4 гита)
	set(&g.LoginDecbufLen, 34)   // asm arg3=0x22: user14+pwd16+otp4
	setb(&g.ForkOrigAddr, "127.0.0.1")
	set(&g.ForkOrigPort, 2109)   // ориг AuthGateD (AionGateOrig) на VM
	set(&g.WorldPort, 7777)
	set(&g.ServerID, 1)
	set(&g.SmAuthGgWire, 42)     // live-форма (эталонная 50 — только по явному конфигу)
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
