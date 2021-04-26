package pidof

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

//go:generate mockery --name Commander --inpackage --output .

var ErrPIDNotFound = fmt.Errorf("could not find pid")

type Commander interface {
	Output(cmd string) ([]byte, error)
}

type Cdr struct{}

func (c Cdr) Output(cmd string) ([]byte, error) {
	cm := exec.Command("/bin/sh", "-c", cmd)

	log.WithField("cmd", cm.String()).Debug("command to run")

	// nolint:wrapcheck
	return cm.Output()
}

func Pidof(name string, cdr Commander) (int, error) {
	p, err := cdr.Output(fmt.Sprintf("ps a|grep '%s'", name))
	if err != nil {
		log.WithField("output", p).Debug(p)

		return 0, fmt.Errorf("could not execute ps command: %w", err)
	}

	for _, l := range strings.Split(string(p), "\n") {
		if strings.Contains(l, name) && !strings.Contains(l, "grep") {
			// nolint:gocritic
			rePID, err := regexp.Compile(`(\d+)\s.+`)
			if err != nil {
				return 0, fmt.Errorf("could not compile re: %w", err)
			}

			matches := rePID.FindStringSubmatch(l)
			// nolint: gomnd
			if len(matches) == 2 {
				pid, err := strconv.Atoi(matches[1])
				if err != nil {
					return 0, fmt.Errorf("could not convert string to int: %w", err)
				}

				return pid, nil
			}

			break
		}
	}

	return 0, ErrPIDNotFound
}

func Pkill(name string) error {
	c := Cdr{}

	pid, err := Pidof(name, c)
	if err != nil {
		if errors.Is(err, ErrPIDNotFound) {
			log.Debug("looks like the process is not running")

			return nil
		}

		return fmt.Errorf("could not get pid: %w", err)
	}

	if err := syscall.Kill(pid, 9); err != nil {
		return fmt.Errorf("could not kill process: %w", err)
	}

	return nil
}
