package bot

import (
	"flag"
	"strings"
	"sync"

	gomap "bitbucket.org/magmeng/go-utils/go-map"
	"github.com/BurntSushi/toml"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

var debugOn = flag.Bool("debug", false, "debug mode")

func newUsers() *gomap.GoMap {
	u := gomap.NewMap(make(GroupUsersMap))
	go u.Handler()
	return u
}

func newGroups() *gomap.GoMap {
	g := gomap.NewMap(make(GroupsMap))
	go g.Handler()
	return g
}

func newGroupsUsers() *gomap.GoMap {
	gu := gomap.NewMap(make(GroupUsersRelationMap))
	go gu.Handler()
	return gu
}

func newConfig() FaqBotConfig {
	var botConf FaqBotConfig
	botConf.groups = newGroups()
	botConf.users = newUsers()
	botConf.groupsUsers = newGroupsUsers()
	return botConf
}

func decodeConfig(path string) FaqBotConfig {
	botConf := newConfig()
	_, err := toml.DecodeFile(path, &botConf)
	if err != nil {
		panic(err)
	}
	return botConf
}

func getMe(conf FaqBotConfig) *tgbotapi.BotAPI {
	api, err := tgbotapi.NewBotAPI(conf.Token)
	if err != nil {
		panic(err)
	}
	api.Debug = *debugOn
	DebugLog("Authorized on account %s", api.Self.UserName)
	return api
}

func newUpdateConfig() tgbotapi.UpdateConfig {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	return u
}

func joinedGroupsUsersKey(cid int64, uid int) string {
	return strings.Join([]string{toString(cid), toString(uid)}, "_")
}

func isChatGroup(chat *tgbotapi.Chat) bool {
	return chat != nil && (chat.IsSuperGroup() || chat.IsGroup())
}

func (b *FaqBotConfig) checkJoinedChat(chat *tgbotapi.Chat) {
	DebugLog("check chat")
	var c *tgbotapi.Chat
	b.groups.Query(chat.ID, &c)
	if c == nil {
		b.groups.Add(chat.ID, chat)
		b.writeJoinedGroups()
	}
}

func (b *FaqBotConfig) checkUserInfo(user *tgbotapi.User) {
	DebugLog("check user")
	var u *tgbotapi.User
	b.users.Query(user.ID, &u)
	if u == nil {
		b.users.Add(user.ID, user)
		b.writeUserInfos()
	}
}

func (b *FaqBotConfig) checkGroupsUsers(cid int64, uid int) {
	DebugLog("check relation")
	var ok bool
	key := joinedGroupsUsersKey(cid, uid)
	b.groupsUsers.Query(key, &ok)
	if !ok {
		b.groupsUsers.Add(key, true)
		b.writeGroupsUsers()
	}
}

func (b *FaqBotConfig) updateJoinedChat(update tgbotapi.Update) {
	if isChatGroup(update.Message.Chat) {
		b.checkUserInfo(update.Message.From)
		b.checkJoinedChat(update.Message.Chat)
		b.checkGroupsUsers(update.Message.Chat.ID, update.Message.From.ID)
	}
}

func getUpdates(api *tgbotapi.BotAPI, config FaqBotConfig) {
	u := newUpdateConfig()
	updates, err := api.GetUpdatesChan(u)
	if err != nil {
		panic(err)
	}

	for update := range updates {
		config.updateJoinedChat(update)
	}
}

func (b FaqBotConfig) writeJoinedGroups() {
	writeFile(b.JoinedGroupsFilePath, "joined_groups.json", b.groups.Interface())
}

func (b *FaqBotConfig) readJoinedGroups() {
	gm := make(GroupsMap)
	readFile(b.JoinedGroupsFilePath, "joined_groups.json", &gm)
	b.groups.Set(gm)
}

func (b FaqBotConfig) writeUserInfos() {
	writeFile(b.UserInfosFilePath, "user_infos.json", b.users.Interface())
}

func (b *FaqBotConfig) readUserInfos() {
	um := make(GroupUsersMap)
	readFile(b.JoinedGroupsFilePath, "user_infos.json", &um)
	b.users.Set(um)
}

func (b FaqBotConfig) writeGroupsUsers() {
	writeFile(b.GroupsUsersFilePath, "groups_users.json", b.groupsUsers.Interface())
}

func (b *FaqBotConfig) readGroupsUsers() {
	jm := make(GroupUsersRelationMap)
	readFile(b.GroupsUsersFilePath, "groups_users.json", &jm)
	b.groupsUsers.Set(jm)
}

func Run(confPath string) {
	DebugLog("decode config")
	botConf := decodeConfig(confPath)
	DebugLog("decode config finish")

	var g sync.WaitGroup
	go func() {
		g.Add(1)
		DebugLog("read groups")
		botConf.readJoinedGroups()
		g.Done()
	}()

	go func() {
		g.Add(1)
		DebugLog("read users")
		botConf.readUserInfos()
		g.Done()
	}()

	go func() {
		g.Add(1)
		DebugLog("read group-user-ship")
		botConf.readGroupsUsers()
		g.Done()
	}()
	g.Wait()

	DebugLog("chats: %+#v", botConf.groups)
	DebugLog("users: %+#v", botConf.users)
	DebugLog("relations: %+#v", botConf.groupsUsers)

	api := getMe(botConf)
	getUpdates(api, botConf)
}
