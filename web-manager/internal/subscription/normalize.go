package subscription

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

const (
	MaxInputBytes    = 8 * 1024 * 1024
	MaxExpandedBytes = 16 * 1024 * 1024
	MaxDepth         = 32
	MaxItems         = 100000
	MaxProxies       = 4096
)

var (
	ErrInvalid       = errors.New("unsupported subscription data")
	ErrLimit         = errors.New("subscription limit exceeded")
	ErrFileReference = errors.New("external file reference is not allowed")
	ErrReference     = errors.New("invalid node dependency")
	ErrDuplicate     = errors.New("duplicate subscription node")
	protocolPattern  = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
)

type Node struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type Result struct {
	Provider []byte
	Nodes    []Node
	Proxies  []map[string]any
}

func Normalize(raw []byte) (Result, error) {
	if len(raw) > MaxInputBytes {
		return Result{}, ErrLimit
	}
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(raw) || len(bytes.TrimSpace(raw)) == 0 {
		return Result{}, ErrInvalid
	}
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	var doc, extra yaml.Node
	if err := decoder.Decode(&doc); err != nil {
		return Result{}, ErrInvalid
	}
	if err := decoder.Decode(&extra); err != io.EOF {
		return Result{}, ErrInvalid
	}
	w := walker{active: make(map[*yaml.Node]bool), remaining: MaxItems, bytesLeft: MaxExpandedBytes}
	value, err := w.plain(&doc, 0)
	if err != nil {
		return Result{}, err
	}
	if root, ok := value.(map[string]any); ok {
		value = root["proxies"]
	}
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return Result{}, ErrInvalid
	}
	if len(items) > MaxProxies {
		return Result{}, ErrLimit
	}
	proxies := make([]map[string]any, len(items))
	nodes := make([]Node, len(items))
	byName := make(map[string]int, len(items))
	deps := make([]int, len(items))
	for i, item := range items {
		p, ok := item.(map[string]any)
		if !ok {
			return Result{}, ErrInvalid
		}
		name, ok := p["name"].(string)
		if !ok || strings.TrimSpace(name) == "" || len(name) > 512 || strings.ContainsFunc(name, unicode.IsControl) {
			return Result{}, ErrInvalid
		}
		if _, exists := byName[name]; exists {
			return Result{}, ErrDuplicate
		}
		protocol, ok := p["type"].(string)
		protocol = strings.ToLower(protocol)
		if !ok || !protocolPattern.MatchString(protocol) {
			return Result{}, ErrInvalid
		}
		switch protocol {
		case "direct", "reject", "reject-drop", "dns", "pass":
			return Result{}, ErrInvalid
		}
		if err := sanitize(p, protocol); err != nil {
			return Result{}, err
		}
		delete(p, "name")
		p["type"] = protocol
		proxies[i], nodes[i], deps[i] = p, Node{Name: name, Type: protocol}, -1
		byName[name] = i
	}
	for i, p := range proxies {
		if ref, exists := p["dialer-proxy"]; exists {
			name, ok := ref.(string)
			if !ok {
				return Result{}, ErrReference
			}
			target, exists := byName[name]
			if !exists {
				return Result{}, ErrReference
			}
			deps[i] = target
		}
	}
	// Hash dependency identities instead of source names. Renaming or reordering
	// a referenced node cannot change or accidentally retarget a saved selection.
	states := make([]uint8, len(proxies))
	var identify func(int) error
	identify = func(i int) error {
		if states[i] == 2 {
			return nil
		}
		if states[i] == 1 {
			return ErrReference
		}
		states[i] = 1
		if deps[i] >= 0 {
			if err := identify(deps[i]); err != nil {
				return err
			}
			proxies[i]["dialer-proxy"] = nodes[deps[i]].ID
		}
		canonical, err := json.Marshal(proxies[i])
		if err != nil {
			return ErrInvalid
		}
		digest := sha256.Sum256(canonical)
		nodes[i].ID = "n-" + hex.EncodeToString(digest[:])
		states[i] = 2
		return nil
	}
	for i := range nodes {
		if err := identify(i); err != nil {
			return Result{}, err
		}
	}
	// Providers sometimes advertise one connection under several display names.
	// Deduplicate only after all references have resolved to stable identities;
	// choose a deterministic display name so input order does not rename a node.
	seenIDs := make(map[string]int, len(nodes))
	uniqueNodes := make([]Node, 0, len(nodes))
	uniqueProxies := make([]map[string]any, 0, len(proxies))
	for i, node := range nodes {
		if existing, ok := seenIDs[node.ID]; ok {
			if node.Name < uniqueNodes[existing].Name {
				uniqueNodes[existing].Name = node.Name
			}
			continue
		}
		seenIDs[node.ID] = len(uniqueNodes)
		proxies[i]["name"] = node.ID
		uniqueNodes = append(uniqueNodes, node)
		uniqueProxies = append(uniqueProxies, proxies[i])
	}
	provider, err := json.Marshal(map[string]any{"proxies": uniqueProxies})
	if err != nil {
		return Result{}, ErrInvalid
	}
	return Result{Provider: append(provider, '\n'), Nodes: uniqueNodes, Proxies: uniqueProxies}, nil
}

type walker struct {
	active    map[*yaml.Node]bool
	remaining int
	bytesLeft int
}

func (w *walker) plain(n *yaml.Node, depth int) (any, error) {
	w.remaining--
	if w.remaining < 0 || depth > MaxDepth {
		return nil, ErrLimit
	}
	if n == nil || w.active[n] {
		return nil, ErrInvalid
	}
	w.active[n] = true
	defer delete(w.active, n)
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) != 1 {
			return nil, ErrInvalid
		}
		return w.plain(n.Content[0], depth)
	case yaml.AliasNode:
		return w.plain(n.Alias, depth)
	case yaml.MappingNode:
		if n.Tag != "!!map" || len(n.Content)%2 != 0 {
			return nil, ErrInvalid
		}
		out := make(map[string]any, len(n.Content)/2)
		var merge *yaml.Node
		for i := 0; i < len(n.Content); i += 2 {
			key, child := n.Content[i], n.Content[i+1]
			w.remaining--
			if w.remaining < 0 {
				return nil, ErrLimit
			}
			if key.Tag == "!!merge" && key.Value == "<<" {
				if merge != nil {
					return nil, ErrInvalid
				}
				merge = child
				continue
			}
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" {
				return nil, ErrInvalid
			}
			w.bytesLeft -= len(key.Value)
			if w.bytesLeft < 0 {
				return nil, ErrLimit
			}
			if _, exists := out[key.Value]; exists {
				return nil, ErrInvalid
			}
			value, err := w.plain(child, depth+1)
			if err != nil {
				return nil, err
			}
			out[key.Value] = value
		}
		if merge != nil {
			value, err := w.plain(merge, depth+1)
			if err != nil {
				return nil, err
			}
			sources, ok := value.([]any)
			if !ok {
				sources = []any{value}
			}
			for _, source := range sources {
				m, ok := source.(map[string]any)
				if !ok {
					return nil, ErrInvalid
				}
				for key, value := range m {
					if _, exists := out[key]; !exists {
						out[key] = value
					}
				}
			}
		}
		return out, nil
	case yaml.SequenceNode:
		if n.Tag != "!!seq" {
			return nil, ErrInvalid
		}
		out := make([]any, 0, len(n.Content))
		for _, child := range n.Content {
			value, err := w.plain(child, depth+1)
			if err != nil {
				return nil, err
			}
			out = append(out, value)
		}
		return out, nil
	case yaml.ScalarNode:
		w.bytesLeft -= len(n.Value)
		if w.bytesLeft < 0 {
			return nil, ErrLimit
		}
		switch n.Tag {
		case "!!str":
			return n.Value, nil
		case "!!null":
			return nil, nil
		case "!!bool", "!!int", "!!float":
			var scalar any
			if err := n.Decode(&scalar); err != nil {
				return nil, ErrInvalid
			}
			switch v := scalar.(type) {
			case bool, int, int64, uint64:
				return v, nil
			case float64:
				if !math.IsInf(v, 0) && !math.IsNaN(v) {
					return v, nil
				}
			}
		}
	}
	return nil, ErrInvalid
}

func sanitize(value any, protocol string) error {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			k := strings.ReplaceAll(strings.ToLower(key), "_", "-")
			if k == "interface-name" || k == "routing-mark" {
				delete(v, key)
				continue
			}
			if child != nil && child != "" {
				file := strings.HasSuffix(k, "-path") || strings.HasSuffix(k, "-file")
				switch k {
				case "ca", "certificate", "client-certificate", "client-key":
					pem, ok := child.(string)
					file = !ok || !strings.HasPrefix(strings.TrimSpace(pem), "-----BEGIN ")
				case "private-key":
					if protocol == "ssh" {
						pem, ok := child.(string)
						file = !ok || !strings.HasPrefix(strings.TrimSpace(pem), "-----BEGIN ")
					}
				}
				if file {
					return ErrFileReference
				}
			}
			if err := sanitize(child, protocol); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range v {
			if err := sanitize(child, protocol); err != nil {
				return err
			}
		}
	}
	return nil
}
