#ifndef __IDDATAMAP__
#define __IDDATAMAP__

#include "iddata.h"

class IdDataMap : public map <int, IdData>
{
	private:
		map <int, IdData>::const_reverse_iterator getIteratorByUid_ (int uid) const;
	public:
		int getIdByUid (int uid) const;
		String getIpByUid (int uid) const;
		String getAccountByUid (int uid) const;
};


#endif
