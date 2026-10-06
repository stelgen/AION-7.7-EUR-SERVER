HEADERS       =	gameserver.h \
		data.h \
		serverlist.h \
		authmainwindow.h \
		server.h \
		serverclasses/mysql.h \
		serverclasses/bfish.h \
		serverclasses/rsa.h \
		serverclasses/security.h \
		serverclasses/iddatamap.h \
		serverclasses/iddata.h \
		packets.h   \
		b32.h \
		catx/parser.h  \
		catx/catxfunc.h \
		catx/catxbase.h \
		catx/catxstring.h 
SOURCES       = main.cpp \
		data.cpp \
		servervector.cpp \
		packets/init.cpp \
		packets/authok.cpp \
		packets/authfail.cpp \
		packets/func.cpp \
		packets/serverlist.cpp \
		packets/playok.cpp \
		packets/eula.cpp \
		packets/playfail.cpp \
		gamepackets/package.cpp \
		gamepackets/ping.cpp \
		gamepackets/playerlogin.cpp \
		gamepackets/authkick.cpp \
		server.cpp \
		authmainwindow.cpp \
		gameserver.cpp \
		serverclasses/mysql.cpp \
		serverclasses/server_signal_slot.cpp \
		serverclasses/server_client_thread.cpp \
		serverclasses/server_interaction.cpp \
		serverclasses/bfish.cpp \
		serverclasses/rsa.cpp \
		serverclasses/security.cpp \
		serverclasses/iddatamap.cpp \
		gameserverclasses/gameserver_signal_slot.cpp \
		gameserverclasses/gameserver_client_thread.cpp \
		gameserverclasses/gameserver_interaction.cpp \
		gameserverclasses/iddatags.cpp \
		catx/catxfunc.cpp \
		catx/parser.cpp   \
		catx/catxbase.cpp \
		catx/catxstring.cpp 
QT           += network sql core
CONFIG += static release
LIBS += libeay32
