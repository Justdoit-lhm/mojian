//go:build windows

package session

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                              = syscall.NewLazyDLL("kernel32.dll")
	procCreatePseudoConsole               = kernel32.NewProc("CreatePseudoConsole")
	procResizePseudoConsole               = kernel32.NewProc("ResizePseudoConsole")
	procClosePseudoConsole                = kernel32.NewProc("ClosePseudoConsole")
	procInitializeProcThreadAttributeList = kernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute         = kernel32.NewProc("UpdateProcThreadAttribute")
	procDeleteProcThreadAttributeList     = kernel32.NewProc("DeleteProcThreadAttributeList")
)

const (
	procThreadAttributePseudoConsole = 0x00020016
	extendedStartupInfoPresent       = 0x00080000
	createUnicodeEnvironment         = 0x00000400
)

type startupInfoEx struct {
	syscall.StartupInfo
	attributes uintptr
}

type winPTY struct {
	mu      sync.Mutex
	console uintptr
	in      *os.File
	out     *os.File
	process syscall.Handle
	pid     int
	job     windows.Handle
}

func startPTY(argv []string, dir string, env []string, cols, rows int) (ptyProcess, error) {
	if cols <= 0 || rows <= 0 {
		cols, rows = 80, 24
	}
	p := &winPTY{}
	if err := p.start(argv, dir, env, cols, rows); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *winPTY) start(argv []string, dir string, env []string, cols, rows int) error {
	if err := procCreatePseudoConsole.Find(); err != nil {
		return errors.New("session: ConPTY needs Windows 10 1809 or later")
	}
	var inR, inW, outR, outW syscall.Handle
	if err := syscall.CreatePipe(&inR, &inW, nil, 0); err != nil {
		return fmt.Errorf("session: %w", err)
	}
	if err := syscall.CreatePipe(&outR, &outW, nil, 0); err != nil {
		syscall.CloseHandle(inR)
		syscall.CloseHandle(inW)
		return fmt.Errorf("session: %w", err)
	}
	var console uintptr
	r, _, _ := procCreatePseudoConsole.Call(coord(cols, rows), uintptr(inR), uintptr(outW), 0, uintptr(unsafe.Pointer(&console)))
	syscall.CloseHandle(inR)
	syscall.CloseHandle(outW)
	if r != 0 {
		syscall.CloseHandle(inW)
		syscall.CloseHandle(outR)
		return fmt.Errorf("session: CreatePseudoConsole: %w", syscall.Errno(r&0xFFFF))
	}
	p.console = console
	p.in = os.NewFile(uintptr(inW), "conpty-in")
	p.out = os.NewFile(uintptr(outR), "conpty-out")
	if err := p.spawn(argv, dir, env); err != nil {
		_ = p.Close()
		if p.out != nil {
			_ = p.out.Close()
		}
		return err
	}
	p.assignJob()
	return nil
}

func (p *winPTY) spawn(argv []string, dir string, env []string) error {
	var size uintptr
	procInitializeProcThreadAttributeList.Call(0, 1, 0, uintptr(unsafe.Pointer(&size)))
	if size == 0 {
		return errors.New("session: InitializeProcThreadAttributeList returned no size")
	}
	list := make([]byte, size)
	if r, _, err := procInitializeProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&list[0])), 1, 0, uintptr(unsafe.Pointer(&size))); r == 0 {
		return fmt.Errorf("session: InitializeProcThreadAttributeList: %w", err)
	}
	defer procDeleteProcThreadAttributeList.Call(uintptr(unsafe.Pointer(&list[0])))
	if r, _, err := procUpdateProcThreadAttribute.Call(uintptr(unsafe.Pointer(&list[0])), 0, procThreadAttributePseudoConsole, p.console, unsafe.Sizeof(p.console), 0, 0); r == 0 {
		return fmt.Errorf("session: UpdateProcThreadAttribute: %w", err)
	}
	si := startupInfoEx{attributes: uintptr(unsafe.Pointer(&list[0]))}
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = syscall.STARTF_USESTDHANDLES
	si.StdInput, si.StdOutput, si.StdErr = syscall.InvalidHandle, syscall.InvalidHandle, syscall.InvalidHandle

	cmdline, err := syscall.UTF16PtrFromString(commandLine(argv[0], argv))
	if err != nil {
		return err
	}
	var cwd *uint16
	if dir != "" {
		if cwd, err = syscall.UTF16PtrFromString(dir); err != nil {
			return err
		}
	}
	var envp *uint16
	if env != nil {
		block := environmentBlock(env)
		envp = &block[0]
	}
	var pi syscall.ProcessInformation
	err = syscall.CreateProcess(nil, cmdline, nil, nil, false, extendedStartupInfoPresent|createUnicodeEnvironment, envp, cwd, &si.StartupInfo, &pi)
	if err != nil {
		return fmt.Errorf("session: starting %s: %w", argv[0], err)
	}
	syscall.CloseHandle(pi.Thread)
	p.process, p.pid = pi.Process, int(pi.ProcessId)
	return nil
}

func (p *winPTY) assignJob() {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		windows.CloseHandle(job)
		return
	}
	if err := windows.AssignProcessToJobObject(job, windows.Handle(p.process)); err != nil {
		windows.CloseHandle(job)
		return
	}
	p.job = job
}

func (p *winPTY) wait() (int, error) {
	if _, err := syscall.WaitForSingleObject(p.process, syscall.INFINITE); err != nil {
		syscall.CloseHandle(p.process)
		p.closeJob()
		return 0, err
	}
	var code uint32
	err := syscall.GetExitCodeProcess(p.process, &code)
	syscall.CloseHandle(p.process)
	p.closeJob()
	return int(code), err
}

func (p *winPTY) closeJob() {
	p.mu.Lock()
	job := p.job
	p.job = 0
	p.mu.Unlock()
	if job != 0 {
		windows.CloseHandle(job)
	}
}

func (p *winPTY) Read(b []byte) (int, error)  { return p.out.Read(b) }
func (p *winPTY) Write(b []byte) (int, error) { return p.in.Write(b) }

func (p *winPTY) Resize(cols, rows int) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.console == 0 {
		return errors.New("session: console closed")
	}
	if r, _, _ := procResizePseudoConsole.Call(p.console, coord(cols, rows)); r != 0 {
		return fmt.Errorf("session: ResizePseudoConsole: %w", syscall.Errno(r&0xFFFF))
	}
	return nil
}

func (p *winPTY) Pid() int { return p.pid }

func (p *winPTY) Wait() (int, error) { return p.wait() }

func (p *winPTY) Interrupt() error {
	_, err := p.in.Write([]byte{3})
	return err
}

func (p *winPTY) Kill() error {
	p.mu.Lock()
	job := p.job
	p.job = 0
	p.mu.Unlock()
	if job != 0 {
		err := windows.TerminateJobObject(job, 1)
		windows.CloseHandle(job)
		return err
	}
	return syscall.TerminateProcess(p.process, 1)
}

func (p *winPTY) Close() error {
	p.mu.Lock()
	console := p.console
	p.console = 0
	p.mu.Unlock()
	if console == 0 {
		return nil
	}
	if p.in != nil {
		_ = p.in.Close()
	}
	go func() {
		procClosePseudoConsole.Call(console)
		if p.out != nil {
			_ = p.out.Close()
		}
	}()
	return nil
}

func coord(cols, rows int) uintptr {
	return uintptr(uint16(max(1, min(cols, 0x7FFF)))) | uintptr(uint16(max(1, min(rows, 0x7FFF))))<<16
}

func commandLine(path string, args []string) string {
	parts := []string{syscall.EscapeArg(path)}
	if len(args) > 1 {
		for _, a := range args[1:] {
			parts = append(parts, syscall.EscapeArg(a))
		}
	}
	return strings.Join(parts, " ")
}

func environmentBlock(env []string) []uint16 {
	var b []uint16
	for _, kv := range env {
		b = append(b, utf16.Encode([]rune(kv))...)
		b = append(b, 0)
	}
	if len(b) == 0 {
		b = append(b, 0)
	}
	return append(b, 0)
}
