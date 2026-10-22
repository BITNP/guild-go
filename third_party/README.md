# third_party

## bitnp-authz

`bitnp-authz/` 是 [`git.bitnp.net/BITNP/bitnp-authz`](https://git.bitnp.net/BITNP/bitnp-authz) 的 Go 源码快照，让 guild-go 在无法访问 `git.bitnp.net` 的环境中也能构建。

- 上游提交：`643da5d210aa`，即 `v0.0.0-20261019160408-643da5d210aa`，与 `go.mod` 中的版本一致。
- 只包含构建所需的 Go 包文件和 `go.mod`；上游的 `examples/`、`js/`、`python/`、文档与测试没有纳入。
- 根目录 `go.mod` 用 `replace` 指向这里；除该模块外，其余依赖仍由 Go module proxy 提供。
- 上游仓库目前没有 LICENSE 文件；内嵌和再分发前建议先和上游确认授权。

更新快照：

```bash
# 在上游仓库中切到目标提交，然后覆盖本目录的 Go 文件
cd /path/to/bitnp-authz
git checkout <commit>
cp go.mod claims.go doc.go permissions.go positions.go roles.go \
  /path/to/guild-go/third_party/bitnp-authz/
```

同时更新根目录 `go.mod` 里 `bitnp-authz` 的伪版本号，并运行 `go mod tidy`。