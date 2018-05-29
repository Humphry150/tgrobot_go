package bot

import (
	gomap "bitbucket.org/magmeng/go-utils/go-map"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

type FaqBotConfig struct {
	Token                string `toml:"token"`
	JoinedGroupsFilePath string `toml:"joined_groups_file_path"`
	UserInfosFilePath    string `toml:"user_infos_file_path"`
	GroupsUsersFilePath  string `toml:"groups_users_file_path"`

	groups      *gomap.GoMap
	users       *gomap.GoMap
	groupsUsers *gomap.GoMap
}

type GroupsMap map[int64]*tgbotapi.Chat
type GroupUsersMap map[int]*tgbotapi.User
type GroupUsersRelationMap map[string]bool
