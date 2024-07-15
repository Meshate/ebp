package ebp

import "fmt"

func interfacesToStrings(interfaces []interface{}) []string {
	strings := make([]string, len(interfaces))
	for i, v := range interfaces {
		if str, ok := v.(string); ok {
			strings[i] = str
		} else {
			// 处理非字符串的情况，比如将其转换为字符串
			strings[i] = fmt.Sprintf("%v", v)
		}
	}
	return strings
}
