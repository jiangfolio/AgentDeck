package market

import "github.com/jiangbyte/agentdeck/internal/pkg/platform/apps"

/**
 * defaultCatalogSeed 仅在目录表为空时写入一次，之后可自由增删改。
 */
func defaultCatalogSeed() []CatalogApp {
	return []CatalogApp{
		{
			ID: "nginx", Name: "Nginx", Summary: "高性能 HTTP 与反向代理（自行解压官方包到应用目录）",
			Category: "Web 服务器", Bin: "sbin/nginx", Start: []string{"{bin}", "-p", "{install_dir}"}, Runnable: true, Sort: 10,
		},
		{
			ID: "redis", Name: "Redis", Summary: "键值内存数据库（自行解压官方包到应用目录）",
			Category: "数据库", Bin: "bin/redis-server", Start: []string{"{bin}"}, Runnable: true, Sort: 20,
		},
		{
			ID: "mysql", Name: "MySQL", Summary: "关系型数据库（自行解压官方包到应用目录）",
			Category: "数据库", Bin: "bin/mysqld", Start: []string{"{bin}", "--basedir={install_dir}", "--datadir={install_dir}/data"}, Runnable: true, Sort: 30,
		},
		{
			ID: "node", Name: "Node.js", Summary: "JavaScript 运行时（仅目录管理，默认不启停）",
			Category: "运行环境", Bin: "bin/node", Start: nil, Runnable: false, Sort: 40,
		},
	}
}

/**
 * EnsureSeed 目录为空时写入默认示例项，并为每项创建应用目录。
 */
func (s *Service) EnsureSeed() error {
	n, err := s.repo.CountCatalog()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	for _, item := range defaultCatalogSeed() {
		row := item
		if err := s.repo.CreateCatalog(&row); err != nil {
			return err
		}
		if _, err := apps.EnsureAppDir(row.ID); err != nil {
			return err
		}
	}
	return nil
}
