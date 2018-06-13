package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

const (
	host = ""
	port = ""
)

func main() {
	ca, err := ioutil.ReadFile("/etc/openvpn/ca.crt")
	if err != nil {
		log.Println("[error]", err)
		return
	}

	tlscrypt, err := ioutil.ReadFile("/etc/openvpn/tls-crypt.key")
	if err != nil {
		log.Println("[error]", err)
		return
	}

	name := os.Args[1]
	if name == "" {
		log.Fatalln("miss repo name")
	}

	cert, err := ioutil.ReadFile(fmt.Sprintf("/etc/openvpn/client/%s.crt", name))
	if err != nil {
		log.Println("[error]", err)
		return
	}

	key, err := ioutil.ReadFile(fmt.Sprintf("/etc/openvpn/client/%s.key", name))
	if err != nil {
		log.Println("[error]", err)
		return
	}

	fmt.Printf(`
client
dev tun
proto tcp
remote %s %s 
resolv-retry infinite
nobind
persist-key
persist-tun
remote-cert-tls server
comp-lzo
verb 4
auth-nocache
<ca>
%s
</ca>
<tls-crypt>
%s
</tls-crypt>
<cert>
%s
</cert>
<key>
%s
</key>
`, host, port, ca, tlscrypt, cert, key)

}
