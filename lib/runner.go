package gin

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
	"sync"
)

type Runner interface {
	Run() (*exec.Cmd, error)
	Info() (os.FileInfo, error)
	SetWriter(io.Writer)
	Kill() error
}

type runner struct {
	bin       string
	args      []string
	writer    io.Writer
	command   *exec.Cmd
	killOnError bool
	starttime time.Time
	wasKilled bool
	sync.Mutex
}

func NewRunner(bin string, killOnError bool, args ...string) Runner {
	return &runner{
		bin:       bin,
		args:      args,
		writer:    ioutil.Discard,
		starttime: time.Now(),
		killOnError: killOnError,
	}
}

func (r *runner) Run() (*exec.Cmd, error) {
	if r.needsRefresh() {
		err := r.Kill()
		if err != nil {
			log.Print("Error killing process: ", err)
		}
	}

	if r.command == nil || r.Exited() {
		err := r.runBin()
		if err != nil {
			log.Print("Error running: ", err)
		}
		time.Sleep(250 * time.Millisecond)
		return r.command, err
	} else {
		log.Print("Not restarting, apparently still running")
		return r.command, nil
	}

}

func (r *runner) Info() (os.FileInfo, error) {
	return os.Stat(r.bin)
}

func (r *runner) SetWriter(writer io.Writer) {
	r.writer = writer
}

func (r *runner) Kill() error {
	if r.command != nil && r.command.Process != nil {
		r.Lock()
		r.wasKilled = true
		r.Unlock()
		done := make(chan error)
		go func() {
			r.command.Wait()
			close(done)
		}()

		//Trying a "soft" kill first
		if runtime.GOOS == "windows" {
			if err := r.command.Process.Kill(); err != nil {
				return err
			}
		} else if err := r.command.Process.Signal(os.Interrupt); err != nil {
			return err
		}

		//Wait for our process to die before we return or hard kill after 3 sec
		select {
		case <-time.After(3 * time.Second):
			if err := r.command.Process.Kill(); err != nil {
				log.Println("failed to kill: ", err)
			}
		case <-done:
		}
		r.command = nil
	}

	return nil
}

func (r *runner) Exited() bool {
	return r.command != nil && r.command.ProcessState != nil && r.command.ProcessState.Exited()
}

func (r *runner) runBin() error {
	r.command = exec.Command(r.bin, r.args...)
	r.command.Stdout = os.Stdout
	r.command.Stderr = os.Stderr

	err := r.command.Start()
	if err != nil {
		return err
	}

	r.starttime = time.Now()
	go func() {
		err := r.command.Wait()
		r.Lock()
		if err != nil {
			logger.Printf("Process execution failed: %s", err)
		}
		if !r.wasKilled && r.killOnError {
			logger.Println("Exiting, because kill-on-error is true")
			time.Sleep(time.Second * 5)
			os.Exit(1)
		}
		r.wasKilled = false
		r.Unlock()
	}()

	return nil
}

func (r *runner) needsRefresh() bool {
	info, err := r.Info()
	if err != nil {
		return false
	} else {
		return info.ModTime().After(r.starttime)
	}
}
