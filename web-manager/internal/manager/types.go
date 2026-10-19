package manager

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"u30pro/webmanager/internal/subscription"
)

const BasePath = "/data/adb/u30pro-vpn"
const Version = "1.3.0-alpha.1"
const MaxEnvelope = 24576
const envelopeLabel = "u30pro-vpn-web-v1"

type Envelope struct {
	V    int    `json:"v"`
	Key  string `json:"key"`
	IV   string `json:"iv"`
	Data string `json:"data"`
}
type Request struct {
	V        int    `json:"v"`
	IssuedAt int64  `json:"issuedAt"`
	Nonce    string `json:"nonce"`
	Action   string `json:"action"`
	Enabled  *bool  `json:"enabled,omitempty"`
	ID       string `json:"id,omitempty"`
	URL      string `json:"url,omitempty"`
}
type State struct {
	Schema          int                 `json:"schema"`
	DesiredEnabled  bool                `json:"desiredEnabled"`
	SelectedID      string              `json:"selectedId"`
	SubscriptionURL string              `json:"subscriptionUrl,omitempty"`
	UpdatedAt       string              `json:"updatedAt,omitempty"`
	Nodes           []subscription.Node `json:"nodes"`
	Message         string              `json:"message,omitempty"`
}
type Job struct {
	ID           string `json:"id"`
	Kind         string `json:"kind"`
	NodeID       string `json:"nodeId,omitempty"`
	Status       string `json:"status"`
	Message      string `json:"message"`
	PID          int    `json:"-"`
	ProcessStart string `json:"-"`
	BootID       string `json:"-"`
	CreatedAt    int64  `json:"-"`
}
type jobDisk struct {
	Job
	PID          int    `json:"pid"`
	ProcessStart string `json:"processStart"`
	BootID       string `json:"bootId"`
	CreatedAt    int64  `json:"createdAt"`
}
type PublicNode struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Type  string  `json:"type"`
	Delay *uint16 `json:"delay"`
	Alive *bool   `json:"alive"`
}
type PublicSubscription struct {
	Configured bool   `json:"configured"`
	UpdatedAt  string `json:"updatedAt"`
	Host       string `json:"host"`
	NodeCount  int    `json:"nodeCount"`
}
type Status struct {
	OK             bool               `json:"ok"`
	Installed      bool               `json:"installed"`
	DesiredEnabled bool               `json:"desiredEnabled"`
	Running        bool               `json:"running"`
	State          string             `json:"state"`
	SelectedID     string             `json:"selectedId"`
	ActiveID       string             `json:"activeId"`
	Nodes          []PublicNode       `json:"nodes"`
	Subscription   PublicSubscription `json:"subscription"`
	Job            *Job               `json:"job"`
	PublicKeyPEM   string             `json:"publicKeyPem"`
	Message        string             `json:"message"`
	ServerTime     int64              `json:"serverTime"`
}
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type ErrorResult struct {
	OK    bool      `json:"ok"`
	Error ErrorInfo `json:"error"`
}
type codeError string

func (e codeError) Error() string { return string(e) }
func fail(code string) error      { return codeError(code) }
func ErrorCode(err error) string {
	var e codeError
	if errors.As(err, &e) {
		return string(e)
	}
	return "operation_failed"
}
func SafeError(code string) ErrorResult {
	messages := map[string]string{"root_required": "需要 UFI Root 权限", "not_initialized": "管理组件尚未初始化", "busy": "正在处理上一项操作，请稍后重试", "invalid_request": "请求无效或已过期，请刷新页面后重试", "replayed_request": "请求已处理，请刷新页面", "download_failed": "订阅下载失败，已保留原配置", "invalid_subscription": "订阅格式或节点验证失败，已保留原配置", "apply_failed": "节点应用失败，已恢复原配置", "rollback_failed": "恢复未完成，请查看运行状态后重试", "invalid_node": "该节点已不存在，请刷新列表", "control_failed": "VPN 开关操作未完成，请刷新状态", "storage_failed": "设备保存失败", "insecure_storage": "管理目录权限异常，操作已停止", "operation_failed": "操作未完成，请刷新状态", "job_interrupted": "上次操作被中断，已尝试恢复原配置"}
	message, ok := messages[code]
	switch code {
	case "not_configured":
		message, ok = "请先添加有效订阅，再开启 VPN", true
	case "vpn_stopped":
		message, ok = "请先开启 VPN 再测速", true
	case "probe_failed":
		message, ok = "测速未完成，请重试", true
	case "probe_interrupted":
		message, ok = "上次测速被中断，请重试", true
	}
	if !ok {
		code = "operation_failed"
		message = messages[code]
	}
	return ErrorResult{false, ErrorInfo{code, message}}
}

type Controller interface {
	Snapshot(context.Context) (NativeSnapshot, error)
	Refresh(context.Context) error
	Select(context.Context, string) error
	Health(context.Context) error
	Probe(context.Context) error
	ProbeNode(context.Context, string) error
}
type NativeSnapshot struct {
	Running  bool
	Selected string
	Active   string
	Nodes    map[string]NativeNode
}
type NativeNode struct {
	Delay *uint16
	Alive *bool
}
type Manager struct {
	Base     string
	Now      func() time.Time
	Core     Controller
	Control  func(context.Context, string) error
	Fetch    func(context.Context, string) ([]byte, error)
	Validate func(context.Context, []map[string]any) error
	Test     bool
}

func New() *Manager {
	m := &Manager{Base: BasePath, Now: time.Now}
	m.Core = &UnixController{Socket: filepath.Join(BasePath, "controller.sock")}
	m.Control = m.control
	m.Fetch = Download
	m.Validate = m.validate
	return m
}
func NewForTest(base string) *Manager         { return &Manager{Base: base, Now: time.Now, Test: true} }
func (m *Manager) webPath(name string) string { return filepath.Join(m.Base, "web", name) }
func (m *Manager) providerPath() string       { return filepath.Join(m.Base, "cache", "provider.yaml") }
