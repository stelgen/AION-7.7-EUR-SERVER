#include <QApplication>
#include <QtPlugin>

#include "authmainwindow.h"

#define VERSION "0.6.2.0"
#define BUILD "0709"

int main (int argc, char ** argv)
{
	QTextCodec * codec = QTextCodec::codecForName ("windows-1251");
	QTextCodec::setCodecForCStrings (codec);
	QApplication app (argc, argv);
	CatX::String::setDebugMode (true);
	CatX::String::setErrorMode (true);
	AuthMainWindow * a = new AuthMainWindow (VERSION, BUILD);
	a->show ();
	return app.exec ();
}
