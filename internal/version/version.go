package version

import (
	"runtime/debug"
	"strings"
)

// Info はバージョン情報を保持する構造体
type Info struct {
	Version string
	Commit  string
	Date    string
}

// DisplayString はバージョン情報を表示用文字列にフォーマットする。
// commit/dateがデフォルト値の場合は括弧ごと省略する。
func (i Info) DisplayString() string {
	var parts []string
	if i.Commit != "none" {
		parts = append(parts, "commit: "+i.Commit)
	}
	if i.Date != "unknown" {
		parts = append(parts, "built: "+i.Date)
	}
	if len(parts) == 0 {
		return i.Version
	}
	return i.Version + " (" + strings.Join(parts, ", ") + ")"
}

// テスト用に差し替え可能
var readBuildInfo = debug.ReadBuildInfo

// Resolve はldflagsの値を優先しつつ、未設定時はruntime/debug.ReadBuildInfoでフォールバックする。
func Resolve(ldVersion, ldCommit, ldDate string) Info {
	if ldVersion != "dev" {
		return Info{Version: ldVersion, Commit: ldCommit, Date: ldDate}
	}

	info := Info{Version: ldVersion, Commit: ldCommit, Date: ldDate}

	bi, ok := readBuildInfo()
	if !ok {
		return info
	}

	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		info.Version = bi.Main.Version
	}
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			info.Commit = s.Value
		case "vcs.time":
			info.Date = s.Value
		}
	}

	return info
}
