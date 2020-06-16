package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type LoadCommand struct {
	command       string
	minChars      int
	textInResults string
}
type LoadServer struct {
	NumClients          int
	ParallelLoadClients int
	GWHost              string
	GWPort              int
	MaxCommands         int
	Instances           []string
}
type LoadTarget struct {
	Client        *ssh.Client
	Session       *ssh.Session
	in            chan<- string
	out           <-chan string
	commandsCount int
	elapsedms     int
	instanceID    string
}

var targets chan *LoadTarget
var loadServerConfig LoadServer = LoadServer{}

var CommandElapsedms int = 0
var CommandCount int = 0
var CommandTimeOuts int = 0
var ConnectedClients int = 0
var LastTimeOutServer string = ""

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
var wg2 sync.WaitGroup

func main() {

	var wg sync.WaitGroup

	// Load configuration file
	configAbsPath, err := filepath.Abs("load.config.json")
	if err != nil {
		return
	}
	fmt.Println(configAbsPath)

	// todo: Check if files exists
	if fileExists(configAbsPath) {
		// Read File
		configData, err := ioutil.ReadFile(configAbsPath)
		if err != nil {
			return
		}
		err = json.Unmarshal(configData, &loadServerConfig)
		if err != nil {
			return
		}
	}

	// Start Connecting to Clients
	targets = make(chan *LoadTarget, loadServerConfig.NumClients)
	for i := 0; i < loadServerConfig.NumClients/loadServerConfig.ParallelLoadClients; i++ {
		for i := 1; i <= loadServerConfig.ParallelLoadClients; i++ {
			wg.Add(1)
			go connectToRandomHost(&targets, &wg)
		}
		wg.Wait()
	}
	performLoad()

}

func performLoad() {
	// Profiling Stuff init
	cpuprofile := "load.prof"
	f, err := os.Create(cpuprofile)
	if err != nil {
		log.Fatal(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	// Perform Load

	fmt.Println("Load Paramaters")
	fmt.Println("---------------")
	fmt.Println("Max Clients: %d", loadServerConfig.NumClients)
	fmt.Println("Max Commands: %d", loadServerConfig.MaxCommands)
	fmt.Println("Parallel Clients: %d", loadServerConfig.ParallelLoadClients)

	commands := make([]LoadCommand, 4)
	commands[0] = LoadCommand{command: "ls", minChars: 4, textInResults: "ca.pub"}
	commands[1] = LoadCommand{command: "pwd", minChars: 4, textInResults: "ec2-user"}
	commands[2] = LoadCommand{command: "ps -ef", minChars: 50, textInResults: "ps -ef"}
	commands[3] = LoadCommand{command: "netstat -an", minChars: 50, textInResults: "ESTABLISHED"}
	for i := 0; i < len(commands); i++ {
		c := commands[i]
		fmt.Printf("Test Command: %s, Min Chars : %d, Pattern: %s\n", c.command, c.minChars, c.textInResults)
	}

	startTime := time.Now()
	var wg sync.WaitGroup

	var invocationCount = 0
	maxLoops := loadServerConfig.MaxCommands / loadServerConfig.ParallelLoadClients
	for i := 0; i < maxLoops; i++ {
		invocationCount++
		if ConnectedClients >= loadServerConfig.ParallelLoadClients {

			c := commands[rand.Intn(len(commands))]
			invokeBatchStartTime := time.Now()
			for i := 0; i < loadServerConfig.ParallelLoadClients; i++ {
				wg.Add(1)
				go SendCommandToTarget(&targets, c, &wg)
			}
			wg.Wait()
			latency := int(time.Since(invokeBatchStartTime).Milliseconds())
			CommandElapsedms += int(latency)
			if latency < 1000 {
				time.Sleep(time.Duration(1000-latency) * time.Millisecond)
			}
			rate := float64(CommandCount) / float64(time.Since(startTime).Seconds())
			fmt.Printf("\rInvoked: %d, Timeouts %d, Commands per sec %f, Avg latency ms: %f, Last TimoutServer %s",
				CommandCount,
				CommandTimeOuts,
				rate,
				float32(CommandElapsedms)/float32(CommandCount),
				LastTimeOutServer)
		}
		// time.Sleep(5 * time.Second)
	}
}
func SendCommandToTarget(targets *chan *LoadTarget, c LoadCommand, wg *sync.WaitGroup) {

	defer wg.Done()
	CommandCount++
	var output string

	id := fmt.Sprintf("id-%d", CommandCount)
	commandToInvoke := fmt.Sprintf("echo %s;%s;", id, c.command)

	t := <-*targets
	// Check t
	t.in <- commandToInvoke
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(10 * time.Second)
		timeout <- true
	}()

	select {
	case output = <-t.out:
		if len(output) < c.minChars || !strings.Contains(output, c.textInResults) || !strings.Contains(output, id) {
			fmt.Printf("Command output  Failed : command: %s", c.command)
			*targets <- t
		}
	case <-timeout:
		// the read from ch has timed out
		fmt.Printf("!!! Command timeout: %s\n", t.instanceID)
		CommandTimeOuts++
		LastTimeOutServer = t.instanceID
		t.Client.Close()
		wg2.Add(1)
		go connectToRandomHost(targets, &wg2)
		wg2.Wait() // Add another healthy connection

	}
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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

	// fmt.Printf("Connected to host :%s, with user: %s\n", host, user)
	fmt.Printf("\rConnections %d", ConnectedClients)

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

func MuxShell(w io.Writer, r io.Reader) (chan<- string, <-chan string) {
	in := make(chan string, 1)
	out := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1) //for the shell itself
	go func() {
		for cmd := range in {
			wg.Add(1)
			w.Write([]byte(cmd))
			w.Write([]byte("\n"))
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
	// instances := []string{"i-0190f618298ff3693", "i-05d3af1d8a4aaf03c", "i-06316bc63aea813ec", "i-0b5ea0d6954565e52"}
	// instances := []string{"i-0190f618298ff3693"}

	instance := loadServerConfig.Instances[rand.Intn(len(loadServerConfig.Instances))]
	user := strings.Replace(strings.Replace(user_template, "ssh_port", port_str, 1), "instance", instance, 1)
	host := fmt.Sprintf("%s:%d", loadServerConfig.GWHost, loadServerConfig.GWPort)
	var t LoadTarget
	t = LoadTarget{}
	t.instanceID = user
	for i := 0; i < 4; i++ { // Number of retries
		client, session, err := connectToHost(&t, user, host)
		if err != nil {
			fmt.Println("Connection to target failed, retrying")
			time.Sleep(500 * time.Millisecond)
		} else {
			t.Client = client
			t.Session = session
			ConnectedClients++
			break
		}
	}
	if t.Client == nil { // Case didn;t reconnect after fail
		fmt.Println("***0 Connection After Max retries failed *****")
	} else { // Success
		*targets <- &t
	}
}
