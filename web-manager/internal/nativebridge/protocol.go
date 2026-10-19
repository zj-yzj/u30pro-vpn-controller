package nativebridge

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	Version     = "1.0.0"
	Base        = "/data/adb/u30pro-vpn"
	NativeDir   = Base + "/native-ui"
	BridgePath  = NativeDir + "/webbridge"
	TrustPath   = NativeDir + "/trust.json"
	WebctlPath  = Base + "/webctl"
	CmdPath     = "/system/bin/cmd"
	PMPath      = "/system/bin/pm"
	DumpsysPath = "/system/bin/dumpsys"
	PackageName = "com.codex.u30vpn"
	SocketName  = "codex_u30vpn_v1"
	MaxLine     = 32768
	MaxEnvelope = 24576
	MaxResponse = 1 << 20
	Timeout     = 12 * time.Second
)

var errInvalid = errors.New("invalid_request")
var errDenied = errors.New("unauthorized")
var errUnavailable = errors.New("unavailable")
var envelopePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Request struct {
	V        int    `json:"v"`
	Op       string `json:"op"`
	Envelope string `json:"envelope,omitempty"`
}
type Trust struct {
	Schema      int    `json:"schema"`
	PackageName string `json:"packageName"`
	UID         uint32 `json:"uid"`
	APKPath     string `json:"apkPath"`
	APKSHA256   string `json:"apkSha256"`
}

// A map decoder alone silently accepts duplicate keys. Require one flat object,
// with each expected field appearing exactly once, before decoding typed values.
func uniqueObject(data []byte) (map[string]json.RawMessage, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	t, err := d.Token()
	if err != nil || t != json.Delim('{') {
		return nil, errInvalid
	}
	m := map[string]json.RawMessage{}
	for d.More() {
		t, err = d.Token()
		if err != nil {
			return nil, errInvalid
		}
		k, ok := t.(string)
		if !ok {
			return nil, errInvalid
		}
		if _, ok = m[k]; ok {
			return nil, errInvalid
		}
		var v json.RawMessage
		if d.Decode(&v) != nil {
			return nil, errInvalid
		}
		m[k] = v
	}
	if t, err = d.Token(); err != nil || t != json.Delim('}') {
		return nil, errInvalid
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, errInvalid
	}
	return m, nil
}

func ParseRequest(data []byte) (Request, error) {
	var req Request
	if len(data) > MaxLine || len(data) == 0 {
		return req, errInvalid
	}
	m, err := uniqueObject(data)
	if err != nil {
		return req, errInvalid
	}
	if json.Unmarshal(data, &req) != nil || req.V != 1 {
		return Request{}, errInvalid
	}
	switch req.Op {
	case "status":
		if len(m) != 2 || m["v"] == nil || m["op"] == nil || req.Envelope != "" {
			return Request{}, errInvalid
		}
	case "request":
		if len(m) != 3 || m["v"] == nil || m["op"] == nil || m["envelope"] == nil || len(req.Envelope) == 0 || len(req.Envelope) > MaxEnvelope || !envelopePattern.MatchString(req.Envelope) {
			return Request{}, errInvalid
		}
		if _, err := base64.RawURLEncoding.Strict().DecodeString(req.Envelope); err != nil {
			return Request{}, errInvalid
		}
	default:
		return Request{}, errInvalid
	}
	return req, nil
}

func ReadRequest(reader io.Reader) (Request, error) {
	line, err := bufio.NewReaderSize(reader, MaxLine).ReadSlice('\n')
	if err != nil || len(line) > MaxLine {
		return Request{}, errInvalid
	}
	return ParseRequest(bytes.TrimSuffix(line, []byte{'\n'}))
}

func ParseTrust(data []byte) (Trust, error) {
	var trust Trust
	if len(data) > 4096 {
		return trust, errDenied
	}
	m, err := uniqueObject(data)
	if err != nil || len(m) != 5 {
		return trust, errDenied
	}
	for _, k := range []string{"schema", "packageName", "uid", "apkPath", "apkSha256"} {
		if m[k] == nil {
			return trust, errDenied
		}
	}
	if json.Unmarshal(data, &trust) != nil || trust.Schema != 1 || trust.PackageName != PackageName || trust.UID < 10000 || trust.UID > 2147483647 || !digestPattern.MatchString(trust.APKSHA256) || !validAPKPath(trust.APKPath) {
		return Trust{}, errDenied
	}
	return trust, nil
}
func validAPKPath(path string) bool {
	return strings.HasPrefix(path, "/data/app/") && strings.HasSuffix(path, "/base.apk") && cleanAPKPath(path) && len(strings.Split(path, "/")) >= 5 && !strings.ContainsAny(path, "\x00\r\n\t \\")
}
func cleanAPKPath(value string) bool { return path.Clean(value) == value }

type Runner func(context.Context, string, []string, int) ([]byte, error)
type Service struct {
	Run       Runner
	LoadTrust func() (Trust, error)
	HashAPK   func(string) (string, error)
}

func NewService() *Service { return &Service{Run: RunCommand, LoadTrust: LoadTrust, HashAPK: HashAPK} }

func (s *Service) Authorize(ctx context.Context, peerUID uint32) error {
	t, err := s.LoadTrust()
	if err != nil || peerUID != t.UID {
		return errDenied
	}
	uid, err := s.Run(ctx, CmdPath, []string{"package", "list", "packages", "-U", PackageName}, 16384)
	if err != nil || !packageUIDMatches(uid, t.UID) {
		return errDenied
	}
	path, err := s.Run(ctx, PMPath, []string{"path", PackageName}, 16384)
	if err != nil || strings.TrimSpace(string(path)) != "package:"+t.APKPath || strings.Contains(strings.TrimSpace(string(path)), "\n") {
		return errDenied
	}
	hash, err := s.HashAPK(t.APKPath)
	if err != nil || hash != t.APKSHA256 {
		return errDenied
	}
	return nil
}
func packageUIDMatches(out []byte, uid uint32) bool {
	fields := strings.Fields(strings.TrimSpace(string(out)))
	return len(fields) == 2 && fields[0] == "package:"+PackageName && fields[1] == "uid:"+strconv.FormatUint(uint64(uid), 10)
}

func (s *Service) Forward(ctx context.Context, req Request) ([]byte, error) {
	b, _ := json.Marshal(req)
	checked, err := ParseRequest(b)
	if err != nil {
		return nil, errInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	args := []string{checked.Op}
	if checked.Op == "request" {
		args = append(args, checked.Envelope)
	}
	out, runErr := s.Run(ctx, WebctlPath, args, MaxResponse)
	if len(out) > MaxResponse || len(out) == 0 {
		return nil, errUnavailable
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(out, &obj) != nil || obj == nil {
		return nil, errUnavailable
	}
	var ok bool
	if value, has := obj["ok"]; !has || json.Unmarshal(value, &ok) != nil {
		return nil, errUnavailable
	}
	if runErr != nil && ok {
		return nil, errUnavailable
	}
	if checked.Op == "status" {
		homeCtx, homeCancel := context.WithTimeout(ctx, 2*time.Second)
		windows, windowErr := s.Run(homeCtx, DumpsysPath, []string{"window", "displays"}, MaxResponse)
		homeCancel()
		visible := windowErr == nil && len(windows) <= MaxResponse && HomeVisible(windows)
		obj["deviceHomeVisible"], _ = json.Marshal(visible)
	}
	out, err = json.Marshal(obj)
	if err != nil || len(out) > MaxResponse {
		return nil, errUnavailable
	}
	return out, nil
}

// Window titles are application-controlled. Require WindowManager's focused
// ActivityRecord as well, and fail closed on multiple displays/focus records.
func HomeVisible(windows []byte) bool {
	windowFound, activityFound := false, false
	windowHome, activityHome := false, false
	for _, line := range strings.Split(string(windows), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "mCurrentFocus="):
			if windowFound {
				return false
			}
			windowFound = true
			if !strings.HasPrefix(line, "mCurrentFocus=Window{") {
				continue
			}
			window, _, closed := strings.Cut(line, "}")
			if !closed {
				continue
			}
			fields := strings.Fields(window)
			if len(fields) != 3 || fields[1] != "u0" {
				continue
			}
			windowHome = isHomeComponent(fields[2])
		case strings.HasPrefix(line, "mFocusedApp="):
			if activityFound {
				return false
			}
			activityFound = true
			if !strings.HasPrefix(line, "mFocusedApp=ActivityRecord{") {
				continue
			}
			activity, _, closed := strings.Cut(line, "}")
			if !closed {
				continue
			}
			fields := strings.Fields(activity)
			if len(fields) < 3 || fields[1] != "u0" {
				continue
			}
			activityHome = isHomeComponent(fields[2])
		}
	}
	return windowFound && activityFound && windowHome && activityHome
}
func isHomeComponent(component string) bool {
	return component == "com.zte.mifavor.ufi.home/.HomeMainActivity" || component == "com.zte.mifavor.ufi.home/com.zte.mifavor.ufi.home.HomeMainActivity"
}
func SafeError(code string) []byte {
	messages := map[string]string{"invalid_request": "Invalid native control request.", "unauthorized": "Native application identity could not be verified.", "busy": "Native control is busy. Try again.", "root_required": "Native control requires its installed root helper.", "unavailable": "Native control is temporarily unavailable."}
	message, ok := messages[code]
	if !ok {
		code = "unavailable"
		message = messages[code]
	}
	b, _ := json.Marshal(map[string]any{"ok": false, "error": map[string]string{"code": code, "message": message}})
	return b
}
func Digest(data []byte) string { d := sha256.Sum256(data); return hex.EncodeToString(d[:]) }

func ParseProcStartTime(stat string) (string, error) {
	end := strings.LastIndex(stat, ") ")
	if end < 0 {
		return "", errUnavailable
	}
	fields := strings.Fields(stat[end+2:])
	if len(fields) < 20 {
		return "", errUnavailable
	}
	n, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil || n == 0 {
		return "", errUnavailable
	}
	return strconv.FormatUint(n, 10), nil
}
