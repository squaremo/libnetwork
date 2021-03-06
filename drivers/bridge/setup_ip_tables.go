package bridge

import (
	"fmt"
	"net"

	"github.com/docker/libnetwork/iptables"
	"github.com/docker/libnetwork/netutils"
)

// DockerChain: DOCKER iptable chain name
const (
	DockerChain = "DOCKER"
)

func (n *bridgeNetwork) setupIPTables(config *networkConfiguration, i *bridgeInterface) error {
	// Sanity check.
	if config.EnableIPTables == false {
		return IPTableCfgError(config.BridgeName)
	}

	hairpinMode := !config.EnableUserlandProxy

	addrv4, _, err := netutils.GetIfaceAddr(config.BridgeName)
	if err != nil {
		return fmt.Errorf("Failed to setup IP tables, cannot acquire Interface address: %s", err.Error())
	}
	if err = setupIPTablesInternal(config.BridgeName, addrv4, config.EnableICC, config.EnableIPMasquerade, hairpinMode, true); err != nil {
		return fmt.Errorf("Failed to Setup IP tables: %s", err.Error())
	}

	_, err = iptables.NewChain(DockerChain, config.BridgeName, iptables.Nat, hairpinMode)
	if err != nil {
		return fmt.Errorf("Failed to create NAT chain: %s", err.Error())
	}

	chain, err := iptables.NewChain(DockerChain, config.BridgeName, iptables.Filter, hairpinMode)
	if err != nil {
		return fmt.Errorf("Failed to create FILTER chain: %s", err.Error())
	}

	n.portMapper.SetIptablesChain(chain)

	return nil
}

type iptRule struct {
	table   iptables.Table
	chain   string
	preArgs []string
	args    []string
}

func setupIPTablesInternal(bridgeIface string, addr net.Addr, icc, ipmasq, hairpin, enable bool) error {

	var (
		address   = addr.String()
		natRule   = iptRule{table: iptables.Nat, chain: "POSTROUTING", preArgs: []string{"-t", "nat"}, args: []string{"-s", address, "!", "-o", bridgeIface, "-j", "MASQUERADE"}}
		hpNatRule = iptRule{table: iptables.Nat, chain: "POSTROUTING", preArgs: []string{"-t", "nat"}, args: []string{"-m", "addrtype", "--src-type", "LOCAL", "-o", bridgeIface, "-j", "MASQUERADE"}}
		outRule   = iptRule{table: iptables.Filter, chain: "FORWARD", args: []string{"-i", bridgeIface, "!", "-o", bridgeIface, "-j", "ACCEPT"}}
		inRule    = iptRule{table: iptables.Filter, chain: "FORWARD", args: []string{"-o", bridgeIface, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT"}}
	)

	// Set NAT.
	if ipmasq {
		if err := programChainRule(natRule, "NAT", enable); err != nil {
			return err
		}
	}

	// In hairpin mode, masquerade traffic from localhost
	if hairpin {
		if err := programChainRule(hpNatRule, "MASQ LOCAL HOST", enable); err != nil {
			return err
		}
	}

	// Set Inter Container Communication.
	if err := setIcc(bridgeIface, icc, enable); err != nil {
		return err
	}

	// Set Accept on all non-intercontainer outgoing packets.
	if err := programChainRule(outRule, "ACCEPT NON_ICC OUTGOING", enable); err != nil {
		return err
	}

	// Set Accept on incoming packets for existing connections.
	if err := programChainRule(inRule, "ACCEPT INCOMING", enable); err != nil {
		return err
	}

	return nil
}

func programChainRule(rule iptRule, ruleDescr string, insert bool) error {
	var (
		prefix    []string
		operation string
		condition bool
		doesExist = iptables.Exists(rule.table, rule.chain, rule.args...)
	)

	if insert {
		condition = !doesExist
		prefix = []string{"-I", rule.chain}
		operation = "enable"
	} else {
		condition = doesExist
		prefix = []string{"-D", rule.chain}
		operation = "disable"
	}
	if rule.preArgs != nil {
		prefix = append(rule.preArgs, prefix...)
	}

	if condition {
		if output, err := iptables.Raw(append(prefix, rule.args...)...); err != nil {
			return fmt.Errorf("Unable to %s %s rule: %s", operation, ruleDescr, err.Error())
		} else if len(output) != 0 {
			return &iptables.ChainError{Chain: rule.chain, Output: output}
		}
	}

	return nil
}

func setIcc(bridgeIface string, iccEnable, insert bool) error {
	var (
		table      = iptables.Filter
		chain      = "FORWARD"
		args       = []string{"-i", bridgeIface, "-o", bridgeIface, "-j"}
		acceptArgs = append(args, "ACCEPT")
		dropArgs   = append(args, "DROP")
	)

	if insert {
		if !iccEnable {
			iptables.Raw(append([]string{"-D", chain}, acceptArgs...)...)

			if !iptables.Exists(table, chain, dropArgs...) {
				if output, err := iptables.Raw(append([]string{"-A", chain}, dropArgs...)...); err != nil {
					return fmt.Errorf("Unable to prevent intercontainer communication: %s", err.Error())
				} else if len(output) != 0 {
					return fmt.Errorf("Error disabling intercontainer communication: %s", output)
				}
			}
		} else {
			iptables.Raw(append([]string{"-D", chain}, dropArgs...)...)

			if !iptables.Exists(table, chain, acceptArgs...) {
				if output, err := iptables.Raw(append([]string{"-A", chain}, acceptArgs...)...); err != nil {
					return fmt.Errorf("Unable to allow intercontainer communication: %s", err.Error())
				} else if len(output) != 0 {
					return fmt.Errorf("Error enabling intercontainer communication: %s", output)
				}
			}
		}
	} else {
		// Remove any ICC rule.
		if !iccEnable {
			if iptables.Exists(table, chain, dropArgs...) {
				iptables.Raw(append([]string{"-D", chain}, dropArgs...)...)
			}
		} else {
			if iptables.Exists(table, chain, acceptArgs...) {
				iptables.Raw(append([]string{"-D", chain}, acceptArgs...)...)
			}
		}
	}

	return nil
}
