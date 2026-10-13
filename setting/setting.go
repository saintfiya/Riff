package setting

import (
	"fmt"
	"strings"
	"riff/conf"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Init() 加载配置文件
func Init() error {
	//读取配置文件
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath("./conf")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	//加载配置文件
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("viper.ReadInConfig err:%v\n", err)
		return err
	}

	//反序列化
	if err := viper.Unmarshal(conf.Conf); err != nil {
		fmt.Printf("viper.Unmarshal err:%v\n", err)
		return err
	}

	//监控配置文件变化
	viper.WatchConfig()

	//配置文件变化后同步到全局变量Conf
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("配置文件修改了...")
		if err := viper.Unmarshal(conf.Conf); err != nil {
			fmt.Printf("viper.Unmarshal err:%v\n", err)
		}
	})

	return nil

}
