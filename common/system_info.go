package common

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/process"
)

// IsInContainer 检测是否在容器中运行
func IsInContainer() bool {
	// 方法1: 检查 /.dockerenv 文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// 方法2: 检查 /proc/1/cgroup 是否包含 docker/kubepods
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := string(data)
		if strings.Contains(content, "docker") || strings.Contains(content, "kubepods") {
			return true
		}
	}

	// 方法3: 检查环境变量
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	return false
}

// GetK8sContext 获取K8s上下文信息
func GetK8sContext() map[string]string {
	ctx := make(map[string]string)

	// 从环境变量获取 Pod 信息
	if podName := os.Getenv("HOSTNAME"); podName != "" {
		ctx["pod_name"] = podName
	}
	if namespace := os.Getenv("POD_NAMESPACE"); namespace != "" {
		ctx["namespace"] = namespace
	}
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		ctx["node_name"] = nodeName
	}
	if serviceAccount := os.Getenv("SERVICE_ACCOUNT"); serviceAccount != "" {
		ctx["service_account"] = serviceAccount
	}

	// 尝试从 kubectl 获取当前上下文
	if cmd := exec.Command("kubectl", "config", "current-context"); cmd.Err == nil {
		if output, err := cmd.Output(); err == nil {
			ctx["current_context"] = strings.TrimSpace(string(output))
		}
	}

	// 检查是否有 kubectl 访问权限
	if cmd := exec.Command("kubectl", "auth", "can-i", "get", "pods", "--all-namespaces"); cmd.Err == nil {
		if err := cmd.Run(); err == nil {
			ctx["cluster_admin"] = "true"
		} else {
			ctx["cluster_admin"] = "false"
		}
	}

	return ctx
}

func GetSystemInfo() (string, error) {
	info, err := host.Info()
	if err != nil {
		return "", err
	}

	sysInfo := fmt.Sprintf("%s %s", info.Platform, info.PlatformVersion)

	// 如果在容器中运行，添加容器标识
	if IsInContainer() {
		sysInfo += " (Container)"

		// 添加K8s信息
		k8sCtx := GetK8sContext()
		if podName, ok := k8sCtx["pod_name"]; ok {
			sysInfo += fmt.Sprintf(" [Pod: %s", podName)
			if namespace, ok := k8sCtx["namespace"]; ok {
				sysInfo += fmt.Sprintf(", Namespace: %s", namespace)
			}
			sysInfo += "]"
		}
	}

	return sysInfo, nil
}

// golang获取当前运行的shell平台，例如bash、sh、zsh、powershell
func GetShellPlatform() (string, error) {
	ppid := os.Getppid() // 获取父进程ID

	// 创建父进程对象
	p, err := process.NewProcess(int32(ppid))
	if err != nil {
		return "unknown", err
	}

	// 获取父进程的可执行文件路径
	exe, err := p.Exe()
	if err != nil {
		return "unknown", err
	}

	// 提取文件名并处理
	name := filepath.Base(exe)
	name = strings.TrimSuffix(name, ".exe") // 移除.exe扩展名（Windows）
	name = strings.ToLower(name)            // 统一小写

	// 根据文件名判断Shell类型
	switch name {
	case "bash":
		return "bash", nil
	case "zsh":
		return "zsh", nil
	case "sh":
		return "sh", nil
	case "cmd":
		return "cmd", nil
	case "powershell", "pwsh":
		return "powershell", nil
	default:
		return name, nil
	}
}

func GetUser() (string, error) {
	user, err := user.Current()
	if err != nil {
		return "", err
	}
	return user.Username, nil
}

func GetPwd() (string, error) {
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return pwd, nil
}
