using System;
using System.IO.Ports;

namespace AuthHost
{
    public class SerialCom : IDisposable
    {
        private SerialPort _serialPort;

        // 定义一个事件，与 TCPServer 类的 DataReceived 事件保持一致
        public event EventHandler<byte[]> DataReceived;
        public delegate void LogDelegate(string message);
        public event LogDelegate LogNeeded;

        // 构造函数
        public SerialCom()
        {
            _serialPort = new SerialPort();
            _serialPort.DataReceived += SerialPort_DataReceived;
        }

        public void Dispose()
        {
            if (_serialPort != null)
            {
                if (_serialPort.IsOpen)
                {
                    _serialPort.Close();
                }
                _serialPort.Dispose();
            }
        }

        // 打开串口
        public void OpenPort(string portName, int baudRate)
        {
            try
            {
                if (!_serialPort.IsOpen)
                {
                    _serialPort.PortName = portName;
                    _serialPort.BaudRate = baudRate;
                    _serialPort.Parity = Parity.None;
                    _serialPort.DataBits = 8;
                    _serialPort.StopBits = StopBits.One;

                    _serialPort.Open();
                }
            }
            catch (Exception ex)
            {
                LogNeeded?.Invoke("Error opening serial port: " + ex.Message);
            }

            if(_serialPort.IsOpen)
            {
                LogNeeded?.Invoke($"{portName} opened successfully");
            }
        }

        // 关闭串口
        public void ClosePort()
        {
            if (_serialPort.IsOpen)
            {
                _serialPort.Close();
            }
        }

        // 发送数据
        public void SendData(byte[] data)
        {
            if (_serialPort.IsOpen)
            {
                _serialPort.Write(data, 0, data.Length);
            }
        }

        // 数据接收事件处理
        private void SerialPort_DataReceived(object sender, SerialDataReceivedEventArgs e)
        {
            if (e.EventType == SerialData.Chars)
            {
                byte[] buffer = new byte[_serialPort.BytesToRead];
                _serialPort.Read(buffer, 0, buffer.Length);

                // 触发 DataReceived 事件
                DataReceived?.Invoke(this, buffer);
            }
        }
    }
}
