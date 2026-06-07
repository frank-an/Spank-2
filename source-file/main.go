package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	"github.com/taigrr/apple-silicon-accelerometer/detector"
	"github.com/taigrr/apple-silicon-accelerometer/sensor"
	"github.com/taigrr/apple-silicon-accelerometer/shm"
)

var version = "dev"

type sensitivity float64

const (
	sensHigh sensitivity = 0.015
	sensMid  sensitivity = 0.05
	sensLow  sensitivity = 0.12
)

func parseSensitivity(s string) (sensitivity, error) {
	switch strings.ToLower(s) {
	case "high":
		return sensHigh, nil
	case "mid":
		return sensMid, nil
	case "low":
		return sensLow, nil
	default:
		return sensMid, fmt.Errorf("invalid sensitivity: %q (must be high/mid/low)", s)
	}
}

var (
	keyName    string
	mouseBtn   int
	wordMode   bool
	sensStr    string
	cooldownMs int
	cmdStr     string
)

type slapTracker struct {
	mu       sync.Mutex
	count    int
	lastTime time.Time
}

func newSlapTracker() *slapTracker {
	return &slapTracker{}
}

func (st *slapTracker) record(now time.Time) int {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.count++
	st.lastTime = now
	return st.count
}

// keyCodeMap maps key names to macOS CGKeyCode values.
var keyCodeMap = map[string]int{
	"enter":     0x24,
	"return":    0x24,
	"space":     0x31,
	"escape":    0x35,
	"esc":       0x35,
	"tab":       0x30,
	"backspace": 0x33,
	"delete":    0x33,
	"up":        0x7E,
	"down":      0x7D,
	"left":      0x7B,
	"right":     0x7C,
	"home":      0x73,
	"end":       0x77,
	"pageup":    0x74,
	"pagedown":  0x79,
}

// charToKeyCode returns the CGKeyCode and whether shift is needed for a character.
func charToKeyCode(ch rune) (keyCode int, shift bool) {
	switch {
	case ch >= 'a' && ch <= 'z':
		return int(ch - 'a'), false
	case ch >= 'A' && ch <= 'Z':
		return int(ch - 'A'), true
	case ch >= '0' && ch <= '9':
		nums := []int{0x1D, 0x12, 0x13, 0x14, 0x15, 0x17, 0x16, 0x1A, 0x1C, 0x19}
		return nums[ch-'0'], false
	default:
		symMap := map[rune]struct {
			kc    int
			shift bool
		}{
			'-': {0x1B, false}, '=': {0x18, false},
			'[': {0x21, false}, ']': {0x1E, false},
			'\\': {0x2A, false},
			';': {0x29, false}, '\'': {0x27, false},
			',': {0x2B, false}, '.': {0x2F, false}, '/': {0x2C, false},
			'`': {0x32, false},
			'~': {0x32, true}, '!': {0x12, true},
			'@': {0x13, true}, '#': {0x14, true},
			'$': {0x15, true}, '%': {0x17, true},
			'^': {0x16, true}, '&': {0x1A, true},
			'*': {0x1C, true}, '(': {0x19, true},
			')': {0x1D, true}, '_': {0x1B, true},
			'+': {0x18, true}, '{': {0x21, true},
			'}': {0x1E, true}, '|': {0x2A, true},
			':': {0x29, true}, '"': {0x27, true},
			'<': {0x2B, true}, '>': {0x2F, true},
			'?': {0x2C, true},
		}
		if s, ok := symMap[ch]; ok {
			return s.kc, s.shift
		}
		return 0, false
	}
}

// processEscapes handles escape sequences: \n, \t, \\, and r"..." raw strings.
func processEscapes(s string) string {
	if strings.HasPrefix(s, `r"`) && strings.HasSuffix(s, `"`) {
		return s[2 : len(s)-1]
	}
	s = strings.ReplaceAll(s, `\\`, "\x00")
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	s = strings.ReplaceAll(s, "\x00", `\`)
	return s
}

// copyAndPaste copies text to clipboard and simulates Cmd+V to paste.
func copyAndPaste(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pbcopy: %w", err)
	}
	time.Sleep(50 * time.Millisecond)
	pressCmdV()
	return nil
}

// shouldTrigger checks if we should trigger based on amplitude and cooldown.
func shouldTrigger(amplitude float64, lastTrigger time.Time, minAmp float64, cd time.Duration) bool {
	if amplitude < minAmp {
		return false
	}
	if time.Since(lastTrigger) < cd {
		return false
	}
	return true
}

var (
	sensorReady = make(chan struct{})
	sensorErr   = make(chan error, 1)
)

func run(cmd *cobra.Command, args []string) error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("spank 需要 root 权限，请使用: sudo spank")
	}

	sens, err := parseSensitivity(sensStr)
	if err != nil {
		return err
	}
	minAmp := float64(sens)
	cd := time.Duration(cooldownMs) * time.Millisecond

	// Get original user for running commands (spank runs as root via sudo)
	runUser := os.Getenv("SUDO_USER")

	// Validate key (skip when --mouse or --command is active)
	if cmdStr == "" && mouseBtn < 0 && keyName != "" && !wordMode {
		if _, ok := keyCodeMap[strings.ToLower(keyName)]; !ok {
			if len(keyName) != 1 || (len(keyName) == 1 && !unicode.IsPrint(rune(keyName[0]))) {
				return fmt.Errorf("unknown key: %s", keyName)
			}
		}
	}

	// Segment mode
	var segments []string
	var segIdx int
	segSrc := ""
	if wordMode && strings.HasPrefix(keyName, "s=") {
		segSrc = keyName
	} else if cmdStr != "" && strings.HasPrefix(cmdStr, "s=") {
		segSrc = cmdStr
	}
	if segSrc != "" {
		raw := segSrc[2:]
		if len(raw) > 0 {
			delim := string(raw[0])
			rest := raw[1:]
			segments = strings.Split(rest, delim)
			if len(segments) <= 1 {
				segments = nil
			}
		}
	}

	ctx, cancel := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create shared memory for accelerometer data
	accelRing, err := shm.CreateRing(shm.NameAccel)
	if err != nil {
		return fmt.Errorf("creating accel shm: %w", err)
	}
	defer accelRing.Close()
	defer accelRing.Unlink()

	// Start sensor worker
	go func() {
		close(sensorReady)
		if err := sensor.Run(sensor.Config{
			AccelRing: accelRing,
			Restarts:  0,
		}); err != nil {
			sensorErr <- err
		}
	}()

	select {
	case <-sensorReady:
	case err := <-sensorErr:
		return fmt.Errorf("sensor worker failed: %w", err)
	case <-ctx.Done():
		return nil
	}

	time.Sleep(100 * time.Millisecond)

	// Determine what to do on trigger.
	// Each trigger function returns a description of what was output.
	var trigger func() string
	if cmdStr != "" {
		userHome := "/Users/" + runUser
		runCmd := func(c string) {
			cmd := exec.Command("zsh", "-c", "source ~/.zshrc; "+c)
			cmd.Env = append(os.Environ(), "HOME="+userHome)
			cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			cmd.Run()
		}
		if segments != nil {
			trigger = func() string {
				seg := segments[segIdx%len(segments)]
				segIdx++
				runCmd(seg)
				return "cmd=" + seg
			}
		} else {
			trigger = func() string {
				runCmd(cmdStr)
				return "cmd=" + cmdStr
			}
		}
	} else if mouseBtn >= 0 {
		btn := mouseBtn
		trigger = func() string {
			clickMouse(btn)
			return fmt.Sprintf("mouse=%d", btn)
		}
	} else if wordMode {
		if segments != nil {
			trigger = func() string {
				seg := segments[segIdx%len(segments)]
				segIdx++
				seg = processEscapes(seg)
				if len(seg) == 1 {
					kc, shift := charToKeyCode(rune(seg[0]))
					if shift {
						pressKeyShifted(kc)
					} else {
						pressKey(kc)
					}
				} else {
					if err := copyAndPaste(seg); err != nil {
						fmt.Fprintf(os.Stderr, "spank: paste error: %v\n", err)
					}
				}
				return "key=" + seg
			}
		} else if len(keyName) == 1 {
			rawKey := keyName
			trigger = func() string {
				kc, shift := charToKeyCode(rune(rawKey[0]))
				if shift {
					pressKeyShifted(kc)
				} else {
					pressKey(kc)
				}
				return "key=" + rawKey
			}
		} else {
			text := processEscapes(keyName)
			trigger = func() string {
				if err := copyAndPaste(text); err != nil {
					fmt.Fprintf(os.Stderr, "spank: paste error: %v\n", err)
				}
				return "key=" + text
			}
		}
	} else {
		// Direct key press mode
		key := strings.ToLower(keyName)
		if kc, ok := keyCodeMap[key]; ok {
			trigger = func() string {
				pressKey(kc)
				return "key=" + keyName
			}
		} else if len(keyName) == 1 {
			ch := rune(keyName[0])
			if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
				kc, shift := charToKeyCode(ch)
				trigger = func() string {
					if shift {
						pressKeyShifted(kc)
					} else {
						pressKey(kc)
					}
					return "key=" + keyName
				}
			} else {
				return fmt.Errorf("unknown key: %s", keyName)
			}
		} else {
			return fmt.Errorf("unknown key: %s", keyName)
		}
	}

	tracker := newSlapTracker()
	det := detector.New()
	var lastAccelTotal uint64
	var lastTrigger time.Time

	// Startup log shows the raw configuration
	var startupDesc string
	if cmdStr != "" {
		startupDesc = "cmd=" + cmdStr
	} else if mouseBtn >= 0 {
		startupDesc = fmt.Sprintf("mouse=%d", mouseBtn)
	} else {
		startupDesc = "key=" + keyName
	}
	fmt.Printf("spank: 监听中 → %s [灵敏度: %s] (Ctrl+C 退出)\n", startupDesc, sensStr)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nbye!")
			return nil
		case err := <-sensorErr:
			return fmt.Errorf("sensor error: %w", err)
		case <-ticker.C:
		}

		now := time.Now()
		tNow := float64(now.UnixNano()) / 1e9

		samples, newTotal := accelRing.ReadNew(lastAccelTotal, shm.AccelScale)
		lastAccelTotal = newTotal
		if len(samples) > 200 {
			samples = samples[len(samples)-200:]
		}

		nSamples := len(samples)
		for idx, sample := range samples {
			tSample := tNow - float64(nSamples-idx-1)/float64(det.FS)
			det.Process(sample.X, sample.Y, sample.Z, tSample)
		}

		if len(det.Events) == 0 {
			continue
		}

		ev := det.Events[len(det.Events)-1]

		if !shouldTrigger(ev.Amplitude, lastTrigger, minAmp, cd) {
			continue
		}

		lastTrigger = now
		num := tracker.record(now)
		desc := trigger()
		fmt.Printf("slap #%d [%s] → %s\n", num, ev.Severity, desc)
	}
}

func main() {
	cmd := &cobra.Command{
		Use:   "spank",
		Short: "拍打 MacBook 时自动模拟按键或输入文字",
		Long: `spank 利用 Apple Silicon 加速计检测敲击/拍打,
通过 CoreGraphics 模拟键盘事件或鼠标点击。

需要 sudo 权限 (加速计需要 IOKit 直接访问)。`,
		Version:           version,
		RunE:              run,
		SilenceUsage:      true,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}

	cmd.Flags().StringVarP(&keyName, "key", "k", "enter", "按下的按键 (space/enter/escape/tab/... 或单个字符)")
	cmd.Flags().IntVarP(&mouseBtn, "mouse", "m", -1, "模拟鼠标点击 (0=左键, 1=右键)")
	cmd.Flags().BoolVarP(&wordMode, "word", "w", false, "文本模式: 多字符复制粘贴, 单字符直接按键, s=分段输出")
	cmd.Flags().StringVarP(&sensStr, "sensitivity", "v", "mid", "灵敏度: high/mid/low")
	cmd.Flags().IntVar(&cooldownMs, "cooldown", 500, "触发冷却时间 (毫秒)")
	cmd.Flags().StringVarP(&cmdStr, "command", "c", "", "拍打时执行的终端命令 (如 'open ~/.zshrc')")

	cmd.Flags().BoolP("version", "V", false, "显示版本信息")

	cmd.SetVersionTemplate("spank version {{.Version}}\n")

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "spank: %v\n", err)
		os.Exit(1)
	}
}
