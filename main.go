package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/eiannone/keyboard"

	"github.com/miekg/dns"
)

var configPath string
var msg string

var goos string

func main() {
	defer WaitExit()
	flag.Parse()
	if len(flag.Args()) > 0 {
		configPath = flag.Args()[0]
	} else {
		configPath = "./config.json"
	}

	goos = runtime.GOOS
	fmt.Println(goos)

	var config = readConfig(configPath)
	if len(config.Dnss) == 0 || len(config.Domains) == 0 {
		fmt.Println("no configs")
		return
	}
	var ch = make(chan A)
	var wg sync.WaitGroup

	var as = make(map[string]ASlice, len(config.Domains))

	for _, ds := range config.Dnss {
		var dnsCopy = ds[0]
		for _, domain := range config.Domains {
			var domainCopy = domain
			wg.Add(1)
			go dnsQuery(ch, &wg, domainCopy, dnsCopy)
		}
	}

	go func() {
		for a := range ch {
			as[a.Domain] = append(as[a.Domain], a)
		}
	}()

	wg.Wait()
	close(ch)

	var localDnsSb strings.Builder

	for addr, a := range as {
		if len(a) == 0 {
			continue
		}
		sort.Sort(a)
		localDnsSb.WriteString(fmt.Sprintf("%s\t%s\t# %d %s\n", a[0].Ip, addr, a[0].Ttl, a[0].DnsIp))
	}

	var localDnsText = localDnsSb.String()
	if localDnsText == "" {
		msg = "all dns detect failed"
		return
	}

	fmt.Println()
	fmt.Println(localDnsText)

	hostsOld := readHosts()
	if hostsOld == "" {
		msg = "read hosts fail"
		return
	}

//	hostsNew := ReplaceHosts(hostsOld, localDnsText)
//	fmt.Println(hostsNew)
//	if err := saveHosts(hostsNew); err != nil {
//		msg = "write hosts fail, need admin permission"
//		return
//	}
	if err := flushDns(); err != nil {
		fmt.Println(err)
		msg = "flush dns fail"
		return
	}

	if msg == "" {
		msg = "update hosts success. "
	}
}

func WaitExit() {
	fmt.Println(msg + "\n press any key to exit... or exit after 30 seconds.")

	go func() {
		_, _, err := keyboard.GetSingleKey()
		if err != nil {
			panic(err)
		}
		os.Exit(0)
	}()

	time.Sleep(30 * time.Second)
}

type A struct {
	Domain string
	Ip     string
	DnsIp  string
	Ttl    int
}

type ASlice []A

func (a ASlice) Len() int           { return len(a) }
func (a ASlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ASlice) Less(i, j int) bool { return a[i].Ttl < a[j].Ttl }

func dnsQuery(ch chan A, wg *sync.WaitGroup, domain string, dnsIp string) {
	defer func() {
		recover()
		wg.Done()
	}()
	var client = &dns.Client{
		Timeout: 5000 * time.Millisecond,
	}
	var msg = &dns.Msg{}
	msg.SetQuestion(domain+".", dns.TypeA)
	r, _, err := client.Exchange(msg, dnsIp+":53")
	if err != nil {
		return
	}
	for _, ans := range r.Answer {
		record, isType := ans.(*dns.A)
		if isType {
			ip := record.A.String()
			fmt.Printf("%15s : %15s %s\n", dnsIp, ip, domain)
			_, _, avg := pingTtl(ip)
			if avg == DefaultMaxNanoSeconds {
				fmt.Printf("%15s : %s %15s %s\n", dnsIp, "no response", domain, ip)
				return
			}
			ch <- A{
				Ip:     ip,
				DnsIp:  dnsIp,
				Ttl:    avg,
				Domain: domain,
			}
		}
	}
}
