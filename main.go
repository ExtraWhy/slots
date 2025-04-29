package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/slotopol/server/api"
	"github.com/slotopol/server/cmd" /*
		func main() {
			cmd.Execute()
		}
	*/

	"github.com/gin-gonic/gin"
	cfg "github.com/slotopol/server/config"
)

// set of bets per line
var betset = []float64{0.1, 0.2, 0.5, 1, 2, 5, 10}

// set of sums to add to wallet
var sumset = []float64{
	50, 50, 50, 50,
	100, 100, 100, 100, 100, 100,
	200, 200, 200, 200,
	250, 250, 250, 250, 250, 250,
	300, 300,
	500, 500, 500, 500, 500, 500, 500, 500,
	600,
	700,
	800,
	1000, 1000, 1000, 1000, 1000, 1000, 1000, 1000,
	1500, 1500,
	2000, 2000,
	5000, 5000,
	10000,
}

func ping(r *gin.Engine) {
	var req = httptest.NewRequest("GET", "/ping", nil)
	var w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
	}
	var resp = w.Result()
	if !strings.HasPrefix(resp.Header.Get("Server"), "slotopol/") {
	}
}

func post(r *gin.Engine, path string, token string, arg any) (ret gin.H) {
	var err error
	var b []byte

	if b, err = json.Marshal(arg); err != nil {
	}
	var req = httptest.NewRequest("POST", path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	var w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code == http.StatusNoContent {
		return
	}
	if w.Code != http.StatusOK {
	}
	var resp = w.Result()
	if b, err = io.ReadAll(resp.Body); err != nil {
	}
	if err = json.Unmarshal(b, &ret); err != nil {
	}
	if w.Code != http.StatusOK {
	}
	return
}

func TestPlay() {
	fmt.Println("cryptowin test")
	const cid, uid uint64 = 1, 3
	var arg, ret gin.H
	var admtoken, usrtoken string
	var gid uint64
	var wallet, gain float64
	var fsr int

	// Prepare in-memory database
	cfg.CfgPath = "appdata"
	if err := cmd.Init(); err != nil {
	}

	gin.SetMode(gin.TestMode)
	var r = gin.New()
	r.HandleMethodNotAllowed = true
	api.SetupRouter(r)

	// Send "ping" and check that server is our
	ping(r)

	// Sign-in admin in order to top up player account

	arg = gin.H{
		"email":  "admin@example.org",
		"secret": "0YBoaT",
	}

	ret = post(r, "/signin", "", arg)
	admtoken = ret["access"].(string)

	// Sign-in player with predefined credentials

	arg = gin.H{
		"email":  "player@example.org",
		"secret": "iVI05M",
	}

	ret = post(r, "/signin", "", arg)
	usrtoken = ret["access"].(string)

	// Join game

	arg = gin.H{
		"cid":   cid, // 'virtual' club
		"uid":   uid, // player ID
		"alias": "IGT / Cleopatra",
	}

	ret = post(r, "/game/new", usrtoken, arg)
	gid = uint64(ret["gid"].(float64))
	wallet = ret["wallet"].(float64)

	var bet, sel = 1., 5

	arg = gin.H{
		"gid": gid,
		"bet": bet,
	}

	post(r, "/slot/bet/set", usrtoken, arg)

	arg = gin.H{
		"gid": gid,
		"sel": sel,
	}

	post(r, "/slot/sel/set", usrtoken, arg)

	// Play the game with 100 spins

	for range 100 {
		// check money at wallet
		if wallet < bet*float64(sel) {
			var sum float64
			for wallet+sum < bet*float64(sel) {
				sum = sumset[rand.N(len(sumset))]
			}
			arg = gin.H{
				"cid": cid,
				"uid": uid,
				"sum": sum,
			}
			ret = post(r, "/prop/wallet/add", admtoken, arg)
			wallet = ret["wallet"].(float64)
		}

		// make spin
		arg = gin.H{
			"gid": gid,
		}
		ret = post(r, "/slot/spin", usrtoken, arg)
		fmt.Println(ret)
		var game = ret["game"].(map[string]any)
		if v, ok := game["gain"]; ok {
			gain = v.(float64)
		} else {
			gain = 0
		}
		if v, ok := game["fsr"]; ok {
			fsr = int(v.(float64))
		} else {
			fsr = 0
		}
		wallet = ret["wallet"].(float64)

		// no any more actions on free spins
		if fsr > 0 {
			continue
		}

		// if there has a win, make double-ups sometime
		if gain > 0 && rand.Float64() < 0.3 {
			for {
				arg = gin.H{
					"gid":  gid,
					"mult": 2,
				}
				ret = post(r, "/slot/doubleup", usrtoken, arg)
				fmt.Println(ret)
				var gain = ret["gain"].(float64)
				wallet = ret["wallet"].(float64)
				if gain == 0 {
					break
				}
				if rand.Float64() < 0.5 {
					arg = gin.H{
						"gid": gid,
					}
					post(r, "/slot/collect", usrtoken, arg)
					break
				}
			}
		}

		// change bet value sometimes
		if rand.Float64() < 1./25. {
			bet = betset[rand.N(len(betset))]
			arg = gin.H{
				"gid": gid,
				"bet": bet,
			}
			post(r, "/slot/bet/set", usrtoken, arg)
		}

		// change selected bet lines sometimes
		if rand.Float64() < 1./25. {
			sel = 3 + rand.N(8)
			arg = gin.H{
				"gid": gid,
				"sel": sel,
			}
			post(r, "/slot/sel/set", usrtoken, arg)
		}
	}

}

func main() {
	TestPlay()
}
