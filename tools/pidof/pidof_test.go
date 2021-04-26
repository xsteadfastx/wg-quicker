// nolint:gochecknoglobals,paralleltest,goerr113,funlen
package pidof_test

import (
	"embed"
	"fmt"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.xsfx.dev/wg-quicker/tools/pidof"
)

//go:embed testdata/*
var testdata embed.FS

func TestPidof(t *testing.T) {
	assert := assert.New(t)

	tables := []struct {
		name string
		data string
		pid  int
		err  error
	}{
		{
			"/home/linuxbrew/.linuxbrew/bin/tmux new-session -t local",
			"psa_ubuntu_1.txt",
			20814,
			nil,
		},
		{
			"/root/.byteexec/wireguard-go w1nd50r",
			"psa_alpine_1.txt",
			2360,
			nil,
		},
		{
			".byteexec/wireguard-go w1nd50r",
			"psa_alpine_1.txt",
			2360,
			nil,
		},
		{
			"/root/.byteexec/wireguard-go w2nd50r",
			"psa_alpine_1.txt",
			0,
			fmt.Errorf("could not find pid"),
		},
		{
			"/root/.byteexec/wireguard-go w1nd50r",
			"psa_alpine_2.txt",
			2501,
			nil,
		},
		{
			"/root/.byteexec/wireguard-go w1nd50r",
			"psa_alpine_3.txt",
			0,
			pidof.ErrPIDNotFound,
		},
	}

	for _, table := range tables {
		log.Printf("testfile: %s", table.data)
		out, err := testdata.ReadFile("testdata/" + table.data)
		assert.NoError(err)

		mock := new(pidof.MockCommander)
		mock.On("Output", fmt.Sprintf("ps a|grep '%s'", table.name)).Return(out, table.err)
		pid, err := pidof.Pidof(table.name, mock)

		if table.err != nil {
			assert.Error(table.err, err)
		} else {
			assert.NoError(err)
		}

		assert.Equal(table.pid, pid)

		mock.AssertExpectations(t)
	}
}
