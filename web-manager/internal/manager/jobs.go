package manager

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func processIdentity(pid int) (string, string) {
	b, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return "", ""
	}
	index := strings.LastIndex(string(b), ")")
	if index < 0 {
		return "", ""
	}
	parts := strings.Fields(string(b)[index+1:])
	if len(parts) < 20 {
		return "", ""
	}
	boot, _ := os.ReadFile("/proc/sys/kernel/random/boot_id")
	return parts[19], strings.TrimSpace(string(boot))
}
func (m *Manager) readJob() (jobDisk, error) {
	var j jobDisk
	b, err := readBounded(m.webPath("job.json"), 8192)
	if err != nil || json.Unmarshal(b, &j) != nil || !jobIDPattern.MatchString(j.ID) {
		return j, fail("storage_failed")
	}
	return j, nil
}
func (m *Manager) writeJob(j jobDisk) error {
	b, err := json.Marshal(j)
	if err != nil {
		return fail("storage_failed")
	}
	return atomicWrite(m.webPath("job.json"), b, 0600)
}
func (m *Manager) reapJob() {
	j, err := m.readJob()
	if err != nil || j.Status != "running" {
		return
	}
	start, boot := processIdentity(j.PID)
	alive := j.PID > 0 && start != "" && start == j.ProcessStart && boot == j.BootID
	if alive {
		return
	}
	if j.PID == 0 && m.Now().UnixMilli()-j.CreatedAt < 3000 {
		return
	}
	j.Status = "failed"
	j.Message = SafeError("job_interrupted").Error.Message
	if j.Kind == "probe-nodes" || j.Kind == "probe-node" {
		j.Message = SafeError("probe_interrupted").Error.Message
	}
	m.writeJob(j)
	os.Remove(m.webPath("job-request.json"))
}

type RequestResult struct {
	OK  bool `json:"ok"`
	Job Job  `json:"job"`
}

func (m *Manager) Queue(encoded string) (RequestResult, error) {
	out := RequestResult{}
	if err := m.storageReady(); err != nil {
		return out, err
	}
	unlock, err := acquireLock(m.webPath("manager.lock"), false)
	if err != nil {
		return out, err
	}
	defer unlock()
	if err = m.recoverTransaction(); err != nil {
		return out, err
	}
	m.reapJob()
	if j, e := m.readJob(); e == nil && j.Status == "running" {
		return out, fail("busy")
	}
	key, err := m.privateKey()
	if err != nil {
		return out, err
	}
	r, err := Decrypt(encoded, key, m.Now())
	if err != nil {
		return out, err
	}
	if err = m.rememberNonce(r.Nonce); err != nil {
		return out, err
	}
	idBytes := make([]byte, 16)
	if _, err = rand.Read(idBytes); err != nil {
		return out, fail("storage_failed")
	}
	id := hex.EncodeToString(idBytes)
	requestBytes, _ := json.Marshal(r)
	if err = atomicWrite(m.webPath("job-request.json"), requestBytes, 0600); err != nil {
		return out, err
	}
	job := jobDisk{Job: Job{ID: id, Kind: r.Action, Status: "running", Message: "正在处理"}, CreatedAt: m.Now().UnixMilli()}
	if r.Action == "probe-nodes" || r.Action == "probe-node" {
		job.Message = "正在测速"
	}
	if r.Action == "probe-node" {
		job.NodeID = r.ID
	}
	if err = m.writeJob(job); err != nil {
		return out, err
	}
	executable := filepath.Join(m.Base, "webctl")
	cmd := exec.Command(executable, "worker", id)
	detach(cmd)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Stdin = nil
	if err = cmd.Start(); err != nil {
		job.Status = "failed"
		job.Message = SafeError("operation_failed").Error.Message
		m.writeJob(job)
		os.Remove(m.webPath("job-request.json"))
		return out, fail("operation_failed")
	}
	job.PID = cmd.Process.Pid
	job.ProcessStart, job.BootID = processIdentity(job.PID)
	if job.ProcessStart == "" {
		job.PID = 0
	}
	if err = m.writeJob(job); err != nil {
		cmd.Process.Kill()
		cmd.Process.Wait()
		return out, err
	}
	cmd.Process.Release()
	return RequestResult{OK: true, Job: job.Job}, nil
}
func (m *Manager) Worker(id string) error {
	if !jobIDPattern.MatchString(id) {
		return fail("invalid_request")
	}
	if err := m.storageReady(); err != nil {
		return err
	}
	unlock, err := acquireLock(m.webPath("manager.lock"), true)
	if err != nil {
		return err
	}
	defer unlock()
	j, err := m.readJob()
	if err != nil || j.ID != id || j.Status != "running" {
		return fail("invalid_request")
	}
	j.PID = os.Getpid()
	j.ProcessStart, j.BootID = processIdentity(j.PID)
	if err = m.writeJob(j); err != nil {
		return err
	}
	b, err := readBounded(m.webPath("job-request.json"), 16384)
	var r Request
	invalidSingleProbe := func() error {
		j.Status = "failed"
		j.Message = SafeError("invalid_request").Error.Message
		if err := m.writeJob(j); err != nil {
			return err
		}
		os.Remove(m.webPath("job-request.json"))
		return fail("invalid_request")
	}
	if err != nil || strictJSON(b, &r) != nil {
		if j.Kind == "probe-node" {
			return invalidSingleProbe()
		}
		return fail("invalid_request")
	}
	if r.Action == "probe-nodes" && !probeRequestShape(b) {
		return fail("invalid_request")
	}
	if r.Action == "probe-node" || j.Kind == "probe-node" {
		if !singleProbeRequestShape(b) || j.Kind != "probe-node" || j.NodeID != r.ID {
			return invalidSingleProbe()
		}
	}
	defer func() { os.Remove(m.webPath("job-request.json")) }()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	err = m.perform(ctx, r)
	if err != nil {
		j.Status = "failed"
		j.Message = SafeError(ErrorCode(err)).Error.Message
	} else {
		j.Status = "succeeded"
		j.Message = "操作已完成"
		if r.Action == "probe-nodes" {
			j.Message = "测速已完成"
		}
		if r.Action == "probe-node" {
			j.Message = "单节点测速已完成"
		}
	}
	if writeErr := m.writeJob(j); writeErr != nil {
		return writeErr
	}
	return err
}
