#ifndef __IDDATA__
#define __IDDATA__

class IdData
{
	private:
		String ip_;
		String account_name_;
		int uid_;
		int rsa_;	
		int lastWorld_;
	public:
		IdData () : uid_ (0), rsa_ (0), lastWorld_ (0) {}
		void setIp (String ip) {ip_ = ip;}
		String getIp () const {return ip_;}
		void setAccountName (String an) {account_name_ = an;}
		String getAccountName () const {return account_name_;}
		void setUid (int uid) {uid_ = uid;}
		int getUid () const {return uid_;}
		void setRsa (int rsa) {rsa_ = rsa;}
		int getRsa () const {return rsa_;}
		void setLastWorld (int lw) {lastWorld_ = lw;}
		int getLastWorld () const {return lastWorld_;}
};

#endif
