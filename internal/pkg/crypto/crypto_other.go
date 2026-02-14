//go:build !windows

package crypto

func getMachineGUID() string {
	return ""
}
