#include "catxstring.h"


CatX::String::String () : data_ (NULL), len_ (0), alloc_ (0)
{
}


CatX::String::String (char c) : data_ (NULL), len_ (0), alloc_ (0)
{
	this->append (c);
}

CatX::String::String (const int val) : data_ (NULL), len_ (0), alloc_ (0)
{
	this->append (val);
}

CatX::String::String (const double val) : data_ (NULL), len_ (0), alloc_ (0)
{
	this->append (val);
}


CatX::String::String (const char * str)
{
	len_ = strlen (str);
	data_ = (char *) malloc ((strlen (str) + 1) * sizeof (char));
	alloc_ = len_ + 1;
	if (data_)
	{
		len_ = strlen (str);
		memcpy (data_, str, len_);
		data_ [len_] = '\0';
	}
	else
	{
		catxError ("Memory allocation fail");
		exit (1);
	}
}

CatX::String::String (const char * str, const unsigned int len) : len_ (len)
{
	data_ = (char *) malloc ((len_ + 1) * sizeof (char));
	alloc_ = len_ + 1;
	if (data_)
	{
		memcpy (data_, str, len_);
		data_ [len_] = '\0';
	}
	else
	{
		catxError ("Memory allocation fail");
		exit (1);
	}	
}

CatX::String::~String ()
{
	if (data_)
	{
		free (data_);
	}
}

unsigned int CatX::String::length () const
{
	return len_;
}

char * CatX::String::c_str () const
{
	return data_;
}

char * CatX::String::c_str (unsigned int pos) const
{
	if (pos < len_)
	{
		return &(data_[pos]);
	}
	else
	{
		catxDebug ("String memory violation", data_);
		return data_;
	}
}

CatX::String & CatX::String::operator += (const char c)
{
	allocate_ (len_ + 1);
	data_ [len_] = c;
	data_ [len_ + 1] = '\0';
	++len_;
	return *this;			
}

CatX::String::String (const String & el)
{
	len_ = el.length ();
	alloc_ = len_ + 1;
	data_ = (char *) malloc (alloc_ * sizeof (char));
	if (data_)
	{
		memcpy (data_, el.c_str (), len_);
		data_ [len_] = '\0';
	}
	else
	{
		catxError ("Memory allocation failed!");
		exit (1);
	}
}

CatX::String::String (const std::string & el)
{
	len_ = el.length ();
	alloc_ = len_ + 1;
	data_ = (char *) malloc (alloc_ * sizeof (char));
	if (data_)
	{
		memcpy (data_, el.c_str (), len_);
		data_ [len_] = '\0';
	}
	else
	{
		catxError ("Memory allocation failed!");
		exit (1);
	}
}


void CatX::String::clear ()
{
	alloc_ = 0;
	len_ = 0;
	free (data_);
	data_ = NULL;
}

CatX::String CatX::String::operator + (const char c) const
{
	CatX::String nya (*this);
	return nya += c;
}


CatX::String CatX::String::operator + (const char * str) const
{
	CatX::String nya (*this);
	return nya += str;
}

CatX::String CatX::String::operator + (const CatX::String & str) const
{
	CatX::String nya (*this);
	return nya += str.c_str ();
}

void CatX::String::operator = (const String & el)
{
	allocate_ (el.length () + 1);
	len_ = el.length ();
	memcpy (data_, el.c_str (), len_);
	data_ [len_] = '\0';
}

void CatX::String::operator = (const std::string & el)
{
	allocate_ (el.length () + 1);
	len_ = el.length ();
	memcpy (data_, el.c_str (), len_);
	data_ [len_] = '\0';
}

void CatX::String::operator = (const char * el)
{	
	len_ = strlen (el);
	allocate_ (len_ + 1);
	memcpy (data_, el, len_);
	data_ [len_] = '\0';
}



char & CatX::String::operator [] (const unsigned int id)
{
	return data_ [id];
}

char CatX::String::operator [] (const unsigned int id) const
{
	return data_ [id];
}

char CatX::String::at (const unsigned int id) const
{
	if (id < len_)
	{
		return data_ [id];
	}
	else
	{
		catxDebug ("String memory violation", data_);
		return data_ [len_ - 1];
	}
}


CatX::String & CatX::String::operator += (const char * str)
{
	this->append (str, strlen (str));
	return *this;
}

CatX::String & CatX::String::operator += (const CatX::String & el)
{
	for (unsigned int i = 0; i < el.length (); ++i)
	{
		*this += el[i];
	}
	return *this;
}

CatX::String & CatX::String::operator += (const std::string & el)
{
	for (unsigned int i = 0; i < el.length (); ++i)
	{
		*this += el[i];
	}
	return *this;
}


void CatX::String::append (const char * str, const unsigned int len)
{
	CatX::String newstr (str);
	allocate_ (len_ + len + 1);
	char * segm = & (data_ [len_]);
	memcpy (segm, newstr.c_str (), len);
	len_ += len;
	data_[len_] = '\0';
}

void CatX::String::append (const char * str)
{
	append (str, strlen (str));
}

void CatX::String::append (const char c)
{
	*this += c;
}

void CatX::String::append (const CatX::String & el)
{
	*this += el;
}

void CatX::String::append (const int el)
{
	char str [11];
	sprintf (str, "%d", el);
	*this += str;
}

void CatX::String::append (const double el)
{
	char val[1024];
	sprintf (val, "%lf", el);
	*this += val;
}

void CatX::String::freeEx ()
{
	data_ = (char *) realloc (data_, len_ + 1);
	alloc_ = len_ + 1;
}

void CatX::String::allocate_ (const unsigned int len)
{
	if (len >= alloc_)
	{
		alloc_ = len * 2;
		data_ = (char *) realloc (data_, alloc_ * sizeof (char));
		if (!data_)
		{		
			catxError ("Memory allocation failed!");
			exit (1);
		}		
	}
}

CatX::String CatX::String::substr (const unsigned int pos, const unsigned int len) const
{
	if (pos < len_)
	{		
		CatX::String ret;
		unsigned int glen = len;
		if (pos + len > len_)
		{
			glen = len_ - pos;
		}
		for (unsigned int i = 0; i < glen; ++i)
		{
			ret += (*this)[pos + i];
		}
		return ret;
	}
	else
	{
		catxDebug ("String memory violation", data_);		
		return substr (0, len);
	}
}

int CatX::String::toInt () const
{
	if (!len_)
	{
		return 0;
	}
	for (unsigned int i = 0; i < len_; ++i)
	{
		if (!i && data_[i] == '-')
		{
			continue;
		}
		else if (data_[i] < '0' || data_[i] > '9')
		{
			return CatX::E_INVALID_NUMBER;
		}
	}
	double check;
	sscanf (data_, "%lf", &check);
	if (check >= INT_MIN && check <= INT_MAX)
	{
		int ret;
		sscanf (data_, "%d", &ret);
		return ret;
	}
	else
	{
		return CatX::E_OVERFLOW;
	}
}

bool CatX::String::operator == (const CatX::String & el) const
{
	if (len_ == el.length ())
	{
		for (unsigned int i = 0; i < len_; ++i)
		{
			if (data_ [i] != el[i])
			{
				return false;
			}
		}
		return true;
	}
	else
	{
		return false;
	}
}

bool CatX::String::operator == (const std::string & el) const
{
	if (len_ == el.length ())
	{
		for (unsigned int i = 0; i < len_; ++i)
		{
			if (data_ [i] != el[i])
			{
				return false;
			}
		}
		return true;
	}
	else
	{
		return false;
	}
}


bool CatX::String::operator == (const char * str) const
{
	if (len_ == strlen (str))
	{
		for (unsigned int i = 0; i < len_; ++i)
		{
			if (data_ [i] != str[i])
			{
				return false;
			}
		}
		return true;
	}
	else
	{
		return false;
	}
}

bool CatX::String::operator != (const char * str) const
{
	return !(*this == str);
}

bool CatX::String::operator != (const String & str) const
{
	return !(*this == str);
}

bool CatX::String::operator != (const std::string & str) const
{
	return !(*this == str);
}


bool CatX::String::operator < (const char * str) const
{
	return std::string (data_) < str;
}

bool CatX::String::operator < (const CatX::String & el) const
{
	return std::string (data_) < el.c_str ();
}


double CatX::String::toDouble () const
{
	if (!len_)
	{
		return 0;
	}
	int points = 0;
	bool error = false;
	for (unsigned int i = 0; i < len_; ++i)
	{
		if (!i && data_[i] == '-')
		{
			continue;
		}
		else if (data_ [i] == '.')
		{
			if (points++)
			{
				error = true;
			}
		}
		else if (data_[i] < '0' || data_[i] > '9')
		{
			error = true;
		}
		if (error)
		{
			return CatX::E_INVALID_NUMBER;
		}
	}
	double ret;
	sscanf (data_, "%lf", &ret);
	return ret;
}


CatX::String CatX::String::windows_str () const
{
	CatX::String ret;
	unsigned int pos = 0;
	for (unsigned int i = 0; i < len_; ++i)
	{
		if (data_ [i] == '\n' && data_ [i - 1] != '\r')
		{
			ret.append (this->substr (pos, i - pos));
			ret.append ("\r\n");
			pos = i + 1;
		}
	}
	if (len_ > pos)
	{
		ret.append (this->substr (pos, len_ - pos));
	}
	return ret;
}

CatX::String CatX::String::toUtf () const
{
	CatX::String ret;
	for (unsigned int i = 0; i < len_; ++i)
	{
		ret.append (data_[i]);
		ret.append (static_cast <char> (0));
	}
	return ret;
}

