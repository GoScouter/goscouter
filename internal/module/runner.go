package module

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"

	"goscouter/pkg/requests"

	"github.com/GoScouter/sdk"
)

const (
	socketPath = "/tmp/gs.sock"
)

type Runner struct {
	Listener net.Listener

	mu          sync.RWMutex
	modulesData map[sdk.ModuleNamespace]json.RawMessage
}

func CreateRunner() (*Runner, error) {
	err := os.Setenv("GS_SOCKET", socketPath)
	if err != nil {
		return nil, err
	}

	_ = os.Remove(socketPath)

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}

	return &Runner{
		Listener:    l,
		modulesData: make(map[sdk.ModuleNamespace]json.RawMessage),
	}, nil
}

func (r *Runner) Report(namespace sdk.ModuleNamespace, data json.RawMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.modulesData == nil {
		r.modulesData = make(map[sdk.ModuleNamespace]json.RawMessage)
	}
	r.modulesData[namespace] = data
}

func (r *Runner) Data(namespace sdk.ModuleNamespace) (json.RawMessage, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, ok := r.modulesData[namespace]
	return data, ok
}

func (r *Runner) CleanupState() {
	r.mu.Lock()
	defer r.mu.Unlock()

	clear(r.modulesData)
}

func (r *Runner) Start(ctx context.Context) error {
	if r.Listener == nil {
		return errors.New("runner: listener not initialized, use CreateRunner")
	}

	defer func() {
		_ = r.Listener.Close()
		_ = os.Remove(socketPath)
	}()

	go func() {
		<-ctx.Done()
		_ = r.Listener.Close()
	}()

	for {
		conn, err := r.Listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			warn(fmt.Sprintf("runner: accept: %v", err))
			continue
		}

		go r.handleClient(conn)
	}
}

func (r *Runner) handleClient(conn net.Conn) {
	defer conn.Close()

	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)

	for {
		var req sdk.Request
		if err := dec.Decode(&req); err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				warn(fmt.Sprintf("runner: decode request: %v", err))
			}
			return
		}

		if err := r.handle(enc, &req); err != nil {
			warn(fmt.Sprintf("runner: %q request: %v", req.Type, err))
		}
	}
}

func (r *Runner) handle(enc *json.Encoder, req *sdk.Request) error {
	switch req.Type {
	case requests.MethodReport:
		var report requests.ReportRequest
		if err := json.Unmarshal(req.Data, &report); err != nil {
			return fmt.Errorf("decode report: %w", err)
		}

		r.Report(report.Namespace, report.Data)
		return nil

	case sdk.MethodAsk:
		var ask sdk.AskRequest
		if err := json.Unmarshal(req.Data, &ask); err != nil {
			_ = respond(enc, req, nil, fmt.Errorf("malformed ask request"))
			return fmt.Errorf("decode ask: %w", err)
		}

		namespace := sdk.ModuleNamespace(ask)
		data, ok := r.Data(namespace)
		if !ok {
			return respond(enc, req, nil, fmt.Errorf("cannot find %s data", Key(namespace)))
		}

		return respond(enc, req, data, nil)

	default:
		return fmt.Errorf("unknown request type")
	}
}

func respond(enc *json.Encoder, req *sdk.Request, data json.RawMessage, respErr error) error {
	response := sdk.Response{
		ClientID:  req.ClientID,
		RequestID: req.RequestID,
		Data:      data,
	}
	if respErr != nil {
		response.Error = respErr.Error()
	}

	if err := enc.Encode(response); err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}

type Reporter struct {
	mu   sync.Mutex
	conn net.Conn
	enc  *json.Encoder
}

func DialReporter() (*Reporter, error) {
	conn, err := net.Dial("unix", os.Getenv("GS_SOCKET"))
	if err != nil {
		return nil, err
	}

	return &Reporter{conn: conn, enc: json.NewEncoder(conn)}, nil
}

func (r *Reporter) Close() error {
	if r == nil {
		return nil
	}
	return r.conn.Close()
}

func (r *Reporter) Report(namespace sdk.ModuleNamespace, data json.RawMessage) error {
	if r == nil {
		return errors.New("module: not connected to the runner socket")
	}

	payload, err := json.Marshal(requests.ReportRequest{
		Namespace: namespace,
		Data:      data,
	})
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	return r.enc.Encode(sdk.Request{
		Type: requests.MethodReport,
		Data: payload,
	})
}

func RunInOrder(info sdk.ModuleInfo, target string, args []string, manager *Manager, reporter *Reporter) ([]json.RawMessage, []string, error) {
	if Namespace(info) == Namespace(SdModuleInfo) {
		sd := manager.GetInternal(Key(Namespace(SdModuleInfo)))
		data, err := sd.Scout(target, args)
		if err != nil {
			return nil, nil, err
		}

		if err := reporter.Report(Namespace(SdModuleInfo), data); err != nil {
			return nil, nil, err
		}

		return []json.RawMessage{data}, []string{sd.Render(data)}, nil
	}

	plan, err := manager.Graph.Plan(Key(Namespace(info)))
	if err != nil {
		return nil, nil, err
	}

	dataOutput := make([]json.RawMessage, 0, len(plan))
	viewOutput := make([]string, 0, len(plan))

	for _, key := range plan {
		data, view, err := RunModule(target, args, key, manager, reporter)
		if err != nil {
			return nil, nil, err
		}

		dataOutput = append(dataOutput, data)
		viewOutput = append(viewOutput, view)
	}

	return dataOutput, viewOutput, nil
}

func RunModule(target string, args []string, key string, manager *Manager, reporter *Reporter) (json.RawMessage, string, error) {
	author, name, _ := strings.Cut(key, ":")
	ns := sdk.ModuleNamespace{Author: author, Name: name}

	if ns.Author != internalAuthor {
		m := manager.GetExternal(key)
		if m == nil {
			return nil, "", fmt.Errorf("unknown module %q", key)
		}

		data, view, err := m.Scout(context.Background(), target, args)
		if err != nil {
			return nil, "", err
		}

		if err := reporter.Report(ns, data); err != nil {
			return nil, "", err
		}

		return data, view, nil
	}

	m := manager.GetInternal(key)
	if m == nil {
		return nil, "", fmt.Errorf("unknown module %q", key)
	}

	data, err := m.Scout(target, args)
	if err != nil {
		return nil, "", err
	}

	if err := reporter.Report(ns, data); err != nil {
		return nil, "", err
	}

	return data, m.Render(data), nil
}
