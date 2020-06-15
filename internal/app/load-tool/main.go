package main

import (
	"fmt"
	"io"
	"math/rand"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type LoadTarget struct {
	Client        *ssh.Client
	Session       *ssh.Session
	in            chan<- string
	out           <-chan string
	commandsCount int
	elapsedms     int
}

var targets chan *LoadTarget = make(chan *LoadTarget, NUM_CLIENT)

const NUM_CLIENT = 3000
const PARALLEL_RUN = 50

var CommandElapsedms int = 0
var CommandCount int = 0

func main() {
	var wg sync.WaitGroup
	for i := 0; i < NUM_CLIENT/PARALLEL_RUN; i++ {
		for i := 1; i <= PARALLEL_RUN; i++ {
			wg.Add(1)
			go connectToRandomHost(&targets, &wg)
		}
		wg.Wait()
	}
	_ = 1

	// Perform Load
	commands := []string{"ls", "pwd", "whoami", "netstat -anp", "ps -ef"}

	startTime := time.Now()
	var amount_commands int = 20
	for {
		command := commands[rand.Intn(len(commands))]
		for i := 0; i < amount_commands; i++ {
			go SendCommandToTarget(&targets, command)
		}
		CommandElapsedms += int(time.Since(startTime).Milliseconds())
		CommandCount += amount_commands
		time.Sleep(1 * time.Second)
		rate := float64(CommandCount) / float64(time.Since(startTime).Seconds())
		fmt.Printf("Invoked: %f, Commands per sec %f, Avg latency ms: %f\n", CommandCount,
			rate,
			CommandElapsedms/CommandCount)
	}

}
func connectToHost(target *LoadTarget, user string, host string) (*ssh.Client, *ssh.Session, error) {
	var pass string
	pass = "gil"

	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.Password(pass)},
	}
	sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	client, err := ssh.Dial("tcp", host, sshConfig)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, nil, err
	}

	if err != nil {
		return nil, nil, err
	}
	// defer session.Close()
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return nil, nil, err
	}

	w, err := session.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	r, err := session.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	in, out := MuxShell(w, r)
	if err := session.Start("/bin/sh"); err != nil {
		return nil, nil, err
	}
	<-out //ignore the shell output
	target.in = in
	target.out = out
	return client, session, nil
}

func SendCommandToTarget(targets *chan *LoadTarget, command string) {

	t := <-*targets
	t.in <- command
	_ = <-t.out
	// fmt.Printf("time: %s, command: %s,  output: %s\n", time.Now(), command, output)

	*targets <- t

}

func MuxShell(w io.Writer, r io.Reader) (chan<- string, <-chan string) {
	in := make(chan string, 1)
	out := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1) //for the shell itself
	go func() {
		for cmd := range in {
			wg.Add(1)
			w.Write([]byte(cmd + "\n"))
			wg.Wait()
		}
	}()
	go func() {
		var (
			buf [65 * 1024]byte
			t   int
		)
		for {
			n, err := r.Read(buf[t:])
			if err != nil {
				close(in)
				close(out)
				return
			}
			t += n
			if buf[t-2] == '$' { //assuming the $PS1 == 'sh-4.3$ '
				out <- string(buf[:t])
				t = 0
				wg.Done()
			}
		}
	}()
	return in, out
}

func connectToRandomHost(targets *chan *LoadTarget, wg *sync.WaitGroup) {

	defer wg.Done()

	user_template := "gadda@ec2-user@aws#instance:ssh_port"
	// user_template := "gadda@ec2-user@instance:ssh_port"

	port := rand.Intn(2046-2022) + 2022
	port_str := fmt.Sprintf("%d", port)
	_ = port_str
	// instances := []string{"3.250.40.60", "52.209.12.114", "34.245.62.153", "34.242.130.17"}
	instances := []string{"i-0190f618298ff3693", "i-05d3af1d8a4aaf03c", "i-06316bc63aea813ec", "i-0b5ea0d6954565e52"}
	// instances := []string{"i-0190f618298ff3693"}

	instance := instances[rand.Intn(len(instances))]
	user := strings.Replace(strings.Replace(user_template, "ssh_port", port_str, 1), "instance", instance, 1)
	host := "34.241.121.178:2222"
	var t LoadTarget
	t = LoadTarget{}
	client, session, err := connectToHost(&t, user, host)
	t.Client = client
	t.Session = session

	//Push
	_ = err
	time.Sleep(1)
	*targets <- &t
}
