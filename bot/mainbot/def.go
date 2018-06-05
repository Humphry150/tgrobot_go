package bot

// 配置
type BotConfig struct {
	Token             string              `toml:"token"`
	ChatsToManage     []int64             `toml:"chats_to_manage"`
	Commands          []*BotCommand       `toml:"commands"`
	DeleteBotMsgDelay int64               `toml:"delete_bot_msg_delay"`
	ReceiveStartTime  int64               `toml:"receive_start_time"`
	ReceiveEndTime    int64               `toml:"receive_end_time"`
	TipsInterval      int64               `toml:"tips_interval"`
	WhiteList         []int               `toml:"white_list"`
	URLWhiteList      []string            `toml:"url_white_list"`
	WelcomeExtraMsg   string              `toml:"welcome_extra_msg"`
	Tips              map[string][]string `toml:"tips"`
	RollerConfig      rollerConfig        `toml:"roller_config"`
	Redis             redisConfig         `toml:"redis"`
	CoinHotAddress    string              `toml:"coinhot_address"`
}


type rollerConfig struct {
	Admins                []int    `toml:"admins"`
	Tips                  []string `toml:"tips"`
	RankPath              string   `toml:"rank_path"`
	HistoryPath           string   `toml:"history_path"`
	RegisteredMembersPath string   `toml:"registered_members_path"`
	RollerChats           []int64  `toml:"roller_chats"`
	RewardWhiteList       []string `toml:"reward_white_list"`
	Rules                 *rollerRules
}

type redisConfig struct {
	Addr     string `toml:"addr"`
	DB       int    `toml:"db"`
	Password string `toml:"password"`
}

type BotCommand struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
	Type    string   `toml:"type"`
	Text    string   `toml:"text"`
}

type BotMethod map[string]func(args ...string) string

type cryptoCurrency struct {
	CoinID int     `json:"coin_id"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`
}

type rewardLaw struct {
	Count  int     `json:"count"`
	Reward float64 `json:"reward"`
}

type rewardRule struct {
	From         int          `json:"from"`
	To           int          `json:"to"`
	RewardCount  int          `json:"reward_count"`
	RewardRatio  float64      `json:"reward_ratio"`
	TopRewards   []*rewardLaw `json:"top_rewards"`
	OtherRewards float64      `json:"other_rewards"`
}

type rollerGeneralRules struct {
	MaxTries       int64  `json:"max_tries"`
	MaxNum         int64  `json:"max_num"`
	RankCount      int    `json:"rank_count"`
	RewardCount    int    `json:"reward_count"`
	RewardCountMin int    `json:"reward_count_min"`
	RewardType     string `json:"reward_type"`
	Announcement   string `json:"announcement"`
}

type rollerRules struct {
	GeneralRules rollerGeneralRules `json:"general_rules"`
	RewardRules  []*rewardRule      `json:"reward_rules"`
}
