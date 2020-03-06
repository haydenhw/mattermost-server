// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package app

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

type permissionTransformation struct {
	On     func(string, map[string]map[string]bool) bool
	Add    []string
	Remove []string
}
type permissionsMap []permissionTransformation

const (
	PERMISSION_MANAGE_SYSTEM                     = "manage_system"
	PERMISSION_MANAGE_EMOJIS                     = "manage_emojis"
	PERMISSION_MANAGE_OTHERS_EMOJIS              = "manage_others_emojis"
	PERMISSION_CREATE_EMOJIS                     = "create_emojis"
	PERMISSION_DELETE_EMOJIS                     = "delete_emojis"
	PERMISSION_DELETE_OTHERS_EMOJIS              = "delete_others_emojis"
	PERMISSION_MANAGE_WEBHOOKS                   = "manage_webhooks"
	PERMISSION_MANAGE_OTHERS_WEBHOOKS            = "manage_others_webhooks"
	PERMISSION_MANAGE_INCOMING_WEBHOOKS          = "manage_incoming_webhooks"
	PERMISSION_MANAGE_OTHERS_INCOMING_WEBHOOKS   = "manage_others_incoming_webhooks"
	PERMISSION_MANAGE_OUTGOING_WEBHOOKS          = "manage_outgoing_webhooks"
	PERMISSION_MANAGE_OTHERS_OUTGOING_WEBHOOKS   = "manage_others_outgoing_webhooks"
	PERMISSION_LIST_PUBLIC_TEAMS                 = "list_public_teams"
	PERMISSION_LIST_PRIVATE_TEAMS                = "list_private_teams"
	PERMISSION_JOIN_PUBLIC_TEAMS                 = "join_public_teams"
	PERMISSION_JOIN_PRIVATE_TEAMS                = "join_private_teams"
	PERMISSION_PERMANENT_DELETE_USER             = "permanent_delete_user"
	PERMISSION_CREATE_BOT                        = "create_bot"
	PERMISSION_READ_BOTS                         = "read_bots"
	PERMISSION_READ_OTHERS_BOTS                  = "read_others_bots"
	PERMISSION_MANAGE_BOTS                       = "manage_bots"
	PERMISSION_MANAGE_OTHERS_BOTS                = "manage_others_bots"
	PERMISSION_DELETE_PUBLIC_CHANNEL             = "delete_public_channel"
	PERMISSION_DELETE_PRIVATE_CHANNEL            = "delete_private_channel"
	PERMISSION_MANAGE_PUBLIC_CHANNEL_PROPERTIES  = "manage_public_channel_properties"
	PERMISSION_MANAGE_PRIVATE_CHANNEL_PROPERTIES = "manage_private_channel_properties"
	PERMISSION_VIEW_MEMBERS                      = "view_members"
	PERMISSION_INVITE_USER                       = "invite_user"
	PERMISSION_INVITE_GUEST                      = "invite_guest"
	PERMISSION_PROMOTE_GUEST                     = "promote_guest"
	PERMISSION_DEMOTE_TO_GUEST                   = "demote_to_guest"
	PERMISSION_USE_CHANNEL_MENTIONS              = "use_channel_mentions"
	PERMISSION_CREATE_POST                       = "create_post"
	PERMISSION_CREATE_POST_PUBLIC                = "create_post_public"
)

func isRole(role string) func(string, map[string]map[string]bool) bool {
	return func(roleName string, permissionsMap map[string]map[string]bool) bool {
		return roleName == role
	}
}

func permissionExists(permission string) func(string, map[string]map[string]bool) bool {
	return func(roleName string, permissionsMap map[string]map[string]bool) bool {
		val, ok := permissionsMap[roleName][permission]
		return ok && val
	}
}

func permissionNotExists(permission string) func(string, map[string]map[string]bool) bool {
	return func(roleName string, permissionsMap map[string]map[string]bool) bool {
		val, ok := permissionsMap[roleName][permission]
		return !(ok && val)
	}
}

func onOtherRole(otherRole string, function func(string, map[string]map[string]bool) bool) func(string, map[string]map[string]bool) bool {
	return func(roleName string, permissionsMap map[string]map[string]bool) bool {
		return function(otherRole, permissionsMap)
	}
}

func permissionOr(funcs ...func(string, map[string]map[string]bool) bool) func(string, map[string]map[string]bool) bool {
	return func(roleName string, permissionsMap map[string]map[string]bool) bool {
		for _, f := range funcs {
			if f(roleName, permissionsMap) {
				return true
			}
		}
		return false
	}
}

func permissionAnd(funcs ...func(string, map[string]map[string]bool) bool) func(string, map[string]map[string]bool) bool {
	return func(roleName string, permissionsMap map[string]map[string]bool) bool {
		for _, f := range funcs {
			if !f(roleName, permissionsMap) {
				return false
			}
		}
		return true
	}
}

func applyPermissionsMap(roleName string, roleMap map[string]map[string]bool, migrationMap permissionsMap) []string {
	var result []string

	for _, transformation := range migrationMap {
		if transformation.On(roleName, roleMap) {
			for _, permission := range transformation.Add {
				roleMap[roleName][permission] = true
			}
			for _, permission := range transformation.Remove {
				roleMap[roleName][permission] = false
			}
		}
	}

	for key, active := range roleMap[roleName] {
		if active {
			result = append(result, key)
		}
	}
	return result
}

func (a *App) doPermissionsMigration(key string, migrationMap permissionsMap) *model.AppError {
	if _, err := a.Srv().Store.System().GetByName(key); err == nil {
		return nil
	}

	roles, err := a.GetAllRoles()
	if err != nil {
		return err
	}

	roleMap := make(map[string]map[string]bool)
	for _, role := range roles {
		roleMap[role.Name] = make(map[string]bool)
		for _, permission := range role.Permissions {
			roleMap[role.Name][permission] = true
		}
	}

	for _, role := range roles {
		role.Permissions = applyPermissionsMap(role.Name, roleMap, migrationMap)
		if _, err := a.Srv().Store.Role().Save(role); err != nil {
			return err
		}
	}

	if err := a.Srv().Store.System().Save(&model.System{Name: key, Value: "true"}); err != nil {
		return err
	}
	return nil
}

func getEmojisPermissionsSplitMigration(a *App) (permissionsMap, error) {
	return permissionsMap{
		permissionTransformation{
			On:     permissionExists(PERMISSION_MANAGE_EMOJIS),
			Add:    []string{PERMISSION_CREATE_EMOJIS, PERMISSION_DELETE_EMOJIS},
			Remove: []string{PERMISSION_MANAGE_EMOJIS},
		},
		permissionTransformation{
			On:     permissionExists(PERMISSION_MANAGE_OTHERS_EMOJIS),
			Add:    []string{PERMISSION_DELETE_OTHERS_EMOJIS},
			Remove: []string{PERMISSION_MANAGE_OTHERS_EMOJIS},
		},
	}, nil
}

func getWebhooksPermissionsSplitMigration(a *App) (permissionsMap, error) {
	return permissionsMap{
		permissionTransformation{
			On:     permissionExists(PERMISSION_MANAGE_WEBHOOKS),
			Add:    []string{PERMISSION_MANAGE_INCOMING_WEBHOOKS, PERMISSION_MANAGE_OUTGOING_WEBHOOKS},
			Remove: []string{PERMISSION_MANAGE_WEBHOOKS},
		},
		permissionTransformation{
			On:     permissionExists(PERMISSION_MANAGE_OTHERS_WEBHOOKS),
			Add:    []string{PERMISSION_MANAGE_OTHERS_INCOMING_WEBHOOKS, PERMISSION_MANAGE_OTHERS_OUTGOING_WEBHOOKS},
			Remove: []string{PERMISSION_MANAGE_OTHERS_WEBHOOKS},
		},
	}, nil
}

func getListJoinPublicPrivateTeamsPermissionsMigration(a *App) (permissionsMap, error) {
	return permissionsMap{
		permissionTransformation{
			On:     isRole(model.SYSTEM_ADMIN_ROLE_ID),
			Add:    []string{PERMISSION_LIST_PRIVATE_TEAMS, PERMISSION_JOIN_PRIVATE_TEAMS},
			Remove: []string{},
		},
		permissionTransformation{
			On:     isRole(model.SYSTEM_USER_ROLE_ID),
			Add:    []string{PERMISSION_LIST_PUBLIC_TEAMS, PERMISSION_JOIN_PUBLIC_TEAMS},
			Remove: []string{},
		},
	}, nil
}

func removePermanentDeleteUserMigration(a *App) (permissionsMap, error) {
	return permissionsMap{
		permissionTransformation{
			On:     permissionExists(PERMISSION_PERMANENT_DELETE_USER),
			Remove: []string{PERMISSION_PERMANENT_DELETE_USER},
		},
	}, nil
}

func getAddBotPermissionsMigration(a *App) (permissionsMap, error) {
	return permissionsMap{
		permissionTransformation{
			On:     isRole(model.SYSTEM_ADMIN_ROLE_ID),
			Add:    []string{PERMISSION_CREATE_BOT, PERMISSION_READ_BOTS, PERMISSION_READ_OTHERS_BOTS, PERMISSION_MANAGE_BOTS, PERMISSION_MANAGE_OTHERS_BOTS},
			Remove: []string{},
		},
	}, nil
}

func applyChannelManageDeleteToChannelUser(a *App) (permissionsMap, error) {
	return permissionsMap{
		permissionTransformation{
			On:  permissionAnd(isRole(model.CHANNEL_USER_ROLE_ID), onOtherRole(model.TEAM_USER_ROLE_ID, permissionExists(PERMISSION_MANAGE_PRIVATE_CHANNEL_PROPERTIES))),
			Add: []string{PERMISSION_MANAGE_PRIVATE_CHANNEL_PROPERTIES},
		},
		permissionTransformation{
			On:  permissionAnd(isRole(model.CHANNEL_USER_ROLE_ID), onOtherRole(model.TEAM_USER_ROLE_ID, permissionExists(PERMISSION_DELETE_PRIVATE_CHANNEL))),
			Add: []string{PERMISSION_DELETE_PRIVATE_CHANNEL},
		},
		permissionTransformation{
			On:  permissionAnd(isRole(model.CHANNEL_USER_ROLE_ID), onOtherRole(model.TEAM_USER_ROLE_ID, permissionExists(PERMISSION_MANAGE_PUBLIC_CHANNEL_PROPERTIES))),
			Add: []string{PERMISSION_MANAGE_PUBLIC_CHANNEL_PROPERTIES},
		},
		permissionTransformation{
			On:  permissionAnd(isRole(model.CHANNEL_USER_ROLE_ID), onOtherRole(model.TEAM_USER_ROLE_ID, permissionExists(PERMISSION_DELETE_PUBLIC_CHANNEL))),
			Add: []string{PERMISSION_DELETE_PUBLIC_CHANNEL},
		},
	}, nil
}

func removeChannelManageDeleteFromTeamUser(a *App) (permissionsMap, error) {
	return permissionsMap{
		permissionTransformation{
			On:     permissionAnd(isRole(model.TEAM_USER_ROLE_ID), permissionExists(PERMISSION_MANAGE_PRIVATE_CHANNEL_PROPERTIES)),
			Remove: []string{PERMISSION_MANAGE_PRIVATE_CHANNEL_PROPERTIES},
		},
		permissionTransformation{
			On:     permissionAnd(isRole(model.TEAM_USER_ROLE_ID), permissionExists(PERMISSION_DELETE_PRIVATE_CHANNEL)),
			Remove: []string{model.PERMISSION_DELETE_PRIVATE_CHANNEL.Id},
		},
		permissionTransformation{
			On:     permissionAnd(isRole(model.TEAM_USER_ROLE_ID), permissionExists(PERMISSION_MANAGE_PUBLIC_CHANNEL_PROPERTIES)),
			Remove: []string{PERMISSION_MANAGE_PUBLIC_CHANNEL_PROPERTIES},
		},
		permissionTransformation{
			On:     permissionAnd(isRole(model.TEAM_USER_ROLE_ID), permissionExists(PERMISSION_DELETE_PUBLIC_CHANNEL)),
			Remove: []string{PERMISSION_DELETE_PUBLIC_CHANNEL},
		},
	}, nil
}

func getViewMembersPermissionMigration(a *App) (permissionsMap, error) {
	return permissionsMap{
		permissionTransformation{
			On:  isRole(model.SYSTEM_USER_ROLE_ID),
			Add: []string{PERMISSION_VIEW_MEMBERS},
		},
		permissionTransformation{
			On:  isRole(model.SYSTEM_ADMIN_ROLE_ID),
			Add: []string{PERMISSION_VIEW_MEMBERS},
		},
	}, nil
}

func getAddManageGuestsPermissionsMigration(a *App) (permissionsMap, error) {
	return permissionsMap{
		permissionTransformation{
			On:  isRole(model.SYSTEM_ADMIN_ROLE_ID),
			Add: []string{PERMISSION_PROMOTE_GUEST, PERMISSION_DEMOTE_TO_GUEST, PERMISSION_INVITE_GUEST},
		},
	}, nil
}

func channelModerationPermissionsMigration(a *App) (permissionsMap, error) {
	transformations := permissionsMap{}

	type rolePair struct {
		targetRole string
		otherRole  string
	}

	rolePairs := []rolePair{
		{targetRole: model.CHANNEL_ADMIN_ROLE_ID, otherRole: model.CHANNEL_USER_ROLE_ID},
	}

	var allTeamSchemes []*model.Scheme

	next := a.SchemesIterator(model.SCHEME_SCOPE_TEAM, 100)
	var schemeBatch []*model.Scheme
	for schemeBatch = next(); len(schemeBatch) > 0; schemeBatch = next() {
		allTeamSchemes = append(allTeamSchemes, schemeBatch...)
	}

	for _, ts := range allTeamSchemes {
		rolePairs = append(rolePairs, rolePair{targetRole: ts.DefaultChannelAdminRole, otherRole: ts.DefaultChannelUserRole})
	}

	for _, rolePair := range rolePairs {
		for perm := range model.ModeratedPermissions {
			trans := permissionTransformation{
				On:  permissionAnd(isRole(rolePair.targetRole), onOtherRole(rolePair.otherRole, permissionExists(perm))),
				Add: []string{perm},
			}
			transformations = append(transformations, trans)
		}
	}

	transformations = append(transformations, permissionTransformation{
		On:  permissionOr(permissionExists(PERMISSION_CREATE_POST), permissionExists(PERMISSION_CREATE_POST_PUBLIC)),
		Add: []string{PERMISSION_USE_CHANNEL_MENTIONS},
	})

	return transformations, nil
}

// DoPermissionsMigrations execute all the permissions migrations need by the current version.
func (a *App) DoPermissionsMigrations() error {
	PermissionsMigrations := []struct {
		Key       string
		Migration func(*App) (permissionsMap, error)
	}{
		{Key: model.MIGRATION_KEY_EMOJI_PERMISSIONS_SPLIT, Migration: getEmojisPermissionsSplitMigration},
		{Key: model.MIGRATION_KEY_WEBHOOK_PERMISSIONS_SPLIT, Migration: getWebhooksPermissionsSplitMigration},
		{Key: model.MIGRATION_KEY_LIST_JOIN_PUBLIC_PRIVATE_TEAMS, Migration: getListJoinPublicPrivateTeamsPermissionsMigration},
		{Key: model.MIGRATION_KEY_REMOVE_PERMANENT_DELETE_USER, Migration: removePermanentDeleteUserMigration},
		{Key: model.MIGRATION_KEY_ADD_BOT_PERMISSIONS, Migration: getAddBotPermissionsMigration},
		{Key: model.MIGRATION_KEY_APPLY_CHANNEL_MANAGE_DELETE_TO_CHANNEL_USER, Migration: applyChannelManageDeleteToChannelUser},
		{Key: model.MIGRATION_KEY_REMOVE_CHANNEL_MANAGE_DELETE_FROM_TEAM_USER, Migration: removeChannelManageDeleteFromTeamUser},
		{Key: model.MIGRATION_KEY_VIEW_MEMBERS_NEW_PERMISSION, Migration: getViewMembersPermissionMigration},
		{Key: model.MIGRATION_KEY_ADD_MANAGE_GUESTS_PERMISSIONS, Migration: getAddManageGuestsPermissionsMigration},
		{Key: model.MIGRATION_KEY_CHANNEL_MODERATIONS_PERMISSIONS, Migration: channelModerationPermissionsMigration},
	}

	for _, migration := range PermissionsMigrations {
		migMap, err := migration.Migration(a)
		if err != nil {
			return err
		}
		if err := a.doPermissionsMigration(migration.Key, migMap); err != nil {
			return err
		}
	}
	return nil
}
