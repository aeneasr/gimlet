package cmd

import (
	"errors"
	"fmt"
	"github.com/arekkas/gimlet/lib"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

var (
	startTime  = time.Now()
	logger     = log.New(os.Stdout, "[gin] ", 0)
	immediate  = false
	buildError error
)

func build(interval int, builder gin.Builder, runner gin.Runner, logger *log.Logger, killOnErr bool) {
	err := builder.Build()
	if err != nil {
		buildError = err
		logger.Print("ERROR! Build failed.\n")
		fmt.Println(builder.Errors())
		if killOnErr {
			logger.Println("Exiting, because kill-on-error is true")
			os.Exit(1)
		}
	} else {
		logger.Print("Build Successful\n")
		buildError = nil
		if immediate {
			_, err := runner.Run()
			if err != nil {
				logger.Printf("An error occurred %s\n", err)
				if killOnErr {
					logger.Println("Exiting, because kill-on-error is true")
					os.Exit(1)
				}
			}
		}
	}

	time.Sleep(time.Duration(interval) * time.Millisecond)
}

type scanCallback func(path string)

func scanChanges(interval int, watchPath string, excludeDirs []string, cb scanCallback) {
	for {
		filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
			if path == ".git" {
				return filepath.SkipDir
			}
			for _, x := range excludeDirs {
				if x == path {
					return filepath.SkipDir
				}
			}
			logger.Printf("Path %s changed", path)

			// ignore hidden files
			if filepath.Base(path)[0] == '.' {
				return nil
			}

			if filepath.Ext(path) == ".go" && info.ModTime().After(startTime) {
				cb(path)
				startTime = time.Now()
				return errors.New("done")
			}

			return nil
		})
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

func shutdown(runner gin.Runner) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-c
		log.Println("Got signal: ", s)
		err := runner.Kill()
		if err != nil {
			log.Print("Error killing: ", err)
		}
		os.Exit(1)
	}()
}
