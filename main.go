package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/stanlyzoolo/exprgen"
)

type msg struct {
	id     int    // id worker
	serial int    // serial of worker
	expr   string // expression
	res    int    // result of calculation
	err    string // error of calculation
}

type fnCalculate func(string) (int, error)

type task struct {
	id        int
	serial    int // serial of worker
	expLen    uint8
	calculate fnCalculate
}

type fnWorker func(task, chan msg)

func main() {

	poolSize := 10
	exprLen := uint8(5)
	api, err := url.Parse("http://127.0.0.1:8080/") // u *url.URL
	if err != nil {
		log.Fatal(err)
	}

	NewPool(poolSize, exprLen, worker, api)

}

func NewPool(poolSize int, exprLen uint8, worker fnWorker, api *url.URL) {

	var serial int

	ch := make(chan msg, poolSize) // NOTE: poolSize?

	// init
	for i := 0; i < poolSize; i++ {
		go worker(
			task{
				id: i, serial: serial, expLen: exprLen,
				calculate: newCalculate(api), // TODO
			}, ch,
		)
		serial++
	}

	// wait for complete
	for {
		msg := <-ch
		// TODO: output
		fmt.Printf("%+v\n", msg)
		go worker(task{
			id: msg.id, serial: serial, expLen: exprLen,
			calculate: newCalculate(api),
		}, ch,
		)
		serial++
	}

}

func worker(t task, output chan msg) {
	expr := generate(t.expLen)
	res, err := t.calculate(expr)
	m := msg{id: t.id, serial: t.serial, expr: expr, res: res}
	if err != nil {
		m.err = err.Error()
	}
	output <- m
}

func generate(len uint8) string {
	return exprgen.Generate(len)
}

func newCalculate(template *url.URL) fnCalculate {
	// template <- 
	return func(expr string) (int, error) {

		var api url.URL
		api = *template

		uv := url.Values{}
		uv.Set("expr", expr)
		api.RawQuery = uv.Encode()
		uri := api.String()

		cl := newClient()
		req, err := http.NewRequest(http.MethodGet, uri, nil) //.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		//req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return 0, err
		}

		resp, err := cl.Do(req)
		if err != nil {
			return 0, err
		}

		defer func() {
			resp.Body.Close() // nolint
		}()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}

		aux := struct {
			Result int
			Error  string
		}{}

		if err := json.Unmarshal(data, &aux); err != nil {
			return 0, fmt.Errorf("unmarshal failed: %w", err)
		}

		if aux.Error != "" {
			return aux.Result, errors.New(aux.Error)
		}

		return aux.Result, nil

	}
}

func newClient() *http.Client {

	const defaultTimeout = 5

	var transport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: defaultTimeout * time.Second,
		}).Dial,
		TLSHandshakeTimeout: defaultTimeout * time.Second,
	}

	return &http.Client{
		Timeout:   time.Second * defaultTimeout,
		Transport: transport,
	}
}
