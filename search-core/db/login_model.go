package db

import "github.com/SaiNageswarS/go-api-boot/odm"

type LoginModel struct {
	UserId         string `bson:"_id"`
	EmailId        string `bson:"email"`
	HashedPassword string `bson:"password"`
	CreatedOn      int64  `bson:"createdOn"`
}

func (m LoginModel) Id() string {
	if len(m.UserId) == 0 {
		m.UserId, _ = odm.HashedKey(m.EmailId)
	}

	return m.UserId
}

func (m LoginModel) CollectionName() string { return "login" }
