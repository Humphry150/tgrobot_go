package bot

import (
	"fmt"

	gomap "bitbucket.org/magmeng/go-utils/go-map"
)

// 绑定并且填写了接收奖励地址的用户
type registeredMembersMap map[int]bool

var registeredMembers *gomap.GoMap

func readRegisterdMembersFile() {
	rm := make(registeredMembersMap)
	readFile(conf.RollerConfig.RegisteredMembersPath, "registered_members", &rm)
	registeredMembers.Set(rm)
}

func writeRegisteredMembersFile() {
	writeFile(conf.RollerConfig.RegisteredMembersPath, "registered_members", registeredMembers.Interface())
}

func initRegisterdMembers() {
	registeredMembers = gomap.NewMap(registeredMembersMap{})
	go registeredMembers.Handler()
	readRegisterdMembersFile()
}

func isLocalRegisteredMember(uid int) bool {
	var ok bool
	registeredMembers.Query(uid, &ok)
	if ok {
		return true
	}
	return false
}

func addRegisteredMember(uid int) {
	registeredMembers.Add(uid, true)
	writeRegisteredMembersFile()
}

func isRegisteredMember(uid int) bool {
	if isLocalRegisteredMember(uid) {
		return true
	}

	status, address := queryBindRequest(fmt.Sprintf("%d", uid))
	if status == BIND_STATUS_ALREADY_BIND && address != "" {
		addRegisteredMember(uid)
		return true
	}

	return false
}
