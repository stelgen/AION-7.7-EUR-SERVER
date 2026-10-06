using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using System.Xml;
using System.IO;
using System.Windows.Forms;
using System.Xml.Linq;

namespace AuthHost
{
    class Config
    {
        public delegate void LogDelegate(string message);
        public event LogDelegate LogNeeded;
        public delegate void MessageDelegate(string message);
        public event MessageDelegate MessageNeeded;
        public delegate void SendDataDelegate(byte[] data);
        public event SendDataDelegate SendDataNeeded;
        public string AIDCfgName;
        public string CAPKCfgName;
        public string DRLCfgName;
        public string ExcpFileCfgName;
        public string TermParCfgName;
        public string RevokeyCfgName;
        public List<string> BrandList = new List<string>();
        public List<string> AIDList = new List<string>();
        public List<string> CAPKList = new List<string>();
        public List<string> DRLList = new List<string>();
        public List<string> ExcpFileList = new List<string>();
        public List<string> TermParList = new List<string>();
        public List<string> RevokeyList = new List<string>();
        public string CfgDir;
        public string curBrandName;
        public XmlDocument docAID;
        public XmlDocument docCAPK;
        public XmlDocument docDRL;
        public XmlDocument docExcpfile;
        public XmlDocument docTermparam;
        public XmlDocument docRevokey;
        public int AIDNum;
        public int CAPKNum;
        public int DRLNum;
        public int ExcpFileNum;
        public int TermParNum;
        public int RevokeyNum;
        private int curAIDIndex;
        private int curCAPKIndex;
        private int curDRLIndex;
        private int curExcpFileIndex;
        private int curTermParIndex;
        private int curRevokeyIndex;

        public enum CfgType
        {
            CfgAID,
            CfgCAPK,
            CfgDRL,
            CfgExcpFile,
            CfgTermParm,
            CfgRevokey
        };

        public void initDir(string CfgDir)
        {
            this.CfgDir = CfgDir + "\\Config";
        }

        public void initBrand()
        {
            MainWnd.Logger.Info("Config Dir: " + this.CfgDir);
            if (Directory.Exists(this.CfgDir))
            {
                string[] files = Directory.GetDirectories(this.CfgDir);
                foreach (string file in files)
                {
                    MainWnd.Logger.Info("Brand Name: " + Path.GetFileName(file));
                    this.BrandList.Add(Path.GetFileName(file));
                }
            }
        }

        public void SetBrandLocaliton(string curBrand)
        {
            MainWnd.Logger.Info("info: Input Brand Name: " + curBrand);
            this.curBrandName = curBrand;
        }

        public void initConfig()
        {
            int index;
            if(Directory.Exists(this.CfgDir))
            {
                if (this.curBrandName == null)
                {
                    AlertHelper.ShowAlert("Warning", "Cur Brand is null");
                    MainWnd.Logger.Info("Error: Get Cur Brand From ComboBox Error");
                    return;
                }

                MainWnd.Logger.Info("Cur Brand Name:" + this.curBrandName);
                string curBrandDir = this.CfgDir + "\\" + this.curBrandName;

                this.AIDCfgName = curBrandDir + "\\AID";
                this.CAPKCfgName = curBrandDir + "\\CAPK";
                this.DRLCfgName = curBrandDir + "\\DRL";
                this.ExcpFileCfgName = curBrandDir + "\\Exception_File";
                this.TermParCfgName = curBrandDir + "\\SimData";
                this.RevokeyCfgName = curBrandDir + "\\Revocation_CAPK";

                MainWnd.Logger.Info("AID Dir Name: " + this.AIDCfgName);
                MainWnd.Logger.Info("CAPK Dir Name: " + this.CAPKCfgName);
                MainWnd.Logger.Info("Exception_File Dir Name: " + this.ExcpFileCfgName);
                MainWnd.Logger.Info("SimData Dir Name: " + this.TermParCfgName);
                MainWnd.Logger.Info("Revocation_CAPK Dir Name: " + this.RevokeyCfgName);
                MainWnd.Logger.Info("DRL Dir Name: " + this.DRLCfgName);

                if (Directory.Exists(this.AIDCfgName))
                {
                    string [] files = Directory.GetFiles(this.AIDCfgName);
                    if(this.AIDList.Count > 0)
                    {
                        this.AIDList.Clear();
                    }
                    foreach (string file in files)
                    {
                        string fileName = Path.GetFileName(file);
                        index = fileName.IndexOf("_");
                        if(index > -1)
                        {
                            fileName = fileName.Substring(index+1);
                        }
                        index = fileName.IndexOf('.');
                        if(index > -1)
                        {
                            fileName = fileName.Substring(0, index);
                        }
                        this.AIDList.Add(fileName);
                    }
                }
                else
                {
                    AlertHelper.ShowAlert("Warning", "Detect no AID File,Plz Check");
                    MainWnd.Logger.Info("Warning: AID DIR path not exist");

                }
                if (Directory.Exists(this.CAPKCfgName))
                {
                    string[] files = Directory.GetFiles(this.CAPKCfgName);
                    if(this.CAPKList.Count > 0)
                    {
                        this.CAPKList.Clear();
                    }
                    foreach (string file in files)
                    {
                        string fileName = Path.GetFileName(file);
                        index = fileName.IndexOf("_");
                        if (index > -1)
                        {
                            fileName = fileName.Substring(index + 1);
                        }
                        index = fileName.IndexOf('.');
                        if (index > -1)
                        {
                            fileName = fileName.Substring(0, index);
                        }
                        this.CAPKList.Add(fileName);
                    }
                }
                else
                {
                    AlertHelper.ShowAlert("Warning", "Detect no CAPK File,Plz Check");
                    MainWnd.Logger.Info("Warning: CAPK DIR path not exist");
                }
                if (Directory.Exists(this.DRLCfgName))
                {
                    string[] files = Directory.GetFiles(this.DRLCfgName);
                    if(this.DRLList.Count > 0)
                    {
                        this.DRLList.Clear();
                    }
                    foreach (string file in files)
                    {
                        string fileName = Path.GetFileName(file);
                        index = fileName.IndexOf("_");
                        if (index > -1)
                        {
                            fileName = fileName.Substring(index + 1);
                        }
                        index = fileName.IndexOf('.');
                        if (index > -1)
                        {
                            fileName = fileName.Substring(0, index);
                        }
                        this.DRLList.Add(fileName);
                    }
                }
                else
                {
                    AlertHelper.ShowAlert("Warning", "Detect no DRL File,Plz Check");
                    MainWnd.Logger.Info("Warning: DRL DIR path not exist");
                }
                if (Directory.Exists(this.ExcpFileCfgName))
                {
                    string[] files = Directory.GetFiles(this.ExcpFileCfgName);
                    if(this.ExcpFileList.Count > 0)
                    {
                        this.ExcpFileList.Clear();
                    }
                    foreach (string file in files)
                    {
                        string fileName = Path.GetFileName(file);
                        index = fileName.IndexOf("_");
                        if (index > -1)
                        {
                            fileName = fileName.Substring(index + 1);
                        }
                        index = fileName.IndexOf('.');
                        if (index > -1)
                        {
                            fileName = fileName.Substring(0, index);
                        }
                        this.ExcpFileList.Add(fileName);
                    }
                }
                else
                {
                    AlertHelper.ShowAlert("Warning", "Detect no Excption File,Plz Check");
                    MainWnd.Logger.Info("Warning: Excption File DIR path not exist");
                }
                if (Directory.Exists(this.TermParCfgName))
                {
                    string[] files = Directory.GetFiles(this.TermParCfgName);
                    if(this.TermParList.Count > 0)
                    {
                        this.TermParList.Clear();
                    }
                    foreach (string file in files)
                    {
                        string fileName = Path.GetFileName(file);
                        index = fileName.IndexOf("_");
                        if (index > -1)
                        {
                            fileName = fileName.Substring(index + 1);
                        }
                        index = fileName.IndexOf('.');
                        if (index > -1)
                        {
                            fileName = fileName.Substring(0, index);
                        }
                        this.TermParList.Add(fileName);
                    }
                }
                else
                {
                    AlertHelper.ShowAlert("Warning", "Detect no Term Param File,Plz Check");
                    MainWnd.Logger.Info("Warning: Term Param DIR path not exist");
                }
                if (Directory.Exists(this.RevokeyCfgName))
                {
                    string[] files = Directory.GetFiles(this.RevokeyCfgName);
                    if(this.RevokeyList.Count > 0)
                    {
                        this.RevokeyList.Clear();
                    }
                    foreach (string file in files)
                    {
                        string fileName = Path.GetFileName(file);
                        index = fileName.IndexOf("_");
                        if (index > -1)
                        {
                            fileName = fileName.Substring(index + 1);
                        }
                        index = fileName.IndexOf('.');
                        if (index > -1)
                        {
                            fileName = fileName.Substring(0, index);
                        }
                        this.RevokeyList.Add(fileName);
                    }
                }
                else
                {
                    AlertHelper.ShowAlert("Warning", "Detect no Revokey File,Plz Check");
                    MainWnd.Logger.Info("Warning: Revokey DIR path not exist");
                }
            }
            else
            {
                AlertHelper.ShowAlert("Warning", "Detect no Config Dir,Plz Check");
                MainWnd.Logger.Info("Warning: Config DIR path is null");
            }
            docAID = new XmlDocument();
            docCAPK = new XmlDocument();
            docDRL = new XmlDocument();
            docExcpfile = new XmlDocument();
            docRevokey = new XmlDocument();
            docTermparam = new XmlDocument();
        }

        public void LoadXml(CfgType type, int index)
        {
            string directoryPath;
            string[] files;
            switch (type)
            {
                case CfgType.CfgAID:
                    directoryPath = AIDCfgName;
                    if(Directory.Exists(directoryPath))
                    {
                        files = Directory.GetFiles(directoryPath);
                        docAID.Load(files[index]);
                        MainWnd.Logger.Info("Load AID File:" + files[index]);
                        XmlNode root = docAID.DocumentElement;
                        AIDNum = root.ChildNodes.Count;
                        MessageNeeded?.Invoke("AID Num:" + AIDNum);
                        curAIDIndex = 0;
                    }
                    break;
                case CfgType.CfgCAPK:
                    directoryPath = CAPKCfgName;
                    if (Directory.Exists(directoryPath))
                    {
                        files = Directory.GetFiles(directoryPath);
                        docCAPK.Load(files[index]);
                        MainWnd.Logger.Info("Load CAPK File:" + files[index]);
                        XmlNode root = docCAPK.DocumentElement;
                        CAPKNum = root.ChildNodes.Count;
                        MessageNeeded?.Invoke("CAPK Num:" + CAPKNum);
                        curCAPKIndex = 0;
                    }
                    break;
                case CfgType.CfgDRL:
                    directoryPath = DRLCfgName;
                    if (Directory.Exists(directoryPath))
                    {
                        files = Directory.GetFiles(directoryPath);
                        docDRL.Load(files[index]);
                        MainWnd.Logger.Info("Load DRL File:" + files[index]);
                        XmlNode root = docDRL.DocumentElement;
                        DRLNum = root.ChildNodes.Count;
                        MessageNeeded?.Invoke("DRL Num:" + DRLNum);
                        curDRLIndex = 0;
                    }
                    break;
                case CfgType.CfgExcpFile:
                    directoryPath = ExcpFileCfgName;
                    if (Directory.Exists(directoryPath))
                    {
                        files = Directory.GetFiles(directoryPath);
                        docExcpfile.Load(files[index]);
                        MainWnd.Logger.Info("Load Exception File:" + files[index]);
                        //LogNeeded?.Invoke("Load Exception File:" + files[index]);
                        XmlNode root = docExcpfile.DocumentElement;
                        ExcpFileNum = root.ChildNodes.Count;
                        MessageNeeded?.Invoke("ExcpFile Num:" + ExcpFileNum);
                        curExcpFileIndex = 0;
                    }
                    break;
                case CfgType.CfgTermParm:
                    directoryPath = TermParCfgName;
                    if (Directory.Exists(directoryPath))
                    {
                        files = Directory.GetFiles(directoryPath);
                        docTermparam.Load(files[index]);
                        MainWnd.Logger.Info("Load Term Param File:" + files[index]);
                        XmlNode root = docTermparam.DocumentElement;
                        TermParNum = root.ChildNodes.Count;
                        MessageNeeded?.Invoke("Term Param Num:" + TermParNum);
                        curTermParIndex = 0;
                    }
                    break;
                case CfgType.CfgRevokey:
                    directoryPath = RevokeyCfgName;
                    if (Directory.Exists(directoryPath))
                    {
                        files = Directory.GetFiles(directoryPath);
                        docRevokey.Load(files[index]);
                        MainWnd.Logger.Info("Load Revokey File:" + files[index]);
                        //LogNeeded?.Invoke("Load Revokey File:" + files[index]);
                        XmlNode root = docRevokey.DocumentElement;
                        RevokeyNum = root.ChildNodes.Count;
                        MessageNeeded?.Invoke("Revokey Num:" + RevokeyNum);
                        curRevokeyIndex = 0;
                    }
                    break;
                default:
                    MainWnd mainWnd = new MainWnd();
                    MainWnd.Logger.Info("Error: Input Cfg Type invalid");
                    MessageNeeded?.Invoke("Error: Input Cfg Type invalid");
                    break;
            }
        }

        public void DownloadXml(CfgType type)
        {
            byte[] bytes = new byte[2044];
            byte[] sendData;
            string s = "";
            int len = 0,i;
            XmlNode root;
            byte high = 0;
            byte low = 0;

            switch (type)
            {
                case CfgType.CfgAID:
                    root = docAID.DocumentElement;
                    XmlNode aidnode = root.FirstChild;
                    MainWnd.Logger.Info("Cur AID Index:" + curAIDIndex);
                    if (curAIDIndex == AIDNum)
                    {
                        MessageNeeded?.Invoke("Finish Download AID");
                        curAIDIndex = 0;
                        sendData = new byte[] { 0x02, 0x83, 0x00, 0x03, 0x03, 0x01, 0x00 };
                        break;
                    }

                    i = curAIDIndex;
                    while(i > 0)
                    {
                        if(aidnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        aidnode = aidnode.NextSibling;
                        i--;
                    }

                    if(aidnode == null)
                    {
                        return;
                    }

                    foreach (XmlNode aidcfgnode in aidnode.ChildNodes)
                    {
                        if(aidcfgnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        XmlAttribute attribute = aidcfgnode.Attributes["label"];
                        if (attribute != null)
                        {
                            string attributeValue = attribute.Value;
                            s += attributeValue;    //Tag
                            string nodeValue = aidcfgnode.InnerText;
                            len = nodeValue.Length / 2;   //Length,if Length greater than 127,length area should has 2 bytes
                            if (len > 127 && len <= 255)
                            {
                                s += "81";
                            }
                            s += len.ToString("X2");
                            s += nodeValue; //Value
                        }
                        else
                        {
                            return;
                        }
                    }
                    len = s.Length;
                    MainWnd.Logger.Info("s.Length: " + len);

                    high = (byte)(((len / 2) >> 8) & 0xFF);
                    low = (byte)((len / 2) & 0xFF);

                    MainWnd.Logger.Info("s: " + s);
                    bytes = Tool.StringToBCD(s);

                    sendData = new byte[len / 2 + 4];
                    Array.Copy(bytes, 0, sendData, 4, len / 2);
                    sendData[0] = 0x02;
                    sendData[1] = 0x83;
                    sendData[2] = high;
                    sendData[3] = low;

                    MainWnd.Logger.Info("Send Data: " + Tool.ByteArrayToBcdString(sendData));
                    curAIDIndex++;
                    break;

                case CfgType.CfgCAPK:
                    root = docCAPK.DocumentElement;
                    XmlNode capknode = root.FirstChild;
                    //LogNeeded?.Invoke("Cur CAPK Index:" + curCAPKIndex);
                    if (curCAPKIndex == CAPKNum)
                    {
                        LogNeeded?.Invoke("Finish Download CAPK");
                        curCAPKIndex = 0;
                        sendData = new byte[] {0x02,0x82,0x00,0x03,0x03,0x01,0x00};
                        break;
                    }

                    i = curCAPKIndex;
                    while( i > 0)
                    {
                        if(capknode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        capknode = capknode.NextSibling;
                        i--;
                    }

                    if(capknode == null)
                    {
                        return;
                    }

                    foreach (XmlNode capkcfgnode in capknode.ChildNodes)
                    {
                        if(capkcfgnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        XmlAttribute attribute = capkcfgnode.Attributes["label"];
                        if (attribute != null)
                        {
                            string attributeValue = attribute.Value;
                            s += attributeValue;    //Tag
                            string nodeValue = capkcfgnode.InnerText;
                            if (attribute.Value == "9F22")
                            {
                                LogNeeded.Invoke("cur CAPKI:" + nodeValue);
                            }
                            len = nodeValue.Length / 2;   //Length,if Length greater than 127,length area should has 2 bytes
                            if (len > 127 && len <= 255)
                            {
                                s += "81";
                            }
                            s += len.ToString("X2");
                            s += nodeValue; //Value
                        }
                    }
      
                    len = s.Length;
                    MainWnd.Logger.Info("s.Length: " + len);

                    high = (byte)(((len/2) >> 8) & 0xFF);
                    low = (byte)((len/2) & 0xFF);

                    MainWnd.Logger.Info("s: " + s);
                    bytes = Tool.StringToBCD(s);

                    sendData = new byte[len / 2 + 4];
                    Array.Copy(bytes, 0, sendData, 4, len / 2);
                    sendData[0] = 0x02;
                    sendData[1] = 0x82;
                    sendData[2] = high;
                    sendData[3] = low;

                    MainWnd.Logger.Info("Send Data: " + Tool.ByteArrayToBcdString(sendData));
                    curCAPKIndex++;
                    break;

                case CfgType.CfgDRL:
                    root = docDRL.DocumentElement;
                    XmlNode drlnode = root.FirstChild;
                    MainWnd.Logger.Info("Cur DRL Index:" + curDRLIndex);
                    if(curDRLIndex == DRLNum)
                    {
                        LogNeeded?.Invoke("Finish Download DRL");
                        curDRLIndex = 0;
                        sendData = new byte[] { 0x02, 0x88, 0x00, 0x03, 0x03, 0x01, 0x00 };
                        break;
                    }

                    i = curDRLIndex;
                    while(i > 0)
                    {
                        if(drlnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        drlnode = drlnode.NextSibling;
                        i--;
                    }

                    if(drlnode == null)
                    {
                        return;
                    }

                    foreach (XmlNode drlcfgnode in drlnode.ChildNodes)
                    {
                        if(drlnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        XmlAttribute attribute = drlcfgnode.Attributes["label"];
                        if (attribute != null)
                        {
                            string attributeValue = attribute.Value;
                            s += attributeValue;    //Tag
                            string nodeValue = drlcfgnode.InnerText;
                            len = nodeValue.Length / 2;   //Length,if Length greater than 127,length area should has 2 bytes
                            if (len > 127 && len <= 255)
                            {
                                s += "81";
                            }
                            s += len.ToString("X2");
                            s += nodeValue; //Value
                        }
                    }

                    len = s.Length;
                    MainWnd.Logger.Info("s.Length: " + len);

                    high = (byte)(((len / 2) >> 8) & 0xFF);
                    low = (byte)((len / 2) & 0xFF);

                    MainWnd.Logger.Info("s: " + s);
                    bytes = Tool.StringToBCD(s);

                    sendData = new byte[len / 2 + 4];
                    Array.Copy(bytes, 0, sendData, 4, len / 2);
                    sendData[0] = 0x02;
                    sendData[1] = 0x88;
                    sendData[2] = high;
                    sendData[3] = low;

                    MainWnd.Logger.Info("Send Data: " + Tool.ByteArrayToBcdString(sendData));
                    curDRLIndex++;
                    break;

                case CfgType.CfgExcpFile:
                    root = docExcpfile.DocumentElement;
                    XmlNode excpnode = root.FirstChild;
                    MainWnd.Logger.Info("Cur ExcpFile Index:" + curExcpFileIndex);
                    if(curExcpFileIndex == ExcpFileNum)
                    {
                        LogNeeded?.Invoke("Finish Download Excption File");
                        curExcpFileIndex = 0;
                        sendData = new byte[] { 0x02, 0x85, 0x00, 0x03, 0x03, 0x01, 0x00 };
                        break;
                    }
                    i = curExcpFileIndex;
                    while(i > 0)
                    {
                        if(excpnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        excpnode = excpnode.NextSibling;
                        i--;
                    }

                    if(excpnode == null)
                    {
                        return;
                    }

                    foreach (XmlNode excpcfgnode in excpnode.ChildNodes)
                    {
                        if(excpcfgnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        XmlAttribute attribute = excpcfgnode.Attributes["label"];
                        if (attribute != null)
                        {
                            string attributeValue = attribute.Value;
                            s += attributeValue;    //Tag
                            string nodeValue = excpcfgnode.InnerText;
                            len = nodeValue.Length / 2;   //Length,if Length greater than 127,length area should has 2 bytes
                            if (len > 127 && len <= 255)
                            {
                                s += "81";
                            }
                            s += len.ToString("X2");
                            s += nodeValue; //Value
                        }
                    }
                    len = s.Length;
                    MainWnd.Logger.Info("s.Length: " + len);

                    high = (byte)(((len / 2) >> 8) & 0xFF);
                    low = (byte)((len / 2) & 0xFF);

                    MainWnd.Logger.Info("s: " + s);
                    bytes = Tool.StringToBCD(s);

                    sendData = new byte[len / 2 + 4];
                    Array.Copy(bytes, 0, sendData, 4, len / 2);
                    sendData[0] = 0x02;
                    sendData[1] = 0x85;
                    sendData[2] = high;
                    sendData[3] = low;

                    MainWnd.Logger.Info("Send Data: " + Tool.ByteArrayToBcdString(sendData));
                    curExcpFileIndex++;
                    break;

                case CfgType.CfgRevokey:
                    root = docRevokey.DocumentElement;
                    XmlNode revokeynode = root.FirstChild;
                    MainWnd.Logger.Info("Cur RevoKey Index:" + curRevokeyIndex);
                    if(curRevokeyIndex == RevokeyNum)
                    {
                        LogNeeded?.Invoke("Finish Download Revokey");
                        curRevokeyIndex = 0;
                        sendData = new byte[] { 0x02, 0x86, 0x00, 0x03, 0x03, 0x01, 0x00 };
                        break;
                    }
                    i = curRevokeyIndex;
                    while(i > 0)
                    {
                        if(revokeynode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        revokeynode = revokeynode.NextSibling;
                        i--;
                    }

                    if(revokeynode == null)
                    {
                        return;
                    }

                    foreach (XmlNode revokeycfgnode in revokeynode.ChildNodes)
                    {
                        if(revokeycfgnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        XmlAttribute attribute = revokeycfgnode.Attributes["label"];
                        if (attribute != null)
                        {
                            string attributeValue = attribute.Value;
                            s += attributeValue;    //Tag
                            string nodeValue = revokeycfgnode.InnerText;
                            len = nodeValue.Length / 2;   //Length,if Length greater than 127,length area should has 2 bytes
                            if (len > 127 && len <= 255)
                            {
                                s += "81";
                            }
                            s += len.ToString("X2");
                            s += nodeValue; //Value
                        }
                    }

                    len = s.Length;
                    MainWnd.Logger.Info("s.Length: " + len);

                    high = (byte)(((len / 2) >> 8) & 0xFF);
                    low = (byte)((len / 2) & 0xFF);

                    MainWnd.Logger.Info("s: " + s);
                    bytes = Tool.StringToBCD(s);

                    sendData = new byte[len / 2 + 4];
                    Array.Copy(bytes, 0, sendData, 4, len / 2);
                    sendData[0] = 0x02;
                    sendData[1] = 0x86;
                    sendData[2] = high;
                    sendData[3] = low;

                    MainWnd.Logger.Info("Send Data: " + Tool.ByteArrayToBcdString(sendData));
                    curRevokeyIndex++;
                    break;

                case CfgType.CfgTermParm:
                    root = docTermparam.DocumentElement;
                    XmlNode termparnode = root.FirstChild;
                    MainWnd.Logger.Info("Cur TermPar Index:" + curTermParIndex);
                    if(curTermParIndex == TermParNum)
                    {
                        LogNeeded?.Invoke("Finish Download Term Param");
                        curTermParIndex = 0;
                        sendData = new byte[] { 0x02, 0x84, 0x00, 0x03, 0x03, 0x01, 0x00 };
                        break;
                    }
                    i = curTermParIndex;
                    while(i > 0)
                    {
                        if(termparnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        termparnode = termparnode.NextSibling;
                        i--;
                    }
                    if(termparnode == null)
                    {
                        return;
                    }

                    foreach (XmlNode termparcfgnode in termparnode.ChildNodes)
                    {
                        if(termparcfgnode.NodeType == XmlNodeType.Comment)
                        {
                            continue;
                        }
                        XmlAttribute attribute = termparcfgnode.Attributes["label"];
                        if (attribute != null)
                        {
                            string attributeValue = attribute.Value;
                            s += attributeValue;
                            string nodeValue = termparcfgnode.InnerText;
                            len = nodeValue.Length / 2;   //Length,if Length greater than 127,length area should has 2 bytes
                            if (len > 127 && len <= 255)
                            {
                                s += "81";
                            }
                            s += len.ToString("X2");
                            s += nodeValue; //Value
                        }
                    }

                    len = s.Length;
                    MainWnd.Logger.Info("s.Length: " + len);

                    high = (byte)(((len / 2) >> 8) & 0xFF);
                    low = (byte)((len / 2) & 0xFF);

                    MainWnd.Logger.Info("s: " + s);
                    bytes = Tool.StringToBCD(s);

                    sendData = new byte[len / 2 + 4];
                    Array.Copy(bytes, 0, sendData, 4, len / 2);
                    sendData[0] = 0x02;
                    sendData[1] = 0x84;
                    sendData[2] = high;
                    sendData[3] = low;

                    MainWnd.Logger.Info("Send Term Param Data: " + Tool.ByteArrayToBcdString(sendData));
                    curTermParIndex++;
                    break;

                default:
                    return ;
            }

            SendDataNeeded?.Invoke(sendData);
        }        
    }

}
