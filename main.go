package main

//SNOW WAS HERE

import (
	"bufio"
	//"flag"
	"fmt"
	"github.com/akamensky/argparse"
	koanfLib "github.com/knadh/koanf"
	koanfToml "github.com/knadh/koanf/parsers/toml"
	_ "github.com/leon332157/replish/client"
	"github.com/leon332157/replish/netstat"
	"github.com/leon332157/replish/server"
	//koanfFile "github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	log "github.com/sirupsen/logrus"
	//flag "github.com/spf13/pflag"
	_ "io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	dotreplit    DotReplit
	port         uint16
	globalConfig ReplishConfig
	//hasReplishField bool = false
	konfConf = koanfLib.Conf{
		Delim:       ".",
		StrictMerge: true,
	}
	koanf = koanfLib.NewWithConf(konfConf)
)

const (
	ModeHelpString = "Mode of operation, can be client or server"
	UrlHelpString  = "URL of the repl (repl.co link)"
	ConfHelpString = "Path to config file"
)

type DotReplit struct {
	Run      string
	Language string
	onBoot   string
	packager map[string]interface{}
	Replish  map[string]interface{}
}

type ReplishConfig struct {
	mode        string
	replUrl     string
	listenPort  uint16
	forwardPort uint16
	ConfigFile  os.File
}

func init() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetReportCaller(false)
	log.SetLevel(log.DebugLevel)
	// Create new parser object
	parser := argparse.NewParser("replish", "Command line tool for replit")
	// Create string flag
	configFile := parser.File("C", "config", os.O_RDONLY, 0777, &argparse.Options{Help: ConfHelpString, Default: ".replit"})
	mode := parser.Selector("m", "mode", []string{"c", "client", "s", "server"}, &argparse.Options{Help: ModeHelpString, Default: "client"})
	replUrl := parser.String("c", "remote-url", &argparse.Options{Help: UrlHelpString})
	listenPort := parser.Int("p", "listen-port", &argparse.Options{Help: "Port to listen on", Default: 8080})
	/*if globalConfig.mode == "client" && *replUrl != "" {
		globalConfig.replUrl = *replUrl
	} else {
		log.Errorf("Invalid repl URL!")
		log.Exit(1)
	}*/
	// Parse input
	server.UNUSED(listenPort)
	err := parser.Parse(os.Args)
	if *mode == "c" || *mode == "client" {
		globalConfig.mode = "client"
	} else {
		globalConfig.mode = "server"
	}
	globalConfig.replUrl = *replUrl
	globalConfig.ConfigFile = *configFile
	if err != nil {
		// In case of error print error and print usage
		// This can also be done by passing -h or --help flags
		//fmt.Println(err)
		fmt.Print(parser.Usage(err))
		log.Exit(1)
	}
}

func startBasicHttp() {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Hello: %s", req.URL.Path)
	})
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
func main() {
	readConfigFile()
	//go startBasicHttp()
	//time.Sleep(1 * time.Second) // wait for server to come online
	//getPort()
	port = 8080
	log.Debugf("[Main] Got port: %v\n", port)
	//go server.StartMain(7777, port)
	//go client.StartWS("ws://127.0.0.1:7777", 0, 10*time.Second)
	/*run, ok := dotreplit.Replish["run"].(string)
	if !ok {
		log.Warn("Reading 'run' field failed")
	}
	go client.ExecCommand(run)*/
	for {
		time.Sleep(1 * time.Second)
	}
}

func getPortAuto() {
	addrs, err := netstat.TCPSocks(func(s *netstat.SockTabEntry) bool {
		if s.Process == nil { // Process can be nil, discard it
			return false
		} else if strings.Contains(s.Process.Name, "System") {
			return false // Discard System process
		}
		return net.IP.IsLoopback(s.LocalAddr.IP) && s.State == netstat.Listen
	})
	if err != nil {
		log.Fatalf("Reading ports failed:%v", err)
	}
	if len(addrs) == 0 {
		log.Fatalf("Looks like we aren't finding any open ports, are you listening on localhost (127.0.0.1)?")
	} else if len(addrs) > 1 {
		fmt.Printf("Multiple ports detected: %v\n", len(addrs))
		for index, sock := range addrs {
			if sock.Process != nil {
				fmt.Printf("%v. %v %v\n", index+1, sock.Process, sock.LocalAddr.Port)
			}
		}
		fmt.Print("Choose port/process: ")
		reader := bufio.NewReader(os.Stdin)
		inp, err := reader.ReadString('\n')
		inp = strings.TrimSuffix(inp, "\r\n")
		if err != nil {
			log.Panic(err)
		}
		sel, err := strconv.ParseInt(inp, 10, 64)
		if err != nil {
			log.Panic(err)
		}
		if sel > int64(len(addrs)) { // Input is a port selection
			port = checkPort(sel)
		} else { // Input is list index
			temp := addrs[sel-1]
			port = temp.LocalAddr.Port
		}
	} else {
		port = addrs[0].LocalAddr.Port
	}
}

func checkPort(p int64) uint16 {
	if p > 65535 || p < 1 {
		log.Fatalf("port %v is out of range(1-65535)", p)
	}
	return uint16(p)
}

func getPort() {
	log.Debug("Getting port")
	rawPort, ok := dotreplit.Replish["port"] // Check if port exist
	if !ok {
		log.Warn("Port is missing, defaulting to auto")
		getPortAuto()
		return
	}
	intPort, ok := rawPort.(int64)
	if ok { // port is int
		port = checkPort(intPort)
		return
	}
	strPort, ok := rawPort.(string)
	if ok {
		// Port is string
		if strPort == "auto" {
			getPortAuto()
			return
		} else {
			intPort, err := strconv.Atoi(strPort)
			temp := int64(intPort)
			if err == nil {
				port = checkPort(temp)
			} else {
				log.Errorf("Error when converting port: %v, defaulting to auto\n", err)
				getPortAuto()
				return
			}
		}
	}
}

func readConfigFile() {
	contents := make([]byte, 1024)
	read, err := globalConfig.ConfigFile.Read(contents)
	if err != nil {
		log.Errorf("Reading .replit failed %s", err)
	}
	contents = contents[:read]
	err = koanf.Load(rawbytes.Provider(contents), koanfToml.Parser())
	if err != nil {
		log.Fatalf("Loading .replit failed %s", err)
	}
	koanf.Print()
}
