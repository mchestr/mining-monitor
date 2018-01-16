package mining_monitor

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	getStatsMethod102 = "miner_getstat2"
	getStatsMethod98  = "miner_getstat1"
)

type ClaymoreClient struct {
	addr         string
	password     string
	version      float64
	readOnly     bool
	failOnWrites bool

	ps PowerService
}

func NewClaymoreClient(addr, password string, version float64) Client {
	return &ClaymoreClient{addr: addr, password: password, version: version}
}

func NewClaymoreClientWithPowerService(addr, password string, version float64, ps PowerService) Client {
	return &ClaymoreClient{addr: addr, password: password, version: version, ps: ps}
}

func (c *ClaymoreClient) SetReadOnly(readOnly, failOnWrites bool) {
	c.readOnly = readOnly
	c.failOnWrites = failOnWrites
}

type claymoreRequest struct {
	ID       int    `json:"id"`
	JsonRpc  string `json:"jsonrpc"`
	Method   string `json:"method"`
	Password string `json:"psw,omitempty"`
}

type claymoreResponse struct {
	ID     int      `json:"id"`
	Result []string `json:"result"`
	Error  string   `json:"error"`
}

func (c *ClaymoreClient) send(method string, expectReply bool) (*claymoreResponse, error) {
	req := &claymoreRequest{
		ID:       0,
		JsonRpc:  "2.0",
		Method:   method,
		Password: c.password,
	}
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal claymore request: %s", err)
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", c.addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tcp addr %s: %s", c.addr, err)
	}
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to remote addr %s: %s", c.addr, err)
	}
	if _, err := conn.Write(b); err != nil {
		return nil, fmt.Errorf("failed to write to remote addr %s: %s", c.addr, err)
	}

	if expectReply {
		reply := make([]byte, 1024)
		n, err := conn.Read(reply)
		if err != nil {
			return nil, fmt.Errorf("failed to read remote addr %s reply: %s", c.addr, err)
		}
		var response claymoreResponse
		if err := json.Unmarshal(reply[:n], &response); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response from remote addr %s got %s: %s", c.addr, string(b[:n]), err)
		}
		return &response, nil
	}
	return nil, nil
}

func parseFloatFromSeparatedString(s, sep string) ([]float64, error) {
	var res []float64
	for _, fs := range strings.Split(s, sep) {
		f, err := strconv.ParseFloat(fs, 64)
		if err != nil {
			return nil, err
		}
		res = append(res, f)
	}
	return res, nil
}

func parseIntFromSeparatedString(s, sep string) ([]int, error) {
	var res []int
	for _, fs := range strings.Split(s, sep) {
		f, err := strconv.Atoi(fs)
		if err != nil {
			return nil, err
		}
		res = append(res, f)
	}
	return res, nil
}

func (c *ClaymoreClient) Stats() (*Statistics, error) {
	getStatMethod := getStatsMethod98
	if c.version >= 10.2 {
		getStatMethod = getStatsMethod102
	}
	resp, err := c.send(getStatMethod, true)
	if err != nil {
		return nil, err
	}
	stats := &Statistics{
		Version: resp.Result[0],
	}
	miningPools := strings.Split(resp.Result[7], ";")
	stats.MainMiningPool = miningPools[0]
	if len(miningPools) > 1 {
		stats.AltMiningPool = miningPools[1]
	}

	runningTime, err := strconv.Atoi(resp.Result[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse running time from %s: %s", resp.Result[1], err)
	}
	stats.RunningTime = runningTime

	ethInfo, err := parseFloatFromSeparatedString(resp.Result[2], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse eth info from %s: %s", resp.Result[2], err)
	}
	stats.MainHashRate = ethInfo[0]
	stats.MainShares = int(ethInfo[1])
	stats.MainRejectedShares = int(ethInfo[2])

	ethHashRates, err := parseFloatFromSeparatedString(resp.Result[3], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse eth gpu hashrates from %s: %s", resp.Result[3], err)
	}
	stats.MainGpuHashRate = ethHashRates
	altInfo, err := parseFloatFromSeparatedString(resp.Result[4], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse altcoin info from %s: %s", resp.Result[4], err)
	}
	stats.AltHashRate = altInfo[0]
	stats.AltShares = int(altInfo[1])
	stats.AltRejectedShares = int(altInfo[2])

	altHashRates, err := parseFloatFromSeparatedString(resp.Result[5], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse gpu alt hashrates from %s: %s", resp.Result[5], err)
	}
	stats.AltGpuHashRate = altHashRates

	gpuInfo := strings.Split(resp.Result[6], ";")
	for i := 0; i < len(gpuInfo); i += 2 {
		temp, err := strconv.ParseFloat(gpuInfo[i], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gpu temp from %s: %s", gpuInfo[i], err)
		}
		fanPercent, err := strconv.ParseFloat(gpuInfo[i+1], 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse fan percent from %s: %s", gpuInfo[i], err)
		}
		stats.GpuTemperatures = append(stats.GpuTemperatures, temp)
		stats.GpuFanPercents = append(stats.GpuFanPercents, fanPercent)
	}
	miningInfo, err := parseFloatFromSeparatedString(resp.Result[8], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse mining info from %s: %s", miningInfo, err)
	}
	stats.MainInvalidShares = int(miningInfo[0])
	stats.MainPoolSwitches = int(miningInfo[1])
	stats.AltInvalidShares = int(miningInfo[2])
	stats.AltPoolSwitches = int(miningInfo[3])

	gpuEthAccepted, err := parseIntFromSeparatedString(resp.Result[9], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse gpu eth accepted from %s: %s", resp.Result[9], err)
	}
	stats.MainGpuShares = gpuEthAccepted
	gpuEthRejected, err := parseIntFromSeparatedString(resp.Result[10], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse gpu eth rejected from %s: %s", resp.Result[10], err)
	}
	stats.MainGpuRejectedShares = gpuEthRejected
	gpuEthInvalid, err := parseIntFromSeparatedString(resp.Result[11], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse gpu gpu eth invalid from %s: %s", resp.Result[11], err)
	}
	stats.MainGpuInvalidShares = gpuEthInvalid
	gpuAltAccepted, err := parseIntFromSeparatedString(resp.Result[12], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse gpu alt accepted from %s: %s", resp.Result[12], err)
	}
	stats.AltGpuShares = gpuAltAccepted
	gpuAltRejected, err := parseIntFromSeparatedString(resp.Result[13], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse gpu alt rejected from %s: %s", resp.Result[13], err)
	}
	stats.AltGpuRejectedShares = gpuAltRejected
	gpuAltInvalid, err := parseIntFromSeparatedString(resp.Result[14], ";")
	if err != nil {
		return nil, fmt.Errorf("failed to parse gpu alt invalid from %s: %s", resp.Result[14], err)
	}
	stats.AltGpuInvalidShares = gpuAltInvalid
	return stats, nil
}

func (c *ClaymoreClient) Reboot() error {
	if c.readOnly {
		if c.failOnWrites {
			return fmt.Errorf("client is read only")
		}
		return nil
	}
	if c.password == "" {
		return fmt.Errorf("remote console does not have a password set and is insecure, " +
			"please set a password to use this functionality")
	}
	_, err := c.send("miner_reboot", false)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClaymoreClient) Restart() error {
	if c.readOnly {
		if c.failOnWrites {
			return fmt.Errorf("client is read only")
		}
		return nil
	}
	if c.password == "" {
		return fmt.Errorf("remote console does not have a password set and is insecure, " +
			"please set a password to use this functionality")
	}
	_, err := c.send("miner_restart", false)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClaymoreClient) PowerCycleEnabled() bool {
	return c.ps != nil
}

func (c *ClaymoreClient) PowerCycle() error {
	return nil
	if c.readOnly {
		if c.failOnWrites {
			return fmt.Errorf("client is read only")
		}
		return nil
	}
	if c.ps == nil {
		return fmt.Errorf("power cycle not enabled on this client, no power service available")
	}
	state, err := c.ps.State()
	if err != nil {
		return err
	}

	if state.On {
		if err := c.ps.Off(); err != nil {
			return fmt.Errorf("failed to turn power off: %s", err)
		}
		time.Sleep(10 * time.Second)
	}

	if err := c.ps.On(); err != nil {
		return fmt.Errorf("failed to turn power on: %s", err)
	}
	return nil
}

func (c *ClaymoreClient) ReadOnly() bool {
	return c.readOnly
}

func (c *ClaymoreClient) IP() string {
	return c.addr
}
