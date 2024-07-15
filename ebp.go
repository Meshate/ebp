package ebp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Parser interface {
	// last parsed block
	GetCurrentBlock() int
	// add address to observer
	Subscribe(address string) bool
	// list of inbound or outbound transactions for an address
	GetTransactions(address string) []Transaction
}

type ethParser struct {
	currentBlock atomic.Int64
	subscribeMap RwMap[string, *blockInfo]
	client       http.Client
}

func NewParser() Parser {
	p := &ethParser{
		currentBlock: atomic.Int64{},
		subscribeMap: NewRwMap[string, *blockInfo](),
		client:       http.Client{Timeout: 30 * time.Second},
	}
	go p.loop(10 * time.Second)
	return p
}

func (p *ethParser) loop(t time.Duration) {
	log.Println("INFO", "started background subscribe address update")
	ticker := time.NewTicker(t)
	defer ticker.Stop()
	for range ticker.C {
		addresses := p.subscribeMap.Keys()
		wg := sync.WaitGroup{}
		for _, addr := range addresses {
			wg.Add(1)
			go func(addr string) {
				defer wg.Done()
				_ = p.updateAddress(addr)
			}(addr)
		}
		wg.Wait()
		log.Println("INFO", "subscribe address updated")
	}
}

func (p *ethParser) GetCurrentBlock() int {
	resp, err := p.sendRequest(blockNumber, []interface{}{})
	if err != nil {
		log.Println("ERROR", err.Error())
		return -1
	}
	rm := make(map[string]interface{})
	err = json.Unmarshal(resp, &rm)
	if err != nil {
		log.Println("ERROR", err.Error())
		return -1
	}
	value, err := strconv.ParseInt(rm["result"].(string)[2:], 16, 0)
	if err != nil {
		log.Println("ERROR", err.Error())
		return -1
	}
	return int(value)
}

func (p *ethParser) Subscribe(address string) bool {
	if _, exist := p.subscribeMap.Get(address); exist {
		return false
	}
	err := p.updateAddress(address)
	if err != nil {
		return false
	}
	return true
}

func (p *ethParser) updateAddress(address string) error {
	resp, err := p.sendRequest(getBlockByHash, []interface{}{address, false})
	if err != nil {
		log.Println("ERROR", err.Error())
		return err
	}
	rm := make(map[string]interface{})
	err = json.Unmarshal(resp, &rm)
	if err != nil {
		log.Println("ERROR", err.Error())
		return err
	}
	result := rm["result"].(map[string]interface{})
	block := &blockInfo{
		Hash:         result["hash"].(string),
		Number:       result["number"].(string),
		Transactions: interfacesToStrings(result["transactions"].([]interface{})),
	}
	p.subscribeMap.Set(address, block)
	return nil
}

func (p *ethParser) GetTransactions(address string) []Transaction {
	if block, exist := p.subscribeMap.Get(address); !exist {
		return []Transaction{}
	} else {
		var wg sync.WaitGroup
		ch := make(chan Transaction)

		for _, t := range block.Transactions {
			wg.Add(1)
			go func(txHash string) {
				defer wg.Done()
				resp, err := p.sendRequest(getTransactionByHash, []interface{}{txHash})
				if err != nil {
					log.Println("ERROR", err.Error())
					return
				}
				r := &TransactionResponse{}
				_ = json.Unmarshal(resp, r)
				//log.Println("DEBUG", "add", r.Result.Hash)
				ch <- r.Result

			}(t)
		}

		go func() {
			wg.Wait()
			close(ch)
		}()

		res := make([]Transaction, 0, len(block.Transactions))
		for t := range ch {
			res = append(res, t)
		}
		return res
	}
}

func (p *ethParser) sendRequest(method string, params []interface{}) ([]byte, error) {
	req := rpcRequest{
		Jsonrpc: jsonrpc,
		Method:  method,
		Params:  params,
		Id:      1,
	}.Json()
	var (
		r   *http.Response
		err error
	)
	for i := 0; i < maxRetry; i++ {
		r, err = p.client.Post(url, contentType, bytes.NewReader(req))
		if err == nil {
			defer r.Body.Close()
			data, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("failed to send request after %d retries: %w", maxRetry, err)
}
