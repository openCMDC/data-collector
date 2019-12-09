package capturer

import (
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"log"
	"testing"
	"time"
)

type a struct {
	mm map[string]int
}

func Test1(t *testing.T) {
	b := new(a)
	b.mm["1"] = 1
}

func TestRangeChange(t *testing.T) {
	c := make(chan int, 1)
	go func() {
		c <- 1
		time.Sleep(500 * time.Millisecond)
		c <- 1

	}()

	go func() {
		for i := range c {
			fmt.Println(i)
		}
		fmt.Println("end range")
	}()
	time.Sleep(1 * time.Second)
	close(c)
	time.Sleep(1 * time.Hour)
}

func TestPcap(t *testing.T) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatal(err)
	}

	// Print device information
	fmt.Println("Devices found:")
	for _, d := range devices {
		fmt.Println("\nName: ", d.Name)
		fmt.Println("Description: ", d.Description)
		fmt.Println("Devices addresses: ", d.Description)

		for _, address := range d.Addresses {
			fmt.Println("- IP address: ", address.IP)
			fmt.Println("- Subnet mask: ", address.Netmask)
		}
	}
}

func TestSniffer(t *testing.T) {
	handle, err := pcap.OpenLive(`\Device\NPF_{7D0089A9-CD14-4C31-926D-B5D61B002A60}`, 65535, false, pcap.BlockForever)
	if err != nil {
		t.Fail()
		return
	}
	err = handle.SetBPFFilter("dst port 8085")
	if err != nil {
		t.Fail()
		return
	}
	src := gopacket.NewPacketSource(handle, handle.LinkType())
	packets := src.Packets()
	for p := range packets {
		fmt.Println(p.String())
	}
}
