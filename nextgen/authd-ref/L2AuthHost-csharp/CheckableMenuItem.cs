using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace AuthHost
{
    public interface CheckableMenuItem
    {
        bool IsLogShowScreenEnabled { get; set; }
        bool IsLogWriteFileEnabled {  get; set; }
    }
}
