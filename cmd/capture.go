package cmd

import (
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/hdm/bounty/pkg/bounty"
	log "github.com/sirupsen/logrus"
)

var protocolCount = 0
var cleanupHandlers = []func(){}

func startCapture(cmd *cobra.Command, args []string) {
	done := false
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done = true
	}()

	// Process CLI arguments
	protocols := make(map[string]bool)
	for _, pname := range strings.Split(params.Protocols, ",") {
		pname = strings.TrimSpace(pname)
		protocols[pname] = true
	}

	// TODO: Configure output actions

	// Setup protocol listeners

	// SNMP
	if _, enabled := protocols["snmp"]; enabled {
		setupSNMP()
	}

	// SSH
	if _, enabled := protocols["ssh"]; enabled {
		setupSSH()
	}

	// Make sure at least one capture is running
	if protocolCount == 0 {
		log.Fatalf("at least one protocol must be enabled")
	}

	// Main loop
	for {
		if done {
			log.Printf("shutting down...")
			for _, cleanupHandler := range cleanupHandlers {
				cleanupHandler()
			}
			break
		}
		time.Sleep(time.Second)
	}
}

func setupSSH() {

	sshHostKey := ""
	if params.SSHHostKey != "" {
		data, err := ioutil.ReadFile(params.SSHHostKey)
		if err != nil {
			log.Fatalf("failed to read ssh host key %s: %s", params.SSHHostKey, err)
		}
		sshHostKey = string(data)
	}

	// Create a listener for each port
	sshPorts, err := bounty.CrackPorts(params.SSHPorts)
	if err != nil {
		log.Fatalf("failed to process ssh ports %s: %s", params.SSHPorts, err)
	}
	for _, port := range sshPorts {
		port := port
		sshConf := bounty.NewConfSSH()
		sshConf.PrivateKey = sshHostKey
		sshConf.BindPort = uint16(port)
		if err := bounty.SpawnSSH(sshConf); err != nil {
			log.Fatalf("failed to start ssh server: %q", err)
		}
		cleanupHandlers = append(cleanupHandlers, func() { sshConf.Shutdown() })
	}

	protocolCount++
}

func setupSNMP() {

	// Create a listener for each port
	snmpPorts, err := bounty.CrackPorts(params.SNMPPorts)
	if err != nil {
		log.Fatalf("failed to process snmp ports %s: %s", params.SSHPorts, err)
	}

	for _, port := range snmpPorts {
		port := port
		snmpConf := bounty.NewConfSNMP()
		snmpConf.BindPort = uint16(port)
		if err := bounty.SpawnSNMP(snmpConf); err != nil {
			log.Fatalf("failed to start snmp server: %q", err)
		}
		cleanupHandlers = append(cleanupHandlers, func() { snmpConf.Shutdown() })
	}

	protocolCount++
}
