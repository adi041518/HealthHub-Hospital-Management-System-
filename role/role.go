package role

import "time"

type Role struct {
	RoleName   string                   `json:"roleName" bson:"roleName"`
	RoleCode   string                   `json:"roleCode" bson:"roleCode"`
	Privileges []map[string]interface{} `json:"privileges" bson:"privileges"`
	CreatedAt  time.Time                `json:"createdAt" bson:"createdAt"`
	CreatedBy  string                   `json:"createdBy" bson:"createdBy"`
	UpdatedAt  time.Time                `json:"updatedAt" bson:"updatedAt"`
	UpdatedBy  string                   `json:"updatedBy" bson:"updatedBy"`
}
