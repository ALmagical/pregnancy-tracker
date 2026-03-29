package content

import "time"

// WeekPayload 对齐设计文档 6.5.1 结构（静态部分 + 服务端合并任务/知识）
func WeekPayload(week, day int, tasksDone map[string]bool, knowledge []map[string]interface{}) map[string]interface{} {
	if week < 1 {
		week = 1
	}
	if week > 40 {
		week = 40
	}
	fetus := map[string]interface{}{
		"lengthCm":   30.0 + float64(week-24)*0.8,
		"weightG":    300 + week*25,
		"compareTo":  "椰子",
		"highlights": []string{"听力逐渐完善", "肺部继续发育"},
	}
	if week < 12 {
		fetus["compareTo"] = "蓝莓"
		fetus["highlights"] = []string{"器官形成关键期", "注意补充叶酸"}
	} else if week < 28 {
		fetus["compareTo"] = "牛油果"
		fetus["highlights"] = []string{"胎动逐渐明显", "骨骼变硬"}
	}

	tasks := []map[string]interface{}{
		{"id": "t_001", "title": "准备大排畸资料", "done": tasksDone["t_001"], "source": "system"},
		{"id": "t_002", "title": "本周至少记录3次体重", "done": tasksDone["t_002"], "source": "system"},
	}
	if len(knowledge) == 0 {
		knowledge = []map[string]interface{}{
			{"id": "a_1001", "title": "大排畸怎么做", "cover": "", "tags": []string{"产检"}, "readMinutes": 5},
		}
	}

	return map[string]interface{}{
		"week":    week,
		"day":     day,
		"fetus":   fetus,
		"mom": map[string]interface{}{
			"changes": []string{"可能出现腰酸", "睡眠变浅"},
			"tips":    []string{"适度散步", "少量多餐"},
			"warning": []map[string]interface{}{
				{"level": "high", "text": "出现阴道出血/剧烈腹痛请及时就医", "disclaimer": true},
			},
		},
		"tasks":     tasks,
		"knowledge": knowledge,
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	}
}
