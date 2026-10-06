#include "security.h"

Security::Security (const int maxThreads, const String & filename)
{
	maxThreads_ = maxThreads;
	fileName_ = filename;
}

void Security::loadBlackList ()
{
	if (fileName_.length ())
	{
		FILE *f = fopen (fileName_.c_str (), "r");
		if (f == NULL)
		{
			return;
		}
		char ip [20];
		while (fscanf (f, "%18s", ip) == 1)
		{
			blackIpData_ [ip] = 1;
		}
		fclose (f);
	}
}

bool Security::isBlackIp (const String & ip)
{
	String check;
	for (unsigned int i = 0; i < ip.length (); ++i)
	{
		check += ip[i];
		if (ip[i] == '.')
		{
			if (blackIpData_[check])
			{
				return true;
			}
		}
	}
	if (blackIpData_[check])
	{
		return true;
	}
	return false;
}

bool Security::isDDoS (const int runningThreads)
{	
	if (runningThreads >= (maxThreads_ << 1))
	{
		return true;
	}
	return false;
}

int Security::maxThreads ()
{
	return maxThreads_;
}

void Security::initTime (const String & ip)
{
	mutex_.lock ();
	banTime_[ip] = time (0);
	mutex_.unlock ();
}

bool Security::checkTime (const String & ip)
{
	if (time (0) - banTime_[ip] <= 1)
	{
		return false;
	}
	return true;
}

void Security::ban (const String & ip, const int reason, int currentThreads)
{
	if (banTime_[ip] < time (0))
	{
		mutex_.lock ();
		banTime_[ip] = time (0);
		mutex_.unlock ();
	}
	switch (reason)
	{
		case E_SECURITY_BRUT:
			if (this->isDDoS (currentThreads))
			{
				mutex_.lock ();
				banTime_[ip] += E_BAN_DDOS;
				mutex_.unlock ();
			}
			else
			{
				mutex_.lock ();
				banTime_[ip] += E_BAN_BRUT;
				mutex_.unlock ();
			}
			break;
		case E_SECURITY_INVALID_PACKET:
			mutex_.lock ();
			banTime_[ip] += E_BAN_INVALID_PACKET;
			mutex_.unlock ();
			break;
		case E_SECURITY_INVALID_PASSWORD:
			if (illegalPassword_[ip] >= E_LOGIN_ATTEMPS)
			{
				mutex_.lock ();
				banTime_[ip] += (E_BAN_BRUT * (illegalPassword_[ip] - 2));
				mutex_.unlock ();
			}
			break;
	}
	if (banTime_[ip] - time (0) > E_BAN_MAX)
	{
		mutex_.lock ();
		banTime_[ip] = E_BAN_MAX;
		mutex_.unlock ();
	}
}

void Security::clearIllegal (const String & ip)
{
	mutex_.lock ();
	illegalPassword_[ip] = 0;
	mutex_.unlock ();
}

void Security::incIllegal (const String & ip)
{
	mutex_.lock ();
	++illegalPassword_[ip];
	mutex_.unlock ();
}
