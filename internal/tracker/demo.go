package tracker

import (
	"fmt"
	"time"
)

// DemoApplications returns 18 unsaved, explicitly marked fictional applications.
// Dates follow the calendar day in now's location, not elapsed 24-hour periods.
// All companies, people and hiring details are fictional; links use example.com.
func DemoApplications(now time.Time) []Application {
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, now.Location())
	date := func(days int) string { return today.AddDate(0, 0, days).Format(time.DateOnly) }
	items := []Application{
		{
			Company: "星屿云研", Role: "Go 后端工程师", City: "上海", Source: "内推", SourcePerson: "林知夏",
			Status: "已投递", AppliedDate: date(0), FollowUpDate: date(3), Priority: "高", Salary: "30–40K · 15薪",
			Notes: "朋友已帮忙转交简历，关注云平台团队的后续反馈。",
		},
		{
			Company: "雾岭数航", Role: "数据开发工程师", City: "杭州", Source: "Boss直聘",
			Status: "已投递", AppliedDate: date(-1), FollowUpDate: date(2), Priority: "普通", Salary: "25–35K · 14薪",
			Notes: "已发送数据平台项目经历，等待招聘负责人查看。",
		},
		{
			Company: "澄月智序", Role: "平台研发工程师", City: "北京", Source: "猎聘",
			Status: "简历筛选", AppliedDate: date(-3), FollowUpDate: date(0), Priority: "高", Salary: "32–45K · 16薪",
			Notes: "今天确认简历筛选结果，补充高并发服务项目介绍。",
		},
		{
			Company: "松鲸互联", Role: "前端工程师", City: "深圳", Source: "猎头", SourcePerson: "周予安",
			Status: "简历筛选", AppliedDate: date(-5), FollowUpDate: date(-1), Priority: "普通", Salary: "28–38K · 14薪",
			Notes: "约定反馈时间已过，向猎头确认团队筛选进展。",
		},
		{
			Company: "栖光算坊", Role: "算法工程师", City: "杭州", Source: "官网",
			Status: "笔试", AppliedDate: date(-7), FollowUpDate: date(1), Priority: "高", Salary: "35–50K · 15薪",
			Notes: "已收到在线测评，今晚完成算法与工程能力题。",
		},
		{
			Company: "青岚工场", Role: "测试开发工程师", City: "成都", Source: "其他",
			Status: "笔试", AppliedDate: date(-6), FollowUpDate: date(-2), Priority: "低", Salary: "20–28K · 13薪",
			Notes: "技术社区获取职位，笔试已提交，尚未收到反馈。",
		},
		{
			Company: "泊星引擎", Role: "资深 Go 工程师", City: "上海", Source: "内推", SourcePerson: "沈清禾",
			Status: "一面", AppliedDate: date(-9), FollowUpDate: date(0), InterviewAt: date(1) + " 10:30",
			Priority: "高", Salary: "35–45K · 16薪", Notes: "明天技术一面，今天确认会议链接并复习服务治理案例。",
		},
		{
			Company: "霁川微栈", Role: "Java 后端工程师", City: "南京", Source: "Boss直聘",
			Status: "一面", AppliedDate: date(-8), FollowUpDate: date(2), InterviewAt: date(2) + " 14:00",
			Priority: "普通", Salary: "25–35K · 14薪", Notes: "准备订单系统设计与数据库调优的项目复盘。",
		},
		{
			Company: "鹿汐数科", Role: "数据平台工程师", City: "北京", Source: "猎聘",
			Status: "二面", AppliedDate: date(-15), FollowUpDate: date(1), InterviewAt: date(1) + " 15:00",
			Priority: "高", Salary: "35–48K · 15薪", Notes: "一面反馈积极，明天下午与负责人讨论平台架构。",
		},
		{
			Company: "云獭实验室", Role: "客户端工程师", City: "深圳", Source: "猎头", SourcePerson: "许星澜",
			Status: "二面", AppliedDate: date(-12), FollowUpDate: date(3), InterviewAt: date(3) + " 11:00",
			Priority: "普通", Salary: "30–42K · 14薪", Notes: "二面包含跨端方案讨论，整理性能优化作品集。",
		},
		{
			Company: "苔原星图", Role: "基础架构工程师", City: "武汉", Source: "官网",
			Status: "HR面", AppliedDate: date(-18), FollowUpDate: date(0), InterviewAt: date(0) + " 16:00",
			Priority: "高", Salary: "28–38K · 15薪", Notes: "技术面已通过，今天沟通职级、薪酬与到岗时间。",
		},
		{
			Company: "晚枫像素", Role: "产品设计师", City: "广州", Source: "其他",
			Status: "HR面", AppliedDate: date(-20), FollowUpDate: date(2), InterviewAt: date(2) + " 10:00",
			Priority: "普通", Salary: "22–32K · 14薪", Notes: "设计社区推荐，确认设计团队协作方式与试用期目标。",
		},
		{
			Company: "拾贝云桥", Role: "后端技术专家", City: "上海", Source: "内推", SourcePerson: "顾南乔",
			Status: "Offer", AppliedDate: date(-25), Priority: "高", Salary: "42–55K · 16薪",
			Notes: "已收到书面 Offer，正在综合比较团队方向与成长空间。",
		},
		{
			Company: "浮岛智造", Role: "嵌入式软件工程师", City: "苏州", Source: "Boss直聘",
			Status: "Offer", AppliedDate: date(-22), Priority: "普通", Salary: "28–38K · 15薪",
			Notes: "录用流程完成，等待确认最终入职日期。",
		},
		{
			Company: "银杏语图", Role: "机器学习工程师", City: "北京", Source: "猎聘",
			Status: "被拒", AppliedDate: date(-14), Priority: "普通", Salary: "32–45K · 14薪",
			Notes: "团队更看重推荐系统经验，记录面试问题供后续复盘。",
		},
		{
			Company: "曦谷流光", Role: "运维开发工程师", City: "成都", Source: "猎头", SourcePerson: "江以宁",
			Status: "被拒", AppliedDate: date(-17), Priority: "低", Salary: "22–30K · 13薪",
			Notes: "岗位招聘计划调整，猎头已确认本轮流程结束。",
		},
		{
			Company: "海盐矩阵", Role: "全栈工程师", City: "厦门", Source: "官网",
			Status: "放弃", AppliedDate: date(-21), Priority: "低", Salary: "20–30K · 13薪",
			Notes: "沟通后发现岗位偏交付，与个人发展方向不符，主动结束。",
		},
		{
			Company: "蓝桉拾序", Role: "技术产品经理", City: "杭州", Source: "其他",
			Status: "放弃", AppliedDate: date(-28), Priority: "普通", Salary: "28–40K · 14薪",
			Notes: "线下交流获得的机会，因通勤与出差安排不合适而放弃。",
		},
	}
	for i := range items {
		items[i].IsDemo = true
		items[i].URL = fmt.Sprintf("https://example.com/jobs/demo-%02d", i+1)
		items[i].JD = items[i].Role + "：负责核心业务能力建设，与产品和研发团队协作，重视工程质量与用户体验。"
		items[i].Contact = "招聘沟通：demo@example.com"
		items[i].Notes += "（公司、人物与招聘信息均为虚构示例）"
	}
	return items
}
