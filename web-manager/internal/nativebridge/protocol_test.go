package nativebridge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReadRequestRequiresBoundedNewlineTerminatedMessage(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  bool
	}{
		{"{\"v\":1,\"op\":\"status\"}\n", true},
		{"{\"v\":1,\"op\":\"status\"}", false},
		{strings.Repeat(" ", MaxLine) + "\n", false},
		{"\n", false},
	} {
		_, err := ReadRequest(strings.NewReader(tc.input))
		if (err == nil) != tc.want {
			t.Errorf("read request mismatch: %v", err)
		}
	}
}

func TestBoundedCommandOutputDoesNotKeepOversizedData(t *testing.T) {
	b := &boundedOutput{limit: 5}
	if n, err := b.Write([]byte("hello")); n != 5 || err != nil {
		t.Fatal("bounded writer failed")
	}
	if _, err := b.Write([]byte("secret")); err == nil || !b.overflow || string(b.buf) != "hello" {
		t.Fatal("overflow retained output")
	}
}

func TestStatusHidesWindowDetailsAndFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		out     string
		err     error
		visible bool
	}{
		{"mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}, [type=1]\nmFocusedApp=ActivityRecord{b1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity t2 d0}", nil, true},
		{"mCurrentFocus=Window{a1 u0 com.android.settings/com.zte.settings.hotspot.U30UserLockActivity}", nil, false},
		{"mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}", errors.New("service unavailable"), false},
	} {
		s := Service{Run: func(ctx context.Context, path string, args []string, limit int) ([]byte, error) {
			if path == WebctlPath {
				return []byte(`{"ok":true}`), nil
			}
			if path == DumpsysPath {
				if strings.Join(args, " ") != "window displays" {
					t.Fatal("window query must be fixed")
				}
				deadline, _ := ctx.Deadline()
				if time.Until(deadline) > 2100*time.Millisecond {
					t.Fatal("window query timeout too long")
				}
				return []byte(tc.out), tc.err
			}
			t.Fatal("unexpected command")
			return nil, nil
		}}
		out, err := s.Forward(context.Background(), Request{V: 1, Op: "status"})
		if err != nil {
			t.Fatal(err)
		}
		var obj map[string]any
		json.Unmarshal(out, &obj)
		if obj["deviceHomeVisible"] != tc.visible {
			t.Fatal("wrong visibility")
		}
		if bytes.Contains(out, []byte("mCurrentFocus")) || bytes.Contains(out, []byte("HomeMainActivity")) || bytes.Contains(out, []byte("U30UserLockActivity")) {
			t.Fatal("window details exposed")
		}
	}
}

func TestParseRequestAllowsOnlyTwoFixedOperations(t *testing.T) {
	for _, input := range []string{`{"v":1,"op":"status"}`, `{"v":1,"op":"request","envelope":"AQID"}`} {
		if _, err := ParseRequest([]byte(input)); err != nil {
			t.Fatalf("valid request rejected: %v", err)
		}
	}
	bad := []string{
		``, `[]`, `null`, `{`, `{"v":2,"op":"status"}`, `{"v":1,"op":"shell"}`,
		`{"v":1,"op":"status","envelope":"AQID"}`, `{"v":1,"op":"request"}`,
		`{"v":1,"op":"request","envelope":"AQID;id"}`,
		`{"v":1,"op":"request","envelope":"AQID="}`,
		`{"v":1,"op":"request","envelope":"A"}`,
		`{"v":1,"op":"status","command":"id"}`,
		`{"v":1,"v":1,"op":"status"}`, `{"v":1,"op":"status"} {}`,
		`{"v":1,"op":"request","envelope":"` + strings.Repeat("A", MaxEnvelope+4) + `"}`,
		strings.Repeat(" ", MaxLine) + `{"v":1,"op":"status"}`,
	}
	for i, input := range bad {
		if _, err := ParseRequest([]byte(input)); err == nil {
			t.Errorf("bad case %d accepted", i)
		}
	}
}

func goodTrust() Trust {
	return Trust{Schema: 1, PackageName: PackageName, UID: 10123, APKPath: "/data/app/~~token/com.codex.u30vpn-abc/base.apk", APKSHA256: strings.Repeat("a", 64)}
}

func TestTrustRejectsUnknownDuplicateAndUnsafeFields(t *testing.T) {
	good := `{"schema":1,"packageName":"com.codex.u30vpn","uid":10123,"apkPath":"/data/app/~~token/com.codex.u30vpn-abc/base.apk","apkSha256":"` + strings.Repeat("a", 64) + `"}`
	if _, err := ParseTrust([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{
		strings.Replace(good, `"schema":1`, `"schema":1,"schema":1`, 1),
		strings.Replace(good, `"schema":1`, `"schema":1,"extra":true`, 1),
		strings.Replace(good, `10123`, `0`, 1),
		strings.Replace(good, PackageName, "com.codex.u30vpn.evil", 1),
		strings.Replace(good, "/data/app/~~token/com.codex.u30vpn-abc/base.apk", "/data/app/../base.apk", 1),
		strings.Replace(good, "/data/app/~~token/com.codex.u30vpn-abc/base.apk", "/sdcard/base.apk", 1),
		strings.Replace(good, strings.Repeat("a", 64), "abc", 1),
	} {
		if _, err := ParseTrust([]byte(input)); err == nil {
			t.Error("unsafe trust accepted")
		}
	}
}

func TestAuthorizationRechecksPackageUIDPathAndAPKHash(t *testing.T) {
	trust := goodTrust()
	for _, tc := range []struct {
		name            string
		peer            uint32
		uid, path, hash string
		want            bool
	}{
		{"valid", 10123, "package:com.codex.u30vpn uid:10123\n", "package:" + trust.APKPath + "\n", trust.APKSHA256, true},
		{"wrong-peer", 10124, "package:com.codex.u30vpn uid:10123\n", "package:" + trust.APKPath + "\n", trust.APKSHA256, false},
		{"uid-reused", 10123, "package:com.codex.u30vpn uid:10124\n", "package:" + trust.APKPath + "\n", trust.APKSHA256, false},
		{"package-substring", 10123, "package:com.codex.u30vpn.evil uid:10123\n", "package:" + trust.APKPath + "\n", trust.APKSHA256, false},
		{"shared-uid", 10123, "package:com.codex.u30vpn uid:10123\npackage:other uid:10123\n", "package:" + trust.APKPath + "\n", trust.APKSHA256, false},
		{"apk-moved", 10123, "package:com.codex.u30vpn uid:10123\n", "package:/data/app/new/base.apk\n", trust.APKSHA256, false},
		{"split-apk", 10123, "package:com.codex.u30vpn uid:10123\n", "package:" + trust.APKPath + "\npackage:/data/app/x/split.apk\n", trust.APKSHA256, false},
		{"apk-replaced", 10123, "package:com.codex.u30vpn uid:10123\n", "package:" + trust.APKPath + "\n", strings.Repeat("b", 64), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			s := Service{LoadTrust: func() (Trust, error) { return trust, nil }, HashAPK: func(path string) (string, error) {
				if path != trust.APKPath {
					t.Fatal("unfixed apk path")
				}
				return tc.hash, nil
			}, Run: func(ctx context.Context, path string, args []string, limit int) ([]byte, error) {
				calls++
				switch path {
				case CmdPath:
					if strings.Join(args, " ") != "package list packages -U "+PackageName {
						t.Fatal("unsafe cmd args")
					}
					return []byte(tc.uid), nil
				case PMPath:
					if strings.Join(args, " ") != "path "+PackageName {
						t.Fatal("unsafe pm args")
					}
					return []byte(tc.path), nil
				}
				t.Fatal("unexpected command")
				return nil, nil
			}}
			if err := s.Authorize(context.Background(), tc.peer); (err == nil) != tc.want {
				t.Fatalf("authorization mismatch: %v", err)
			}
			if tc.want {
				if calls != 2 {
					t.Fatal("missing package recheck")
				}
				if err := s.Authorize(context.Background(), tc.peer); err != nil || calls != 4 {
					t.Fatal("authorization must recheck each connection")
				}
			}
		})
	}
}

func TestForwardExecutesOnlyFixedWebctlAndBoundsOutput(t *testing.T) {
	for _, tc := range []struct {
		name    string
		req     Request
		output  string
		execErr error
		wantOK  bool
	}{
		{"status", Request{V: 1, Op: "status"}, `{"ok":true,"running":true}`, nil, true},
		{"request", Request{V: 1, Op: "request", Envelope: "AQID"}, `{"ok":true,"job":{"id":"x"}}`, nil, true},
		{"safe-command-error", Request{V: 1, Op: "status"}, `{"ok":false,"error":{"code":"busy","message":"busy"}}`, errors.New("exit 1"), true},
		{"bad-json", Request{V: 1, Op: "status"}, `secret: not json`, nil, false},
		{"missing-ok", Request{V: 1, Op: "status"}, `{"result":"secret"}`, nil, false},
		{"too-large", Request{V: 1, Op: "status"}, strings.Repeat("x", MaxResponse+1), nil, false},
		{"bad-request", Request{V: 1, Op: "shell"}, `{"ok":true}`, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			s := Service{Run: func(ctx context.Context, path string, args []string, limit int) ([]byte, error) {
				if path == DumpsysPath {
					if strings.Join(args, " ") != "window displays" {
						t.Fatal("unsafe dumpsys args")
					}
					return []byte("mCurrentFocus=null"), nil
				}
				calls++
				if path != WebctlPath || limit != MaxResponse {
					t.Fatal("unexpected exec target/limit")
				}
				want := tc.req.Op
				if tc.req.Envelope != "" {
					want += " " + tc.req.Envelope
				}
				if strings.Join(args, " ") != want {
					t.Fatal("unsafe args")
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("missing exec deadline")
				}
				return []byte(tc.output), tc.execErr
			}}
			out, err := s.Forward(context.Background(), tc.req)
			if (err == nil) != tc.wantOK {
				t.Fatalf("forward result mismatch %s %v", out, err)
			}
			if tc.name == "bad-request" && calls != 0 {
				t.Fatal("invalid request reached exec")
			}
		})
	}
}

func TestAPKDigestRejectsUnexpectedContents(t *testing.T) {
	data := []byte("signed APK bytes")
	want := sha256.Sum256(data)
	if got := Digest(data); got != hex.EncodeToString(want[:]) {
		t.Fatal("wrong SHA256")
	}
}

func TestProcStartTimeHandlesParenthesesInCommand(t *testing.T) {
	fields := make([]string, 22)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0] = "S"
	fields[19] = "87654"
	if got, err := ParseProcStartTime("42 (strange ) name) " + strings.Join(fields, " ")); err != nil || got != "87654" {
		t.Fatalf("proc parse failed: %q %v", got, err)
	}
	for _, input := range []string{"", "42 no delimiter", "42 (x) S", "42 (x) " + strings.Repeat("x ", 22)} {
		if _, err := ParseProcStartTime(input); err == nil {
			t.Fatal("malformed proc identity accepted")
		}
	}
}

func TestHomeVisibilityRequiresExactWindowAndFocusedActivity(t *testing.T) {
	home := "\nmFocusedApp=ActivityRecord{b1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity t2 d0}"
	for _, tc := range []struct {
		input string
		want  bool
	}{
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}" + home, true},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/com.zte.mifavor.ufi.home.HomeMainActivity}" + home, true},
		{"  mCurrentFocus=Window{c679ce0 u0 com.zte.mifavor.ufi.home/com.zte.mifavor.ufi.home.HomeMainActivity}, [type=1,alpha=1.0]" + home, true},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}", false},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}\nmFocusedApp=ActivityRecord{b1 u0 com.evil/.Fake t2 d0}", false},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}" + home + home, false},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}\nmFocusedApp=ActivityRecord{b1 u0 com.zte.mifavor.ufi.home.evil/.HomeMainActivity t2 d0}", false},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}\nmFocusedApp=ActivityRecord{b1 u0 com.zte.mifavor.ufi.home/.OtherActivity t2 d0}", false},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}\nmFocusedApp=ActivityRecord{b1 u10 com.zte.mifavor.ufi.home/.HomeMainActivity t2 d0}", false},
		{"  mCurrentFocus=Window{a1 u0 com.android.settings/com.zte.settings.hotspot.U30UserLockActivity}", false},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home.evil/.HomeMainActivity}", false},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.OtherActivity}", false},
		{"  mFocusedApp=com.zte.mifavor.ufi.home/.HomeMainActivity", false},
		{"  mCurrentFocus=null\nOther com.zte.mifavor.ufi.home/.HomeMainActivity", false},
		{"  mCurrentFocus=Window{a1 u0 com.zte.mifavor.ufi.home/.HomeMainActivity}\n  mCurrentFocus=null", false},
	} {
		if got := HomeVisible([]byte(tc.input)); got != tc.want {
			t.Errorf("home visibility=%v want %v", got, tc.want)
		}
	}
}
