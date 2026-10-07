// Package ship — высылка телеметрии authd во внешний стак (стандарт TELEMETRY-SPEC).
//
// Копия internal/ship из aion-logd «как есть» (метод трека B), кроме:
// App-имя в syslog-шапке = aion-authd, procid = 2110 (SPEC §3: App = имя сервиса).
package ship
