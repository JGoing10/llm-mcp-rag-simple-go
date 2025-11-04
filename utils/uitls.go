package utils

import (
	"fmt"
	"github.com/fatih/color"
	"strings"
)

// 打印带有彩色边框的格式化标题
func LogTitle(message string) {
	const totalLength = 80 //// 总长度固定为80字符
	messageLength := len(message)
	padding := totalLength - messageLength - 4 //计算填充长度
	if padding < 0 {
		padding = 0
	}
	leftPadding := padding / 2
	rightPadding := padding - leftPadding
	paddedMessage := fmt.Sprintf("%s %s %s", strings.Repeat("=", leftPadding), message, strings.Repeat("=", rightPadding))
	color.New(color.FgCyan, color.Bold).Println(paddedMessage)
}

//打印信息级别日志消息

func LogInfo(message string) {
	color.New(color.FgGreen).Printf("[INFO]%s\n", message)
}

// 警告级别日志
func LogWarn(message string) {
	color.New(color.FgYellow).Printf("[WARN]%s\n", message)
}

// 错误级别日志
func LogError(message string) {
	color.New(color.FgRed).Printf("[ERROR]%s\n", message)
}

// 调试级别日志
func LogDebug(message string) {
	color.New(color.FgMagenta).Printf("[DEBUG]%s\n", message)
}

// 截断字符串到指定长度并添加省略号
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s // 字符串长度未超过限制，直接返回
	}
	if maxLen <= 3 {
		return s[:maxLen] // 最大长度太小，直接截断
	}
	return s[:maxLen-3] + "..." // 截断并添加省略号
}
