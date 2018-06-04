package bot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/parnurzeal/gorequest"
	"math/rand"
	"encoding/json"
	"crypto/md5"
	"encoding/hex"
)

func makeRequest(method, url string, p map[string]string) *gorequest.SuperAgent {
	req := gorequest.New().CustomMethod(method, url)
	for k, v := range p {
		req.Param(k, v)
	}
	if *testOn {
		req.Proxy("http://127.0.0.1:1087")
	}
	return req
}

func getNonce() (nonce string) {
	s := "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	t := time.Now().UTC()
	r := rand.New(rand.NewSource(t.UnixNano()))
	for i := 0; i < 8; i++ {
		a := s[r.Intn(len(s)-1)]
		nonce += string(a)
	}
	return
}

func ywreceiveCode(code string) string {
	req := makeRequest("POST", fmt.Sprintf("%s/api/user/act/tg/snet/prize/","https://a.yunex.io"), nil)
	jsonStr := fmt.Sprintf(`{"code":"%s"}`, code)

	var jsonData map[string]interface{}
	json.Unmarshal([]byte(jsonStr), &jsonData)

	ts := fmt.Sprintf("%v", time.Now().Unix())
	nonce := getNonce()
	secret := "7P534HD2BVOL"
	signStr := fmt.Sprintf("%s%s%s%s", jsonStr, ts, nonce, secret)

	fmt.Printf("\n\n\nbody:%s ts:%s nonce:%s secret:%s\n\n", jsonStr, ts, nonce, secret)
	fmt.Printf("signStr:%s\n\n\n", signStr)


	h := md5.New()
	h.Write([]byte(signStr))
	sign := hex.EncodeToString(h.Sum(nil))
	fmt.Printf("md5 is:%s\n\n\n", sign)

	req.Header["x-bitex-ts"] = ts
	req.Header["x-bitex-nonce"] = nonce
	req.Header["x-bitex-sign"] = sign
	req.Header["Content-Type"] = "application/json"
	req.SendStruct(jsonData)

	var resp map[string]interface{}
	req.EndStruct(&resp)
	if resp["ok"] == "true" {
		return ""
	}else {
		if (resp["reason"] != nil) {
			return resp["reason"].(string)
		}else {
			return "sys error"
		}
	}

}

// dragonex.io listcoin api
func listCoin() {
	type coinlistData struct {
		Code   string  `json:"code"`
		Name   string  `json:"name"`
		CoinID int     `json:"coin_id"`
		Price  float64 `json:"price,string"`
	}
	type coinList struct {
		OK   bool            `json:"ok"`
		Msg  string          `json:"msg"`
		Code int             `json:"code"`
		Data []*coinlistData `json:"data"`
	}
	var resp coinList
	makeRequest("GET", "https://a.dragonex.im/coin/list/", nil).EndStruct(&resp)
	if resp.OK {
		var cids []int
		for _, data := range resp.Data {
			cids = append(cids, data.CoinID)
			var c cryptoCurrency
			c.CoinID = data.CoinID
			c.Name = strings.ToUpper(data.Code)
			c.Price = data.Price * 6.5
			coins[strings.ToLower(c.Name)] = &c
		}
	}
}

func dragonexPrice() {
	coins = make(map[string]*cryptoCurrency)
	for {
		listCoin()
		time.Sleep(time.Second * 10)
	}
}

func getDividends() (btc, eth string, date time.Time) {
	url := "https://coinhot.io/dividends/today/?format=json"
	type devidendData struct {
		ID               int64     `json:"id"`
		Date             string    `json:"date"`
		Coin             string    `json:"coin"`
		Quantity         string    `json:"quantity"`
		Fee              string    `json:"fee"`
		Hots             string    `json:"hots"`
		Dividable        string    `json:"dividable"`
		DividendPerShare string    `json:"dividend_per_share"`
		UsdPerShare      string    `json:"usd_per_share"`
		CreatedTime      time.Time `json:"created_time"`
	}

	type coinhotResponse struct {
		Count    int            `json:"count"`
		Next     string         `json:"next"`
		Previous string         `json:"previous,omitempty"`
		Result   []devidendData `json:"results"`
	}

	var resp coinhotResponse
	req := makeRequest("GET", url, nil)
	req.EndStruct(&resp)
	if len(resp.Result) > 1 {
		if resp.Result[0].Date == resp.Result[1].Date {
			date, err := time.Parse("2006-01-02", resp.Result[0].Date)
			if err == nil {
				return resp.Result[0].DividendPerShare, resp.Result[1].DividendPerShare, date
			}
		}
	}
	return "", "", time.Time{}
}

func conhotDividend() {
	for {
		// 北京时间
		now := time.Now().In(time.FixedZone("", 8*3600))
		// 北京时间上午8时
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		// 北京时间上午10时
		startAt := today.Add(time.Hour * 2)
		sleepTime := startAt.Sub(now)
		if now.After(startAt) {
			if dividendDate.Before(today.AddDate(0, 0, -1)) {
				dividendBTC, dividendETH, dividendDate = getDividends()
				DebugLog("devidends: %v, %v, %v", dividendBTC, dividendETH, dividendDate)
				sleepTime = time.Second * 5
			} else {
				sleepTime = startAt.AddDate(0, 0, 1).Sub(now)
			}
		}
		time.Sleep(sleepTime)
	}
}

type bindStatus int

const (
	BIND_STATUS_NOT_BIND = iota
	BIND_STATUS_ALREADY_BIND
	BIND_STATUS_UNKOWN
)

type coinHotUser struct {
	UserName     string `json:"username"`
	UserID       int64  `json:"user_id"`
	PhoneDisplay string `json:"phone_display"`
	ETHAddress   string `json:"eth_address"`
}

func queryBindRequest(uid string) (bindStatus, string) {
	u := conf.CoinHotAddress + "/inner/accounts/telegram/query/"

	p := map[string]string{
		"telegram_uid": uid,
	}

	req := makeRequest("GET", u, p)
	type queryBindResponse struct {
		Reason string `json:"reason"`
		coinHotUser
	}

	var resp queryBindResponse
	_, body, errs := req.EndStruct(&resp)
	DebugLog("body: %s", body)
	if len(errs) != 0 {
		Errorf("query %s bind failed, error: %v", uid, errs)
		return BIND_STATUS_UNKOWN, ""
	}

	if resp.Reason == "" {
		return BIND_STATUS_ALREADY_BIND, resp.ETHAddress
	}

	if resp.Reason == "NOT_BIND" {
		return BIND_STATUS_NOT_BIND, ""
	}

	Warnf("query %s bind, resp: %+v\n", resp)
	return BIND_STATUS_UNKOWN, ""
}

func bindUserRequest(code, uid, uname, firstname, lastname string) string {
	url := conf.CoinHotAddress + "/inner/accounts/telegram/bind/"
	p := map[string]string{
		"code":              code,
		"telegram_uid":      uid,
		"telegram_username": uname,
		"firstname":         firstname,
		"lastname":          lastname,
	}
	type bindResponse struct {
		Reason string `json:"reason"`
		coinHotUser
	}

	var resp bindResponse
	req := makeRequest("GET", url, p)
	_, _, errs := req.EndStruct(&resp)

	if len(errs) != 0 {
		Errorf("bind %s error: %v", uid, errs)
		return "绑定失败，请联系客服。"
	}

	if resp.Reason == "" {
		return fmt.Sprintf("绑定CoinHot账号成功，请【私聊】[CoinHot机器宝宝](https://t.me/coinhot_bot)使用`/setaddress`设置您接收活动奖励的地址。\nCoinHot后续的活动奖励都将发放到该地址中，请尽快添加。推荐使用 [DragonEX](https://dragonex.im) 平台充值地址。")
	}

	if resp.Reason == "CODE_NOT_EXIST" {
		return "绑定失败，请确认您的绑定码输入正确，然后再次尝试绑定。"
	}

	DebugLog("resp: %+v", resp)
	return "绑定失败，请联系客服。"
}

func unbindUserRequest(uid string) string {
	url := conf.CoinHotAddress + "/inner/accounts/telegram/unbind/"
	p := map[string]string{
		"telegram_uid": uid,
	}

	req := makeRequest("GET", url, p)
	_, body, errs := req.End()
	DebugLog("body: %s", body)
	if len(errs) != 0 {
		Errorf("bind %s error: %v", uid, errs)
		return "解绑失败，请联系客服。"
	}

	if body == "{}" {
		return "解绑成功。您可以再次使用'/bind'来绑定您的账号。"
	}

	DebugLog("%s unbind failed. body: %s", uid, body)
	return "解绑失败，请联系客服。"
}

func setAddressRequest(uid, address string) string {
	url := conf.CoinHotAddress + "/inner/accounts/telegram/address/set/"
	p := map[string]string{
		"telegram_uid": uid,
		"address":      address,
	}
	req := makeRequest("GET", url, p)
	_, body, errs := req.End()
	DebugLog("body: %s", body)
	if errs != nil {
		Warnf("set %s address failed. error: %v", errs)
		return "设置奖励地址失败，请稍后再尝试。"
	}

	if body == "{}" {
		isPrivateChatRequest(uid)
		uidNum, _ := strconv.ParseInt(uid, 0, 64)
		if !isLocalRegisteredMember(int(uidNum)) {
			addRegisteredMember(int(uidNum))
		}
		return "设置地址成功。若要修改地址，使用本命令和新的地址即可。"
	}

	Warnf("%s set address %s failed. body: %s", uid, address, body)
	return "设置失败，请呼叫客服，或者稍后再试。"
}

func isPrivateChatRequest(uid string) {
	url := conf.CoinHotAddress + "/inner/accounts/telegram/communicate/"
	p := map[string]string{
		"telegram_uid": uid,
	}
	req := makeRequest("GET", url, p)
	_, _, errs := req.End()
	var n int
	for len(errs) != 0 {
		Errorf("private request error: %v", errs)
		if n >= 2 {
			return
		}
		n++
		_, _, errs = req.End()
	}
	return
}

func recordInviteshipRequest(inviter int, invitees []int, gid int64, gname string) {
	u := conf.CoinHotAddress + "/inner/accounts/telegram/invite/"
	inviterStr := toString(inviter)
	var inviteesSlice []string
	for _, invitee := range invitees {
		inviteesSlice = append(inviteesSlice, toString(invitee))
	}
	type inviterGroup struct {
		Inviter   string   `json:"inviter_uid"`
		Invitees  []string `json:"invited_uids"`
		GroupID   string   `json:"group_id"`
		GroupName string   `json:"group_name"`
	}

	var g inviterGroup
	g.Inviter = inviterStr
	g.Invitees = inviteesSlice
	g.GroupID = toString(gid)
	g.GroupName = gname

	req := makeRequest("POST", u, nil)
	req.SendStruct(g)
	req.Header["Content-Type"] = "application/json"
	_, body, errs := req.End()
	if len(errs) != 0 {
		Errorf("record inviteship %+v failed. error: %v", g, errs)
		return
	}

	if body != "{}" {
		Errorf("recode inviteship %+v failed. body: %v", g, body)
	}

	return
}

func parseFrozenBalanceFromHTML(h string) string {
	reg := regexp.MustCompile(`frozenBalance\s\&nbsp;\s<i\sclass='fa\sfa-long-arrow-right'></i>\s([\d]*)`)
	s := reg.FindAllString(h, 1)
	if len(s) > 0 {
		ss := reg.ReplaceAllString(s[0], "$1")
		b, _ := strconv.ParseInt(ss, 0, 64)
		bf := float64(b) / 1e8
		return fmt.Sprintf("• 通过 etherscan 查询到目前CHT冻结总量为：%.8f\n\n• %s\n\n• %s\n\n• %s\n", bf, "CHT合约[点击查看](https://etherscan.io/address/0x792e0fc822ac6ff5531e46425f13540f1f68a7a8#readContract)", "CHT解禁方案请[点击查看](https://help.coinhot.io/hc/zh-cn/articles/360003300193-%E5%B9%B3%E5%8F%B0%E5%86%BB%E7%BB%93%E7%9A%84%E7%83%AD%E5%B8%81%E5%A6%82%E4%BD%95%E8%A7%A3%E7%A6%81)", "CHT冻结使用合约技术性冻结，具体查询方法请[点击查看](https://help.coinhot.io/hc/zh-cn/articles/360003273814-%E7%83%AD%E5%B8%81%E5%86%BB%E7%BB%93%E7%9A%84%E5%B8%81%E5%9C%A8%E5%93%AA%E9%87%8C-)")
	}
	Warnf("html: %s", h)
	return "查询出错，请联系客服"
}

func contractRequest() string {
	u := "https://etherscan.io/readContract?a=0x792e0fc822ac6ff5531e46425f13540f1f68a7a8&v=0x792e0fc822ac6ff5531e46425f13540f1f68a7a8"
	_, body, err := makeRequest("GET", u, nil).End()
	if err != nil {
		Errorf("get contract failed. error: %v", err)
		return "因网络原因暂时查询不到，请稍后再查询。"
	}

	return parseFrozenBalanceFromHTML(body)
}

func getRewardRequest(uid int, rewardType string) string {
	u := conf.CoinHotAddress + "/inner/accounts/activity/reward/accept/"
	p := map[string]string{
		"telegram_uid": toString(uid),
		"reward_type":  rewardType,
	}

	req := makeRequest("GET", u, p)
	type resp struct {
		Reason string `reason`
	}

	var r resp
	_, _, errs := req.EndStruct(&r)
	if errs != nil {
		Errorf("get reward failed. error: %v", errs)
		return fmt.Sprintf("因系统原因，获取奖励失败，请联系管理员。")
	}

	var tail string
	switch rewardType {
	case "register":
		tail = "活动期间注册成为CoinHot.com的新用户，即可随机获赠 10-100CHT。"
	case "trade":
		tail = "活动期间与专属活动标记的广告商交易一笔，即可随机获赠 100-10000CHT。每个用户仅限领取一次，如果同个用户多次领取，我们有权撤销奖励。"
	case "invitation":
		tail = "活动期间邀请好友注册，好友领取奖励后，即可随机获赠 10-100 CHT。"
	case "telegram_group":
		tail = "活动期间加入龙网电报群，即可获赠 10 CHT。"
	case "identification":
		tail = "活动期间完成实名认证，即可获赠 30 CHT。"
	default:
	}

	var failedReason string
	switch r.Reason {
	case "":
		return fmt.Sprintf("奖励领取成功，请前往网站或App的“钱包”中查看。\n\n%s", tail)
	case "NOT_BIND":
		failedReason = "用户未绑定。请使用命令`/bind`进行绑定"
	case "NO_REWARD_FOUND":
		failedReason = "用户无领取奖励资格"
	case "ALREADY_DELIVERED":
		failedReason = "奖励已经发放，请勿重复领取"
	case "NO_BIND_ADDRESS":
		failedReason = "还未设置收奖地址，请【私聊】[CoinHot机器宝宝](https://t.me/coinhot_bot)使用命令`/setaddress`来设置收奖地址"
	case "NEED_IDENTIFICATION":
		failedReason = "用户未实名认证。请前往官网进行实名认证后再领取奖励"
	default:
		failedReason = fmt.Sprintf("未知错误%q，请联系管理员", r.Reason)
	}
	return fmt.Sprintf("奖励领取不成功，原因：%s。\n\n%s", failedReason, tail)
}

func (r Ranks) sendRollerRewardRequest(api *tgbotapi.BotAPI) {
	u := conf.CoinHotAddress + "/inner/accounts/telegram/game/reward/"
	var rewardMap = make(map[string]string)
	for _, user := range r {
		rewardMap[toString(user.User.ID)] = toString(user.RewardAmount)
	}

	req := makeRequest("POST", u, nil)
	_, _, errs := req.SendMap(rewardMap).End()
	if errs != nil {
		Errorf("reward request failed. error: %v rewardMap: %+v", errs, rewardMap)
		for _, admin := range conf.RollerConfig.Admins {
			sendTextMessage(api, int64(admin), "本次奖励发放失败，请尽快查找原因。", 0, false)
		}
		return
	}
}
