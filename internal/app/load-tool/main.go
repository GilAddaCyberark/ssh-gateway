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
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
)

var mux sync.Mutex
var muxCommand sync.Mutex

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

// var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to `file`")
// var memprofile = flag.String("memprofile", "", "write memory profile to `file`")
// var wg2 sync.WaitGroup

func init() {
	// Init Flags
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

	setCommandLineArgs()

}

func SetupCloseHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\r- Ctrl+C pressed in Terminal")
		for target := range targets {
			if target.Client != nil {
				target.Client.Close()
			}
		}
		os.Exit(0)
	}()
}
func main() {
	SetupCloseHandler()

	flag.Parse()

	// Run Loaders
	go performLoad()
	// Start Connecting to Clients
	go createConnections()

	for {
		time.Sleep(time.Second)
	}

}

func createConnections() {

	targets = make(chan *LoadTarget, loadServerConfig.NumClients)
	var parallel = loadServerConfig.ParallelLoadClients
	for {
		for ConnectedClients < loadServerConfig.NumClients {
			// timeout := make(chan bool, 1)

			// go func() {
			// 	time.Sleep(time.Millisecond * time.Duration(2500))
			// 	timeout <- true
			// }()
			var wg sync.WaitGroup

			for i := 1; i <= parallel; i++ {
				wg.Add(1)
				go AddConnectionToRandomHost(&targets, &wg)
			}
			for i := 1; i <= parallel; i++ {
				wg.Add(1)
				go AddConnectionToRandomHost(&targets, &wg)
			}

			// wait for sync to finish
			timeoutSec := 15
			if waitTimeout(&wg, 15*time.Second) {
				fmt.Printf("Timed out waiting for wait group : %d\n", timeoutSec)
			} else {
				fmt.Printf("\n*Wait group finished, number of connections : %d\n", ConnectedClients)
			}
		}
	}
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false // completed normally
	case <-time.After(timeout):
		return true // timed out
	}
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
	fmt.Printf("GW Host: %s\n", loadServerConfig.GWHost)
	fmt.Printf("GW Port: %d\n", loadServerConfig.GWPort)
	fmt.Printf("Max Clients: %d\n", loadServerConfig.NumClients)
	fmt.Printf("Max Commands: %d\n", loadServerConfig.MaxCommands)
	fmt.Printf("Parallel Clients: %d\n", loadServerConfig.ParallelLoadClients)

	commands := make([]LoadCommand, 5)
	commands[0] = LoadCommand{command: "ls", minChars: 4, textInResults: "ca.pub"}
	commands[1] = LoadCommand{command: "pwd", minChars: 4, textInResults: "ec2-user"}
	commands[2] = LoadCommand{command: "ulimit", minChars: 50, textInResults: "unlimited"}
	commands[3] = LoadCommand{command: "whoami", minChars: 50, textInResults: "ec2-user"}
	commands[3] = LoadCommand{command: "uname", minChars: 50, textInResults: "Linux"}

	for i := 0; i < len(commands); i++ {
		c := commands[i]
		fmt.Printf("Test Command: %s, Min Chars : %d, Pattern: %s\n", c.command, c.minChars, c.textInResults)
	}

	startTime := time.Now()

	var invocationCount = 0
	var sleepBetweenCommandsMs = 800 / loadServerConfig.ParallelLoadClients
	maxLoops := loadServerConfig.MaxCommands / loadServerConfig.ParallelLoadClients
	for i := 0; i < maxLoops; i++ {
		invocationCount++
		// if ConnectedClients >= loadServerConfig.ParallelLoadClients {

		c := commands[rand.Intn(len(commands))]
		invokeBatchStartTime := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < loadServerConfig.ParallelLoadClients; i++ {
			time.Sleep(time.Duration(sleepBetweenCommandsMs) * time.Millisecond)
			go SendCommandToTarget(&targets, c, &wg)
		}
		if waitTimeout(&wg, 1*time.Second) {
			fmt.Printf("Timed out waiting for commands :\n")
		}
		// wg.Wait()
		latency := int(time.Since(invokeBatchStartTime).Milliseconds())
		CommandElapsedms += int(latency)
		if latency < 1000 {
			time.Sleep(time.Duration(1000-latency) * time.Millisecond)
		}
		rate := float64(CommandCount) / float64(time.Since(startTime).Seconds())
		fmt.Printf("\nClients: %d, Commands Invoked: %d, Commands Timeouts %d, Commands per sec %f, Avg latency ms: %f, Last TimoutServer %s",
			ConnectedClients,
			CommandCount,
			CommandTimeOuts,
			rate,
			float32(CommandElapsedms)/float32(CommandCount),
			LastTimeOutServer)
	}
}
func SendCommandToTarget(targets *chan *LoadTarget, c LoadCommand, wg *sync.WaitGroup) {

	defer wg.Done()
	wg.Add(1)
	CommandCount++
	var output string

	id := fmt.Sprintf("id-%d", CommandCount)
	// commandToInvoke := fmt.Sprintf("echo %s;%s;", id, c.command)
	commandToInvoke := fmt.Sprintf("echo %s;", id)

	t := <-*targets
	t.in <- commandToInvoke
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Millisecond * time.Duration(2500))
		timeout <- true
	}()

	select {
	case output = <-t.out:
		// if len(output) > c.minChars && strings.Contains(output, c.textInResults) && strings.Contains(output, id) {
		if output != "" && len(output) > 0 && strings.Contains(output, id) {

			*targets <- t
			// fmt.Printf("Command Invoked: %s\n", commandToInvoke)
		}
	case <-timeout:
		// the read from ch has timed out
		fmt.Printf("!!! Command timeout: %s (%s)\n", t.instanceID, commandToInvoke)
		CommandTimeOuts++
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

	mux.Lock()
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
	mux.Unlock()
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
				fmt.Println(err.Error())
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

func AddConnectionToRandomHost(targets *chan *LoadTarget, wg *sync.WaitGroup) {

	defer wg.Done()

	user_template := "gadda@ec2-user@aws#instance:ssh_port"
	// user_template := "gadda@ec2-user@instance:ssh_port"
	countPorts := 2046 - 2022
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	port := r1.Intn(countPorts) + 2022
	port_str := fmt.Sprintf("%d", port)
	_ = port_str
	s2 := rand.NewSource(time.Now().UnixNano())

	r2 := rand.New(s2)
	instanceIdx := r2.Intn(len(loadServerConfig.Instances))
	instance := loadServerConfig.Instances[instanceIdx]

	user := strings.Replace(strings.Replace(user_template, "ssh_port", port_str, 1), "instance", instance, 1)
	host := fmt.Sprintf("%s:%d", loadServerConfig.GWHost, loadServerConfig.GWPort)
	var t LoadTarget
	t = LoadTarget{}
	t.instanceID = user
	for i := 0; i < 4; i++ { // Number of retries
		client, session, err := connectToHost(&t, user, host)
		ConnectedClients++
		if err != nil {
			fmt.Println("Connection to target failed, retrying")
			time.Sleep(time.Duration(20) * time.Millisecond)
			ConnectedClients--
			if client != nil {
				client.Close()
			}
		} else {
			t.Client = client
			t.Session = session
			break
		}
	}
	if t.Client == nil { // Case didn;t reconnect after fail
		fmt.Println("***0 Connection After Max retries failed *****")
	} else { // Success
		*targets <- &t
	}
}

func setCommandLineArgs() {

	const (
		defaultGWPort = 2222
	)

	flag.IntVar(&loadServerConfig.GWPort, "port", defaultGWPort, "The port of the ssh gateway")
	flag.StringVar(&loadServerConfig.GWHost, "host", "54.171.15.94", "The IP Address of the SSH GW")
	flag.IntVar(&loadServerConfig.NumClients, "clients", 3000, "help message for flagname")
	flag.IntVar(&loadServerConfig.ParallelLoadClients, "parallel", 20, "help message for flagname")
	flag.IntVar(&loadServerConfig.MaxCommands, "max", 10000, "Max Command to run and then stop")

}
