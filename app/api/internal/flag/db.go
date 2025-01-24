package flag

import (
	"Gocument/app/api/global"
	"Gocument/app/api/internal/model"
	"fmt"
)

func DatabaseAutoMigrate() {
	var err error

	//自动建表**
	err = global.MysqlDB.Set("gorm:table_option", "Engine=InnoDB").
		AutoMigrate(
			&model.File{},
			&model.User{},
			&model.FileAccess{},
		)

	if err != nil {
		fmt.Println("自动建表失败")
	} else {
		fmt.Println("自动建表成功")
	}
}
