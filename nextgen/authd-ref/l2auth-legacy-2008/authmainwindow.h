#ifndef __AUTH_MAIN_WINDOW_H__
#define __AUTH_MAIN_WINDOW_H__

#include <QMainWindow>
#include <QTimer>
#include <QLabel>
#include <QVBoxLayout>
#include <QString>
#include <time.h>
#include "server.h"
#include "gameserver.h"

class AuthMainWindow : public QMainWindow
{
	Q_OBJECT
	private:
		time_t timer_;
		QTimer * timeTimer_;
		QLabel * timeLabel_;
		MySQL * pSql_;
	public:
		AuthMainWindow (const QString &, const QString &);
		~AuthMainWindow ();
	private slots:
		void slotTimeOut_ ();		
		void slotInitServers ();
	signals:
		void signalInitServerList ();
};

#endif
