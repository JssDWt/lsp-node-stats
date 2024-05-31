package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type ListForwardsResp struct {
	Forwards []*Forward `json:"forwards"`
}
type Forward struct {
	InChannel    string  `json:"in_channel"`
	OutChannel   string  `json:"out_channel"`
	FeeMsat      uint64  `json:"fee_msat"`
	OutMsat      uint64  `json:"out_msat"`
	ReceivedTime float64 `json:"received_time"`
}

type ListPeerChannelsResp struct {
	Channels []*Channel `json:"channels"`
}
type ListClosedChannelsResp struct {
	Channels []*Channel `json:"closedchannels"`
}
type Channel struct {
	ShortChannelId string `json:"short_channel_id"`
	Alias          *Alias `json:"alias"`
	Peer           string `json:"peer_id"`
}

type Alias struct {
	LocalAlias  string `json:"local"`
	RemoteAlias string `json:"remote"`
}

const (
	month                    string = "2024-04"
	BreezcChannelsFile       string = "breezc-listpeerchannels-2024-05-06.json.gz"
	BreezcClosedChannelsFile string = "breezc-listclosedchannels-2024-05-06.json.gz"
	BreezcForwardsFile       string = "breezc-listforwards-settled-2024-05-06.json.gz"
)

type interestingPeer struct {
	name   string
	pubkey string
}

var routingNodes []*interestingPeer = []*interestingPeer{
	{
		name:   "BreezR",
		pubkey: "02442d4249f9a93464aaf8cd8d522faa869356707b5f1537a8d6def2af50058c5b",
	},
	{
		name:   "Breez",
		pubkey: "031015a7839468a3c266d662d5bb21ea4cea24226936e2864a7ca4f2c3939836e0",
	},
}

type lspNodeData struct {
	name              string
	pubkey            string
	forwards          []*Forward
	channels          []*Channel
	channelPeerLookup map[string]string
}

func initializeNodes() (*lspNodeData, error) {
	var err error
	breezc := &lspNodeData{
		name:   "breezc",
		pubkey: "02c811e575be2df47d8b48dab3d3f1c9b0f6e16d0d40b5ed78253308fc2bd7170d",
	}
	breezc.forwards, err = readForwards(BreezcForwardsFile)
	if err != nil {
		return nil, fmt.Errorf("breezc: %w", err)
	}
	breezc.channels, err = readChannels(BreezcChannelsFile)
	if err != nil {
		return nil, fmt.Errorf("breezc: %w", err)
	}
	cc, err := readClosedChannels(BreezcClosedChannelsFile)
	if err != nil {
		return nil, fmt.Errorf("breezc: %w", err)
	}
	breezc.channels = append(breezc.channels, cc...)
	breezc.channelPeerLookup = make(map[string]string)
	for _, channel := range breezc.channels {
		breezc.channelPeerLookup[channel.ShortChannelId] = channel.Peer
		breezc.channelPeerLookup[channel.Alias.LocalAlias] = channel.Peer
	}

	return breezc, nil
}

func readForwards(fileName string) ([]*Forward, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open forwards file: %w", err)
	}

	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	var forwards ListForwardsResp
	err = json.NewDecoder(reader).Decode(&forwards)
	if err != nil {
		return nil, fmt.Errorf("failed to decode forwards json: %w", err)
	}

	return forwards.Forwards, nil
}

func readChannels(fileName string) ([]*Channel, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open channels file: %w", err)
	}

	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	var channels ListPeerChannelsResp
	err = json.NewDecoder(reader).Decode(&channels)
	if err != nil {
		return nil, fmt.Errorf("failed to decode channels json: %w", err)
	}

	return channels.Channels, nil
}

func readClosedChannels(fileName string) ([]*Channel, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open closed channels file: %w", err)
	}

	reader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	var channels ListClosedChannelsResp
	err = json.NewDecoder(reader).Decode(&channels)
	if err != nil {
		return nil, fmt.Errorf("failed to decode closed channels json: %w", err)
	}

	return channels.Channels, nil
}

func main() {
	start, err := time.ParseInLocation("2006-01", month, time.Local)
	if err != nil {
		fmt.Printf("failed to parse month: %v", err)
		os.Exit(1)
	}
	end := AddMonth(start, 1)
	fmt.Printf("start: %v\n", start)
	fmt.Printf("end:   %v\n", end)
	fmt.Printf("DID YOU MAKE SURE THE CHANNELS AND FORWARDS ARE UP-TO-DATE?\n")

	breezc, err := initializeNodes()
	if err != nil {
		fmt.Printf("failed to initialize nodes: %v", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println()
	fmt.Println("*********************************************************")
	fmt.Printf("*****************  Report for %s   *****************\n", month)
	fmt.Println("*********************************************************")
	fmt.Println()
	fmt.Println()

	err = lsp_stats(float64(start.Unix()), float64(end.Unix()), breezc, routingNodes)
	if err != nil {
		fmt.Printf("failed to get lsp stats for %s: %v", breezc.name, err)
		os.Exit(1)
	}
}

type routingStats struct {
	amountMsat uint64
	feeMsat    uint64
	count      uint64
}

type lspStats struct {
	amountMsat               uint64
	feeMsat                  uint64
	count                    uint64
	amountMsatExcludingOpens uint64
	feeMsatExcludingOpens    uint64
	countExcludingOpens      uint64
}

func lsp_stats(start, end float64, node *lspNodeData, routingPeers []*interestingPeer) error {
	totalstats := &lspStats{}
	routingstats := &routingStats{}
	routingLookup := make(map[string]bool)
	for _, routingPeer := range routingPeers {
		routingLookup[routingPeer.pubkey] = true
	}

	for _, forward := range node.forwards {
		if forward.ReceivedTime < start || forward.ReceivedTime >= end {
			continue
		}

		totalstats.amountMsat += forward.OutMsat
		totalstats.count++
		totalstats.feeMsat += forward.FeeMsat
		if (forward.FeeMsat*1000000)/forward.OutMsat >= 3999 && forward.OutMsat >= 500000 {
		} else {
			totalstats.amountMsatExcludingOpens += forward.OutMsat
			totalstats.countExcludingOpens++
			totalstats.feeMsatExcludingOpens += forward.FeeMsat
		}

		inPeerPubkey, ok := node.channelPeerLookup[forward.InChannel]
		if !ok {
			return fmt.Errorf("channel '%s' was not in channelPeerLookup", forward.InChannel)
		}
		outPeerPubkey, ok := node.channelPeerLookup[forward.OutChannel]
		if !ok {
			return fmt.Errorf("channel '%s' was not in channelPeerLookup", forward.OutChannel)
		}
		inRouter := routingLookup[inPeerPubkey]
		outRouter := routingLookup[outPeerPubkey]
		if inRouter && outRouter {
			routingstats.amountMsat += forward.OutMsat
			routingstats.count++
			routingstats.feeMsat += forward.FeeMsat
		}
	}

	var routingnames []string
	for _, p := range routingPeers {
		routingnames = append(routingnames, p.name)
	}

	fmt.Println("*********************************************************")
	fmt.Printf("LSP node stats - %s\n", node.name)
	fmt.Println("---------------------------------------------------------")
	fmt.Println("Totals all routing")
	fmt.Println("Includes all forwards, but fees for probable channel opens are excluded.")
	fmt.Println("count,amount_msat,fee_msat,count_excluding_opens,amount_msat_excluding_opens,fee_msat_excluding_opens")
	fmt.Printf("%d,%d,%d,%d,%d,%d\n", totalstats.count, totalstats.amountMsat, totalstats.feeMsat, totalstats.countExcludingOpens, totalstats.amountMsatExcludingOpens, totalstats.feeMsatExcludingOpens)
	fmt.Println("---------------------------------------------------------")
	fmt.Printf("Routing to/from only routing nodes '%s'\n", strings.Join(routingnames, ", "))
	fmt.Println("count,amount_msat,fee_msat")
	fmt.Printf("%d,%d,%d\n", routingstats.count, routingstats.amountMsat, routingstats.feeMsat)
	fmt.Println("---------------------------------------------------------")
	fmt.Println("*********************************************************")
	return nil
}

func AddMonth(t time.Time, m int) time.Time {
	x := t.AddDate(0, m, 0)
	if d := x.Day(); d != t.Day() {
		return x.AddDate(0, 0, -d)
	}
	return x
}
