package monitor

import (
	"fmt"
	"net/url"
	"strings"
)

// nodeBrief extracts a credential-free "scheme@host:port" summary from a node
// URI for log lines. Credentials in the URI are dropped. Returns "" when the
// URI cannot be parsed (e.g. base64-encoded vmess links).
func nodeBrief(uri string) string {
	if uri == "" {
		return ""
	}
	u, err := url.Parse(uri)
	if err != nil || u.Scheme == "" {
		return ""
	}
	host := u.Hostname()
	if host == "" {
		return ""
	}
	if port := u.Port(); port != "" {
		return fmt.Sprintf("%s@%s:%s", u.Scheme, host, port)
	}
	return fmt.Sprintf("%s@%s", u.Scheme, host)
}

// classifyProbeError maps a raw probe error to a stable category code and a
// human-readable summary, so log readers can tell *why* a node failed without
// parsing opaque sing-box / net error strings.
//
// Cases are matched top-down; more specific signatures must come before generic
// ones (probe cancellation before timeouts, transport handshake before a bare
// status code, dial-stage failures before read-stage timeouts).
func classifyProbeError(err error) (category, summary string) {
	if err == nil {
		return "", ""
	}
	s := err.Error()
	switch {
	// Probe was cancelled (overall batch deadline or client disconnect), not a
	// node fault. Checked first so it isn't misread as a slow-link read timeout.
	case strings.Contains(s, "context deadline exceeded") || strings.Contains(s, "context canceled"):
		return "cancelled", "探测被取消(批量超时或连接中断,非节点故障)"

	// Placeholder address left by broken subscription parsing. Only the sentinel
	// 127.127.127.1 is treated as a parse artifact; a genuine 127.0.0.1 dial
	// error falls through to the dial-stage bucket below for an accurate cause.
	case strings.Contains(s, "127.127.127.1"):
		return "addr_invalid", "节点地址无效(疑似订阅解析异常或占位地址)"

	// Transport-layer handshake failures from sing-box outbounds: the server or
	// a CDN in front of it rejected the websocket / http-upgrade / v2ray upgrade.
	case strings.Contains(s, "http-upgrade") || strings.Contains(s, "httpupgrade") ||
		strings.Contains(s, "unexpected status") || strings.Contains(s, "unexpected HTTP") ||
		strings.Contains(s, "v2ray-"):
		return "transport_handshake", "传输层握手失败(节点服务端异常或被CDN拦截)"

	// TLS handshake / certificate verification failures (strict probe mode).
	case strings.Contains(s, "tls:") || strings.Contains(s, "handshake") || strings.Contains(s, "certificate"):
		return "tls_failed", "TLS握手失败(证书无效或疑似劫持)"

	// Response byte stream is not the expected protocol frame.
	case strings.Contains(s, "unknown version") || strings.Contains(s, "malformed"):
		return "proto_mismatch", "协议不匹配(响应非预期,疑似TLS/HTTP混淆或劫持)"

	// TCP dial-stage failures: the node host itself is unreachable.
	case strings.Contains(s, "dial tcp") || strings.Contains(s, "dial udp"):
		switch {
		case strings.Contains(s, "i/o timeout") || strings.Contains(s, "timeout"):
			return "dial_timeout", "节点不可达(TCP连接超时,可能被封或已下线)"
		case strings.Contains(s, "connection refused"):
			return "dial_refused", "端口未开放(连接被拒绝)"
		case strings.Contains(s, "no route to host"):
			return "dial_no_route", "网络不可达(无路由到主机)"
		case strings.Contains(s, "connection reset"):
			return "dial_reset", "TCP连接被重置(疑似网络封锁)"
		default:
			return "dial_failed", "TCP连接失败(节点不可达)"
		}

	// Read-stage timeouts: connected through the node, but the probe response
	// did not arrive in time (slow / unstable exit link).
	case strings.Contains(s, "i/o timeout") || strings.Contains(s, "deadline exceeded"):
		return "read_timeout", "节点响应超时(链路过慢或不稳定)"

	// Connection torn down mid-probe.
	case strings.Contains(s, "EOF") || strings.Contains(s, "reset by peer") ||
		strings.Contains(s, "broken pipe") || strings.Contains(s, "connection reset"):
		return "conn_reset", "连接被对端中断(EOF,节点异常或疑似劫持)"

	default:
		return "other", "探测失败(未知原因)"
	}
}

// FormatProbeFailure renders a single-line, human-readable probe failure that
// identifies the node (tag + protocol@host:port) and classifies the error.
// Example:
//
//	"example [vless@1.2.3.4:443] (dial_timeout) 节点不可达(TCP连接超时,可能被封或已下线) | dial tcp 1.2.3.4:443: i/o timeout"
func FormatProbeFailure(tag, uri string, err error) string {
	brief := nodeBrief(uri)
	category, summary := classifyProbeError(err)
	var b strings.Builder
	b.WriteString(tag)
	if brief != "" {
		b.WriteString(" [")
		b.WriteString(brief)
		b.WriteString("]")
	}
	b.WriteString(" (")
	b.WriteString(category)
	b.WriteString(") ")
	b.WriteString(summary)
	b.WriteString(" | ")
	b.WriteString(err.Error())
	return b.String()
}
