package compression

import "os/exec"

func lookPathOK(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}
