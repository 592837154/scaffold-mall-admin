# Mall Admin 启动说明

本项目包含三个 Docker 服务：

- `frontend`：前端管理系统，访问地址 `http://localhost:8000`
- `goods-api`：Go + Gin 后端接口，访问地址 `http://localhost:8080`
- `mysql`：MySQL 8.4 数据库，供后端容器内部使用

## 启动服务

在项目根目录执行：

```bash
cd /d/learn/f
docker compose up -d --build
```

启动后访问：

```text
http://localhost:8000/product-manage
```

## 查看服务状态

```bash
docker compose ps
```

正常情况下可以看到：

```text
mall-admin-frontend
mall-admin-goods-api
mall-admin-mysql
```

## 查看日志

查看全部服务日志：

```bash
docker compose logs -f
```

只看后端日志：

```bash
docker compose logs -f goods-api
```

只看 MySQL 日志：

```bash
docker compose logs -f mysql
```

## 关闭服务

关闭并删除容器，但保留 MySQL 数据：

```bash
docker compose down
```

## 清空数据库

关闭服务并删除 MySQL 数据卷：

```bash
docker compose down -v
```

下次重新启动时，后端会重新自动建表并插入初始化商品数据。

## 修改前端后重新发布

前端 Docker 镜像使用本地 `frontend/dist` 目录，所以修改前端代码后需要先重新构建前端：

```bash
cd /d/learn/f/frontend
pnpm build
cd ..
docker compose up -d --build
```

## 修改后端后重新发布

后端 Docker 镜像使用本地编译好的 Linux 二进制 `backend/server`，所以修改 Go 代码后需要先重新编译后端：

```bash
cd /d/learn/f/backend
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o server main.go
cd ..
docker compose up -d --build
```

如果在 PowerShell 中执行，使用：

```powershell
cd D:\learn\f\backend
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"
go build -o server main.go
cd ..
docker compose up -d --build
```

## 接口测试

```bash
curl "http://localhost:8080/api/goods/list?current=1&pageSize=10"
```

如果返回 `success: true`，说明后端和 MySQL 连接正常。
