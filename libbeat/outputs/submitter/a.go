package submitter

import (
	"fmt"
	"regexp"
)

func main() {
	strMessage := "[2020-09-26 08:34:41] GET http://local.test.com:8010/index/index/test?name=张三&id=9 0.412191"
	fmt.Println(strMessage)
	//text := "Hello 世界！123 Go."

	// regexp.Compile, 创建正则表达式对象, 还有一个方法与它类似，
	// regexp.MustCompile, 但在解析失败的时候回panic，常用于全局正则表达变量的安全初始化
	//reg, _ := regexp.Compile(`\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] \w+ \S+ \S+`) // 查找连续的小写字母
	reg, _ := regexp.Compile(` \S+ `) // 查找连续的小写字母

	// regexp.Regexp.FindAll 于 FindAllString 类似
	fmt.Printf("%q\n", reg.FindAllString(strMessage, -1)) // ["ello" "o"]

}
