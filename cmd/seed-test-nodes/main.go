// Command seed-test-nodes adds a repeatable set of demo nodes to a local
// development database. It intentionally lives outside the server binary.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"

	"git.bitnp.net/BITNP/guild-go/internal/database"
)

type nodeSeed struct {
	Title       string
	Description string
	Tags        []string
	DueInDays   int
	Completed   bool
	Children    []nodeSeed
}

func main() {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		exitWithError(errors.New("DATABASE_URL is not set"))
	}

	pool, err := database.Open(ctx, dsn)
	if err != nil {
		exitWithError(err)
	}
	defer pool.Close()

	owner := os.Getenv("SEED_OWNER_SUBJECT")
	if owner == "" {
		err = pool.QueryRow(ctx, `SELECT subject FROM users ORDER BY created_at, subject LIMIT 1`).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) {
			owner = "seed-user"
			_, err = pool.Exec(ctx, `INSERT INTO users (subject) VALUES ($1)`, owner)
		}
	}
	if err != nil {
		exitWithError(fmt.Errorf("resolve seed owner: %w", err))
	}

	created := 0
	err = database.WithTx(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (subject) VALUES ($1)
			ON CONFLICT (subject) DO NOTHING`, owner); err != nil {
			return err
		}

		for index, seed := range testNodes {
			id, wasCreated, err := ensureNode(ctx, tx, nil, seed, owner, int64(index))
			if err != nil {
				return err
			}
			if wasCreated {
				created++
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO node_members (node_id, subject, role)
				VALUES ($1, $2, 'leader')
				ON CONFLICT (node_id, subject) DO UPDATE SET role='leader'`,
				id, owner,
			); err != nil {
				return err
			}
			count, err := ensureChildren(ctx, tx, id, seed.Children, owner)
			if err != nil {
				return err
			}
			created += count
		}
		return nil
	})
	if err != nil {
		exitWithError(err)
	}

	var total int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM nodes
		WHERE title LIKE '测试：%'
		   OR description LIKE '[测试样例]%'`).Scan(&total); err != nil {
		exitWithError(err)
	}
	fmt.Printf("seed complete: %d nodes created, %d test nodes present, owner=%s\n",
		created, total, owner)
}

func ensureChildren(
	ctx context.Context,
	tx pgx.Tx,
	parentID int64,
	children []nodeSeed,
	owner string,
) (int, error) {
	created := 0
	for index, seed := range children {
		childID, wasCreated, err := ensureNode(
			ctx,
			tx,
			&parentID,
			seed,
			owner,
			int64(index),
		)
		if err != nil {
			return created, err
		}
		if wasCreated {
			created++
		}
		count, err := ensureChildren(ctx, tx, childID, seed.Children, owner)
		if err != nil {
			return created, err
		}
		created += count
	}
	return created, nil
}

func ensureNode(
	ctx context.Context,
	tx pgx.Tx,
	parentID *int64,
	seed nodeSeed,
	owner string,
	sortOrder int64,
) (int64, bool, error) {
	var id int64
	err := tx.QueryRow(ctx, `
		SELECT id FROM nodes
		WHERE parent_id IS NOT DISTINCT FROM $1 AND title=$2
		ORDER BY id LIMIT 1`,
		parentID, seed.Title,
	).Scan(&id)
	if err == nil {
		return id, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, err
	}

	var dueAt *time.Time
	if seed.DueInDays > 0 {
		due := time.Now().AddDate(0, 0, seed.DueInDays)
		dueAt = &due
	}
	var completedAt *time.Time
	if seed.Completed {
		completed := time.Now().Add(-24 * time.Hour)
		completedAt = &completed
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO nodes (
			parent_id, title, description, due_at, completed_at,
			tags, sort_order, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		parentID,
		seed.Title,
		"[测试样例] "+seed.Description,
		dueAt,
		completedAt,
		seed.Tags,
		sortOrder,
		owner,
	).Scan(&id)
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

var testNodes = []nodeSeed{
	{
		Title:       "测试：深空探索计划",
		Description: "用于验证跨层级扇区布局、完成状态和标签密度。",
		Tags:        []string{"测试", "航天", "多层级"},
		DueInDays:   45,
		Children: []nodeSeed{
			{
				Title:       "任务规划与总体设计",
				Description: "验证父节点向多个子扇区展开时的间距。",
				Tags:        []string{"规划", "架构"},
				DueInDays:   20,
				Children: []nodeSeed{
					{Title: "轨道与窗口设计", Description: "核对首发窗口和转移轨道。", Tags: []string{"轨道"}, Completed: true},
					{Title: "资源预算与风险清单", Description: "整理预算基线和高风险项。", Tags: []string{"预算", "风险"}, DueInDays: 18},
				},
			},
			{
				Title:       "技术验证",
				Description: "覆盖完成态、进行态和较长标题。",
				Tags:        []string{"研发", "验证"},
				DueInDays:   30,
				Children: []nodeSeed{
					{Title: "电推进样机测试", Description: "记录连续点火和热平衡结果。", Tags: []string{"推进", "测试"}, Completed: true},
					{Title: "长期生命保障实验", Description: "验证循环生保系统的稳定性。", Tags: []string{"生命保障", "长期实验"}, DueInDays: 35},
					{Title: "自主导航算法", Description: "评估深空环境下的轨迹修正能力。", Tags: []string{"导航", "算法"}, DueInDays: 28},
				},
			},
			{
				Title:       "地面系统",
				Description: "验证含较少子节点的非均衡扇区。",
				Tags:        []string{"地面", "运维"},
				DueInDays:   24,
				Children: []nodeSeed{
					{Title: "测控网络联调", Description: "串联站点、链路与任务控制中心。", Tags: []string{"测控"}, DueInDays: 15},
					{Title: "数据中心扩容", Description: "补充遥测存储和实时检索容量。", Tags: []string{"数据", "基础设施"}, DueInDays: 22},
				},
			},
		},
	},
	{
		Title:       "测试：开源社区运营",
		Description: "用于验证多根节点、活动状态和不同分支深度。",
		Tags:        []string{"测试", "社区"},
		DueInDays:   60,
		Children: []nodeSeed{
			{
				Title:       "内容与活动",
				Description: "验证月度和专题活动的标签显示。",
				Tags:        []string{"内容", "活动"},
				DueInDays:   14,
				Children: []nodeSeed{
					{Title: "月度技术分享", Description: "准备十一月主题与讲者名单。", Tags: []string{"分享"}, DueInDays: 9},
					{Title: "新手工作坊", Description: "整理首次贡献者的上手材料。", Tags: []string{"新人", "工作坊"}, Completed: true},
				},
			},
			{
				Title:       "工程协作",
				Description: "模拟一个工程流程较深的社区分支。",
				Tags:        []string{"协作", "工程"},
				DueInDays:   40,
				Children: []nodeSeed{
					{Title: "贡献指南改版", Description: "合并代码规范与评审清单。", Tags: []string{"文档", "贡献"}, DueInDays: 12},
					{
						Title:       "发布流程自动化",
						Description: "验证第四级节点的布局和聚焦路径。",
						Tags:        []string{"发布", "自动化"},
						DueInDays:   32,
						Children: []nodeSeed{
							{Title: "灰度发布检查表", Description: "补齐回滚判定和观察指标。", Tags: []string{"灰度", "检查表"}, DueInDays: 16},
						},
					},
				},
			},
			{
				Title:       "社区健康度",
				Description: "验证观察类节点和较长描述。",
				Tags:        []string{"社区", "数据"},
				DueInDays:   55,
				Children: []nodeSeed{
					{Title: "反馈渠道盘点", Description: "汇总论坛、群聊和工单中的高频问题。", Tags: []string{"反馈"}, DueInDays: 21},
					{Title: "贡献者访谈", Description: "安排新老贡献者的对照访谈。", Tags: []string{"访谈", "贡献者"}, DueInDays: 38},
				},
			},
		},
	},
	{
		Title:       "测试：知识库重构",
		Description: "用于验证不同完成比例和迁移项目节点。",
		Tags:        []string{"测试", "知识库"},
		DueInDays:   75,
		Children: []nodeSeed{
			{
				Title:       "信息架构",
				Description: "验证搜索和导航相关节点的分组。",
				Tags:        []string{"信息架构"},
				DueInDays:   26,
				Children: []nodeSeed{
					{Title: "内容分层方案", Description: "确定概念、指南和参考文档的边界。", Tags: []string{"内容", "方案"}, Completed: true},
					{Title: "导航与搜索", Description: "设计跨栏目检索和推荐入口。", Tags: []string{"导航", "搜索"}, DueInDays: 19},
				},
			},
			{
				Title:       "内容治理",
				Description: "验证需要定期维护的节点样式。",
				Tags:        []string{"治理", "维护"},
				DueInDays:   48,
				Children: []nodeSeed{
					{Title: "过期文档清理", Description: "标记半年未更新且无引用的文档。", Tags: []string{"清理"}, DueInDays: 11},
					{Title: "责任人轮值", Description: "建立按季度轮换的内容责任人名单。", Tags: []string{"责任", "流程"}, DueInDays: 27},
				},
			},
			{
				Title:       "迁移验证",
				Description: "验证上线前后并行运行的测试节点。",
				Tags:        []string{"迁移", "验证"},
				DueInDays:   65,
				Children: []nodeSeed{
					{Title: "双轨发布", Description: "新旧知识库在一周内同步更新。", Tags: []string{"发布"}, DueInDays: 33},
					{Title: "访问数据分析", Description: "对比迁移前后的搜索成功率和跳出率。", Tags: []string{"数据", "指标"}, DueInDays: 52},
				},
			},
		},
	},
}
