#ifndef __SERVERLIST_H__
#define __SERVERLIST_H__

#include "catx/catxstring.h"
#include <map>
#include <vector>

using namespace CatX;
using namespace std;

class Server_s
{
	private:
		int id_;
		String ip_;
		String innerIp_;
		int port_;
		bool ageLimit_;
		bool pvp_;
		int online_;
		int maxOnline_;
		bool isOnline_;
	public:
		Server_s () : id_ (0), port_ (0), online_ (0), maxOnline_ (0), isOnline_ (false) {}
		int id () const {return id_;}
		void setId (int id) {id_ = id;}
		String ip () const {return ip_;}
		void setIp (const String & ip) {ip_ = ip;}
		String innerIp () const {return innerIp_;}
		void setInnerIp (const String & innerIp) {innerIp_ = innerIp;}
		int port () const {return port_;}
		void setPort (int port) {port_ = port;}
		bool ageLimit () const {return ageLimit_;}
		void setAgeLimit (bool al) {ageLimit_ = al;}
		bool pvp () const {return pvp_;}
		void setPvp (bool pvp) {pvp_ = pvp;}
		int online () const {return online_;}
		void setOnline (int online) {online_ = online;}
		int maxOnline () const {return maxOnline_;}
		void setMaxOnline (int maxOnline) {maxOnline_ = maxOnline;}
		bool isOnline () const {return isOnline_;}
		void setIsOnline (bool val) {isOnline_ = val;}		
};

class ServerVector : public vector <Server_s>
{
	public:
		void init (const vector <Server_s> & data);
		bool isOnline (int id) const;
		String innerIp (int id) const;
		Server_s & getServerByIp (const String & ip);
};

#endif // __SERVERLIST_H__
