package proxmox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type ProxmoxClient struct {
	baseUrl    *url.URL
	httpClient *http.Client
	Token      string // API Token 认证（格式：PVEAPIToken=userId=userToken）
	// 高权限认证（可选）：如果设置了 Ticket 和 CSRFToken，将优先使用 Cookie + CSRF 方式
	Ticket    string // Proxmox 高权限票据（用于 Cookie: PVEAuthCookie=<ticket>）
	CSRFToken string // CSRF 防护令牌（用于 Header: CSRFPreventionToken: <token>）
}

func NewProxmoxClient(apiURL string, userId, userToken string) (*ProxmoxClient, error) {
	baseUrl, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	return &ProxmoxClient{
		baseUrl: baseUrl,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		Token: fmt.Sprintf("PVEAPIToken=%s=%s", userId, userToken),
	}, nil
}

// NewProxmoxClientWithTicket 使用高权限 ticket 和 CSRF token 创建 ProxmoxClient
// 这种方式使用 Cookie + CSRF 认证，通常具有更高的权限（如 root 账号）
func NewProxmoxClientWithTicket(apiURL string, ticket, csrfToken string) (*ProxmoxClient, error) {
	baseUrl, err := url.Parse(apiURL)
	if err != nil {
		return nil, err
	}
	return &ProxmoxClient{
		baseUrl: baseUrl,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		Ticket:    ticket,
		CSRFToken: csrfToken,
	}, nil
}

func (c *ProxmoxClient) Request(ctx context.Context, req *http.Request, result interface{}) error {
	// 如果提供了 Ticket 和 CSRFToken，使用 Cookie + CSRF 认证方式（高权限）
	if c.Ticket != "" && c.CSRFToken != "" {
		req.Header.Set("CSRFPreventionToken", c.CSRFToken)
		req.AddCookie(&http.Cookie{
			Name:  "PVEAuthCookie",
			Value: c.Ticket,
		})
	} else {
		// 否则使用 API Token 认证方式
	req.Header.Set("Authorization", c.Token)
	}

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		// 尝试解析错误详情
		var errResp struct {
			Data   interface{}            `json:"data"`
			Errors map[string]interface{} `json:"errors,omitempty"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			if len(errResp.Errors) > 0 {
				return fmt.Errorf("proxmox API error (status %d): %v", resp.StatusCode, errResp.Errors)
			}
		}
		// 如果无法解析为标准错误格式，返回原始响应体
		// 尝试提取更多信息
		var rawResp map[string]interface{}
		if json.Unmarshal(body, &rawResp) == nil {
			if msg, ok := rawResp["message"].(string); ok {
				return fmt.Errorf("proxmox API error (status %d): %s", resp.StatusCode, msg)
			}
		}
		return fmt.Errorf("proxmox API error (status %d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		var apiResp struct {
			Data interface{} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return err
		}
		if apiResp.Data != nil {
			data, _ := json.Marshal(apiResp.Data)
			return json.Unmarshal(data, result)
		}
	}
	return nil
}

func (c *ProxmoxClient) Get(ctx context.Context, path string, result interface{}) error {
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	return c.Request(ctx, req, result)
}

// GetVersion 获取 Proxmox VE 版本信息（用于验证连接）
// GET /api2/json/version
// 返回字段：version, release, repoid
func (c *ProxmoxClient) GetVersion(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := c.Get(ctx, "/version", &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ProxmoxClient) Post(ctx context.Context, path string, body, result interface{}) error {
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.Request(ctx, req, result)
}

func (c *ProxmoxClient) Put(ctx context.Context, path string, body, result interface{}) error {
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, reqBody)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.Request(ctx, req, result)
}

func (c *ProxmoxClient) Delete(ctx context.Context, path string) error {
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	return c.Request(ctx, req, nil)
}

// PostForm 发送 application/x-www-form-urlencoded 请求
func (c *ProxmoxClient) PostForm(ctx context.Context, path string, form url.Values, result interface{}) error {
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	formData := form.Encode()
	// 当 form 为空时，Encode() 返回空字符串，这是正确的
	// Content-Length 会自动设置为 0
	body := strings.NewReader(formData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return err
	}
	// 即使 body 为空，也设置 Content-Type
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.Request(ctx, req, result)
}

// PutForm 发送 PUT 方法的 application/x-www-form-urlencoded 请求
func (c *ProxmoxClient) PutForm(ctx context.Context, path string, form url.Values, result interface{}) error {
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	formData := form.Encode()
	body := strings.NewReader(formData)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.Request(ctx, req, result)
}

// CreateQemuVM 创建 QEMU 虚拟机（空机/ISO 安装等）
// POST /api2/json/nodes/{node}/qemu
// 注意：Proxmox 该接口通常使用 form 参数（application/x-www-form-urlencoded）
// 返回：UPID（任务ID）
func (c *ProxmoxClient) CreateQemuVM(ctx context.Context, nodeName string, params url.Values) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu", nodeName)
	var upid string
	if err := c.PostForm(ctx, path, params, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// QemuVNCProxy 获取虚拟机 VNC 代理信息（用于 console/noVNC）
// POST /api2/json/nodes/{node}/qemu/{vmid}/vncproxy
// 返回字段通常包含：port、ticket、user、cert 等
func (c *ProxmoxClient) QemuVNCProxy(ctx context.Context, nodeName string, vmID uint32, websocket, generatePassword bool) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/vncproxy", nodeName, vmID)

	params := url.Values{}
	if websocket {
		params.Set("websocket", "1")
	}
	if generatePassword {
		params.Set("generate-password", "1")
	}

	var result map[string]interface{}
	if err := c.PostForm(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ProxmoxClient) WebSocket(path, params string) (*websocket.Conn, *http.Response, error) {
	endpoint := fmt.Sprintf("wss://%s/api2/json%s?%s", c.baseUrl.Host, path, params)
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 30 * time.Second
	dialer.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	dialer.ReadBufferSize = 8192
	dialer.WriteBufferSize = 8192

	requestHeader := http.Header{}
	// 如果提供了 Ticket 和 CSRFToken，使用 Cookie + CSRF 认证方式
	if c.Ticket != "" && c.CSRFToken != "" {
		requestHeader.Add("CSRFPreventionToken", c.CSRFToken)
		requestHeader.Add("Cookie", fmt.Sprintf("PVEAuthCookie=%s", c.Ticket))
	} else {
		// 否则使用 API Token 认证方式
	requestHeader.Add("Authorization", c.Token)
	}

	conn, resp, err := dialer.Dial(endpoint, requestHeader)
	if err != nil {
		return nil, resp, err
	}
	return conn, resp, nil
}

// CloneVMRequest 克隆虚拟机请求参数
type CloneVMRequest struct {
	NewID       uint32 `json:"newid"`                 // 新虚拟机的ID
	Name        string `json:"name,omitempty"`        // 新虚拟机的名称
	Target      string `json:"target,omitempty"`      // 目标节点（可选，如果为空则在同一节点）
	Full        int    `json:"full,omitempty"`        // 是否完整克隆（1=完整克隆，0=链接克隆）
	Storage     string `json:"storage,omitempty"`     // 目标存储
	Format      string `json:"format,omitempty"`      // 存储格式
	Description string `json:"description,omitempty"` // 描述
	Pool        string `json:"pool,omitempty"`        // 资源池
	Snapname    string `json:"snapname,omitempty"`    // 快照名称（如果要从快照克隆）
}

// CloneVM 克隆虚拟机
// 从源虚拟机克隆创建新虚拟机
// Proxmox API 的 clone 操作需要参数通过 URL query string 传递，而不是 JSON body
func (c *ProxmoxClient) CloneVM(ctx context.Context, nodeName string, sourceVMID uint32, req *CloneVMRequest) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/clone", nodeName, sourceVMID)

	// 构建 URL 查询参数
	params := url.Values{}
	params.Set("newid", fmt.Sprintf("%d", req.NewID))
	if req.Name != "" {
		params.Set("name", req.Name)
	}
	if req.Target != "" {
		params.Set("target", req.Target)
	}
	if req.Full > 0 {
		params.Set("full", "1")
	} else {
		params.Set("full", "0")
	}
	if req.Storage != "" {
		params.Set("storage", req.Storage)
	}
	if req.Format != "" {
		params.Set("format", req.Format)
	}
	if req.Description != "" {
		params.Set("description", req.Description)
	}
	if req.Pool != "" {
		params.Set("pool", req.Pool)
	}
	if req.Snapname != "" {
		params.Set("snapname", req.Snapname)
	}

	// 构建完整的 URL
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	// 发送 POST 请求（没有 body）
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.Request(ctx, httpReq, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// GetVMConfig 获取虚拟机配置（用于检查虚拟机是否存在）
func (c *ProxmoxClient) GetVMConfig(ctx context.Context, nodeName string, vmID uint32) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	var config map[string]interface{}
	if err := c.Get(ctx, path, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// GetVMStatus 获取虚拟机状态
func (c *ProxmoxClient) GetVMStatus(ctx context.Context, nodeName string, vmID uint32) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/current", nodeName, vmID)
	var status map[string]interface{}
	if err := c.Get(ctx, path, &status); err != nil {
		return nil, err
	}
	return status, nil
}

// StartVM 启动虚拟机
func (c *ProxmoxClient) StartVM(ctx context.Context, nodeName string, vmID uint32) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/start", nodeName, vmID)
	var upid string
	if err := c.Post(ctx, path, nil, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// StopVM 停止虚拟机
func (c *ProxmoxClient) StopVM(ctx context.Context, nodeName string, vmID uint32) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", nodeName, vmID)
	params := url.Values{}
	params.Set("timeout", "30") // 等待最多30秒

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	endpoint += "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.Request(ctx, req, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// DeleteVM 删除虚拟机
// 注意：删除前需要确保虚拟机已停止
func (c *ProxmoxClient) DeleteVM(ctx context.Context, nodeName string, vmID uint32, purge bool) error {
	path := fmt.Sprintf("/nodes/%s/qemu/%d", nodeName, vmID)

	// 构建 URL 查询参数
	params := url.Values{}
	if purge {
		params.Set("purge", "1")
	}

	// 构建完整的 URL
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	return c.Request(ctx, req, nil)
}

// WaitForTask 等待任务完成（保留占位，当前未使用）
func (c *ProxmoxClient) WaitForTask(ctx context.Context, nodeName, upid string, timeout time.Duration) error {
	// 这里可以实现任务状态查询逻辑
	// 实际使用时可以调用 /api2/json/nodes/{node}/tasks/{upid}/status 来查询
	return nil
}

// GetClusterTasks 获取集群任务列表
// GET /api2/json/cluster/tasks
func (c *ProxmoxClient) GetClusterTasks(ctx context.Context) ([]map[string]interface{}, error) {
	path := "/cluster/tasks"
	var tasks []map[string]interface{}
	if err := c.Get(ctx, path, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetNodeTasks 获取节点任务列表
// GET /api2/json/nodes/{node}/tasks
func (c *ProxmoxClient) GetNodeTasks(ctx context.Context, nodeName string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/tasks", nodeName)
	var tasks []map[string]interface{}
	if err := c.Get(ctx, path, &tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// GetTaskLog 获取任务日志
// GET /api2/json/nodes/{node}/tasks/{upid}/log
func (c *ProxmoxClient) GetTaskLog(ctx context.Context, nodeName, upid string, start, limit int) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/tasks/%s/log", nodeName, upid)

	// 构建查询参数
	params := url.Values{}
	if start > 0 {
		params.Set("start", fmt.Sprintf("%d", start))
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var logs []map[string]interface{}
	if err := c.Request(ctx, req, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// GetTaskStatus 获取任务状态
// GET /api2/json/nodes/{node}/tasks/{upid}/status
func (c *ProxmoxClient) GetTaskStatus(ctx context.Context, nodeName, upid string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/tasks/%s/status", nodeName, upid)
	var status map[string]interface{}
	if err := c.Get(ctx, path, &status); err != nil {
		return nil, err
	}
	return status, nil
}

// StopTask 终止任务
// DELETE /api2/json/nodes/{node}/tasks/{upid}
func (c *ProxmoxClient) StopTask(ctx context.Context, nodeName, upid string) error {
	path := fmt.Sprintf("/nodes/%s/tasks/%s", nodeName, upid)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseUrl.JoinPath("/api2/json", path).String(), nil)
	if err != nil {
		return err
	}
	return c.Request(ctx, req, nil)
}

// GetVMCurrentConfig 获取虚拟机当前配置
// GET /api2/json/nodes/{node}/qemu/{vmid}/config
func (c *ProxmoxClient) GetVMCurrentConfig(ctx context.Context, nodeName string, vmID uint32) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	var config map[string]interface{}
	if err := c.Get(ctx, path, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// RequestExtJS 处理 extjs API 路径的请求（响应格式可能不同）
func (c *ProxmoxClient) RequestExtJS(ctx context.Context, req *http.Request, result interface{}) error {
	req.Header.Set("Authorization", c.Token)

	resp, err := c.httpClient.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		// 尝试解析错误详情
		var errResp struct {
			Data   interface{}            `json:"data"`
			Errors map[string]interface{} `json:"errors,omitempty"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			if len(errResp.Errors) > 0 {
				return fmt.Errorf("proxmox API error (status %d): %v", resp.StatusCode, errResp.Errors)
			}
		}
		return fmt.Errorf("proxmox API error (status %d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		// extjs API 可能直接返回数据，也可能包装在 data 字段中
		// 先尝试直接解析
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		// 尝试解析为标准格式 {"data": ...}
		var apiResp struct {
			Data interface{} `json:"data"`
		}
		if err := json.Unmarshal(bodyBytes, &apiResp); err == nil && apiResp.Data != nil {
			// 如果成功解析且 data 不为空，使用 data 字段
			data, _ := json.Marshal(apiResp.Data)
			return json.Unmarshal(data, result)
		}

		// 否则直接解析整个响应
		return json.Unmarshal(bodyBytes, result)
	}
	return nil
}

// GetVMPendingConfig 获取虚拟机pending配置
// GET /api2/json/nodes/{node}/qemu/{vmid}/pending
// 返回格式是数组，每个元素包含 key, value, pending, delete 等字段
func (c *ProxmoxClient) GetVMPendingConfig(ctx context.Context, nodeName string, vmID uint32) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/pending", nodeName, vmID)
	var result []map[string]interface{}
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetNodeStatus 获取节点状态
// GET /api2/json/nodes/{node}/status
func (c *ProxmoxClient) GetNodeStatus(ctx context.Context, nodeName string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/status", nodeName)
	var status map[string]interface{}
	if err := c.Get(ctx, path, &status); err != nil {
		return nil, err
	}
	return status, nil
}

// GetNodeServices 获取节点服务列表
// GET /api2/json/nodes/{node}/services
func (c *ProxmoxClient) GetNodeServices(ctx context.Context, nodeName string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/services", nodeName)
	var services []map[string]interface{}
	if err := c.Get(ctx, path, &services); err != nil {
		return nil, err
	}
	return services, nil
}

// StartNodeService 启动节点服务
// POST /api2/json/nodes/{node}/services/{service}/start
// 返回: UPID (任务ID)
func (c *ProxmoxClient) StartNodeService(ctx context.Context, nodeName, serviceName string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/services/%s/start", nodeName, serviceName)
	var upid string
	if err := c.Post(ctx, path, nil, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// StopNodeService 停止节点服务
// POST /api2/json/nodes/{node}/services/{service}/stop
// 返回: UPID (任务ID)
func (c *ProxmoxClient) StopNodeService(ctx context.Context, nodeName, serviceName string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/services/%s/stop", nodeName, serviceName)
	var upid string
	if err := c.Post(ctx, path, nil, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// RestartNodeService 重启节点服务
// POST /api2/json/nodes/{node}/services/{service}/restart
// 返回: UPID (任务ID)
func (c *ProxmoxClient) RestartNodeService(ctx context.Context, nodeName, serviceName string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/services/%s/restart", nodeName, serviceName)
	var upid string
	if err := c.Post(ctx, path, nil, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// GetNodeNetworks 获取节点网络列表
// GET /api2/json/nodes/{node}/network
func (c *ProxmoxClient) GetNodeNetworks(ctx context.Context, nodeName string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/network", nodeName)
	var networks []map[string]interface{}
	if err := c.Get(ctx, path, &networks); err != nil {
		return nil, err
	}
	return networks, nil
}

// CreateNodeNetwork 创建网络设备配置
// POST /api2/json/nodes/{node}/network
// 参数通过 form 格式传递（Proxmox API 要求）
func (c *ProxmoxClient) CreateNodeNetwork(ctx context.Context, nodeName string, params url.Values) error {
	path := fmt.Sprintf("/nodes/%s/network", nodeName)
	return c.PostForm(ctx, path, params, nil)
}

// ReloadNodeNetwork 重新加载网络配置
// PUT /api2/json/nodes/{node}/network
func (c *ProxmoxClient) ReloadNodeNetwork(ctx context.Context, nodeName string) error {
	path := fmt.Sprintf("/nodes/%s/network", nodeName)
	return c.Put(ctx, path, nil, nil)
}

// RevertNodeNetwork 恢复网络配置更改
// DELETE /api2/json/nodes/{node}/network
func (c *ProxmoxClient) RevertNodeNetwork(ctx context.Context, nodeName string) error {
	path := fmt.Sprintf("/nodes/%s/network", nodeName)
	return c.Delete(ctx, path)
}

// SetNodeStatus 设置节点状态（重启/关闭）
// POST /api2/json/nodes/{node}/status
// command: reboot (重启) 或 shutdown (关闭)
func (c *ProxmoxClient) SetNodeStatus(ctx context.Context, nodeName, command string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/status", nodeName)
	params := url.Values{}
	params.Set("command", command)
	var upid string
	if err := c.PostForm(ctx, path, params, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// GetNodeRRDData 获取节点RRD监控数据
// GET /api2/json/nodes/{node}/rrddata
// 参数: timeframe (hour|day|week|month|year), cf (AVERAGE|MAX)
func (c *ProxmoxClient) GetNodeRRDData(ctx context.Context, nodeName string, timeframe, cf string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/rrddata", nodeName)

	// 构建查询参数
	params := url.Values{}
	if timeframe != "" {
		params.Set("timeframe", timeframe)
	}
	if cf != "" {
		params.Set("cf", cf)
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := c.Request(ctx, req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetVMRRDData 获取虚拟机RRD监控数据
// GET /api2/json/nodes/{node}/qemu/{vmid}/rrddata
// 参数: timeframe (hour|day|week|month|year), cf (AVERAGE|MAX)
func (c *ProxmoxClient) GetVMRRDData(ctx context.Context, nodeName string, vmID uint32, timeframe, cf string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/rrddata", nodeName, vmID)

	// 构建查询参数
	params := url.Values{}
	if timeframe != "" {
		params.Set("timeframe", timeframe)
	}
	if cf != "" {
		params.Set("cf", cf)
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := c.Request(ctx, req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateVMConfig 更新虚拟机配置
// PUT /api2/extjs/nodes/{node}/qemu/{vmid}/config
func (c *ProxmoxClient) UpdateVMConfig(ctx context.Context, nodeName string, vmID uint32, config map[string]interface{}) error {
	// extjs 路径需要特殊处理
	path := fmt.Sprintf("/nodes/%s/qemu/%d/config", nodeName, vmID)
	endpoint := c.baseUrl.JoinPath("/api2/extjs", path).String()

	var reqBody io.Reader
	if config != nil {
		data, err := json.Marshal(config)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, reqBody)
	if err != nil {
		return err
	}
	if config != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.RequestExtJS(ctx, req, nil)
}

// GetVMCloudInitConfig 获取虚拟机 CloudInit 配置
// GET /api2/json/nodes/{node}/qemu/{vmid}/cloudinit
// 返回包含当前和待处理值的配置
func (c *ProxmoxClient) GetVMCloudInitConfig(ctx context.Context, nodeName string, vmID uint32) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/cloudinit", nodeName, vmID)
	var config map[string]interface{}
	if err := c.Get(ctx, path, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// UpdateVMCloudInitConfig 更新虚拟机 CloudInit 配置
// PUT /api2/json/nodes/{node}/qemu/{vmid}/cloudinit
// 重新生成和更改 cloudinit 配置驱动器
// 参数通过 form 格式传递（Proxmox API 要求）
func (c *ProxmoxClient) UpdateVMCloudInitConfig(ctx context.Context, nodeName string, vmID uint32, params url.Values) error {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/cloudinit", nodeName, vmID)
	return c.PutForm(ctx, path, params, nil)
}

// MigrateVM 同集群迁移虚拟机
// POST /api2/json/nodes/{node}/qemu/{vmid}/migrate
// 参数: target (目标节点), online (在线迁移), bwlimit (带宽限制), 等
// 返回: UPID (任务ID)
func (c *ProxmoxClient) MigrateVM(ctx context.Context, nodeName string, vmID uint32, params map[string]interface{}) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/migrate", nodeName, vmID)

	// Proxmox API 迁移接口使用 URL 查询参数，而不是 JSON body
	queryParams := url.Values{}
	for key, value := range params {
		if value != nil {
			switch v := value.(type) {
			case string:
				if v != "" {
					queryParams.Set(key, v)
				}
			case bool:
				if v {
					queryParams.Set(key, "1")
				} else {
					queryParams.Set(key, "0")
				}
			case int, int32, int64:
				queryParams.Set(key, fmt.Sprintf("%d", v))
			case uint, uint32, uint64:
				queryParams.Set(key, fmt.Sprintf("%d", v))
			default:
				queryParams.Set(key, fmt.Sprintf("%v", v))
			}
		}
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(queryParams) > 0 {
		endpoint += "?" + queryParams.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.Request(ctx, req, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// RemoteMigrateVM 跨集群迁移虚拟机
// POST /api2/json/nodes/{node}/qemu/{vmid}/remote_migrate
// 参数: target (目标节点), target-cluster (目标集群), online (在线迁移), 等
// 返回: UPID (任务ID)
func (c *ProxmoxClient) RemoteMigrateVM(ctx context.Context, nodeName string, vmID uint32, params map[string]interface{}) (string, error) {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/remote_migrate", nodeName, vmID)

	// Proxmox API 迁移接口使用 URL 查询参数，而不是 JSON body
	queryParams := url.Values{}
	for key, value := range params {
		if value != nil {
			switch v := value.(type) {
			case string:
				if v != "" {
					queryParams.Set(key, v)
				}
			case bool:
				if v {
					queryParams.Set(key, "1")
				} else {
					queryParams.Set(key, "0")
				}
			case int, int32, int64:
				queryParams.Set(key, fmt.Sprintf("%d", v))
			case uint, uint32, uint64:
				queryParams.Set(key, fmt.Sprintf("%d", v))
			default:
				queryParams.Set(key, fmt.Sprintf("%v", v))
			}
		}
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(queryParams) > 0 {
		endpoint += "?" + queryParams.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.Request(ctx, req, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// GetNodeCertificatesInfo 获取节点证书信息
// GET /api2/json/nodes/{node}/certificates/info
func (c *ProxmoxClient) GetNodeCertificatesInfo(ctx context.Context, nodeName string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/certificates/info", nodeName)
	var certificates []map[string]interface{}
	if err := c.Get(ctx, path, &certificates); err != nil {
		return nil, err
	}
	return certificates, nil
}

// GetClusterStatus 获取集群状态
// GET /api2/json/cluster/status
func (c *ProxmoxClient) GetClusterStatus(ctx context.Context) ([]map[string]interface{}, error) {
	var status []map[string]interface{}
	if err := c.Get(ctx, "/cluster/status", &status); err != nil {
		return nil, err
	}
	return status, nil
}

// GetClusterResources 获取集群资源
// GET /api2/json/cluster/resources
func (c *ProxmoxClient) GetClusterResources(ctx context.Context) ([]map[string]interface{}, error) {
	var resources []map[string]interface{}
	if err := c.Get(ctx, "/cluster/resources", &resources); err != nil {
		return nil, err
	}
	return resources, nil
}

// GetNodeDisksList 获取节点磁盘列表
// GET /api2/json/nodes/{node}/disks/list
func (c *ProxmoxClient) GetNodeDisksList(ctx context.Context, nodeName string, includePartitions bool) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/disks/list", nodeName)

	// 构建查询参数
	params := url.Values{}
	if includePartitions {
		params.Set("include-partitions", "1")
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var disks []map[string]interface{}
	if err := c.Request(ctx, req, &disks); err != nil {
		return nil, err
	}
	return disks, nil
}

// GetNodeDisksDirectory 获取节点 Directory 存储
// GET /api2/json/nodes/{node}/disks/directory
// Proxmox API 返回的是对象格式，需要转换为数组
func (c *ProxmoxClient) GetNodeDisksDirectory(ctx context.Context, nodeName string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/disks/directory", nodeName)

	// 使用 interface{} 来接收数据，然后根据实际类型处理
	var rawData interface{}
	if err := c.Get(ctx, path, &rawData); err != nil {
		return nil, err
	}

	// 如果返回的是 nil，返回空数组
	if rawData == nil {
		return []map[string]interface{}{}, nil
	}

	// 尝试解析为对象
	directoriesObj, ok := rawData.(map[string]interface{})
	if !ok {
		// 如果不是对象，可能是数组，直接返回
		if arr, ok := rawData.([]interface{}); ok {
			result := make([]map[string]interface{}, 0, len(arr))
			for _, item := range arr {
				if itemMap, ok := item.(map[string]interface{}); ok {
					result = append(result, itemMap)
				}
			}
			return result, nil
		}
		// 其他类型，返回空数组
		return []map[string]interface{}{}, nil
	}

	// 如果返回的是空对象，直接返回空数组
	if len(directoriesObj) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 将对象转换为数组格式
	directories := make([]map[string]interface{}, 0, len(directoriesObj))
	for name, data := range directoriesObj {
		// 处理不同的数据类型
		var dataMap map[string]interface{}
		switch v := data.(type) {
		case map[string]interface{}:
			dataMap = make(map[string]interface{})
			// 复制所有字段
			for k, val := range v {
				dataMap[k] = val
			}
		case []interface{}:
			// 如果是数组，跳过（不应该出现）
			continue
		default:
			// 如果是其他类型，创建一个新的 map
			dataMap = make(map[string]interface{})
			dataMap["value"] = v
		}
		// 添加名称字段
		dataMap["name"] = name
		directories = append(directories, dataMap)
	}
	return directories, nil
}

// GetNodeDisksLVM 获取节点 LVM 存储
// GET /api2/json/nodes/{node}/disks/lvm
// Proxmox API 返回的是嵌套的树形结构，需要递归解析并扁平化
func (c *ProxmoxClient) GetNodeDisksLVM(ctx context.Context, nodeName string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/disks/lvm", nodeName)

	// 接收原始数据
	var rawData interface{}
	if err := c.Get(ctx, path, &rawData); err != nil {
		return nil, err
	}

	// 如果返回的是 nil，返回空数组
	if rawData == nil {
		return []map[string]interface{}{}, nil
	}

	// 解析树形结构
	lvmsObj, ok := rawData.(map[string]interface{})
	if !ok {
		return []map[string]interface{}{}, nil
	}

	// 递归函数：解析树形结构并提取所有卷组和逻辑卷
	var parseLVMNode func(node map[string]interface{}, parentVG string) []map[string]interface{}
	parseLVMNode = func(node map[string]interface{}, parentVG string) []map[string]interface{} {
		result := make([]map[string]interface{}, 0)

		// 获取节点信息
		name, _ := node["name"].(string)
		leaf, _ := node["leaf"].(float64)

		// 如果是叶子节点（逻辑卷），添加到结果中
		if leaf == 1 {
			item := make(map[string]interface{})
			item["name"] = name
			item["vg_name"] = parentVG
			if size, ok := node["size"].(float64); ok {
				item["size"] = int64(size)
			}
			if free, ok := node["free"].(float64); ok {
				item["free"] = int64(free)
			}
			item["leaf"] = int(leaf)
			result = append(result, item)
		} else {
			// 如果不是叶子节点（卷组），也添加到结果中
			item := make(map[string]interface{})
			item["name"] = name
			item["vg_name"] = name
			if size, ok := node["size"].(float64); ok {
				item["size"] = int64(size)
			}
			if free, ok := node["free"].(float64); ok {
				item["free"] = int64(free)
			}
			if lvcount, ok := node["lvcount"].(float64); ok {
				item["lvcount"] = int(lvcount)
			}
			item["leaf"] = int(leaf)
			result = append(result, item)

			// 更新父卷组名称（用于子节点）
			if parentVG == "" {
				parentVG = name
			}
		}

		// 递归处理子节点
		if children, ok := node["children"].([]interface{}); ok {
			for _, child := range children {
				if childMap, ok := child.(map[string]interface{}); ok {
					childResults := parseLVMNode(childMap, parentVG)
					result = append(result, childResults...)
				}
			}
		}

		return result
	}

	// 从根节点开始解析
	lvms := make([]map[string]interface{}, 0)
	if children, ok := lvmsObj["children"].([]interface{}); ok {
		for _, child := range children {
			if childMap, ok := child.(map[string]interface{}); ok {
				results := parseLVMNode(childMap, "")
				lvms = append(lvms, results...)
			}
		}
	}

	return lvms, nil
}

// GetNodeDisksLVMThin 获取节点 LVM-Thin 存储
// GET /api2/json/nodes/{node}/disks/lvmthin
// Proxmox API 返回的是对象格式，需要转换为数组
func (c *ProxmoxClient) GetNodeDisksLVMThin(ctx context.Context, nodeName string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/disks/lvmthin", nodeName)

	// 使用 interface{} 来接收数据，然后根据实际类型处理
	var rawData interface{}
	if err := c.Get(ctx, path, &rawData); err != nil {
		return nil, err
	}

	// 如果返回的是 nil，返回空数组
	if rawData == nil {
		return []map[string]interface{}{}, nil
	}

	// 尝试解析为对象
	lvmthinsObj, ok := rawData.(map[string]interface{})
	if !ok {
		// 如果不是对象，可能是数组，直接返回
		if arr, ok := rawData.([]interface{}); ok {
			result := make([]map[string]interface{}, 0, len(arr))
			for _, item := range arr {
				if itemMap, ok := item.(map[string]interface{}); ok {
					result = append(result, itemMap)
				}
			}
			return result, nil
		}
		// 其他类型，返回空数组
		return []map[string]interface{}{}, nil
	}

	// 如果返回的是空对象，直接返回空数组
	if len(lvmthinsObj) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 将对象转换为数组格式
	lvmthins := make([]map[string]interface{}, 0, len(lvmthinsObj))
	for name, data := range lvmthinsObj {
		// 处理不同的数据类型
		var dataMap map[string]interface{}
		switch v := data.(type) {
		case map[string]interface{}:
			dataMap = make(map[string]interface{})
			// 复制所有字段
			for k, val := range v {
				dataMap[k] = val
			}
		case []interface{}:
			// 如果是数组，跳过（不应该出现）
			continue
		default:
			// 如果是其他类型，创建一个新的 map
			dataMap = make(map[string]interface{})
			dataMap["value"] = v
		}
		// 添加名称字段
		dataMap["name"] = name
		lvmthins = append(lvmthins, dataMap)
	}
	return lvmthins, nil
}

// GetNodeDisksZFS 获取节点 ZFS 存储
// GET /api2/json/nodes/{node}/disks/zfs
// Proxmox API 返回的是对象格式，需要转换为数组
func (c *ProxmoxClient) GetNodeDisksZFS(ctx context.Context, nodeName string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/disks/zfs", nodeName)

	// 使用 interface{} 来接收数据，然后根据实际类型处理
	var rawData interface{}
	if err := c.Get(ctx, path, &rawData); err != nil {
		return nil, err
	}

	// 如果返回的是 nil，返回空数组
	if rawData == nil {
		return []map[string]interface{}{}, nil
	}

	// 尝试解析为对象
	zfssObj, ok := rawData.(map[string]interface{})
	if !ok {
		// 如果不是对象，可能是数组，直接返回
		if arr, ok := rawData.([]interface{}); ok {
			result := make([]map[string]interface{}, 0, len(arr))
			for _, item := range arr {
				if itemMap, ok := item.(map[string]interface{}); ok {
					result = append(result, itemMap)
				}
			}
			return result, nil
		}
		// 其他类型，返回空数组
		return []map[string]interface{}{}, nil
	}

	// 如果返回的是空对象，直接返回空数组
	if len(zfssObj) == 0 {
		return []map[string]interface{}{}, nil
	}

	// 将对象转换为数组格式
	zfss := make([]map[string]interface{}, 0, len(zfssObj))
	for name, data := range zfssObj {
		// 处理不同的数据类型
		var dataMap map[string]interface{}
		switch v := data.(type) {
		case map[string]interface{}:
			dataMap = make(map[string]interface{})
			// 复制所有字段
			for k, val := range v {
				dataMap[k] = val
			}
		case []interface{}:
			// 如果是数组，跳过（不应该出现）
			continue
		default:
			// 如果是其他类型，创建一个新的 map
			dataMap = make(map[string]interface{})
			dataMap["value"] = v
		}
		// 添加名称字段
		dataMap["name"] = name
		zfss = append(zfss, dataMap)
	}
	return zfss, nil
}

// InitGPTDisk 初始化 GPT 磁盘
// POST /api2/json/nodes/{node}/disks/initgpt
// 参数通过 URL query string 传递: disk (磁盘设备名，如 /dev/sdb)
func (c *ProxmoxClient) InitGPTDisk(ctx context.Context, nodeName string, disk string) (string, error) {
	path := fmt.Sprintf("/nodes/%s/disks/initgpt", nodeName)

	// 构建 URL 查询参数
	params := url.Values{}
	params.Set("disk", disk)

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.Request(ctx, req, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// WipeDisk 擦除磁盘或分区
// PUT /api2/json/nodes/{node}/disks/wipedisk
// 参数通过 URL query string 传递: disk (磁盘设备名，如 /dev/sdb), partition (可选，分区号)
func (c *ProxmoxClient) WipeDisk(ctx context.Context, nodeName string, disk string, partition *int) (string, error) {
	path := fmt.Sprintf("/nodes/%s/disks/wipedisk", nodeName)

	// 构建 URL 查询参数
	params := url.Values{}
	params.Set("disk", disk)
	if partition != nil {
		params.Set("partition", fmt.Sprintf("%d", *partition))
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.Request(ctx, req, &upid); err != nil {
		return "", err
	}
	return upid, nil
}

// GetStorageStatus 获取存储状态
// GET /api2/json/nodes/{node}/storage/{storage}/status
func (c *ProxmoxClient) GetStorageStatus(ctx context.Context, nodeName, storage string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/storage/%s/status", nodeName, storage)
	var status map[string]interface{}
	if err := c.Get(ctx, path, &status); err != nil {
		return nil, err
	}
	return status, nil
}

// GetStorageRRDData 获取存储 RRD 监控数据
// GET /api2/json/nodes/{node}/storage/{storage}/rrddata
// 参数: timeframe (hour|day|week|month|year), cf (AVERAGE|MAX)
func (c *ProxmoxClient) GetStorageRRDData(ctx context.Context, nodeName, storage, timeframe, cf string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/storage/%s/rrddata", nodeName, storage)

	params := url.Values{}
	if timeframe != "" {
		params.Set("timeframe", timeframe)
	}
	if cf != "" {
		params.Set("cf", cf)
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := c.Request(ctx, req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetStorageContent 获取存储内容列表
// GET /api2/json/nodes/{node}/storage/{storage}/content
// 可通过 content 过滤类型: images,iso,backup 等
func (c *ProxmoxClient) GetStorageContent(ctx context.Context, nodeName, storage, content string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/storage/%s/content", nodeName, storage)

	params := url.Values{}
	if content != "" {
		params.Set("content", content)
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := c.Request(ctx, req, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetStorageVolume 获取单个卷属性
// GET /api2/json/nodes/{node}/storage/{storage}/content/{volume}
func (c *ProxmoxClient) GetStorageVolume(ctx context.Context, nodeName, storage, volume string) (map[string]interface{}, error) {
	escapedVolume := url.PathEscape(volume)
	path := fmt.Sprintf("/nodes/%s/storage/%s/content/%s", nodeName, storage, escapedVolume)

	var info map[string]interface{}
	if err := c.Get(ctx, path, &info); err != nil {
		return nil, err
	}
	return info, nil
}

// UploadStorageContent 上传模板 / ISO / OVA / VM 镜像到存储
// POST /api2/json/nodes/{node}/storage/{storage}/upload
// 参数：content (iso/vztmpl/backup/images...)，文件字段名必须是 "filename"
func (c *ProxmoxClient) UploadStorageContent(
	ctx context.Context,
	nodeName, storage, content, filename string,
	file multipart.File,
) (interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/storage/%s/upload", nodeName, storage)
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 可选的 content 类型
	if content != "" {
		if err := writer.WriteField("content", content); err != nil {
			return nil, err
		}
	}

	// Proxmox 要求文件字段名为 "filename"
	fw, err := writer.CreateFormFile("filename", filename)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(fw, file); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", c.Token)

	uploadClient := &http.Client{
		Timeout: 60 * time.Minute, // 60分钟超时
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := uploadClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		// 尝试解析错误详情
		var errResp struct {
			Data   interface{}            `json:"data"`
			Errors map[string]interface{} `json:"errors,omitempty"`
		}
		if json.Unmarshal(body, &errResp) == nil {
			if len(errResp.Errors) > 0 {
				return nil, fmt.Errorf("proxmox API error (status %d): %v", resp.StatusCode, errResp.Errors)
			}
		}
		return nil, fmt.Errorf("proxmox API error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Data interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}
	// upload 的 data 通常是 UPID 字符串，也可能是对象；这里直接透传给上层
	return apiResp.Data, nil
}

// DeleteStorageContent 删除存储内容（镜像 / ISO / OVA / VM 镜像等）
// DELETE /api2/json/nodes/{node}/storage/{storage}/content/{volume}
// 参数：volume 需要 URL 编码（例如：/local-dir:iso/ubuntu-22.04-server-amd64.iso）
// delay 为可选的延迟删除时间（秒）
func (c *ProxmoxClient) DeleteStorageContent(ctx context.Context, nodeName, storage, volume string, delay *int) error {
	escapedVolume := url.PathEscape(volume)
	path := fmt.Sprintf("/nodes/%s/storage/%s/content/%s", nodeName, storage, escapedVolume)

	// 构建查询参数
	params := url.Values{}
	if delay != nil && *delay > 0 {
		params.Set("delay", fmt.Sprintf("%d", *delay))
	}

	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	return c.Request(ctx, req, nil)
}

// AccessTicketResult 封装 /access/ticket 返回的数据
type AccessTicketResult struct {
	Username            string `json:"username"`
	Ticket              string `json:"ticket"`
	CSRFPreventionToken string `json:"CSRFPreventionToken"`
}

// GetAccessTicket 调用 Proxmox 原生 /access/ticket 接口，使用用户名/密码获取高权限票据。
// 这是一个独立的登录接口，不依赖 PVEAPIToken。
//
// POST /api2/json/access/ticket
// form:
//   - username=root
//   - realm=pam
//   - password=xxxx
func GetAccessTicket(ctx context.Context, apiURL, username, realm, password string) (*AccessTicketResult, error) {
	if strings.TrimSpace(apiURL) == "" {
		return nil, fmt.Errorf("apiURL is required")
	}
	baseURL, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid apiURL: %w", err)
	}

	endpoint := baseURL.JoinPath("/api2/json", "/access/ticket").String()

	form := url.Values{}
	form.Set("username", username)
	form.Set("realm", realm)
	form.Set("password", password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		// 尝试解析错误详情
		var errResp struct {
			Data   interface{}            `json:"data"`
			Errors map[string]interface{} `json:"errors,omitempty"`
		}
		if json.Unmarshal(bodyBytes, &errResp) == nil {
			if len(errResp.Errors) > 0 {
				return nil, fmt.Errorf("proxmox access ticket error (status %d): %v", resp.StatusCode, errResp.Errors)
			}
		}
		return nil, fmt.Errorf("proxmox access ticket error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// 标准响应结构：{"data": { ... }}
	var wrapper struct {
		Data AccessTicketResult `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &wrapper); err != nil {
		return nil, err
	}
	return &wrapper.Data, nil
}

// NodeTermProxy 获取节点终端代理信息（用于 SSH-like 终端）
// POST /api2/json/nodes/{node}/termproxy
// 注意：根据实际测试，termproxy 返回的数据结构与 vncshell 相同（包含 port、ticket、user、upid 等）
// 返回字段通常包含：user、ticket、port、upid 等
func (c *ProxmoxClient) NodeTermProxy(ctx context.Context, nodeName string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/termproxy", nodeName)
	// termproxy 接口不需要任何参数，使用空的 url.Values{}
	var result map[string]interface{}
	if err := c.PostForm(ctx, path, url.Values{}, &result); err != nil {
		return nil, fmt.Errorf("failed to call termproxy for node %s: %w", nodeName, err)
	}
	return result, nil
}

// NodeVncShell 获取节点 VNC Shell 信息（用于图形界面控制台）
// POST /api2/json/nodes/{node}/vncshell
// 返回字段通常包含：port、ticket、user、cert 等
func (c *ProxmoxClient) NodeVncShell(ctx context.Context, nodeName string, websocket, generatePassword bool) (map[string]interface{}, error) {
	path := fmt.Sprintf("/nodes/%s/vncshell", nodeName)

	params := url.Values{}
	if websocket {
		params.Set("websocket", "1")
	}
	if generatePassword {
		params.Set("generate-password", "1")
	}

	var result map[string]interface{}
	if err := c.PostForm(ctx, path, params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// ConvertToTemplate 将虚拟机转换为模板
// POST /api2/json/nodes/{node}/qemu/{vmid}/template
// 参数：
//   - disk: 可选，要转换为基础镜像的磁盘（格式：scsi0, ide0 等）
// 返回：nil（成功）或 error
func (c *ProxmoxClient) ConvertToTemplate(ctx context.Context, nodeName string, vmID uint32, disk string) error {
	path := fmt.Sprintf("/nodes/%s/qemu/%d/template", nodeName, vmID)

	params := url.Values{}
	if disk != "" {
		params.Set("disk", disk)
	}

	// 使用 PostForm 发送请求（即使参数为空）
	if err := c.PostForm(ctx, path, params, nil); err != nil {
		return fmt.Errorf("failed to convert VM %d to template on node %s: %w", vmID, nodeName, err)
	}

	return nil
}

// GetNextFreeVMID 获取集群中下一个可用的 VMID
// GET /api2/json/cluster/nextid
// 返回：可用的 VMID（通常从 100 开始）
// 注意：Proxmox API 返回的是字符串格式的数字，需要转换
func (c *ProxmoxClient) GetNextFreeVMID(ctx context.Context) (uint32, error) {
	path := "/cluster/nextid"

	// Proxmox API 返回的可能是字符串格式，例如 "100"
	var result interface{}
	if err := c.Get(ctx, path, &result); err != nil {
		return 0, fmt.Errorf("failed to get next free vmid: %w", err)
	}

	// 尝试解析为不同类型
	switch v := result.(type) {
	case float64:
		// JSON 数字默认解析为 float64
		return uint32(v), nil
	case string:
		// 字符串格式，需要转换
		vmid, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("failed to parse vmid string '%s': %w", v, err)
		}
		return uint32(vmid), nil
	case int:
		return uint32(v), nil
	case int64:
		return uint32(v), nil
	default:
		return 0, fmt.Errorf("unexpected vmid type: %T, value: %v", result, result)
	}
}

// CreateBackupRequest 创建备份请求参数
// 参考: https://pve.proxmox.com/pve-docs/api-viewer/#/nodes/{node}/vzdump
type CreateBackupRequest struct {
	VMID      uint32 `json:"vmid"`                 // 虚拟机ID（必填）
	Storage   string `json:"storage,omitempty"`    // 存储名称（可选，默认使用配置的存储）
	Compress  string `json:"compress,omitempty"`   // 压缩格式：zst, lzo, gz（可选）
	Mode      string `json:"mode,omitempty"`        // 备份模式：snapshot, suspend, stop（可选，默认 snapshot）
	Remove    int    `json:"remove,omitempty"`     // 是否删除旧备份：0=否, 1=是（可选）
	MailTo    string `json:"mailto,omitempty"`     // 备份完成后发送邮件到（可选）
	MailNotification string `json:"mailnotification,omitempty"` // 邮件通知类型：always, failure（可选）
	NotesTemplate string `json:"notes-template,omitempty"` // 备份注释模板（可选）
	Exclude   string `json:"exclude,omitempty"`     // 排除的挂载点（可选，逗号分隔）
	Quiesce   int    `json:"quiesce,omitempty"`     // 是否使用 quiesce：0=否, 1=是（可选，需要 qemu-guest-agent）
	MaxFiles  int    `json:"maxfiles,omitempty"`    // 保留的最大备份文件数（可选）
	Bwlimit   int    `json:"bwlimit,omitempty"`     // 带宽限制（MB/s）（可选）
	Ionice    int    `json:"ionice,omitempty"`      // IO 优先级（可选）
	Stop      int    `json:"stop,omitempty"`        // 是否停止虚拟机：0=否, 1=是（可选）
	StopWait  int    `json:"stopwait,omitempty"`    // 停止等待时间（秒）（可选）
	DumpDir   string `json:"dumpdir,omitempty"`     // 备份目录（可选，覆盖存储配置）
	Zstd      int    `json:"zstd,omitempty"`        // zstd 压缩级别 1-22（可选，仅当 compress=zst 时有效）
}

// CreateBackup 创建虚拟机备份
// POST /api2/json/nodes/{node}/vzdump
// 参考: https://pve.proxmox.com/pve-docs/api-viewer/#/nodes/{node}/vzdump
// 返回: UPID (任务ID)
func (c *ProxmoxClient) CreateBackup(ctx context.Context, nodeName string, req *CreateBackupRequest) (string, error) {
	path := fmt.Sprintf("/nodes/%s/vzdump", nodeName)

	// 构建 URL 查询参数
	params := url.Values{}
	params.Set("vmid", fmt.Sprintf("%d", req.VMID))

	if req.Storage != "" {
		params.Set("storage", req.Storage)
	}
	if req.Compress != "" {
		params.Set("compress", req.Compress)
	}
	if req.Mode != "" {
		params.Set("mode", req.Mode)
	}
	if req.Remove > 0 {
		params.Set("remove", "1")
	}
	if req.MailTo != "" {
		params.Set("mailto", req.MailTo)
	}
	if req.MailNotification != "" {
		params.Set("mailnotification", req.MailNotification)
	}
	if req.NotesTemplate != "" {
		params.Set("notes-template", req.NotesTemplate)
	}
	if req.Exclude != "" {
		params.Set("exclude", req.Exclude)
	}
	if req.Quiesce > 0 {
		params.Set("quiesce", "1")
	}
	if req.MaxFiles > 0 {
		params.Set("maxfiles", fmt.Sprintf("%d", req.MaxFiles))
	}
	if req.Bwlimit > 0 {
		params.Set("bwlimit", fmt.Sprintf("%d", req.Bwlimit))
	}
	if req.Ionice > 0 {
		params.Set("ionice", fmt.Sprintf("%d", req.Ionice))
	}
	if req.Stop > 0 {
		params.Set("stop", "1")
	}
	if req.StopWait > 0 {
		params.Set("stopwait", fmt.Sprintf("%d", req.StopWait))
	}
	if req.DumpDir != "" {
		params.Set("dumpdir", req.DumpDir)
	}
	if req.Zstd > 0 {
		params.Set("zstd", fmt.Sprintf("%d", req.Zstd))
	}

	// 构建完整的 URL
	endpoint := c.baseUrl.JoinPath("/api2/json", path).String()
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	// 发送 POST 请求（没有 body）
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", err
	}

	var upid string
	if err := c.Request(ctx, httpReq, &upid); err != nil {
		return "", err
	}

	return upid, nil
}
