package auth

import (
	"github.com/metal-stack/metal-lib/jwt/sec"
	"github.com/metal-stack/security"
)

var (
	// EditGroups members can edit
	EditGroups = []security.ResourceAccess{
		security.ResourceAccess("ipam-all-all-edit"),
	}

	EditAccess = sec.MergeResourceAccess(EditGroups)

	// EditUser is able to edit content
	EditUser = security.User{
		EMail:  "ipam@metal-stack.io",
		Name:   "ipam",
		Groups: sec.MergeResourceAccess(EditGroups),
		Tenant: "ipam",
	}
)
