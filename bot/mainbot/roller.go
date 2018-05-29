package bot

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	gomap "bitbucket.org/magmeng/go-utils/go-map"
	"bitbucket.org/magmeng/go-utils/log"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

// 掷骰子
// 全局只有一个变量，一旦开启，其他地方也可以用

type RollerNumber struct {
	Num       int64
	Timestamp int64
}

type RollerTries []*RollerNumber

func (r RollerTries) Len() int {
	return len(r)
}

func (r RollerTries) Less(i, j int) bool {
	if r[i].Num < r[j].Num {
		return true
	}
	if r[i].Num == r[j].Num {
		return r[i].Timestamp >= r[j].Timestamp
	}
	return false
}

func (r RollerTries) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r RollerTries) GetMaxNumber() *RollerNumber {
	sort.Sort(sort.Reverse(r))
	return r[0]
}

func (r RollerTries) Append(rn ...*RollerNumber) {
	r = append(r, rn...)
}

type UserRank struct {
	Rank         int64
	RollerNumber *RollerNumber
	User         *tgbotapi.User
	RewardAmount float64
}

func (ur UserRank) String(withUID bool) string {
	if withUID {
		return fmt.Sprintf("%04d %6d %d\n", ur.Rank, ur.RollerNumber.Num, ur.User.ID)
	}
	return fmt.Sprintf("%04d %6d %s\n", ur.Rank, ur.RollerNumber.Num, validUserName(fmt.Sprintf("%d", ur.User.ID), ur.User.UserName, ur.User.FirstName, ur.User.LastName))
}

type Ranks []*UserRank

func (r Ranks) Len() int {
	return len(r)
}

func (r Ranks) Less(i, j int) bool {
	if r[i].RollerNumber.Num < r[j].RollerNumber.Num {
		return true
	}

	if r[i].RollerNumber.Num == r[j].RollerNumber.Num {
		return r[i].RollerNumber.Timestamp >= r[j].RollerNumber.Timestamp
	}

	return false
}

func (r Ranks) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r Ranks) ListResult(rankLength int) Ranks {
	if r.Len() < rankLength {
		return r
	}

	return r[:rankLength]
}

func (r Ranks) UpdateRanks() {
	sort.Sort(sort.Reverse(r))
	for i := range r {
		r[i].Rank = int64(i) + 1
	}
}

type UserRoller struct {
	ChatID          int64
	User            *tgbotapi.User
	MaxRollingCount int
	Tries           RollerTries
}

func (ur UserRoller) IsRollingCountRunningOut() bool {
	return ur.Tries.Len() >= ur.MaxRollingCount
}

func (ur *UserRoller) Roll(rollerRange int64) *RollerNumber {
	var r RollerNumber
	r.Timestamp = time.Now().Unix()
	r.Num = rand.Int63n(rollerRange)
	log.Infofln("user %d roll %d", ur.User.ID, r.Num)
	if isUserRewarded(ur.ChatID, ur.User.ID) {
		r.Num = rand.Int63n(rollerRange) * 85 / 100
		log.Infofln("rewarded, change to %d", r.Num)
	}
	ur.Tries = append(ur.Tries, &r)
	return &r
}

func (rn *RollerNumber) BroadcastString() string {
	return fmt.Sprintf("本次结果: %d", rn.Num)
}

func (ur UserRoller) CheckCountingString() string {
	return fmt.Sprintf("%s 还剩 %d 次机会", userString(fmt.Sprintf("%d", ur.User.ID), ur.User.UserName, ur.User.FirstName, ur.User.LastName), ur.MaxRollingCount-ur.Tries.Len())
}

func (u UserRoller) FinishRolling() *UserRank {
	var ur UserRank
	ur.RollerNumber = u.Tries.GetMaxNumber()
	ur.User = u.User
	return &ur
}

func (r Ranks) HeadString() string {
	return fmt.Sprintf("%4s %6s %s\n", "Rank", "Num", "User")
}

type RollingUsersMap map[int]*UserRoller

type RollGame struct {
	MaxRollingCount int64
	ChatID          int64
	StartTime       int64
	MaxNum          int64
	RankLength      int
	RewardLength    int
	RewardLengthMin int
	RollingUsers    *gomap.GoMap
}

func (r RollGame) GetRank() Ranks {
	rollingUserMap := r.RollingUsers.Interface().(RollingUsersMap)
	var ranks Ranks
	for _, v := range rollingUserMap {
		ranks = append(ranks, v.FinishRolling())
	}
	ranks.UpdateRanks()
	return ranks
}

func (r RollGame) GetRollingUser(uid int) *UserRoller {
	var user *UserRoller
	r.RollingUsers.Query(uid, &user)
	return user
}

func NewRollGame(chatID int64) *RollGame {
	var r RollGame
	r.MaxNum = conf.RollerConfig.Rules.GeneralRules.MaxNum
	r.MaxRollingCount = conf.RollerConfig.Rules.GeneralRules.MaxTries
	r.RollingUsers = gomap.NewMap(make(RollingUsersMap))
	r.ChatID = chatID
	r.StartTime = time.Now().Unix()
	r.RewardLength = conf.RollerConfig.Rules.GeneralRules.RewardCount
	r.RankLength = conf.RollerConfig.Rules.GeneralRules.RankCount
	r.RewardLengthMin = conf.RollerConfig.Rules.GeneralRules.RewardCountMin
	go r.RollingUsers.Handler()
	rand.Seed(time.Now().UnixNano())
	return &r
}

// return roller history file name
func (r RollGame) historyFileName() string {
	chatIDTag := "[" + toString(r.ChatID) + "]"
	return chatIDTag + ".history"
}

// return roller rank file name
func (r RollGame) rankFileName() string {
	timeTag := time.Unix(r.StartTime, 0).In(time.FixedZone("BeiJing", 8*3600)).Format("2006-01-02_15:04:05")
	chatIDTag := "[" + toString(r.ChatID) + "]"
	return chatIDTag + "_" + timeTag + ".rank"
}

func (r RollGame) writeHistory() {
	writeFile(conf.RollerConfig.HistoryPath, r.historyFileName(), r.RollingUsers.Interface())
}

func (r RollGame) readHistory() {
	var rollerHistory = make(RollingUsersMap)
	readFile(conf.RollerConfig.HistoryPath, r.historyFileName(), &rollerHistory)
	r.RollingUsers.Set(rollerHistory)
}

func (r RollGame) clearHistory() {
	removeFile(conf.RollerConfig.HistoryPath, r.historyFileName())
}

func (r RollGame) writeRank() {
	writeFile(conf.RollerConfig.RankPath, r.rankFileName(), r.GetRank())
}

func isUserRewarded(chatID int64, uid int) bool {
	return redisClient.Get(strings.Join([]string{toString(chatID), toString(uid)}, "_")).Val() == "recorded"
}

func recordRewardedUser(chatID int64, uid int) {
	redisClient.Set(strings.Join([]string{toString(chatID), toString(uid)}, "_"), "recorded", time.Hour*7*24)
}

func (r Ranks) send(api *tgbotapi.BotAPI, chatID, startTime int64, withUID bool, rewardLength int) {
	head := r.HeadString()
	var body string

	for i, rank := range r {
		body += rank.String(withUID)
		if !withUID {
			recordRewardedUser(chatID, rank.User.ID)
		}
		if i == rewardLength-1 {
			body += "↑↑以上人员中奖↑↑\n"
		}
	}

	if body == "" {
		sendTextMessageHideKeyBoard(
			api,
			chatID,
			fmt.Sprintf("%s 摇奖榜单：\n\n本次无人中奖。", time.Unix(startTime, 0).In(time.FixedZone("BeiJing", 8*3600)).Format("2006/01/02 15:04:05")),
			0,
			false,
		)
		return
	}

	if r.Len() < rewardLength {
		body += "↑↑以上人员中奖↑↑\n"
	}

	if rewardLength == 0 {
		body += "↑↑本次无人中奖↑↑"
	}

	sendTextMessageHideKeyBoard(
		api,
		chatID,
		fmt.Sprintf("%s 摇奖榜单：\n\n`%s%s`", time.Unix(startTime, 0).In(time.FixedZone("BeiJing", 8*3600)).Format("2006/01/02 15:04:05"), head, body),
		0,
		true,
	)
}

func newRollerUser(chatID int64, user *tgbotapi.User, count int) *UserRoller {
	var ur UserRoller
	ur.User = user
	ur.MaxRollingCount = count
	ur.ChatID = chatID
	return &ur
}

func getUserRollingCount() int {
	return int(conf.RollerConfig.Rules.GeneralRules.MaxTries)
}

// tips after each rolling num
func choseRollerTips(num int64) string {
	length := len(conf.RollerConfig.Tips)
	i := int(num) % length
	return conf.RollerConfig.Tips[i]
}

func canStartRoll(chatID int64) bool {
	for _, cid := range conf.RollerConfig.RollerChats {
		if chatID == cid {
			return true
		}
	}
	return false
}

type RollGamesMap map[int64]*RollGame

var gameHolder *gomap.GoMap

func RollerMonitor() {
	gameHolder = gomap.NewMap(make(RollGamesMap))
	gameHolder.Handler()
}

func startRoller(chatID int64, messageID int, api *tgbotapi.BotAPI) {
	r := getRollGame(chatID)
	if r != nil {
		sendTextMessage(
			api,
			chatID,
			"活动已经开始。",
			0,
			false,
			messageID,
		)
		return
	}

	if conf.RollerConfig.Rules == nil {
		sendTextMessage(api, chatID, "请管理员先配置摇奖活动。", 0, false, messageID)
		return
	}

	r = NewRollGame(chatID)
	r.readHistory()
	gameHolder.Add(chatID, r)

	sendTextMessageStartRoll(api, chatID, fmt.Sprintf("%s %s", todayStart(time.Now()).Format("2006/01/02"), "摇奖活动开始"), 0, false, messageID)
}

func getRollGame(chatID int64) *RollGame {
	var r *RollGame
	gameHolder.Query(chatID, &r)
	return r
}

func pingMessage(api *tgbotapi.BotAPI, chatID int64, messageID int) {
	var p tgbotapi.PinChatMessageConfig
	p.ChatID = chatID
	p.MessageID = messageID
	api.PinChatMessage(p)
}

func stopRoller(api *tgbotapi.BotAPI, chatID int64) {
	r := getRollGame(chatID)
	if r == nil {
		sendTextMessage(api, chatID, "活动尚未开始。", 0, false)
		return
	}

	r.writeRank()
	var result Ranks
	participantsCount := r.GetRank().Len()
	var rate float64
	var rr rewardRule

	if r.RewardLength > 0 {
		if r.RewardLengthMin > r.RewardLength {
			r.RewardLength = r.RewardLengthMin
		}
		rate = float64(r.RewardLength) / float64(participantsCount)
	} else {
		for _, rule := range conf.RollerConfig.Rules.RewardRules {
			if participantsCount >= rule.From && participantsCount < rule.To {
				rr = *rule
				if rule.RewardCount != 0 {
					r.RewardLength = rule.RewardCount
					rate = float64(r.RewardLength) / float64(participantsCount)
				} else {
					rate = rule.RewardRatio
					r.RewardLength = int(rule.RewardRatio*float64(participantsCount) + 0.5)
				}
				r.RankLength = r.RewardLength + 10
				break
			}
		}
	}

	result = r.GetRank().ListResult(r.RankLength)
	if conf.RollerConfig.Rules.GeneralRules.RewardType == "auto" {
		rewardResult := result[:r.RewardLength]
		rewardResult.rollCHT(rr.TopRewards, rr.OtherRewards)
		result = append(rewardResult, result[r.RewardLength:]...)
		result.sendRewardList(api, r.ChatID, r.StartTime, false, participantsCount, r.RewardLength, rate)

		for _, admin := range conf.RollerConfig.Admins {
			result.sendRewardList(api, int64(admin), r.StartTime, true, participantsCount, r.RewardLength, rate)
		}
		result.sendRollerRewardRequest(api)
	} else if conf.RollerConfig.Rules.GeneralRules.RewardType == "fix" {
		result.send(api, r.ChatID, r.StartTime, false, r.RewardLength)
		for _, admin := range conf.RollerConfig.Admins {
			result.send(api, int64(admin), r.StartTime, true, r.RewardLength)
		}
	}

	r.clearHistory()
	gameHolder.Delete(chatID)
}

func isRollerChat(chatID int64) bool {
	for _, cid := range conf.RollerConfig.RollerChats {
		if cid == chatID {
			return true
		}
	}
	return false
}

func isInRewardWhiteList(username string) bool {
	if username == "" {
		return false
	}

	for _, uname := range conf.RollerConfig.RewardWhiteList {
		if uname == username {
			return true
		}
	}
	return false
}

func ratio(fluctuation float64) float64 {
	r := rand.New(rand.NewSource(time.Now().Unix())).Int63n(int64(fluctuation * 200))
	return 1 + (float64(r)-fluctuation*100)/200
}

func (r Ranks) choseRewardedPerson() *UserRank {
	rand.Seed(time.Now().UnixNano())
	n := rand.Intn(r.Len())
	return r[n]
}

func totalTopRewardCounts(topRewards []*rewardLaw) int {
	var totalTops int
	for _, t := range topRewards {
		totalTops += t.Count
	}
	return totalTops
}

func (r Ranks) topRewardsIndexMap(totalTopRewardCounts int) []int {
	topUsersIndexMap := make(map[int]bool)
	rand.Seed(time.Now().UnixNano())

	for len(topUsersIndexMap) < totalTopRewardCounts {
		topUsersIndexMap[rand.Intn(r.Len())] = true
	}

	var topIndexArray []int
	for k := range topUsersIndexMap {
		topIndexArray = append(topIndexArray, k)
	}

	return topIndexArray
}

func (r Ranks) calculateRewards(topRewardsArray []int, topLaw []*rewardLaw, others float64) {
	var topIndex int
	for _, law := range topLaw {
		for i := 0; i < law.Count; i++ {
			r[topRewardsArray[topIndex]].RewardAmount = law.Reward
			topIndex++
		}
	}
	for i := range r {
		if r[i].RewardAmount == 0 {
			r[i].RewardAmount = others
		}
	}
}

// 计算榜单上的用户应该获得多少CHT奖励
func (r Ranks) rollCHT(top []*rewardLaw, others float64) {
	totalTops := totalTopRewardCounts(top)
	topRewardsArray := r.topRewardsIndexMap(totalTops)
	r.calculateRewards(topRewardsArray, top, others)
}

func (r Ranks) rewardHead() string {
	return fmt.Sprintf("%4s %4s %5s %s\n", "Rank", "Num", "CHT", "User")
}

func (ur *UserRank) rewardLine(withUID bool) string {
	if !withUID {
		return fmt.Sprintf("%04d %04d %5.f %s\n", ur.Rank, ur.RollerNumber.Num, ur.RewardAmount, validUserName(fmt.Sprintf("%d", ur.User.ID), ur.User.UserName, ur.User.FirstName, ur.User.LastName))
	}
	return fmt.Sprintf("%04d %04d %5.f %d\n", ur.Rank, ur.RollerNumber.Num, ur.RewardAmount, ur.User.ID)
}

func (r Ranks) sendRewardList(api *tgbotapi.BotAPI, chatID, startTime int64, withUID bool, participantsCount, rewardCount int, rewardRate float64) {
	head := r.rewardHead()
	var body string

	for i := range r {
		body += r[i].rewardLine(withUID)
		if i == rewardCount-1 {
			body += "↑↑以上人员中奖↑↑\n"
		}
	}

	if r.Len() < rewardCount {
		body += "↑↑以上人员中奖↑↑\n"
	}

	if body == "" {
		sendTextMessageHideKeyBoard(
			api,
			chatID,
			fmt.Sprintf("%s 摇奖\n\n参与人数：%d\n中奖概率：%.2f%%\n中奖人数：%d\n\n参与人数不足，本次不开奖。", time.Unix(startTime, 0).In(time.FixedZone("BeiJing", 8*3600)).Format("2006/01/02 15:04:05"), participantsCount, rewardRate*100, rewardCount),
			0,
			false,
		)
		return
	}
	sendTextMessageHideKeyBoard(
		api,
		chatID,
		fmt.Sprintf("%s 摇奖\n\n参与人数：%d\n中奖概率：%.2f%%\n中奖人数：%d\n\n榜单：\n`%s%s`\n\n稍后奖励会自动发放到中奖用户的CoinHot账户中，请注意查收。", time.Unix(startTime, 0).In(time.FixedZone("BeiJing", 8*3600)).Format("2006/01/02 15:04:05"), participantsCount, rewardRate*100, rewardCount, head, body),
		0,
		true,
	)
}
