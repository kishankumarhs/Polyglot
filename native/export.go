package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	core "polyglot/internal/logger"
)

type nativeInstance struct {
	log       *core.Logger
	errMu     sync.Mutex
	lastErr   string
	errCStr   *C.char
	statsCStr *C.char
}

// maxRetiredHandles bounds the memory held by closed handles. Retiring an
// address instead of reusing it immediately is what lets a stale handle report
// "invalid logger handle" rather than aliasing a different logger. Past this
// many retirements we start recycling the oldest addresses so a create/close
// loop cannot grow memory without limit.
const maxRetiredHandles = 1024

var (
	mu      sync.RWMutex
	loggers = map[uintptr]*nativeInstance{}

	handles handlePool

	globalErrMu   sync.Mutex
	globalLastErr string
	globalErrCStr *C.char

	versionOnce sync.Once
	versionCStr *C.char

	// testPanicCreate is flipped by tests to verify recover wrappers.
	testPanicCreate bool
)

func setGlobalErr(err error) {
	globalErrMu.Lock()
	defer globalErrMu.Unlock()
	if err == nil {
		globalLastErr = ""
		return
	}
	globalLastErr = err.Error()
}

func (n *nativeInstance) setErr(err error) {
	n.errMu.Lock()
	defer n.errMu.Unlock()
	if err == nil {
		n.lastErr = ""
		return
	}
	n.lastErr = err.Error()
}

func goString(s *C.char) string {
	if s == nil {
		return ""
	}
	return C.GoString(s)
}

func handleID(h unsafe.Pointer) uintptr {
	return uintptr(h)
}

// handlePool owns the one-byte C allocations used as opaque handle addresses.
type handlePool struct {
	mu      sync.Mutex
	retired []unsafe.Pointer
	head    int
}

func (p *handlePool) acquire() unsafe.Pointer {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.retired)-p.head >= maxRetiredHandles {
		h := p.retired[p.head]
		p.retired[p.head] = nil
		p.head++
		if p.head >= maxRetiredHandles {
			p.retired = append(p.retired[:0], p.retired[p.head:]...)
			p.head = 0
		}
		return h
	}
	return C.malloc(1)
}

func (p *handlePool) release(h unsafe.Pointer) {
	if h == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.retired = append(p.retired, h)
}

func lookup(h unsafe.Pointer) *nativeInstance {
	if h == nil {
		return nil
	}
	mu.RLock()
	defer mu.RUnlock()
	return loggers[handleID(h)]
}

func registerLogger(log *core.Logger) unsafe.Pointer {
	handle := handles.acquire()
	if handle == nil {
		_ = log.Close()
		setGlobalErr(fmt.Errorf("out of memory allocating logger handle"))
		return nil
	}
	inst := &nativeInstance{log: log}
	mu.Lock()
	loggers[handleID(handle)] = inst
	mu.Unlock()
	setGlobalErr(nil)
	return handle
}

func createFromJSON(configJSON *C.char) (result unsafe.Pointer) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in logger_create: %v", r)
			fmt.Fprintf(os.Stderr, "[polyglot-logger] %v\n", err)
			setGlobalErr(err)
			result = nil
		}
	}()
	if testPanicCreate {
		panic("injected create panic")
	}
	cfg, err := core.ParseConfigJSON([]byte(goString(configJSON)))
	if err != nil {
		setGlobalErr(err)
		return nil
	}
	log, err := core.New(cfg)
	if err != nil {
		setGlobalErr(err)
		return nil
	}
	return registerLogger(log)
}

//export logger_version
func logger_version() *C.char {
	defer func() { _ = recover() }()
	versionOnce.Do(func() {
		versionCStr = C.CString(core.Version)
	})
	return versionCStr
}

//export logger_abi_version
func logger_abi_version() (rc C.int) {
	defer func() {
		if recover() != nil {
			rc = 0
		}
	}()
	return C.int(core.ABIVersion)
}

//export logger_create_v1
func logger_create_v1(configJSON *C.char) unsafe.Pointer {
	return createFromJSON(configJSON)
}

//export logger_create
func logger_create(configJSON *C.char) unsafe.Pointer {
	return createFromJSON(configJSON)
}

//export logger_log
func logger_log(handle unsafe.Pointer, level C.int, message *C.char, fieldsJSON *C.char) (rc C.int) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in logger_log: %v", r)
			fmt.Fprintf(os.Stderr, "[polyglot-logger] %v\n", err)
			setGlobalErr(err)
			rc = -1
		}
	}()
	inst := lookup(handle)
	if inst == nil {
		setGlobalErr(fmt.Errorf("invalid logger handle"))
		return -1
	}
	fields := goString(fieldsJSON)
	if err := inst.log.LogInt(int(level), goString(message), fields); err != nil {
		inst.setErr(err)
		return -1
	}
	inst.setErr(nil)
	return 0
}

//export logger_log_simple
func logger_log_simple(handle unsafe.Pointer, level C.int, message *C.char) C.int {
	return logger_log(handle, level, message, nil)
}

//export logger_with
func logger_with(handle unsafe.Pointer, fieldsJSON *C.char) (result unsafe.Pointer) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in logger_with: %v", r)
			fmt.Fprintf(os.Stderr, "[polyglot-logger] %v\n", err)
			setGlobalErr(err)
			result = nil
		}
	}()
	inst := lookup(handle)
	if inst == nil {
		setGlobalErr(fmt.Errorf("invalid logger handle"))
		return nil
	}
	fields, err := core.ParseFieldsJSON(goString(fieldsJSON))
	if err != nil {
		inst.setErr(err)
		setGlobalErr(err)
		return nil
	}
	child := inst.log.With(fields)
	handleOut := handles.acquire()
	if handleOut == nil {
		setGlobalErr(fmt.Errorf("out of memory allocating logger handle"))
		return nil
	}
	childInst := &nativeInstance{log: child}
	mu.Lock()
	loggers[handleID(handleOut)] = childInst
	mu.Unlock()
	inst.setErr(nil)
	setGlobalErr(nil)
	return handleOut
}

//export logger_set_fields
func logger_set_fields(handle unsafe.Pointer, fieldsJSON *C.char) (rc C.int) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in logger_set_fields: %v", r)
			fmt.Fprintf(os.Stderr, "[polyglot-logger] %v\n", err)
			setGlobalErr(err)
			rc = -1
		}
	}()
	inst := lookup(handle)
	if inst == nil {
		setGlobalErr(fmt.Errorf("invalid logger handle"))
		return -1
	}
	if err := inst.log.SetFieldsJSON(goString(fieldsJSON)); err != nil {
		inst.setErr(err)
		return -1
	}
	inst.setErr(nil)
	return 0
}

//export logger_reload_config
func logger_reload_config(handle unsafe.Pointer, configJSON *C.char) (rc C.int) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in logger_reload_config: %v", r)
			fmt.Fprintf(os.Stderr, "[polyglot-logger] %v\n", err)
			setGlobalErr(err)
			rc = -1
		}
	}()
	inst := lookup(handle)
	if inst == nil {
		setGlobalErr(fmt.Errorf("invalid logger handle"))
		return -1
	}
	cfg, err := core.ParseConfigJSON([]byte(goString(configJSON)))
	if err != nil {
		inst.setErr(err)
		return -1
	}
	if err := inst.log.ReloadConfig(cfg); err != nil {
		inst.setErr(err)
		return -1
	}
	inst.setErr(nil)
	return 0
}

//export logger_flush
func logger_flush(handle unsafe.Pointer) (rc C.int) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in logger_flush: %v", r)
			fmt.Fprintf(os.Stderr, "[polyglot-logger] %v\n", err)
			setGlobalErr(err)
			rc = -1
		}
	}()
	inst := lookup(handle)
	if inst == nil {
		setGlobalErr(fmt.Errorf("invalid logger handle"))
		return -1
	}
	if err := inst.log.Flush(); err != nil {
		inst.setErr(err)
		return -1
	}
	inst.setErr(nil)
	return 0
}

//export logger_close
func logger_close(handle unsafe.Pointer) (rc C.int) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in logger_close: %v", r)
			fmt.Fprintf(os.Stderr, "[polyglot-logger] %v\n", err)
			setGlobalErr(err)
			rc = -1
		}
	}()
	if handle == nil {
		setGlobalErr(fmt.Errorf("invalid logger handle"))
		return -1
	}
	id := handleID(handle)
	mu.Lock()
	inst := loggers[id]
	delete(loggers, id)
	mu.Unlock()
	if inst == nil {
		setGlobalErr(fmt.Errorf("invalid logger handle"))
		return -1
	}
	err := inst.log.Close()
	inst.errMu.Lock()
	if inst.errCStr != nil {
		C.free(unsafe.Pointer(inst.errCStr))
		inst.errCStr = nil
	}
	if inst.statsCStr != nil {
		C.free(unsafe.Pointer(inst.statsCStr))
		inst.statsCStr = nil
	}
	inst.errMu.Unlock()
	handles.release(handle)
	if err != nil {
		inst.setErr(err)
		setGlobalErr(err)
		return -1
	}
	inst.setErr(nil)
	return 0
}

//export logger_stats
func logger_stats(handle unsafe.Pointer) (out *C.char) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in logger_stats: %v", r)
			fmt.Fprintf(os.Stderr, "[polyglot-logger] %v\n", err)
			setGlobalErr(err)
			out = nil
		}
	}()
	inst := lookup(handle)
	if inst == nil {
		setGlobalErr(fmt.Errorf("invalid logger handle"))
		return nil
	}
	inst.errMu.Lock()
	defer inst.errMu.Unlock()
	if inst.statsCStr != nil {
		C.free(unsafe.Pointer(inst.statsCStr))
		inst.statsCStr = nil
	}
	inst.statsCStr = C.CString(inst.log.StatsJSON())
	return inst.statsCStr
}

//export logger_last_error
func logger_last_error(handle unsafe.Pointer) (out *C.char) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	if handle == nil {
		globalErrMu.Lock()
		defer globalErrMu.Unlock()
		if globalErrCStr != nil {
			C.free(unsafe.Pointer(globalErrCStr))
			globalErrCStr = nil
		}
		globalErrCStr = C.CString(globalLastErr)
		return globalErrCStr
	}
	inst := lookup(handle)
	if inst == nil {
		globalErrMu.Lock()
		defer globalErrMu.Unlock()
		if globalErrCStr != nil {
			C.free(unsafe.Pointer(globalErrCStr))
			globalErrCStr = nil
		}
		globalErrCStr = C.CString("invalid logger handle")
		return globalErrCStr
	}
	inst.errMu.Lock()
	defer inst.errMu.Unlock()
	if inst.errCStr != nil {
		C.free(unsafe.Pointer(inst.errCStr))
		inst.errCStr = nil
	}
	inst.errCStr = C.CString(inst.lastErr)
	return inst.errCStr
}

//export logger_free_string
func logger_free_string(s *C.char) {
	defer func() { _ = recover() }()
	// No-op: stats/error/version strings are owned by the library.
	_ = s
}

//export logger_create_from_config_file
func logger_create_from_config_file(configPath *C.char) (result unsafe.Pointer) {
	return logger_create_from_config_file_with_overrides(configPath, nil)
}

//export logger_create_from_config_file_with_overrides
func logger_create_from_config_file_with_overrides(configPath *C.char, overlayJSON *C.char) (result unsafe.Pointer) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in logger_create_from_config_file_with_overrides: %v", r)
			fmt.Fprintf(os.Stderr, "[polyglot-logger] %v\n", err)
			setGlobalErr(err)
			result = nil
		}
	}()

	path := goString(configPath)
	overlay := []byte(goString(overlayJSON))
	
	// Debug logging
	if path != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] logger_create_from_config_file_with_overrides called with explicit path: %q\n", path)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] logger_create_from_config_file_with_overrides called with no explicit path\n")
	}
	
	cfg, _, err := core.CreateConfigFromFileWithOverrides(path, overlay)
	if err != nil {
		setGlobalErr(err)
		return nil
	}

	log, err := core.New(cfg)
	if err != nil {
		setGlobalErr(err)
		return nil
	}
	return registerLogger(log)
}

func main() {}
